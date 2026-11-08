package main

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type QueueLimits struct {
	Concurrency int `json:"concurrency"`
	CPU         int `json:"cpu"`
	RAMMB       int `json:"ramMB"`
}
type QueueEntry struct {
	ID      string     `json:"id"`
	Label   string     `json:"label"`
	State   string     `json:"state"`
	Held    bool       `json:"held"`
	Request RunRequest `json:"request"`
	CPU     int        `json:"cpu"`
	RAMMB   int        `json:"ramMB"`
	Job     *Job       `json:"job"`
	ETA     etaView    `json:"eta"`
	worker  *App
	cancel  context.CancelFunc
}
type JobQueue struct {
	BatchETA    string `json:"batchEta"`
	ExportError string `json:"exportError,omitempty"`
	mu          sync.Mutex
	Jobs        []*QueueEntry `json:"jobs"`
	Limits      QueueLimits   `json:"limits"`
	Paused      bool          `json:"paused"`
	Error       string        `json:"error,omitempty"`
	path        string
	app         *App
}

var palKeyword = regexp.MustCompile(`(?i)\bPAL(\d+)\b`)

func resources(req RunRequest) (cpu, ram int, err error) {
	seenStages := map[string]bool{}
	files := map[string]string{}
	for _, f := range req.Files {
		files[f.Name] = f.Content
	}
	for _, s := range req.Stages {
		key := strings.ToLower(s.Input)
		if seenStages[key] {
			return 0, 0, fmt.Errorf("중복 단계 INP: %s", s.Input)
		}
		seenStages[key] = true
		if s.Opt {
			name := strings.TrimSuffix(s.Input, filepath.Ext(s.Input)) + ".xyz"
			for _, f := range req.Files {
				if strings.EqualFold(f.Name, name) {
					return 0, 0, fmt.Errorf("최적화 산출물 이름이 등록 XYZ와 충돌합니다: %s", name)
				}
			}
		}
		inp := files[s.Input]
		if regexp.MustCompile(`(?im)^\s*(?:%\s*(?:base|moinp|compound)|\$new_job)|\bMOREAD\b`).MatchString(inp) {
			return 0, 0, fmt.Errorf("%s: Queue에서는 %%base, 외부 orbital 및 compound 입력을 지원하지 않습니다", s.Input)
		}
		if len(xyzHeader.FindAllStringSubmatch(inp, -1)) != 1 {
			return 0, 0, fmt.Errorf("%s: Queue는 * xyzfile 입력 한 개가 필요합니다", s.Input)
		}
		p := 1
		procs := nprocsRx.FindAllStringSubmatch(inp, -1)
		pals := palKeyword.FindAllStringSubmatch(inp, -1)
		if len(procs)+len(pals) > 1 {
			return 0, 0, fmt.Errorf("%s: 중복 PAL 설정을 정리하세요", s.Input)
		}
		if len(procs) == 1 {
			p, _ = strconv.Atoi(procs[0][1])
		}
		if len(pals) == 1 {
			p, _ = strconv.Atoi(pals[0][1])
		}
		mem := maxcorePattern.FindAllStringSubmatch(inp, -1)
		if len(mem) != 1 {
			return 0, 0, fmt.Errorf("%s: %%maxcore를 한 번 명시하세요 (코어당 MB)", s.Input)
		}
		m, _ := strconv.Atoi(mem[0][1])
		if p < 1 || p > 65536 || m < 1 || m > 10000000 {
			return 0, 0, fmt.Errorf("CPU/RAM 값이 올바르지 않습니다")
		}
		// ORCA maxcore is per core and is not a hard RSS cap. Reserve 25%% overhead.
		estimate := (int64(p)*int64(m)*5 + 3) / 4
		if estimate > 1000000000 {
			return 0, 0, fmt.Errorf("RAM 요청이 너무 큽니다")
		}
		if p > cpu {
			cpu = p
		}
		if int(estimate) > ram {
			ram = int(estimate)
		}
	}
	return
}
func validLimits(l QueueLimits) error {
	if l.Concurrency < 1 || l.Concurrency > 64 || l.CPU < 1 || l.CPU > runtime.NumCPU() || l.RAMMB < 128 || l.RAMMB > 1000000000 {
		return fmt.Errorf("동시 실행은 1–64개, CPU는 1–%d개, RAM은 128MB 이상으로 설정하세요", runtime.NumCPU())
	}
	return nil
}
func newJobQueue(a *App, path string) *JobQueue {
	q := &JobQueue{app: a, path: path, Jobs: []*QueueEntry{}, Limits: QueueLimits{1, runtime.NumCPU(), 8192}, Paused: true}
	b, err := os.ReadFile(path)
	if err == nil {
		if err = json.Unmarshal(b, q); err != nil {
			q.Error = "Queue 파일을 읽지 못했습니다. 원본을 보존했습니다: " + err.Error()
			q.Paused = true
			return q
		}
		q.Paused = true // No automatic calculation on application restart.
		for _, e := range q.Jobs {
			if e.State == "Running" {
				e.State = "Interrupted"
				if e.Job != nil {
					e.Job.State = "interrupted"
					e.Job.Message = "프로그램이 종료되어 중단됐습니다. 완료 단계는 보존됩니다."
				}
			}
		}
	} else if !os.IsNotExist(err) {
		q.Error = err.Error()
	}
	return q
}
func queueID() string {
	b := make([]byte, 12)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func (q *JobQueue) snapshotLocked() {
	defer func() { q.BatchETA = q.batchETA() }()
	for _, e := range q.Jobs {
		if e.worker != nil {
			e.worker.mu.Lock()
			j := *e.worker.job
			j.CompletedStages = append([]string(nil), j.CompletedStages...)
			e.worker.mu.Unlock()
			e.Job = &j
			e.ETA = q.app.currentETA(&j)
		}
	}
}
func (q *JobQueue) saveLocked() error {
	q.snapshotLocked()
	if q.Error != "" {
		return fmt.Errorf("Queue 저장 중단: %s", q.Error)
	}
	b, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(q.path), 0700); err == nil {
		err = os.WriteFile(q.path+".tmp", b, 0600)
	}
	if err == nil {
		err = os.Rename(q.path+".tmp", q.path)
	}
	if err != nil {
		q.Error = "Queue 저장 실패: " + err.Error()
		q.Paused = true
	}
	return err
}
func (q *JobQueue) activeLocked() bool {
	for _, e := range q.Jobs {
		if e.worker != nil || e.State == "Pending" {
			return true
		}
	}
	return false
}
func (a *App) queueBusy() bool {
	if a.queue == nil {
		return false
	}
	a.queue.mu.Lock()
	defer a.queue.mu.Unlock()
	return a.queue.activeLocked()
}
func (q *JobQueue) dispatchLocked() {
	if q.Paused || q.Error != "" {
		return
	}
	cpu, ram, n := 0, 0, 0
	for _, e := range q.Jobs {
		if e.worker != nil {
			cpu += e.CPU
			ram += e.RAMMB
			n++
		}
	}
	for _, e := range q.Jobs {
		if e.State != "Pending" || e.Held || n >= q.Limits.Concurrency || cpu+e.CPU > q.Limits.CPU || ram+e.RAMMB > q.Limits.RAMMB {
			continue
		}
		q.app.mu.Lock()
		cfg := q.app.cfg
		legacy := q.app.job != nil && q.app.job.State == "running"
		q.app.mu.Unlock()
		if legacy || cfgUpdating(q.app) {
			return
		}
		if err := q.prepareLocked(e, cfg); err != nil {
			e.State = "Failed"
			e.Job = &Job{State: "failed", Message: err.Error()}
			continue
		}
		ctx, cancel := context.WithCancel(context.Background())
		e.cancel = cancel
		worker := &App{history: q.app.history, job: e.Job}
		e.worker = worker
		e.State = "Running"
		e.Job.State = "running"
		e.Job.Started = time.Now().Format(time.RFC3339)
		e.Job.Finished = ""
		worker.onChange = func() { q.mu.Lock(); defer q.mu.Unlock(); _ = q.saveLocked() }
		n++
		cpu += e.CPU
		ram += e.RAMMB
		if err := q.saveLocked(); err != nil {
			cancel()
			e.worker = nil
			e.State = "Interrupted"
			return
		}
		go func(e *QueueEntry, w *App) {
			w.runJob(ctx, cfg.ORCAExe, e.Request, w.job)
			q.mu.Lock()
			defer q.mu.Unlock()
			q.snapshotLocked()
			switch e.Job.State {
			case "completed":
				e.State = "Completed"
			case "stopped", "interrupted":
				e.State = "Interrupted"
			default:
				e.State = "Failed"
			}
			e.worker = nil
			e.cancel = nil
			q.writeBatchLocked()
			_ = q.saveLocked()
			q.dispatchLocked()
		}(e, worker)
	}
	_ = q.saveLocked()
}
func (q *JobQueue) prepareLocked(e *QueueEntry, cfg Config) error {
	if cfg.ORCAExe == "" {
		return fmt.Errorf("ORCA 실행 파일 경로를 저장하세요")
	}
	fi, err := os.Stat(cfg.ORCAExe)
	if err != nil || fi.IsDir() {
		return fmt.Errorf("ORCA 실행 파일이 없습니다")
	}
	if e.Job != nil && e.Job.Folder != "" {
		return nil
	}
	if err = os.MkdirAll(cfg.WorkRoot, 0755); err != nil {
		return err
	}
	folder, err := os.MkdirTemp(cfg.WorkRoot, safeLabel(e.Label)+"_")
	if err != nil {
		return err
	}
	for _, f := range e.Request.Files {
		if err = os.WriteFile(filepath.Join(folder, f.Name), []byte(f.Content), 0600); err != nil {
			return err
		}
	}
	e.Job = &Job{ID: e.ID, Folder: folder, StageCount: len(e.Request.Stages)}
	atoms := xyzAtomCount(filepath.Join(folder, e.Request.Stages[0].Source))
	for _, s := range e.Request.Stages {
		e.Job.stageKeys = append(e.Job.stageKeys, signatureForStage(folder, s, atoms))
	}
	return nil
}
func (q *JobQueue) action(id, action string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if action == "pause" {
		q.Paused = true
		return q.saveLocked()
	}
	if action == "resume" {
		if cfgUpdating(q.app) {
			return fmt.Errorf("업데이트 적용 중입니다")
		}
		if err := q.saveLocked(); err != nil {
			return err
		}
		q.Paused = false
		q.dispatchLocked()
		return q.saveLocked()
	}
	if action == "stopAll" {
		q.Paused = true
		for _, e := range q.Jobs {
			if e.cancel != nil {
				e.cancel()
			}
			if e.State == "Pending" {
				e.Held = true
			}
		}
		return q.saveLocked()
	}
	for i, e := range q.Jobs {
		if e.ID != id {
			continue
		}
		switch action {
		case "cancel":
			if e.cancel != nil {
				e.cancel()
			} else if e.State == "Pending" {
				e.State = "Cancelled"
			} else {
				return fmt.Errorf("대기 또는 실행 작업만 취소할 수 있습니다")
			}
		case "hold":
			if e.State != "Pending" {
				return fmt.Errorf("대기 작업만 보류할 수 있습니다")
			}
			e.Held = true
		case "release":
			e.Held = false
		case "retry":
			if e.worker != nil || (e.State != "Failed" && e.State != "Interrupted" && e.State != "Cancelled") {
				return fmt.Errorf("실패·중단·취소 작업만 재시작할 수 있습니다")
			}
			if e.Job != nil {
				// Retry in a new folder; copy completed stages and all their artifacts. Never truncate old results.
				old := e.Job
				for _, f := range e.Request.Files {
					if !strings.EqualFold(filepath.Ext(f.Name), ".inp") {
						continue
					}
					b, err := os.ReadFile(filepath.Join(old.Folder, f.Name))
					if err != nil || string(b) != f.Content {
						return fmt.Errorf("원본 INP가 변경됐습니다. 수정된 계산은 새 Job으로 등록하세요: %s", f.Name)
					}
				}
				e.Job = nil
				q.app.mu.Lock()
				cfg := q.app.cfg
				q.app.mu.Unlock()
				if err := q.prepareLocked(e, cfg); err != nil {
					e.Job = old
					return err
				}
				entries, err := os.ReadDir(old.Folder)
				if err != nil {
					e.Job = old
					return err
				}
				for _, f := range entries {
					if f.IsDir() || f.Type()&os.ModeSymlink != 0 {
						continue
					}
					src := filepath.Join(old.Folder, f.Name())
					b, err := os.ReadFile(src)
					if err != nil {
						return err
					}
					if err = os.WriteFile(filepath.Join(e.Job.Folder, f.Name()), b, 0600); err != nil {
						return err
					}
				}
				e.Job.CompletedStages = append([]string(nil), old.CompletedStages...)
				for _, st := range e.Request.Stages {
					done := false
					for _, v := range old.CompletedStages {
						if v == st.Input {
							done = true
						}
					}
					if !done {
						stem := strings.TrimSuffix(st.Input, filepath.Ext(st.Input))
						exts := []string{".out"}
						if st.Opt {
							exts = append(exts, ".xyz")
						}
						for _, ext := range exts {
							if err := os.Remove(filepath.Join(e.Job.Folder, stem+ext)); err != nil && !os.IsNotExist(err) {
								return err
							}
						}
					}
				}
			}
			e.State = "Pending"
			e.Held = false
		case "up":
			if e.State != "Pending" {
				return fmt.Errorf("대기 작업만 이동할 수 있습니다")
			}
			if i > 0 {
				q.Jobs[i-1], q.Jobs[i] = q.Jobs[i], q.Jobs[i-1]
			}
		case "down":
			if e.State != "Pending" {
				return fmt.Errorf("대기 작업만 이동할 수 있습니다")
			}
			if i+1 < len(q.Jobs) {
				q.Jobs[i+1], q.Jobs[i] = q.Jobs[i], q.Jobs[i+1]
			}
		case "reveal":
			if e.Job == nil || e.Job.Folder == "" {
				return fmt.Errorf("결과 폴더가 없습니다")
			}
			revealFolder(e.Job.Folder)
		default:
			return fmt.Errorf("지원하지 않는 Queue 동작")
		}
		if err := q.saveLocked(); err != nil {
			return err
		}
		q.dispatchLocked()
		return nil
	}
	return fmt.Errorf("Job을 찾을 수 없습니다")
}
func (a *App) queueRoute(w http.ResponseWriter, r *http.Request) {
	q := a.queue
	if q == nil {
		fail(w, 503, "Queue를 사용할 수 없습니다")
		return
	}
	switch r.URL.Path {
	case "/api/queue/action":
		if r.Method != "POST" {
			fail(w, 405, "Method not allowed")
			return
		}
		var x struct {
			ID     string `json:"id"`
			Action string `json:"action"`
		}
		if json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&x) != nil {
			fail(w, 400, "잘못된 요청")
			return
		}
		if err := q.action(x.ID, x.Action); err != nil {
			fail(w, 400, err.Error())
			return
		}
		respond(w, 200, map[string]bool{"ok": true})
	case "/api/queue/report", "/api/queue/results":
		if r.Method != "GET" {
			fail(w, 405, "Method not allowed")
			return
		}
		q.mu.Lock()
		path := ""
		for _, e := range q.Jobs {
			if e.ID == r.URL.Query().Get("id") && e.Job != nil {
				name := "analysis.txt"
				if r.URL.Path == "/api/queue/results" {
					name = "results.json"
				}
				path = filepath.Join(e.Job.Folder, name)
			}
		}
		q.mu.Unlock()
		if path == "" {
			fail(w, 404, "보고서가 없습니다")
			return
		}
		b, err := os.ReadFile(path)
		if err != nil {
			fail(w, 404, "보고서가 없습니다")
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if r.URL.Path == "/api/queue/results" {
			w.Header().Set("Content-Type", "application/json")
		}
		w.Write(b)
	case "/api/queue/export":
		if r.Method != "GET" {
			fail(w, 405, "Method not allowed")
			return
		}
		q.mu.Lock()
		defer q.mu.Unlock()
		q.snapshotLocked()
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=batch_results.csv")
		q.exportLocked(w)
	case "/api/queue", "/api/queue/preflight":
		if r.Method == "GET" && r.URL.Path == "/api/queue" {
			q.mu.Lock()
			defer q.mu.Unlock()
			q.snapshotLocked()
			respond(w, 200, q)
			return
		}
		if r.Method != "POST" {
			fail(w, 405, "Method not allowed")
			return
		}
		var x struct {
			Jobs   []RunRequest `json:"jobs"`
			Limits QueueLimits  `json:"limits"`
		}
		if json.NewDecoder(io.LimitReader(r.Body, 64<<20)).Decode(&x) != nil {
			fail(w, 400, "입력을 읽지 못했습니다")
			return
		}
		if err := validLimits(x.Limits); err != nil {
			fail(w, 400, err.Error())
			return
		}
		if len(x.Jobs) == 0 || len(x.Jobs) > 100 {
			fail(w, 400, "Batch는 1–100개 분자여야 합니다")
			return
		}
		reports := []PreflightReport{}
		entries := []*QueueEntry{}
		valid := true
		for _, req := range x.Jobs {
			p := preflight(req)
			cpu, ram, err := resources(req)
			if err != nil {
				p.OK = false
				p.Errors = append(p.Errors, err.Error())
			}
			if cpu > x.Limits.CPU || ram > x.Limits.RAMMB {
				p.OK = false
				p.Errors = append(p.Errors, "작업 CPU/RAM 예약량이 전체 예산보다 큽니다 (RAM은 25% 여유 포함)")
			}
			if !p.OK {
				valid = false
			}
			reports = append(reports, p)
			entries = append(entries, &QueueEntry{ID: queueID(), Label: req.Label, State: "Pending", Request: req, CPU: cpu, RAMMB: ram})
		}
		if r.URL.Path == "/api/queue/preflight" {
			respond(w, 200, map[string]interface{}{"ok": valid, "reports": reports})
			return
		}
		if !valid {
			respond(w, 400, map[string]interface{}{"error": "전체 Batch Preflight를 통과해야 합니다", "reports": reports})
			return
		}
		q.mu.Lock()
		defer q.mu.Unlock()
		if q.Error != "" {
			fail(w, 500, q.Error)
			return
		}
		for _, e := range q.Jobs {
			if e.worker != nil {
				fail(w, 409, "실행 중에는 새 Batch를 등록하지 마세요")
				return
			}
		}
		if len(q.Jobs)+len(entries) > 1000 {
			fail(w, 400, "Queue 최대 1000개입니다")
			return
		}
		if cfgUpdating(q.app) {
			fail(w, 409, "업데이트 적용 중입니다")
			return
		}
		q.app.mu.Lock()
		legacy := q.app.job != nil && q.app.job.State == "running"
		q.app.mu.Unlock()
		if legacy {
			fail(w, 409, "단일 계산이 실행 중입니다")
			return
		}
		q.Limits = x.Limits
		q.Jobs = append(q.Jobs, entries...)
		q.Paused = false
		if err := q.saveLocked(); err != nil {
			fail(w, 500, err.Error())
			return
		}
		q.dispatchLocked()
		respond(w, 200, map[string]bool{"ok": true})
	default:
		http.NotFound(w, r)
	}
}

// Prefix spreadsheet formula characters in untrusted labels and paths.
func csvSafe(s string) string {
	if strings.ContainsAny(strings.TrimLeft(s, " \t\r\n")[:min(1, len(strings.TrimLeft(s, " \t\r\n")))], "=+-@") {
		return "'" + s
	}
	return s
}
func (q *JobQueue) exportLocked(out io.Writer) {
	c := csv.NewWriter(out)
	defer c.Flush()
	c.Write([]string{"Molecule", "Status", "Stage", "Method", "Charge", "Multiplicity", "Geometry", "Electronic_Eh", "Gibbs_Eh", "Spin_population", "Folder", "Message"})
	for _, e := range q.Jobs {
		for _, s := range e.Request.Stages {
			row := []string{csvSafe(e.Label), e.State, csvSafe(s.Name), "Unavailable", "Unavailable", "Unavailable", s.Source, "Not calculated", "Not calculated", "Unavailable", "", ""}
			if e.Job != nil {
				row[10] = csvSafe(e.Job.Folder)
				row[11] = csvSafe(e.Job.Message)
				inp, _ := os.ReadFile(filepath.Join(e.Job.Folder, s.Input))
				out, err := os.Open(filepath.Join(e.Job.Folder, strings.TrimSuffix(s.Input, filepath.Ext(s.Input))+".out"))
				if err == nil {
					a, err := ParseOrcaOutput(out, s.Input, string(inp))
					out.Close()
					if err == nil {
						row[3] = csvSafe(a.Method)
						row[4] = strconv.Itoa(a.Charge)
						row[5] = strconv.Itoa(a.Multiplicity)
						if a.Normal && a.SCFConverged {
							row[7] = "Unavailable"
							row[8] = "Unavailable"
							if a.Electronic != nil {
								row[7] = strconv.FormatFloat(*a.Electronic, 'g', 16, 64)
							}
							if a.Gibbs != nil {
								row[8] = strconv.FormatFloat(*a.Gibbs, 'g', 16, 64)
							}
						} else {
							row[7] = "Failed"
							row[8] = "Failed"
						}
						if len(a.Populations) > 0 {
							b, _ := json.Marshal(a.Populations)
							row[9] = string(b)
						}
					}
				}
			}
			c.Write(row)
		}
	}
}

func cfgUpdating(a *App) bool { a.mu.Lock(); defer a.mu.Unlock(); return a.updating }

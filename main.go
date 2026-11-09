package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode"
)

//go:embed assets/desktop.html assets/ORCA_Nova.ico assets/ORCA_Nova_Logo.png
var assets embed.FS

type Config struct {
	ORCAExe  string `json:"orcaExe"`
	WorkRoot string `json:"workRoot"`
}
type JobFile struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}
type Stage struct {
	Name   string `json:"name"`
	Input  string `json:"input"`
	Source string `json:"source"`
	Opt    bool   `json:"opt"`
}
type RunRequest struct {
	Label  string    `json:"label"`
	Files  []JobFile `json:"files"`
	Stages []Stage   `json:"stages"`
}
type Job struct {
	ID              string   `json:"id"`
	Folder          string   `json:"folder"`
	State           string   `json:"state"`
	StageIndex      int      `json:"stageIndex"`
	StageCount      int      `json:"stageCount"`
	StageName       string   `json:"stageName"`
	Started         string   `json:"started"`
	Finished        string   `json:"finished,omitempty"`
	Message         string   `json:"message"`
	OutFile         string   `json:"outFile,omitempty"`
	LastLog         string   `json:"lastLog,omitempty"`
	ReportFile      string   `json:"reportFile,omitempty"`
	CompletedStages []string `json:"completedStages,omitempty"`
	stageStarted    time.Time
	stageKey        string
	stageKeys       []string
}
type App struct {
	updating bool
	queue    *JobQueue
	onChange func()
	mu       sync.Mutex
	cfg      Config
	job      *Job
	cancel   context.CancelFunc
	process  *os.Process
	token    string
	updates  *updater
	cfgPath  string
	history  *durationHistory
}

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "ORCA Nova", "config.json")
}
func newApp() *App {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	a := &App{cfg: Config{WorkRoot: filepath.Join(home, "Documents", "ORCA Nova Jobs")}, cfgPath: configPath()}
	cfgBytes, cfgErr := os.ReadFile(a.cfgPath)
	if cfgErr != nil {
		// Retain ORCA executable and working-folder preferences from ORCA Studio.
		legacy := filepath.Join(filepath.Dir(filepath.Dir(a.cfgPath)), "ORCA Studio", "config.json")
		cfgBytes, cfgErr = os.ReadFile(legacy)
	}
	if cfgErr == nil {
		b := cfgBytes
		var cfg Config
		if json.Unmarshal(b, &cfg) == nil {
			if cfg.WorkRoot != "" {
				a.cfg.WorkRoot = cfg.WorkRoot
			}
			a.cfg.ORCAExe = cfg.ORCAExe
		}
	}
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	a.token = hex.EncodeToString(b)
	a.updates = newUpdater()
	a.history = newDurationHistory(a.cfgPath)
	a.queue = newJobQueue(a, filepath.Join(filepath.Dir(a.cfgPath), "job_queue.json"))
	return a
}
func (a *App) saveConfig() error {
	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(a.cfgPath), 0700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(a.cfgPath, b, 0600)
}
func respond(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, code int, msg string) {
	respond(w, code, map[string]string{"error": msg})
}
func (a *App) guard(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("session") != a.token {
			fail(w, 403, "Invalid local session")
			return
		}
		if r.Method == http.MethodPost && r.Header.Get("X-ORCA-Session") != a.token {
			fail(w, 403, "Invalid request token")
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next(w, r)
	}
}
func (a *App) route(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/queue") {
		a.queueRoute(w, r)
		return
	}
	switch r.URL.Path {
	case "/":
		if r.Method != http.MethodGet {
			fail(w, 405, "Method not allowed")
			return
		}
		b, _ := assets.ReadFile("assets/desktop.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self' 'unsafe-inline' data: https:; connect-src 'self'; img-src 'self' data:; frame-ancestors 'none'")
		w.Write(b)
	case "/icon.ico":
		b, _ := assets.ReadFile("assets/ORCA_Nova.ico")
		w.Header().Set("Content-Type", "image/x-icon")
		w.Write(b)
	case "/logo.png":
		b, _ := assets.ReadFile("assets/ORCA_Nova_Logo.png")
		w.Header().Set("Content-Type", "image/png")
		w.Write(b)
	case "/api/status":
		if r.Method != http.MethodGet {
			fail(w, 405, "Method not allowed")
			return
		}
		a.mu.Lock()
		cfg := a.cfg
		var job *Job
		if a.job != nil {
			cp := *a.job
			job = &cp
		}
		a.mu.Unlock()
		if job != nil && job.OutFile != "" {
			job.LastLog = tail(job.OutFile, 19000)
		}
		var eta *etaView
		if job != nil {
			v := a.currentETA(job)
			eta = &v
		}
		respond(w, 200, map[string]interface{}{"config": cfg, "job": job, "eta": eta, "platform": runtime.GOOS, "version": appVersion, "update": a.updates.snapshot()})
	case "/api/update/check":
		if r.Method != http.MethodPost {
			fail(w, 405, "Method not allowed")
			return
		}
		respond(w, 200, a.updates.check())
	case "/api/update/install":
		if a.queue != nil {
			a.queue.mu.Lock()
			defer a.queue.mu.Unlock()
			if a.queue.activeLocked() {
				fail(w, 409, "Queue에 실행·대기 작업이 있습니다. 먼저 중단하세요.")
				return
			}
		}
		if r.Method != http.MethodPost {
			fail(w, 405, "Method not allowed")
			return
		}
		a.mu.Lock()
		running := a.job != nil && a.job.State == "running"
		a.mu.Unlock()
		if running {
			fail(w, 409, "계산 중에는 업데이트할 수 없어요. 먼저 계산을 완료하거나 중단하세요.")
			return
		}
		a.mu.Lock()
		if a.updating {
			a.mu.Unlock()
			fail(w, 409, "업데이트 적용 중입니다")
			return
		}
		a.updating = true
		a.mu.Unlock()
		folder, err := a.updates.prepare()
		if err != nil {
			a.mu.Lock()
			a.updating = false
			a.mu.Unlock()
			fail(w, 500, "업데이트 준비 실패: "+err.Error())
			return
		}
		respond(w, 200, map[string]interface{}{"ok": true, "folder": folder, "message": "다운로드·검증 완료. 앱을 닫으면 EXE가 교체됩니다."})
		go func() { time.Sleep(500 * time.Millisecond); os.Exit(0) }()
	case "/api/config":
		if r.Method != http.MethodPost {
			fail(w, 405, "Method not allowed")
			return
		}
		var cfg Config
		if json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&cfg) != nil {
			fail(w, 400, "Invalid settings")
			return
		}
		cfg.ORCAExe = strings.Trim(cfg.ORCAExe, " \t\"\r\n")
		cfg.WorkRoot = strings.Trim(cfg.WorkRoot, " \t\"\r\n")
		if cfg.WorkRoot == "" || !filepath.IsAbs(cfg.WorkRoot) {
			fail(w, 400, "작업 폴더는 절대 경로로 지정하세요.")
			return
		}
		if cfg.ORCAExe != "" {
			if !filepath.IsAbs(cfg.ORCAExe) || strings.ToLower(filepath.Base(cfg.ORCAExe)) != "orca.exe" && runtime.GOOS == "windows" {
				fail(w, 400, "orca.exe 파일의 전체 경로를 입력하세요.")
				return
			}
		}
		a.mu.Lock()
		a.cfg = cfg
		a.mu.Unlock()
		if err := a.saveConfig(); err != nil {
			fail(w, 500, err.Error())
			return
		}
		respond(w, 200, map[string]bool{"ok": true})
	case "/api/preflight":
		if r.Method != http.MethodPost {
			fail(w, 405, "Method not allowed")
			return
		}
		var req RunRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 12<<20)).Decode(&req); err != nil {
			fail(w, 400, "입력 데이터 해석 실패: "+err.Error())
			return
		}
		report := preflight(req)
		respond(w, 200, report)
	case "/api/run":
		if r.Method != http.MethodPost {
			fail(w, 405, "Method not allowed")
			return
		}
		a.start(w, r)
	case "/api/stop":
		if r.Method != http.MethodPost {
			fail(w, 405, "Method not allowed")
			return
		}
		a.mu.Lock()
		can := a.cancel
		p := a.process
		a.mu.Unlock()
		if can != nil {
			can()
		}
		if p != nil {
			killTree(p.Pid)
		}
		respond(w, 200, map[string]bool{"ok": true})
	case "/api/report":
		if r.Method != http.MethodGet {
			fail(w, 405, "Method not allowed")
			return
		}
		a.mu.Lock()
		var report string
		if a.job != nil {
			report = a.job.ReportFile
		}
		a.mu.Unlock()
		if report == "" {
			fail(w, 404, "아직 분석 보고서가 없어요.")
			return
		}
		content, err := os.ReadFile(report)
		if err != nil {
			fail(w, 404, "보고서를 찾을 수 없어요.")
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=ORCA_Nova_Analysis.txt")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(content)
	case "/api/analyze":
		if r.Method != http.MethodPost {
			fail(w, 405, "Method not allowed")
			return
		}
		var payload struct {
			Name   string `json:"name"`
			Output string `json:"output"`
			Input  string `json:"input"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 64<<20)).Decode(&payload); err != nil {
			fail(w, 400, "OUT 데이터를 읽지 못했어요: "+err.Error())
			return
		}
		if len(payload.Output) == 0 || len(payload.Output) > 60<<20 {
			fail(w, 400, "OUT 파일은 60MB 이하여야 합니다.")
			return
		}
		name := filepath.Base(payload.Name)
		if !strings.EqualFold(filepath.Ext(name), ".out") {
			fail(w, 400, ".out 파일을 선택해 주세요.")
			return
		}
		analyzed, err := ParseOrcaOutput(strings.NewReader(payload.Output), name, payload.Input)
		if err != nil {
			fail(w, 400, err.Error())
			return
		}
		report := formatAnalysis(analyzed, Stage{Name: "가져온 OUT 파일", Input: name})
		respond(w, 200, map[string]string{"report": report})
	case "/api/reveal":
		if r.Method != http.MethodPost {
			fail(w, 405, "Method not allowed")
			return
		}
		a.mu.Lock()
		j := a.job
		a.mu.Unlock()
		if j == nil {
			fail(w, 400, "실행한 작업이 없어요.")
			return
		}
		revealFolder(j.Folder)
		respond(w, 200, map[string]bool{"ok": true})
	case "/api/quit":
		if r.Method != http.MethodPost {
			fail(w, 405, "Method not allowed")
			return
		}
		a.mu.Lock()
		running := a.job != nil && a.job.State == "running"
		a.mu.Unlock()
		if running {
			fail(w, 409, "계산이 실행 중입니다. 먼저 중단하세요.")
			return
		}
		if a.queue != nil {
			if err := a.queue.pauseForQuit(); err != nil {
				fail(w, 409, "Queue 종료 보류: "+err.Error())
				return
			}
		}
		respond(w, 200, map[string]bool{"ok": true})
		go func() { time.Sleep(200 * time.Millisecond); os.Exit(0) }()
	default:
		http.NotFound(w, r)
	}
}
func safeName(s string) bool {
	if len(s) < 1 || len(s) > 120 || s == "." || s == ".." || strings.Contains(s, "..") || filepath.Base(s) != s {
		return false
	}
	for _, c := range s {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}
func safeLabel(s string) string {
	var b strings.Builder
	for _, c := range s {
		if unicode.IsLetter(c) || unicode.IsDigit(c) || c == '-' || c == '_' {
			b.WriteRune(c)
		} else if c == ' ' {
			b.WriteRune('_')
		}
	}
	ret := b.String()
	if len(ret) > 42 {
		ret = ret[:42]
	}
	if ret == "" {
		ret = "molecule"
	}
	return ret
}
func (a *App) start(w http.ResponseWriter, r *http.Request) {
	if a.queue != nil {
		a.queue.mu.Lock()
		defer a.queue.mu.Unlock()
		if a.queue.activeLocked() {
			fail(w, 409, "Queue 작업을 먼저 완료하거나 취소하세요.")
			return
		}
	}
	var req RunRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 12<<20)).Decode(&req); err != nil {
		fail(w, 400, "입력 데이터를 읽지 못했어요: "+err.Error())
		return
	}
	if report := preflight(req); !report.OK {
		fail(w, 400, "계산 전 검사 실패: "+strings.Join(report.Errors, " / "))
		return
	}
	if len(req.Files) == 0 || len(req.Files) > 80 || len(req.Stages) == 0 || len(req.Stages) > 30 {
		fail(w, 400, "실행 파일 또는 계산 단계 개수가 올바르지 않아요.")
		return
	}
	seen := map[string]bool{}
	for _, f := range req.Files {
		ext := strings.ToLower(filepath.Ext(f.Name))
		if !safeName(f.Name) || !(ext == ".inp" || ext == ".xyz") {
			fail(w, 400, "파일명이 안전하지 않아요: "+f.Name)
			return
		}
		if seen[strings.ToLower(f.Name)] || len(f.Content) > 4<<20 {
			fail(w, 400, "중복 파일 또는 파일 크기 초과: "+f.Name)
			return
		}
		seen[strings.ToLower(f.Name)] = true
	}
	for _, s := range req.Stages {
		if !safeName(s.Input) || !strings.HasSuffix(strings.ToLower(s.Input), ".inp") || !seen[strings.ToLower(s.Input)] {
			fail(w, 400, "INP 파일이 누락됐어요: "+s.Input)
			return
		}
		if !safeName(s.Source) || !strings.HasSuffix(strings.ToLower(s.Source), ".xyz") {
			fail(w, 400, "XYZ 참조가 잘못됐어요: "+s.Source)
			return
		}
	}
	a.mu.Lock()
	if a.job != nil && a.job.State == "running" {
		a.mu.Unlock()
		fail(w, 409, "이미 ORCA가 실행 중이에요.")
		return
	}
	cfg := a.cfg
	updating := a.updating
	a.mu.Unlock()
	if updating {
		fail(w, 409, "업데이트 적용 중입니다")
		return
	}
	if cfg.ORCAExe == "" {
		fail(w, 400, "먼저 ORCA 실행 파일 경로를 저장하세요.")
		return
	}
	info, err := os.Stat(cfg.ORCAExe)
	if err != nil || info.IsDir() {
		fail(w, 400, "orca.exe를 찾을 수 없어요. 설치 경로를 확인하세요.")
		return
	}
	if err := os.MkdirAll(cfg.WorkRoot, 0755); err != nil {
		fail(w, 500, "작업 폴더를 만들 수 없어요: "+err.Error())
		return
	}
	folder := filepath.Join(cfg.WorkRoot, time.Now().Format("20060102_150405")+"_"+safeLabel(req.Label))
	for i := 1; i < 100; i++ {
		if _, err := os.Stat(folder); errors.Is(err, os.ErrNotExist) {
			break
		}
		folder = filepath.Join(cfg.WorkRoot, time.Now().Format("20060102_150405")+"_"+safeLabel(req.Label)+fmt.Sprintf("_%02d", i))
	}
	if err := os.Mkdir(folder, 0755); err != nil {
		fail(w, 500, err.Error())
		return
	}
	for _, f := range req.Files {
		if err := os.WriteFile(filepath.Join(folder, f.Name), []byte(f.Content), 0644); err != nil {
			fail(w, 500, "파일 저장 실패: "+err.Error())
			return
		}
	}
	// The first stage must have input geometry. Later stages may use the XYZ from the earlier optimization.
	if _, err := os.Stat(filepath.Join(folder, req.Stages[0].Source)); err != nil {
		fail(w, 400, "첫 단계 XYZ가 없어요: "+req.Stages[0].Source+" (폴더: "+folder+")")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	job := &Job{ID: time.Now().Format("20060102_150405"), Folder: folder, State: "running", StageCount: len(req.Stages), StageName: req.Stages[0].Name, Started: time.Now().Format(time.RFC3339), Message: "계산 준비 중"}
	initialAtoms := xyzAtomCount(filepath.Join(folder, req.Stages[0].Source))
	for _, stage := range req.Stages {
		job.stageKeys = append(job.stageKeys, signatureForStage(folder, stage, initialAtoms))
	}
	a.mu.Lock()
	a.job = job
	a.cancel = cancel
	a.process = nil
	a.mu.Unlock()
	respond(w, 200, map[string]interface{}{"ok": true, "folder": folder})
	go a.runJob(ctx, cfg.ORCAExe, req, job)
}
func (a *App) updateJob(j *Job, fn func(*Job)) {
	a.mu.Lock()
	fn(j)
	a.mu.Unlock()
	if a.onChange != nil {
		a.onChange()
	}
}
func tail(path string, limit int) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return ""
	}
	n := fi.Size()
	off := n - int64(limit)
	if off < 0 {
		off = 0
	}
	f.Seek(off, io.SeekStart)
	b, _ := io.ReadAll(io.LimitReader(f, int64(limit)))
	return string(b)
}
func (a *App) failure(j *Job, msg string, outputPath string) {
	analysis := "ORCA Nova failure report\n" + time.Now().Format(time.RFC3339) + "\n" + msg + "\n\nLast output:\n" + tail(outputPath, 14000) + "\n"
	_ = os.WriteFile(filepath.Join(j.Folder, "FAILURE_ANALYSIS.txt"), []byte(analysis), 0644)
	a.updateJob(j, func(j *Job) { j.State = "failed"; j.Message = msg; j.Finished = time.Now().Format(time.RFC3339) })
}
func (a *App) runJob(ctx context.Context, orcaExe string, req RunRequest, j *Job) {
	defer func() {
		path, err := writeAnalysisReports(j.Folder, req)
		if err == nil {
			a.updateJob(j, func(j *Job) { j.ReportFile = path })
		} else {
			a.updateJob(j, func(j *Job) { j.Message += " (보고서 저장 실패: " + err.Error() + ")" })
		}
	}()
	defer func() { a.mu.Lock(); a.process = nil; a.cancel = nil; a.mu.Unlock() }()
	for i, stage := range req.Stages {
		done := false
		for _, name := range j.CompletedStages {
			if name == stage.Input {
				done = true
			}
		}
		if done {
			if err := validateStage(j.Folder, stage); err != nil {
				a.failure(j, "완료 단계 재검증 실패: "+err.Error(), "")
				return
			}
			continue
		}
		if ctx.Err() != nil {
			a.updateJob(j, func(j *Job) {
				j.State = "interrupted"
				j.Message = "사용자 중단"
				j.Finished = time.Now().Format(time.RFC3339)
			})
			return
		}
		outName := strings.TrimSuffix(stage.Input, filepath.Ext(stage.Input)) + ".out"
		outPath := filepath.Join(j.Folder, outName)
		a.updateJob(j, func(j *Job) {
			j.StageIndex = i
			j.StageName = stage.Name
			j.OutFile = outPath
			j.stageStarted = time.Now()
			if i < len(j.stageKeys) {
				j.stageKey = j.stageKeys[i]
			}
			j.Message = fmt.Sprintf("%d/%d 단계 실행 중", i+1, len(req.Stages))
		})
		if _, err := os.Stat(filepath.Join(j.Folder, stage.Source)); err != nil {
			a.failure(j, "필요한 XYZ 파일이 없어요: "+stage.Source, outPath)
			return
		}
		out, err := os.Create(outPath)
		if err != nil {
			a.failure(j, "OUT 파일 생성 실패: "+err.Error(), outPath)
			return
		}
		cmd := exec.CommandContext(ctx, orcaExe, stage.Input)
		cmd.Dir = j.Folder
		configureProcess(cmd)
		cmd.Cancel = func() error { killTree(cmd.Process.Pid); return cmd.Process.Kill() }
		cmd.WaitDelay = 5 * time.Second
		cmd.Stdout = out
		cmd.Stderr = out
		if err := cmd.Start(); err != nil {
			out.Close()
			if ctx.Err() != nil {
				a.updateJob(j, func(j *Job) {
					j.State = "interrupted"
					j.Message = "사용자 중단"
					j.Finished = time.Now().Format(time.RFC3339)
				})
				return
			}
			a.failure(j, "ORCA 실행 시작 실패: "+err.Error(), outPath)
			return
		}
		a.mu.Lock()
		a.process = cmd.Process
		a.mu.Unlock()
		err = cmd.Wait()
		out.Close()
		a.mu.Lock()
		a.process = nil
		a.mu.Unlock()
		if ctx.Err() != nil {
			a.updateJob(j, func(j *Job) {
				j.State = "interrupted"
				j.Message = "사용자 중단"
				j.Finished = time.Now().Format(time.RFC3339)
			})
			return
		}
		if err != nil {
			a.failure(j, "ORCA 실행 실패 (종료 코드 확인): "+err.Error(), outPath)
			return
		}
		normal, optimized, err := checkOut(outPath)
		if err != nil || !normal {
			a.failure(j, "ORCA 정상 종료 문구가 없어요. OUT을 확인하세요.", outPath)
			return
		}
		if stage.Opt && !optimized {
			a.failure(j, "Geometry Optimization 수렴 문구가 없어요.", outPath)
			return
		}
		if err := validateStage(j.Folder, stage); err != nil {
			a.failure(j, err.Error(), outPath)
			return
		}
		a.updateJob(j, func(j *Job) { j.CompletedStages = append(j.CompletedStages, stage.Input) })
		if i < len(j.stageKeys) {
			a.history.observe(j.stageKeys[i], time.Since(j.stageStarted).Seconds())
		}
		if path, err := writeAnalysisReports(j.Folder, req); err == nil {
			a.updateJob(j, func(j *Job) { j.ReportFile = path })
		}
	}
	a.updateJob(j, func(j *Job) {
		j.State = "completed"
		j.Message = "모든 계산 단계가 정상 종료됐어요. 상태·Frequency·Spin을 별도로 검토하세요."
		j.Finished = time.Now().Format(time.RFC3339)
	})
}
func checkOut(path string) (bool, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, false, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4096), 10*1024*1024)
	normal, opt := false, false
	for scanner.Scan() {
		s := scanner.Text()
		if strings.Contains(s, "ORCA TERMINATED NORMALLY") {
			normal = true
		}
		if strings.Contains(s, "THE OPTIMIZATION HAS CONVERGED") {
			opt = true
		}
	}
	return normal, opt, scanner.Err()
}

func revealFolder(path string) {
	if runtime.GOOS == "windows" {
		exec.Command("explorer.exe", path).Start()
	} else {
		exec.Command("xdg-open", path).Start()
	}
}
func launch(url string) error {
	if runtime.GOOS == "windows" {
		if self, err := os.Executable(); err == nil {
			host := filepath.Join(filepath.Dir(self), "ORCA_Nova_UI.exe")
			if info, err := os.Stat(host); err == nil && !info.IsDir() {
				if err := exec.Command(host, url).Start(); err == nil {
					return nil
				}
			}
		}
		options := []string{}
		for _, root := range []string{os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramFiles"), os.Getenv("LOCALAPPDATA")} {
			if root != "" {
				options = append(options, filepath.Join(root, "Microsoft", "Edge", "Application", "msedge.exe"))
			}
		}
		for _, exe := range options {
			if _, err := os.Stat(exe); err == nil {
				cmd := exec.Command(exe, "--app="+url, "--window-size=1450,950", "--no-first-run", "--no-default-browser-check", "--user-data-dir="+filepath.Join(os.Getenv("LOCALAPPDATA"), "ORCA Nova", "edge-profile"))
				if err := cmd.Start(); err != nil {
					return err
				}
				go patchWindowsAppIcon()
				return nil
			}
		}
		return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url).Start()
	}
	if runtime.GOOS == "darwin" {
		return exec.Command("open", url).Start()
	}
	return exec.Command("xdg-open", url).Start()
}
func main() {
	a := newApp()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.guard(a.route))
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, err)
		}
	}()
	url := fmt.Sprintf("http://%s/?session=%s", listener.Addr().String(), a.token)
	if os.Getenv("ORCA_NOVA_TEST_MODE") == "1" {
		fmt.Println("TEST_URL=" + url)
	} else if err := launch(url); err != nil {
		fmt.Fprintln(os.Stderr, "UI launch:", err, "Open URL:", url)
	}
	go func() {
		time.Sleep(3 * time.Second)
		a.updates.check()
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			a.updates.check()
		}
	}()
	select {}
}

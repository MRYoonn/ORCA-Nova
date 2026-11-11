package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The test executable acts as a cross-platform ORCA substitute in child processes.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "mock-tree-leaf" {
		for {
			if err := os.WriteFile(filepath.Join(os.Getenv("ORCA_NOVA_TREE_DIR"), "heartbeat"), []byte(time.Now().String()), 0600); err != nil {
				os.Exit(2)
			}
			time.Sleep(30 * time.Millisecond)
		}
	}
	if len(os.Args) > 1 && os.Args[1] == "mock-tree-parent" {
		exe, _ := os.Executable()
		child := exec.Command(exe, "mock-tree-leaf")
		if err := child.Start(); err != nil {
			os.Exit(3)
		}
		_ = os.WriteFile(filepath.Join(os.Getenv("ORCA_NOVA_TREE_DIR"), "leaf.pid"), []byte(strconv.Itoa(child.Process.Pid)), 0600)
		time.Sleep(20 * time.Second)
		_ = child.Process.Kill()
		os.Exit(0)
	}
	if os.Getenv("ORCA_NOVA_MOCK") == "1" {
		inp, _ := os.ReadFile(os.Args[1])
		s := string(inp)
		_ = os.WriteFile("started", []byte("yes"), 0600)
		time.Sleep(150 * time.Millisecond)
		if strings.Contains(s, "MOCK_FAIL") {
			if _, err := os.Stat(filepath.Join("..", "allow_retry")); err != nil {
				fmt.Println("SCF NOT CONVERGED")
				os.Exit(0)
			}
		}
		if strings.Contains(s, "MOCK_BLOCK") {
			// Deterministic integration fixture: keep the child ORCA process
			// alive until the test explicitly allows the retried job to finish.
			deadline := time.Now().Add(8 * time.Second)
			for {
				if _, err := os.Stat(filepath.Join("..", "allow_continue")); err == nil {
					break
				}
				if time.Now().After(deadline) {
					fmt.Println("MOCK BLOCK TIMEOUT")
					os.Exit(34)
				}
				time.Sleep(25 * time.Millisecond)
			}
		}
		if strings.Contains(s, "MOCK_SLOW") {
			time.Sleep(10 * time.Second)
		}
		if !strings.Contains(s, "MOCK_NO_SCF") {
			fmt.Println("SCF CONVERGED AFTER 10 CYCLES")
		}
		if strings.Contains(s, "Opt") {
			fmt.Println("THE OPTIMIZATION HAS CONVERGED")
			if !strings.Contains(s, "MOCK_NO_XYZ") {
				h := xyzHeader.FindStringSubmatch(s)
				b, _ := os.ReadFile(h[3])
				_ = os.WriteFile(strings.TrimSuffix(os.Args[1], ".inp")+".xyz", b, 0600)
			}
		}
        if os.Getenv("ORCA_NOVA_TRIPLET_MOCK")=="1" {
            stem:=strings.TrimSuffix(os.Args[1], ".inp")
            if strings.Contains(stem,"step2_triplet"){
                _=os.WriteFile("step2_triplet.gbw",[]byte("mock GBW, not a real ORCA wavefunction"),0600)
                if strings.Contains(s," Freq "){
                    fmt.Println("VIBRATIONAL FREQUENCIES\n  0:   123.4500 cm**-1")
                }
            }
            if strings.Contains(stem,"step3_spin"){
                if os.Getenv("ORCA_NOVA_TRIPLET_NO_CUBE")!="1"{
                    _=os.WriteFile("step3_spin.cube",[]byte("synthetic cube for mock testing only"),0600)
                }
                fmt.Println("Expectation value of <S**2> : 2.019")
                fmt.Println("ORBITAL ENERGIES\nSPIN UP ORBITALS\n NO OCC E(Eh) E(eV)\n  0 1.0000 -0.8000 -21.77\n  1 1.0000 -0.3500 -9.52\n  2 1.0000 -0.2000 -5.44\nSPIN DOWN ORBITALS\n NO OCC E(Eh) E(eV)\n  0 1.0000 -0.7500 -20.40\n  1 0.0000 -0.1100 -2.99\n  2 0.0000 0.1200 3.26")
                xyzMatch:=xyzHeader.FindStringSubmatch(s)
                if len(xyzMatch)>3{
                    raw,_:=os.ReadFile(xyzMatch[3])
                    symbols,_,parseErr:=xyzGeometry(string(raw))
                    if parseErr==nil{
                        for _,scheme:=range []string{"MULLIKEN ATOMIC CHARGES AND SPIN POPULATIONS","LOEWDIN ATOMIC CHARGES AND SPIN POPULATIONS","HIRSHFELD ANALYSIS"}{
                            fmt.Println(scheme)
                            for i,e:=range symbols{
                                if scheme=="HIRSHFELD ANALYSIS"{fmt.Printf("%d %s 0.000000 %.6f\n",i,e,float64(i)*0.007)} else{fmt.Printf("%d %s : 0.000000 %.6f\n",i,e,float64(i)*0.009)}
                            }
                            fmt.Println()
                        }
                    }
                }
            }
        }
		fmt.Println("FINAL SINGLE POINT ENERGY -75.1234\nORCA TERMINATED NORMALLY")
		os.Exit(0)
	}
	os.Exit(m.Run())
}
func mockQueue(t *testing.T) *JobQueue {
	t.Helper()
	t.Setenv("ORCA_NOVA_MOCK", "1")
	dir := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	a := &App{cfg: Config{ORCAExe: exe, WorkRoot: filepath.Join(dir, "jobs")}, cfgPath: filepath.Join(dir, "config.json"), history: newDurationHistory(filepath.Join(dir, "config.json")), updates: newUpdater()}
	a.queue = newJobQueue(a, filepath.Join(dir, "queue.json"))
	return a.queue
}
func smallReq(label string) RunRequest {
	r := testReq(0, 1)
	r.Label = label
	r.Files[1].Content = strings.ReplaceAll(strings.ReplaceAll(r.Files[1].Content, "nprocs 16", "nprocs 1"), "maxcore 3000", "maxcore 100")
	r.Files = append(r.Files, JobFile{Name: "triplet.inp", Content: "! PBE0 def2-SVP Opt\n%pal nprocs 1 end\n%maxcore 100\n* xyzfile 0 3 water_opt.xyz\n"})
	r.Stages = append(r.Stages, Stage{Name: "T1 Opt + Spin", Input: "triplet.inp", Source: "water_opt.xyz", Opt: true})
	return r
}
func addEntry(t *testing.T, q *JobQueue, r RunRequest) *QueueEntry {
	t.Helper()
	cpu, ram, err := resources(r)
	if err != nil {
		t.Fatal(err)
	}
	e := &QueueEntry{ID: queueID(), Label: r.Label, Request: r, State: "Pending", CPU: cpu, RAMMB: ram}
	q.Jobs = append(q.Jobs, e)
	return e
}
func waitQueue(t *testing.T, q *JobQueue, condition func([]*QueueEntry) bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		q.mu.Lock()
		q.snapshotLocked()
		ok := condition(q.Jobs)
		q.mu.Unlock()
		if ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("Queue did not reach expected state")
}
func TestQueueFourMoleculesFailureRetryIsolation(t *testing.T) {
	if runtime.NumCPU() < 2 {
		t.Skip("parallel scenario needs 2 CPU")
	}
	q := mockQueue(t)
	q.Limits = QueueLimits{2, 2, 1024}
	for _, label := range []string{"A", "B", "C", "D"} {
		r := smallReq(label)
		if label == "B" {
			r.Files[2].Content += "# MOCK_FAIL\n"
		}
		addEntry(t, q, r)
	}
	if err := q.action("", "resume"); err != nil {
		t.Fatal(err)
	}
	waitQueue(t, q, func(es []*QueueEntry) bool {
		return es[0].State == "Running" && es[1].State == "Running" && es[2].State == "Pending"
	})
	waitQueue(t, q, func(es []*QueueEntry) bool {
		return es[0].State == "Completed" && es[1].State == "Failed" && es[2].State == "Completed" && es[3].State == "Completed"
	})
	q.mu.Lock()
	b := q.Jobs[1]
	old := b.Job.Folder
	if len(b.Job.CompletedStages) != 1 {
		t.Fatal("completed stage lost")
	}
	for i, e := range q.Jobs {
		if len(e.Job.CompletedStages) == 2 && i == 1 {
			t.Fatal("failed stage marked completed")
		}
		for j, f := range q.Jobs {
			if i != j && e.Job.Folder == f.Job.Folder {
				t.Fatal("shared result folders")
			}
		}
	}
	q.mu.Unlock()
	if err := os.WriteFile(filepath.Join(q.app.cfg.WorkRoot, "allow_retry"), []byte("yes"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := q.action(b.ID, "retry"); err != nil {
		t.Fatal(err)
	}
	waitQueue(t, q, func(es []*QueueEntry) bool { return es[1].State == "Completed" })
	q.mu.Lock()
	defer q.mu.Unlock()
	if b.Job.Folder == old {
		t.Fatal("retry overwrote original folder")
	}
	if _, err := os.Stat(filepath.Join(old, "FAILURE_ANALYSIS.txt")); err != nil {
		t.Fatal(err)
	}
	var csv bytes.Buffer
	q.exportLocked(&csv)
	if !strings.Contains(csv.String(), "-75.1234") {
		t.Fatal(csv.String())
	}
	for _, e := range q.Jobs {
		for _, name := range []string{"analysis.txt", "results.json"} {
			if _, err := os.Stat(filepath.Join(e.Job.Folder, name)); err != nil {
				t.Fatal(err)
			}
		}
	}
}
func TestQueueBudgetHoldCancelAndRecovery(t *testing.T) {
	q := mockQueue(t)
	q.Limits = QueueLimits{3, 1, 200}
	r := smallReq("A")
	r.Files[1].Content += "# MOCK_SLOW\n"
	a := addEntry(t, q, r)
	b := addEntry(t, q, smallReq("B"))
	if err := q.action(b.ID, "hold"); err != nil {
		t.Fatal(err)
	}
	_ = q.action("", "resume")
	waitQueue(t, q, func(es []*QueueEntry) bool { return es[0].State == "Running" })
	if err := q.action(a.ID, "cancel"); err != nil {
		t.Fatal(err)
	}
	waitQueue(t, q, func(es []*QueueEntry) bool { return es[0].State == "Interrupted" && es[1].State == "Pending" })
	if err := q.action(b.ID, "release"); err != nil {
		t.Fatal(err)
	}
	waitQueue(t, q, func(es []*QueueEntry) bool { return es[1].State == "Completed" })
	q.mu.Lock()
	b.State = "Running"
	if err := q.saveLocked(); err != nil {
		t.Fatal(err)
	}
	q.mu.Unlock()
	recovered := newJobQueue(q.app, q.path)
	if recovered.Jobs[1].State != "Interrupted" || !recovered.Paused || len(recovered.Jobs[1].Job.CompletedStages) != 2 {
		t.Fatal("restart failed to preserve completed stages or interrupt running job")
	}
}
func TestQueueRejectsInvalidBatchAndUpdateConflict(t *testing.T) {
	q := mockQueue(t)
	a := q.app
	a.token = "token"
	request := func(path string, payload any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(payload)
		r := httptest.NewRequest("POST", path+"?session=token", bytes.NewReader(b))
		r.Header.Set("X-ORCA-Session", "token")
		w := httptest.NewRecorder()
		a.guard(a.route)(w, r)
		return w
	}
	r := smallReq("valid")
	bad := smallReq("invalid")
	bad.Files[0].Content = "broken"
	w := request("/api/queue", map[string]any{"jobs": []RunRequest{r, bad}, "limits": QueueLimits{1, 1, 1024}})
	if w.Code != 400 || len(q.Jobs) != 0 {
		t.Fatal("non-atomic batch validation", w.Body.String())
	}
	e := addEntry(t, q, r)
	w = request("/api/update/install", map[string]any{})
	if w.Code != 409 {
		t.Fatal("updater allowed pending queue", w.Code, w.Body.String())
	}
	_ = q.action(e.ID, "cancel")
	w = request("/api/queue/action", map[string]any{"action": "nonsense", "id": e.ID})
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
	w = request("/api/queue", map[string]any{"jobs": []RunRequest{r}, "limits": QueueLimits{1, 1, 100}})
	if w.Code != 400 {
		t.Fatal("overbudget accepted")
	}
	noSession := httptest.NewRecorder()
	a.guard(a.route)(noSession, httptest.NewRequest(http.MethodGet, "/api/queue", nil))
	if noSession.Code != 403 {
		t.Fatal("queue unauthenticated")
	}
}
func TestStageRequiresSCFAndGeneratedXYZ(t *testing.T) {
	for _, marker := range []string{"MOCK_NO_SCF", "MOCK_NO_XYZ"} {
		t.Run(marker, func(t *testing.T) {
			q := mockQueue(t)
			r := smallReq(marker)
			r.Files[1].Content += "# " + marker
			addEntry(t, q, r)
			_ = q.action("", "resume")
			waitQueue(t, q, func(es []*QueueEntry) bool { return es[0].State == "Failed" })
			q.mu.Lock()
			defer q.mu.Unlock()
			if len(q.Jobs[0].Job.CompletedStages) != 0 {
				t.Fatal("unverified step checkpointed")
			}
		})
	}
}
func TestResourcesPerCoreAndInlinePAL(t *testing.T) {
	r := smallReq("A")
	cpu, ram, err := resources(r)
	if err != nil || cpu != 1 || ram != 125 {
		t.Fatal(cpu, ram, err)
	}
	r.Files[1].Content = strings.ReplaceAll(r.Files[1].Content, "nprocs 1", "nprocs 8")
	cpu, ram, err = resources(r)
	if err != nil || cpu != 8 || ram != 1000 {
		t.Fatal(cpu, ram, err)
	}
}
func TestQueueCorruptStateIsNotOverwritten(t *testing.T) {
	q := mockQueue(t)
	os.WriteFile(q.path, []byte("not json"), 0600)
	r := newJobQueue(q.app, q.path)
	if r.Error == "" {
		t.Fatal("corruption hidden")
	}
	if r.action("", "resume") == nil {
		t.Fatal("corrupt state overwritten")
	}
	b, _ := os.ReadFile(q.path)
	if string(b) != "not json" {
		t.Fatal("original lost")
	}
}

// Explicitly verify cancellation before first stage cannot claim success.
func TestCancelledContextDoesNotComplete(t *testing.T) {
	q := mockQueue(t)
	r := smallReq("cancel")
	e := addEntry(t, q, r)
	q.mu.Lock()
	err := q.prepareLocked(e, q.app.cfg)
	q.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	q.app.runJob(ctx, q.app.cfg.ORCAExe, r, e.Job)
	if e.Job.State == "completed" {
		t.Fatal("cancelled job completed")
	}
}

func TestQueueRetryNonOptPreservesSameStemInputGeometry(t *testing.T) {
	q := mockQueue(t)
	q.Limits = QueueLimits{1, 1, 1024}
	original := smallReq("SP retry")
	inp := strings.Replace(original.Files[1].Content, " def2-SVP Opt", " def2-SVP", 1) + "\n# MOCK_FAIL\n"
	req := RunRequest{Label: "SP retry", Files: []JobFile{original.Files[0], {Name: "water.inp", Content: inp}}, Stages: []Stage{{Name: "SP", Input: "water.inp", Source: "water.xyz"}}}
	e := addEntry(t, q, req)
	if err := q.action("", "resume"); err != nil {
		t.Fatal(err)
	}
	waitQueue(t, q, func(es []*QueueEntry) bool { return es[0].State == "Failed" })
	q.mu.Lock()
	old := e.Job.Folder
	q.mu.Unlock()
	if err := os.WriteFile(filepath.Join(q.app.cfg.WorkRoot, "allow_retry"), []byte("yes"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := q.action(e.ID, "retry"); err != nil {
		t.Fatal(err)
	}
	waitQueue(t, q, func(es []*QueueEntry) bool { return es[0].State == "Completed" })
	q.mu.Lock()
	defer q.mu.Unlock()
	if e.Job.Folder == old {
		t.Fatal("retry reused original result folder")
	}
	for _, folder := range []string{old, e.Job.Folder} {
		if _, err := os.Stat(filepath.Join(folder, "water.xyz")); err != nil {
			t.Fatalf("submitted source geometry removed from %s: %s", folder, err)
		}
	}
}

func TestStageRejectsElementOrderAndEqualChargeCompositionMismatch(t *testing.T) {
	for _, output := range []string{"2\nreordered\nO 0 0 0\nC 1 0 0\n", "2\ndifferent composition, same total atomic number\nN 0 0 0\nN 1 0 0\n"} {
		dir := t.TempDir()
		files := map[string]string{"source.xyz": "2\nsource\nC 0 0 0\nO 1 0 0\n", "opt.xyz": output, "opt.inp": "! PBE0 def2-SVP Opt\n* xyzfile 0 1 source.xyz\n", "opt.out": "SCF CONVERGED AFTER 4 CYCLES\nFINAL SINGLE POINT ENERGY -100.0\nTHE OPTIMIZATION HAS CONVERGED\nORCA TERMINATED NORMALLY\n"}
		for name, content := range files {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
		}
		if err := validateStage(dir, Stage{Name: "Opt", Input: "opt.inp", Source: "source.xyz", Opt: true}); err == nil {
			t.Fatal("changed element identities/order passed stage validation")
		}
	}
}

func TestUpdatingBlocksQueueDispatchAndAdmission(t *testing.T) {
	q := mockQueue(t)
	q.Limits = QueueLimits{1, 1, 1024}
	addEntry(t, q, smallReq("already queued"))
	q.app.mu.Lock()
	q.app.updating = true
	q.app.token = "token"
	q.app.mu.Unlock()
	q.mu.Lock()
	q.Paused = false
	q.dispatchLocked()
	state, worker := q.Jobs[0].State, q.Jobs[0].worker
	q.mu.Unlock()
	if state != "Pending" || worker != nil {
		t.Fatal("update freeze allowed an ORCA process to start")
	}
	payload, _ := json.Marshal(map[string]any{"jobs": []RunRequest{smallReq("new")}, "limits": QueueLimits{1, 1, 1024}})
	req := httptest.NewRequest("POST", "/api/queue?session=token", bytes.NewReader(payload))
	req.Header.Set("X-ORCA-Session", "token")
	w := httptest.NewRecorder()
	q.app.guard(q.app.route)(w, req)
	if w.Code != 409 {
		t.Fatalf("new batch admitted during update: %d %s", w.Code, w.Body.String())
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.Jobs) != 1 {
		t.Fatal("blocked submission modified Queue")
	}
}

func TestResourcesRejectsUnsupportedBaseAndExternalOrbitalDirectives(t *testing.T) {
	for _, directive := range []string{`%base "../shared"`, `%BASE "C:\\shared\\result"`, `%base "other_stem"`, `%moinp "../other_job/orbitals.gbw"`, `! MOREAD`, `%compound`, `$new_job`} {
		t.Run(directive, func(t *testing.T) {
			req := smallReq("unsupported input")
			req.Files[1].Content += "\n" + directive + "\n"
			if _, _, err := resources(req); err == nil {
				t.Fatal("unsupported artifact/reference directive passed Queue validation")
			}
		})
	}
}

func TestStageChecksActualORCAChargeMultiplicityMetadata(t *testing.T) {
	for _, tc := range []struct {
		name, charge, mult string
		valid              bool
	}{{"matching", "0", "1", true}, {"wrong charge", "+1", "1", false}, {"wrong multiplicity", "0", "3", false}} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			inp := "! PBE0 def2-SVP\n* xyzfile 0 1 water.xyz\n"
			out := "Total Charge Charge .... " + tc.charge + "\nMultiplicity Mult .... " + tc.mult + "\nSCF CONVERGED AFTER 4 CYCLES\nFINAL SINGLE POINT ENERGY -75.0\nORCA TERMINATED NORMALLY\n"
			for name, content := range map[string]string{"water.inp": inp, "water.out": out} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			err := validateStage(dir, Stage{Name: "SP", Input: "water.inp", Source: "water.xyz"})
			if (err == nil) != tc.valid {
				t.Fatalf("metadata validity %t, validation error: %v", tc.valid, err)
			}
		})
	}
}

func TestQueueRetryRejectsDiskEditedInputWithoutChangingReservation(t *testing.T) {
	q := mockQueue(t)
	q.Limits = QueueLimits{1, 1, 1024}
	req := smallReq("edited retry")
	req.Files[1].Content += "\n# MOCK_FAIL\n"
	e := addEntry(t, q, req)
	if err := q.action("", "resume"); err != nil {
		t.Fatal(err)
	}
	waitQueue(t, q, func(es []*QueueEntry) bool { return es[0].State == "Failed" })
	q.mu.Lock()
	oldFolder, oldCPU, oldRAM := e.Job.Folder, e.CPU, e.RAMMB
	q.mu.Unlock()
	modified := strings.Replace(req.Files[1].Content, "nprocs 1", "nprocs 8", 1)
	if err := os.WriteFile(filepath.Join(oldFolder, req.Stages[0].Input), []byte(modified), 0600); err != nil {
		t.Fatal(err)
	}
	if err := q.action(e.ID, "retry"); err == nil {
		_ = q.action("", "stopAll")
		waitQueue(t, q, func(es []*QueueEntry) bool { return es[0].worker == nil })
		t.Fatal("retry silently accepted an edited input with larger CPU reservation")
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if e.worker != nil || e.State != "Failed" || e.CPU != oldCPU || e.RAMMB != oldRAM || e.Job.Folder != oldFolder {
		t.Fatal("rejected retry changed the original job or dispatched a worker")
	}
}

func TestQueueQuitKeepsPendingJobsAndPausesDispatch(t *testing.T) {
	q := mockQueue(t)
	e := addEntry(t, q, smallReq("pending across quit"))
	q.Paused = false
	if err := q.pauseForQuit(); err != nil {
		t.Fatal(err)
	}
	if !q.Paused || e.State != "Pending" || e.worker != nil {
		t.Fatal("exit unexpectedly removed or executed pending job")
	}
	recovered := newJobQueue(q.app, q.path)
	if len(recovered.Jobs) != 1 || recovered.Jobs[0].State != "Pending" || !recovered.Paused {
		t.Fatal("pending job did not survive restart")
	}
}

func TestQueueQuitRejectsRunningWorker(t *testing.T) {
	q := mockQueue(t)
	e := addEntry(t, q, smallReq("running"))
	e.worker = &App{job: &Job{State: "running"}}
	if err := q.pauseForQuit(); err == nil {
		t.Fatal("app quit accepted an active worker")
	}
}

func TestRetryDoesNotCopyFailedStageGBWOrHess(t *testing.T) {
	q := mockQueue(t)
	req := smallReq("stale restart")
	e := addEntry(t, q, req)
	q.mu.Lock()
	err := q.prepareLocked(e, q.app.cfg)
	q.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	old := e.Job.Folder
	for _, pair := range [][2]string{
		{"water_opt.xyz", waterXYZ},
		{"water_opt.out", "ORCA TERMINATED NORMALLY"},
		{"water_opt.gbw", "completed orbitals"},
		{"triplet.gbw", "failed orbitals"},
		{"triplet.hess", "failed hessian"},
		{"triplet.out", "SCF NOT CONVERGED"},
		{"FAILURE_ANALYSIS.txt", "failed"},
	} {
		if err := os.WriteFile(filepath.Join(old, pair[0]), []byte(pair[1]), 0600); err != nil {
			t.Fatal(err)
		}
	}
	e.Job.CompletedStages = []string{"water_opt.inp"}
	e.State = "Failed"
	if err := q.action(e.ID, "retry"); err != nil {
		t.Fatal(err)
	}
	fresh := e.Job.Folder
	if old == fresh || e.State != "Pending" {
		t.Fatal("retry did not preserve old result and schedule a fresh attempt")
	}
	for _, name := range []string{"water.xyz", "water_opt.inp", "water_opt.xyz", "water_opt.out", "water_opt.gbw"} {
		if _, err := os.Stat(filepath.Join(fresh, name)); err != nil {
			t.Fatalf("required input/completed stage artifact %s missing: %v", name, err)
		}
	}
	for _, name := range []string{"triplet.gbw", "triplet.hess", "triplet.out", "FAILURE_ANALYSIS.txt"} {
		if _, err := os.Stat(filepath.Join(fresh, name)); !os.IsNotExist(err) {
			t.Fatalf("stale failed artifact %s was reused: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(old, "triplet.gbw")); err != nil {
		t.Fatal("original failure artifacts were removed")
	}
}

func TestBatchFourJobsParallelCancelRetryReports(t *testing.T) {
	if runtime.NumCPU() < 2 {
		t.Skip("parallel scenario requires two CPUs")
	}
	q := mockQueue(t)
	q.Limits = QueueLimits{Concurrency: 2, CPU: 2, RAMMB: 1024}
	for _, name := range []string{"A", "B", "C", "D"} {
		req := smallReq(name)
		if name == "B" {
			req.Files[1].Content += "\n# MOCK_BLOCK\n"
		}
		addEntry(t, q, req)
	}
	if err := q.action("", "resume"); err != nil {
		t.Fatal(err)
	}
	waitQueue(t, q, func(es []*QueueEntry) bool {
		return es[0].State == "Running" && es[1].State == "Running" &&
			es[2].State == "Pending" && es[3].State == "Pending"
	})
	q.mu.Lock()
	b := q.Jobs[1]
	old := b.Job.Folder
	q.mu.Unlock()

	// Ensure that the actual ORCA substitute was started before cancelling.
	startDeadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(old, "started")); err == nil {
			break
		}
		if time.Now().After(startDeadline) {
			t.Fatal("mock ORCA child process never started")
		}
		time.Sleep(15 * time.Millisecond)
	}
	if err := q.action(b.ID, "cancel"); err != nil {
		t.Fatal(err)
	}
	waitQueue(t, q, func(es []*QueueEntry) bool {
		return es[1].State == "Interrupted"
	})
	waitQueue(t, q, func(es []*QueueEntry) bool {
		return es[0].State == "Completed" && es[2].State == "Completed" &&
			es[3].State == "Completed" && es[1].State == "Interrupted"
	})
	if err := os.WriteFile(filepath.Join(q.app.cfg.WorkRoot, "allow_continue"), []byte("yes"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := q.action(b.ID, "retry"); err != nil {
		t.Fatal(err)
	}
	waitQueue(t, q, func(es []*QueueEntry) bool {
		for _, e := range es {
			if e.State != "Completed" {
				return false
			}
		}
		return true
	})
	q.mu.Lock()
	defer q.mu.Unlock()
	if b.Job.Folder == old {
		t.Fatal("cancelled job retried in original folder")
	}
	if len(q.Jobs) != 4 {
		t.Fatalf("expected four jobs, got %d", len(q.Jobs))
	}
	dirs := map[string]bool{}
	for _, e := range q.Jobs {
		if dirs[e.Job.Folder] {
			t.Fatal("two jobs share a calculation folder")
		}
		dirs[e.Job.Folder] = true
		if len(e.Job.CompletedStages) != len(e.Request.Stages) {
			t.Fatalf("%s: missing completed workflow stages", e.Label)
		}
		for _, name := range []string{"analysis.txt", "results.json"} {
			if _, err := os.Stat(filepath.Join(e.Job.Folder, name)); err != nil {
				t.Fatalf("%s: report %s: %v", e.Label, name, err)
			}
		}
	}
	for _, name := range []string{"batch_results.csv", "batch_summary.txt"} {
		b, err := os.ReadFile(filepath.Join(q.app.cfg.WorkRoot, name))
		if err != nil {
			t.Fatalf("batch result %s not written: %v", name, err)
		}
		if len(b) == 0 || !strings.Contains(string(b), "B") {
			t.Fatalf("batch result %s is empty or missing molecule B", name)
		}
	}
	if _, err := os.Stat(old); err != nil {
		t.Fatal("cancelled original job directory was lost:", err)
	}
}

// A real helper process spawns another process, representing ORCA/MPI. This
// same test runs on Windows (taskkill /T) and Unix (process group kill).
func TestCancellationTerminatesDescendantProcess(t *testing.T) {
	dir := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "mock-tree-parent")
	cmd.Env = append(os.Environ(), "ORCA_NOVA_TREE_DIR="+dir)
	configureProcess(cmd)
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		killTree(cmd.Process.Pid)
		_ = cmd.Process.Kill()
		if b, e := os.ReadFile(filepath.Join(dir, "leaf.pid")); e == nil {
			if pid, e := strconv.Atoi(string(b)); e == nil {
				if p, e := os.FindProcess(pid); e == nil {
					_ = p.Kill()
				}
			}
		}
	}()
	heartbeat := filepath.Join(dir, "heartbeat")
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err = os.Stat(heartbeat); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("descendant heartbeat never started")
		}
		time.Sleep(20 * time.Millisecond)
	}
	killTree(cmd.Process.Pid)
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	time.Sleep(100 * time.Millisecond)
	before, err := os.ReadFile(heartbeat)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(180 * time.Millisecond)
	after, err := os.ReadFile(heartbeat)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("descendant kept running after process tree cancellation")
	}
}

func TestRetrySaveFailureRestoresOriginalAttempt(t *testing.T) {
	q := mockQueue(t)
	e := addEntry(t, q, smallReq("retry transaction"))
	q.mu.Lock()
	err := q.prepareLocked(e, q.app.cfg)
	q.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	old := e.Job
	e.State = "Failed"
	e.Held = true
	// An existing directory cannot be atomically replaced by the queue JSON file.
	q.path = q.app.cfg.WorkRoot
	if err = q.action(e.ID, "retry"); err == nil {
		t.Fatal("save failure was hidden")
	}
	if e.Job != old || e.State != "Failed" || !e.Held {
		t.Fatal("failed retry replaced original job metadata")
	}
	if _, err = os.Stat(old.Folder); err != nil {
		t.Fatal("original result directory lost", err)
	}
	if q.Error == "" || !q.Paused {
		t.Fatal("queue did not expose persistence failure and pause")
	}
}

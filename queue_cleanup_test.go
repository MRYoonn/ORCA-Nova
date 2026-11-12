package main

import (
    "os"
    "path/filepath"
    "testing"
)

func TestQueueHistoryCleanupPreservesResultsAndActiveJobs(t *testing.T) {
    q := mockQueue(t)
    resultDir := t.TempDir()
    outPath := filepath.Join(resultDir, "original.out")
    if err := os.WriteFile(outPath, []byte("ORCA output must survive history deletion"), 0600); err != nil {
        t.Fatal(err)
    }

    failed := addEntry(t, q, smallReq("failed"))
    failed.State = "Failed"
    failed.Job = &Job{Folder: resultDir, State: "failed"}
    interrupted := addEntry(t, q, smallReq("interrupted"))
    interrupted.State = "Interrupted"
    cancelled := addEntry(t, q, smallReq("cancelled"))
    cancelled.State = "Cancelled"
    completed := addEntry(t, q, smallReq("completed"))
    completed.State = "Completed"
    pending := addEntry(t, q, smallReq("pending"))
    running := addEntry(t, q, smallReq("running"))
    running.State = "Running"

    q.mu.Lock()
    err := q.saveLocked()
    q.mu.Unlock()
    if err != nil {
        t.Fatal(err)
    }

    for _, e := range []*QueueEntry{pending, running} {
        if err := q.action(e.ID, "remove"); err == nil {
            t.Fatalf("%s job must not be deleted", e.State)
        }
    }
    if err := q.action("missing-id", "remove"); err == nil {
        t.Fatal("unknown job ID must not be accepted")
    }
    if err := q.action(failed.ID, "remove"); err != nil {
        t.Fatal(err)
    }
    if _, err := os.Stat(outPath); err != nil {
        t.Fatalf("removing a queue record removed its result file: %v", err)
    }
    if err := q.action("", "clearFailed"); err != nil {
        t.Fatal(err)
    }
    if len(q.Jobs) != 3 || q.Jobs[0] != completed || q.Jobs[1] != pending || q.Jobs[2] != running {
        t.Fatalf("clearFailed removed unrelated jobs: %+v", q.Jobs)
    }
    if err := q.action("", "clearCompleted"); err != nil {
        t.Fatal(err)
    }
    if err := q.action("", "clearAll"); err != nil {
        t.Fatal(err)
    }
    if len(q.Jobs) != 2 || q.Jobs[0] != pending || q.Jobs[1] != running {
        t.Fatalf("active jobs must remain after clearing all history: %+v", q.Jobs)
    }
    if _, err := os.Stat(outPath); err != nil {
        t.Fatalf("clearing queue history removed the output: %v", err)
    }
}

func TestQueueHistoryCleanupPersistsAndRollsBackOnSaveFailure(t *testing.T) {
    q := mockQueue(t)
    failed := addEntry(t, q, smallReq("old-failure"))
    failed.State = "Failed"
    keep := addEntry(t, q, smallReq("keep"))
    keep.State = "Completed"
    if err := q.action(failed.ID, "remove"); err != nil {
        t.Fatal(err)
    }
    reloaded := newJobQueue(q.app, q.path)
    if len(reloaded.Jobs) != 1 || reloaded.Jobs[0].ID != keep.ID {
        t.Fatalf("deleted record reappeared after restart: %+v", reloaded.Jobs)
    }

    // A pre-existing .tmp directory forces the atomic queue save to fail.
    // The in-memory list must not lose the entry when persistence fails.
    if err := os.Mkdir(q.path+".tmp", 0700); err != nil {
        t.Fatal(err)
    }
    if err := q.action(keep.ID, "remove"); err == nil {
        t.Fatal("expected queue persistence error")
    }
    if len(q.Jobs) != 1 || q.Jobs[0].ID != keep.ID {
        t.Fatalf("queue mutated despite failed save: %+v", q.Jobs)
    }
}

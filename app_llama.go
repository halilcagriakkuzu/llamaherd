package main

import (
	"fmt"
	"sync"
	"time"

	"llamaherd/control"
	"llamaherd/settings"
)

// llamaUpdate tracks the single in-flight update-llamacpp.sh run.
type llamaUpdate struct {
	mu      sync.Mutex
	running bool
	cancel  func()
	last    string // one-line outcome of the previous run
}

// LlamaPanel is everything the "llama.cpp" tab shows.
type LlamaPanel struct {
	Info       control.LlamaInfo   `json:"info"`
	Procs      []control.LlamaProc `json:"procs"`
	Updating   bool                `json:"updating"`
	LastUpdate string              `json:"lastUpdate"`
}

func (a *App) llamaState() (running bool, last string) {
	a.llama.mu.Lock()
	defer a.llama.mu.Unlock()
	return a.llama.running, a.llama.last
}

// GetLlamaPanel returns the checkout/build info plus all llama-server
// processes on the machine (any origin).
func (a *App) GetLlamaPanel() LlamaPanel {
	running, last := a.llamaState()
	return LlamaPanel{
		Info:       control.ReadLlamaInfo(settings.LlamaDir(), a.bin),
		Procs:      control.ListLlamaProcs(a.mgr.Info().PID),
		Updating:   running,
		LastUpdate: last,
	}
}

// CheckLlamaUpdates fetches upstream so the panel can say how far behind
// the checkout is.
func (a *App) CheckLlamaUpdates() LlamaPanel {
	a.Log("llama.cpp: git fetch ...")
	if err := control.FetchLlama(settings.LlamaDir()); err != nil {
		a.Log("llama.cpp: fetch FAILED: " + err.Error())
	}
	return a.GetLlamaPanel()
}

// UpdateLlamaCpp runs scripts/update-llamacpp.sh (git pull, cmake configure,
// cmake build). It refuses while any llama-server is running — stop it (or
// use ForceStopLlama) first — and while another update is in flight.
func (a *App) UpdateLlamaCpp() {
	a.llama.mu.Lock()
	if a.llama.running {
		a.llama.mu.Unlock()
		a.Log("llama.cpp: an update is already running")
		return
	}
	if procs := control.ListLlamaProcs(0); len(procs) > 0 {
		a.llama.mu.Unlock()
		a.Log(fmt.Sprintf("llama.cpp: update refused — llama-server is running (pid %d). Stop it or use Force stop.", procs[0].PID))
		return
	}
	a.llama.running = true
	a.llama.mu.Unlock()

	a.Log("llama.cpp: update started (scripts/update-llamacpp.sh)")
	start := time.Now()
	cancel, done := control.RunUpdateScript("scripts/update-llamacpp.sh", a.Log)
	a.llama.mu.Lock()
	a.llama.cancel = cancel
	a.llama.mu.Unlock()
	a.emitState()

	go func() {
		err := <-done
		took := time.Since(start).Truncate(time.Second)
		msg := fmt.Sprintf("update finished in %s", took)
		if err != nil {
			msg = fmt.Sprintf("update FAILED after %s: %v", took, err)
		}
		a.llama.mu.Lock()
		a.llama.running = false
		a.llama.cancel = nil
		a.llama.last = time.Now().Format("2006-01-02 15:04") + " — " + msg
		a.llama.mu.Unlock()
		a.Log("llama.cpp: " + msg)
		a.emitState()
	}()
}

// CancelLlamaUpdate terminates a running update (whole process group).
func (a *App) CancelLlamaUpdate() {
	a.llama.mu.Lock()
	cancel := a.llama.cancel
	a.llama.mu.Unlock()
	if cancel == nil {
		return
	}
	a.Log("llama.cpp: cancelling update ...")
	cancel()
}

// ForceStopLlama stops every llama-server on the machine, including ones
// LlamaHerd did not start: a graceful Stop for the managed one, then SIGTERM
// and finally SIGKILL for the rest.
func (a *App) ForceStopLlama() {
	procs := control.ListLlamaProcs(a.mgr.Info().PID)
	if len(procs) == 0 {
		a.Log("llama.cpp: no llama-server process found")
		return
	}
	a.Log(fmt.Sprintf("llama.cpp: force-stopping %d llama-server process(es)", len(procs)))
	_ = a.mgr.Stop()
	pids := control.KillLlamaProcs(4 * time.Second)
	a.Log(fmt.Sprintf("llama.cpp: signalled pid(s) %v", pids))
	a.mgr.ResetExternal()
	a.runCfg = nil
	a.emitState()
}

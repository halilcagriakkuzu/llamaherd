package main

import (
	"encoding/json"

	"llamaherd/doctor"
)

// DoctorReport returns the last cached environment report (cheap; safe to call
// from snapshot()).
func (a *App) DoctorReport() doctor.Report {
	a.doctorMu.Lock()
	defer a.doctorMu.Unlock()
	return a.doctorRep
}

// refreshDoctor re-runs the checks and caches the result.
func (a *App) refreshDoctor() doctor.Report {
	rep := doctor.Run()
	a.doctorMu.Lock()
	a.doctorRep = rep
	a.doctorMu.Unlock()
	return rep
}

// GetDoctor runs the checks and returns the report (Wails binding).
func (a *App) GetDoctor() doctor.Report {
	return a.refreshDoctor()
}

// RunDoctor re-checks and pushes the result to the UI (Wails binding).
func (a *App) RunDoctor() doctor.Report {
	rep := a.refreshDoctor()
	a.Log("doctor: " + rep.Summary)
	a.emitState()
	return rep
}

// DoctorJSON marshals the cached report (control API GET /doctor).
func (a *App) DoctorJSON() ([]byte, error) {
	return json.Marshal(a.DoctorReport())
}

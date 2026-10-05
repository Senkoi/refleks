package main

import (
	"aimmeow/internal/practice"
	"encoding/json"
	"time"
)

func (a *App) SaveTrainingSessionNote(id, name, notes string, aliases []string) error {
	return a.runsRuntimeSvc.SaveTrainingSessionNote(id, name, notes, aliases)
}

// GroupPracticeSessions accepts only lightweight timestamps/IDs, never raw
// events or mouse paths. UI and assessment use the same domain calculation.
func (a *App) GroupPracticeSessions(request string) (string, error) {
	var input practice.SessionRequest
	if err := json.Unmarshal([]byte(request), &input); err != nil {
		return "", err
	}
	data, err := json.Marshal(practice.Group(input.Entries, time.Duration(input.GapMinutes)*time.Minute, time.Now()))
	return string(data), err
}

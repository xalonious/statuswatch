package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

type SeenIncident struct {
	Name string `json:"name"`
}

type State struct {
	SeenIncidents     map[string]map[string]SeenIncident `json:"seen_incidents"`
	UnhealthyServices map[string]bool                    `json:"unhealthy_services"`
}

func stateFilePath() string {
	exe, err := os.Executable()
	if err != nil {
		return "statuswatch_state.json"
	}
	return filepath.Join(filepath.Dir(exe), "statuswatch_state.json")
}

func loadState() State {
	data, err := os.ReadFile(stateFilePath())
	if err != nil {
		return newState()
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		log.Printf("Warning: could not parse state file, starting fresh: %v", err)
		return newState()
	}

	if s.SeenIncidents == nil {
		s.SeenIncidents = make(map[string]map[string]SeenIncident)
	}
	if s.UnhealthyServices == nil {
		s.UnhealthyServices = make(map[string]bool)
	}

	return s
}

func newState() State {
	return State{
		SeenIncidents:     make(map[string]map[string]SeenIncident),
		UnhealthyServices: make(map[string]bool),
	}
}

func saveState(s State) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(stateFilePath(), data, 0644)
}
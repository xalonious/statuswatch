package main

import (
	"log"
	"os"
	"time"
)

func main() {
	log.SetOutput(os.Stdout)
	cfg := loadConfig()

	log.Printf("statuswatch started, polling every %d seconds", cfg.PollIntervalSeconds)

	checkAll(cfg)

	ticker := time.NewTicker(time.Duration(cfg.PollIntervalSeconds) * time.Second)
	for range ticker.C {
		checkAll(cfg)
	}
}

func checkAll(cfg Config) {
	start := time.Now()
	state := loadState()
	changed := false

	for _, svc := range cfg.Services {
		result, err := fetchStatus(svc)
		if err != nil {
			log.Printf("Error checking %s: %v", svc.Name, err)
			continue
		}

		if state.SeenIncidents[svc.Name] == nil {
			state.SeenIncidents[svc.Name] = make(map[string]SeenIncident)
		}

		currentIDs := make(map[string]bool)
		for _, inc := range result.Incidents {
			currentIDs[inc.ID] = true

			if !meetsImpactThreshold(inc.Impact, cfg.MinImpact) {
				continue
			}

			existing, alreadySeen := state.SeenIncidents[svc.Name][inc.ID]
			if !alreadySeen {
				log.Printf("[NEW INCIDENT] %s: %s (%s)", svc.Name, inc.Name, inc.Status)
				if err := sendIncidentAlert(cfg.DiscordWebhookURL, result, inc); err != nil {
					log.Printf("Error sending alert for %s: %v", svc.Name, err)
				} else {
					state.SeenIncidents[svc.Name][inc.ID] = SeenIncident{Name: inc.Name, LatestUpdateID: inc.LatestUpdateID}
					changed = true
				}
			} else if inc.LatestUpdateID != "" && inc.LatestUpdateID != existing.LatestUpdateID {
				log.Printf("[UPDATE] %s: %s (%s)", svc.Name, inc.Name, inc.Status)
				if err := sendUpdateAlert(cfg.DiscordWebhookURL, result, inc); err != nil {
					log.Printf("Error sending update alert for %s: %v", svc.Name, err)
				} else {
					state.SeenIncidents[svc.Name][inc.ID] = SeenIncident{Name: inc.Name, LatestUpdateID: inc.LatestUpdateID}
					changed = true
				}
			}
		}

		for id, seen := range state.SeenIncidents[svc.Name] {
			if !currentIDs[id] {
				log.Printf("[RESOLVED] %s: %s", svc.Name, seen.Name)
				if err := sendResolutionAlert(cfg.DiscordWebhookURL, svc.Name, seen.Name); err != nil {
					log.Printf("Error sending resolution alert for %s: %v", svc.Name, err)
				} else {
					delete(state.SeenIncidents[svc.Name], id)
					changed = true
				}
			}
		}

		wasUnhealthy := state.UnhealthyServices[svc.Name]
		if !result.IsHealthy && !wasUnhealthy && len(result.Incidents) == 0 {
			log.Printf("[DEGRADED] %s: %s", svc.Name, result.Overall)
			if err := sendDegradedAlert(cfg.DiscordWebhookURL, result); err != nil {
				log.Printf("Error sending degraded alert for %s: %v", svc.Name, err)
			} else {
				state.UnhealthyServices[svc.Name] = true
				changed = true
			}
		} else if result.IsHealthy && wasUnhealthy {
			log.Printf("[RECOVERED] %s is back to operational", svc.Name)
			if err := sendRecoveryAlert(cfg.DiscordWebhookURL, svc.Name); err != nil {
				log.Printf("Error sending recovery alert for %s: %v", svc.Name, err)
			} else {
				delete(state.UnhealthyServices, svc.Name)
				changed = true
			}
		} else {
			log.Printf("[OK] %s: %s", svc.Name, result.Overall)
		}
	}

	if changed {
		if err := saveState(state); err != nil {
			log.Printf("Error saving state: %v", err)
		}
	}

	log.Printf("Check done in %s", time.Since(start).Round(time.Millisecond))
}
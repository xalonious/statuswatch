package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type StatusResult struct {
	ServiceName string
	Overall     string
	IsHealthy   bool
	Incidents   []Incident
}

type Incident struct {
	ID                 string
	Name               string
	Status             string
	Impact             string
	Body               string
	LatestUpdateID     string
	Updated            time.Time
	AffectedComponents []string
}

type statusIOResponse struct {
	Result statusIOResult `json:"result"`
}

type statusIOResult struct {
	StatusOverall statusIOOverall    `json:"status_overall"`
	Incidents     []statusIOIncident `json:"incidents"`
}

type statusIOOverall struct {
	Status     string `json:"status"`
	StatusCode int    `json:"status_code"`
}

type statusIOIncident struct {
	ID       string                   `json:"_id"`
	Name     string                   `json:"name"`
	Messages []statusIOIncidentUpdate `json:"messages"`
}

type statusIOIncidentUpdate struct {
	Body     string `json:"details"`
	State    int    `json:"state"`
	Status   int    `json:"status"`
	Datetime string `json:"datetime"`
}

type atlassianResponse struct {
	Status     atlassianStatus      `json:"status"`
	Components []atlassianComponent `json:"components"`
	Incidents  []atlassianIncident  `json:"incidents"`
}

type atlassianStatus struct {
	Indicator   string `json:"indicator"`
	Description string `json:"description"`
}

type atlassianComponent struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type atlassianIncident struct {
	ID         string                    `json:"id"`
	Name       string                    `json:"name"`
	Status     string                    `json:"status"`
	Impact     string                    `json:"impact"`
	Updates    []atlassianIncidentUpdate `json:"incident_updates"`
	Components []atlassianComponent      `json:"components"`
}

type atlassianIncidentUpdate struct {
	ID                 string                       `json:"id"`
	Body               string                       `json:"body"`
	UpdatedAt          string                       `json:"updated_at"`
	AffectedComponents []atlassianAffectedComponent `json:"affected_components"`
}

type atlassianAffectedComponent struct {
	Name string `json:"name"`
}

func fetchStatus(svc Service) (StatusResult, error) {
	switch svc.Provider {
	case ProviderStatusIO:
		return fetchStatusIO(svc)
	case ProviderAtlassian:
		return fetchAtlassian(svc)
	default:
		return StatusResult{}, fmt.Errorf("unknown provider: %s", svc.Provider)
	}
}

func fetchStatusIO(svc Service) (StatusResult, error) {
	body, err := httpGet(svc.URL)
	if err != nil {
		return StatusResult{}, fmt.Errorf("fetching %s: %w", svc.Name, err)
	}

	var resp statusIOResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return StatusResult{}, fmt.Errorf("parsing %s response: %w", svc.Name, err)
	}

	result := StatusResult{
		ServiceName: svc.Name,
		Overall:     resp.Result.StatusOverall.Status,
		IsHealthy:   resp.Result.StatusOverall.StatusCode == 100,
	}

	for _, inc := range resp.Result.Incidents {
		var latestBody, latestUpdateID, status, impact string
		var updated time.Time

		if len(inc.Messages) > 0 {
			latest := inc.Messages[len(inc.Messages)-1]
			latestBody = latest.Body
			latestUpdateID = latest.Datetime
			updated, _ = time.Parse(time.RFC3339, latest.Datetime)
			switch latest.State {
			case 100:
				status = "Investigating"
			case 200:
				status = "Identified"
			case 300:
				status = "Monitoring"
			case 400:
				status = "Resolved"
			default:
				status = "Investigating"
			}
			switch latest.Status {
			case 100:
				impact = "none"
			case 200, 300:
				impact = "minor"
			case 400:
				impact = "major"
			case 500:
				impact = "critical"
			default:
				impact = "minor"
			}
		}

		result.Incidents = append(result.Incidents, Incident{
			ID:             inc.ID,
			Name:           inc.Name,
			Status:         status,
			Impact:         impact,
			Body:           latestBody,
			LatestUpdateID: latestUpdateID,
			Updated:        updated,
		})
	}

	return result, nil
}

func fetchAtlassian(svc Service) (StatusResult, error) {
	body, err := httpGet(svc.URL + "/api/v2/summary.json")
	if err != nil {
		return StatusResult{}, fmt.Errorf("fetching %s: %w", svc.Name, err)
	}

	var resp atlassianResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return StatusResult{}, fmt.Errorf("parsing %s response: %w", svc.Name, err)
	}

	isHealthy := resp.Status.Indicator == "none"
	if !isHealthy && len(svc.Components) > 0 {
		isHealthy = !anyFilteredComponentDegraded(resp.Components, svc.Components)
	}

	result := StatusResult{
		ServiceName: svc.Name,
		Overall:     resp.Status.Description,
		IsHealthy:   isHealthy,
	}

	for _, inc := range resp.Incidents {
		var updated time.Time
		var latestBody string
		var components []string
		seen := make(map[string]bool)

		if len(inc.Updates) > 0 {
			updated, _ = time.Parse(time.RFC3339, inc.Updates[0].UpdatedAt)
			latestBody = inc.Updates[0].Body

			for _, update := range inc.Updates {
				for _, c := range update.AffectedComponents {
					if !seen[c.Name] {
						components = append(components, c.Name)
						seen[c.Name] = true
					}
				}
			}
		}
		for _, c := range inc.Components {
			if !seen[c.Name] {
				components = append(components, c.Name)
				seen[c.Name] = true
			}
		}

		var latestUpdateID string
		if len(inc.Updates) > 0 {
			latestUpdateID = inc.Updates[0].ID
		}

		incident := Incident{
			ID:                 inc.ID,
			Name:               inc.Name,
			Status:             inc.Status,
			Impact:             inc.Impact,
			Body:               latestBody,
			LatestUpdateID:     latestUpdateID,
			Updated:            updated,
			AffectedComponents: components,
		}

		if len(svc.Components) == 0 || incidentMatchesFilter(incident, svc.Components) {
			result.Incidents = append(result.Incidents, incident)
		}
	}

	return result, nil
}

func anyFilteredComponentDegraded(components []atlassianComponent, filter []string) bool {
	for _, c := range components {
		if c.Status == "operational" {
			continue
		}
		if componentNameMatchesFilter(c.Name, filter) {
			return true
		}
	}
	return false
}

func incidentMatchesFilter(inc Incident, filter []string) bool {
	if len(inc.AffectedComponents) == 0 {
		return true
	}

	for _, affected := range inc.AffectedComponents {
		if componentNameMatchesFilter(affected, filter) {
			return true
		}
	}

	return false
}

func componentNameMatchesFilter(name string, filter []string) bool {
	for _, wanted := range filter {
		if componentNamesEqual(name, wanted) {
			return true
		}
	}
	return false
}

func componentNamesEqual(name, wanted string) bool {
	name = normalizeComponentName(name)
	wanted = normalizeComponentName(wanted)
	if strings.EqualFold(name, wanted) {
		return true
	}

	return strings.HasSuffix(strings.ToLower(name), " - "+strings.ToLower(wanted))
}

func normalizeComponentName(name string) string {
	return strings.Join(strings.Fields(name), " ")
}

func httpGet(url string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

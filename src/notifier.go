package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type discordWebhookPayload struct {
	Embeds []discordEmbed `json:"embeds"`
}

type discordEmbed struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Color       int            `json:"color"`
	Fields      []discordField `json:"fields,omitempty"`
	Footer      discordFooter  `json:"footer"`
	Timestamp   string         `json:"timestamp"`
}

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type discordFooter struct {
	Text string `json:"text"`
}

const (
	colorGreen  = 3066993
	colorYellow = 16776960
	colorOrange = 15105570
	colorRed    = 15158332
	colorGrey   = 9807270
)

func impactColor(impact string) int {
	switch impact {
	case "minor":
		return colorYellow
	case "major":
		return colorOrange
	case "critical":
		return colorRed
	default:
		return colorOrange
	}
}

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func impactRank(impact string) int {
	switch strings.ToLower(impact) {
	case "critical":
		return 4
	case "major":
		return 3
	case "minor":
		return 2
	case "none":
		return 1
	default:
		return 2 
	}
}

func meetsImpactThreshold(impact, minImpact string) bool {
	if minImpact == "" {
		return true
	}
	return impactRank(impact) >= impactRank(minImpact)
}

func severityColor(overall string) int {
	lower := strings.ToLower(overall)
	switch {
	case strings.Contains(lower, "operational") || strings.Contains(lower, "resolved"):
		return colorGreen
	case strings.Contains(lower, "degraded") || strings.Contains(lower, "minor"):
		return colorYellow
	case strings.Contains(lower, "partial"):
		return colorOrange
	case strings.Contains(lower, "major") || strings.Contains(lower, "critical"):
		return colorRed
	default:
		return colorGrey
	}
}

func sendIncidentAlert(webhookURL string, result StatusResult, inc Incident) error {
	description := fmt.Sprintf("**%s**", inc.Name)
	if inc.Body != "" {
		description += fmt.Sprintf("\n\n%s", inc.Body)
	}

	fields := []discordField{
		{Name: "Status", Value: capitalizeFirst(inc.Status), Inline: true},
		{Name: "Service Health", Value: result.Overall, Inline: true},
	}

	if len(inc.AffectedComponents) > 0 {
		fields = append(fields, discordField{
			Name:   "Affects",
			Value:  strings.Join(inc.AffectedComponents, ", "),
			Inline: false,
		})
	}

	embed := discordEmbed{
		Title:       fmt.Sprintf("🚨 New Incident: %s", result.ServiceName),
		Description: description,
		Color:       impactColor(inc.Impact),
		Fields:      fields,
		Footer:      discordFooter{Text: "statuswatch"},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	return sendWebhook(webhookURL, embed)
}

func sendUpdateAlert(webhookURL string, result StatusResult, inc Incident) error {
	description := fmt.Sprintf("**%s**", inc.Name)
	if inc.Body != "" {
		description += "\n\n" + inc.Body
	}

	fields := []discordField{
		{Name: "Status", Value: capitalizeFirst(inc.Status), Inline: true},
		{Name: "Service Health", Value: result.Overall, Inline: true},
	}

	if len(inc.AffectedComponents) > 0 {
		fields = append(fields, discordField{
			Name:   "Affects",
			Value:  strings.Join(inc.AffectedComponents, ", "),
			Inline: false,
		})
	}

	embed := discordEmbed{
		Title:       fmt.Sprintf("🔄 Update: %s", result.ServiceName),
		Description: description,
		Color:       impactColor(inc.Impact),
		Fields:      fields,
		Footer:      discordFooter{Text: "statuswatch"},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	return sendWebhook(webhookURL, embed)
}

func sendResolutionAlert(webhookURL string, serviceName string, incidentName string) error {
	embed := discordEmbed{
		Title:       fmt.Sprintf("✅ Resolved: %s", serviceName),
		Description: fmt.Sprintf("**%s** has been resolved.", incidentName),
		Color:       colorGreen,
		Footer:      discordFooter{Text: "statuswatch"},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	return sendWebhook(webhookURL, embed)
}

func sendDegradedAlert(webhookURL string, result StatusResult) error {
	embed := discordEmbed{
		Title:       fmt.Sprintf("🔴 %s is reporting issues", result.ServiceName),
		Description: fmt.Sprintf("Overall status: **%s**", result.Overall),
		Color:       severityColor(result.Overall),
		Footer:      discordFooter{Text: "statuswatch"},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	return sendWebhook(webhookURL, embed)
}

func sendRecoveryAlert(webhookURL string, serviceName string) error {
	embed := discordEmbed{
		Title:       fmt.Sprintf("✅ Recovered: %s", serviceName),
		Description: "Service has returned to normal operation.",
		Color:       colorGreen,
		Footer:      discordFooter{Text: "statuswatch"},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	return sendWebhook(webhookURL, embed)
}

func sendWebhook(webhookURL string, embed discordEmbed) error {
	payload := discordWebhookPayload{
		Embeds: []discordEmbed{embed},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling webhook payload: %w", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("posting to Discord: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Discord returned unexpected status: %d", resp.StatusCode)
	}

	return nil
}
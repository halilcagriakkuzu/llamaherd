package hf

// modelinfo.go — licence and gated status of a Hub repository
// (GET /api/models/<repo>). Only the download path asks: a gated repository
// needs its licence accepted on Hugging Face before its files can be fetched.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ModelInfo is the provenance subset of the Hub model API.
type ModelInfo struct {
	Repo string `json:"repo"`
	// License is the SPDX-ish id from the model card ("apache-2.0", "other",
	// …); empty when the card does not declare one.
	License string `json:"license,omitempty"`
	// LicenseName is the card's license_name for custom ("other") licences.
	LicenseName string `json:"licenseName,omitempty"`
	// Gated is "no", "auto" (approved automatically) or "manual".
	Gated string `json:"gated"`
}

// IsGated reports whether downloading needs the licence accepted first.
func (m ModelInfo) IsGated() bool { return m.Gated == "auto" || m.Gated == "manual" }

// RepoURL is the browser URL of a model repository.
func RepoURL(repo string) string { return Endpoint() + "/" + repo }

type modelInfoJSON struct {
	Gated    json.RawMessage `json:"gated"`
	Tags     []string        `json:"tags"`
	CardData struct {
		License     json.RawMessage `json:"license"`
		LicenseName string          `json:"license_name"`
	} `json:"cardData"`
}

// FetchModelInfo asks the Hub for a repository's provenance.
func FetchModelInfo(ctx context.Context, repo, token string) (ModelInfo, error) {
	repo = strings.Trim(strings.TrimSpace(repo), "/")
	if strings.Count(repo, "/") != 1 {
		return ModelInfo{}, fmt.Errorf("invalid repository %q", repo)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, Endpoint()+"/api/models/"+escapePath(repo), nil)
	if err != nil {
		return ModelInfo{}, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return ModelInfo{}, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return ModelInfo{}, fmt.Errorf("%w: repository %s requires authentication (set a Hugging Face token)", ErrAccessDenied, repo)
	case http.StatusNotFound:
		return ModelInfo{}, fmt.Errorf("repository not found: %s", repo)
	default:
		return ModelInfo{}, fmt.Errorf("hub returned %s for %s", resp.Status, repo)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return ModelInfo{}, err
	}
	return parseModelInfo(repo, body)
}

func parseModelInfo(repo string, body []byte) (ModelInfo, error) {
	var raw modelInfoJSON
	if err := json.Unmarshal(body, &raw); err != nil {
		return ModelInfo{}, fmt.Errorf("unexpected hub response for %s: %w", repo, err)
	}
	info := ModelInfo{
		Repo:        repo,
		Gated:       parseGated(raw.Gated),
		LicenseName: raw.CardData.LicenseName,
		License:     firstString(raw.CardData.License),
	}
	for _, t := range raw.Tags {
		if lic, ok := strings.CutPrefix(t, "license:"); ok && info.License == "" {
			info.License = lic
		}
	}
	return info, nil
}

// parseGated maps the API's false | "auto" | "manual" to a string.
func parseGated(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil && s != "" {
		return s
	}
	var b bool
	if json.Unmarshal(raw, &b) == nil && b {
		return "auto"
	}
	return "no"
}

// firstString reads a card field that may be a string or a list of strings.
func firstString(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var list []string
	if json.Unmarshal(raw, &list) == nil && len(list) > 0 {
		return list[0]
	}
	return ""
}

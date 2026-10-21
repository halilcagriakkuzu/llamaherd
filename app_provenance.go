package main

// app_provenance.go — licence and gated status matter only when downloading:
// a gated Hugging Face repository serves its files only after the licence was
// accepted on the website. Once a model is on disk it is never checked again;
// the recorded repo/licence is shown in the details drawer for reference.

import (
	"context"
	"errors"
	"fmt"

	"llamaherd/hf"
	"llamaherd/schema"
	"llamaherd/settings"
)

// provenanceFrom is the library record of a successful model API lookup.
func provenanceFrom(info hf.ModelInfo) *schema.Provenance {
	return &schema.Provenance{
		Repo: info.Repo, URL: hf.RepoURL(info.Repo),
		License: info.License, LicenseName: info.LicenseName,
	}
}

// checkDownloadAccess looks the repository up before a download. A gated repo
// without a token fails right away with instructions; any other lookup
// failure is not fatal (the download itself reports access problems).
func checkDownloadAccess(ctx context.Context, repo, token string) (hf.ModelInfo, error) {
	info, err := hf.FetchModelInfo(ctx, repo, token)
	if err != nil {
		return hf.ModelInfo{Repo: repo}, nil
	}
	if info.IsGated() && token == "" {
		return info, gatedError(info, token)
	}
	return info, nil
}

// gatedError explains what to do when a gated repository refuses a download.
func gatedError(info hf.ModelInfo, token string) error {
	url := hf.RepoURL(info.Repo)
	if token == "" {
		return fmt.Errorf("%s is a gated repository: open %s, sign in and accept the licence, "+
			"then save a Hugging Face access token in the download dialog and try again", info.Repo, url)
	}
	approval := ""
	if info.Gated == "manual" {
		approval = " (this repository approves requests manually, which can take a while)"
	}
	return fmt.Errorf("your Hugging Face account has no access to %s yet: accept the licence at %s%s, "+
		"then try again", info.Repo, url, approval)
}

// downloadAccessError turns a 401/403 on a gated repository into instructions.
func downloadAccessError(err error, info hf.ModelInfo, token string) error {
	if errors.Is(err, hf.ErrAccessDenied) && info.IsGated() {
		return gatedError(info, token)
	}
	return err
}

// OpenRepoPage opens a Hugging Face repository page (owner/name or a Hub
// link), e.g. to accept a gated model's licence before downloading.
func (a *App) OpenRepoPage(ref string) error {
	repo, _, err := hf.ParseRef(ref)
	if err != nil {
		return err
	}
	if a.ctx == nil {
		return fmt.Errorf("opening links is only available in the GUI")
	}
	return settings.OpenExternalURL(hf.RepoURL(repo))
}

// OpenModelSource opens the model's source repository page in the browser.
func (a *App) OpenModelSource(ref string) error {
	a.stateMu.RLock()
	i, err := a.modelIndex(ref)
	var p *schema.Provenance
	if err == nil {
		p = a.models[i].Provenance
	}
	a.stateMu.RUnlock()
	if err != nil {
		return err
	}
	if p == nil || p.URL == "" {
		return fmt.Errorf("no source repository is known for this model")
	}
	if a.ctx == nil {
		return fmt.Errorf("opening links is only available in the GUI")
	}
	return settings.OpenExternalURL(p.URL)
}

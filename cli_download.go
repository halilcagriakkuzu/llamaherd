package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"llamaherd/control"
	"llamaherd/hf"
	"llamaherd/settings"
)

// runDownload is the `llamaherd download` entry point.
func runDownload(args []string) int {
	sub := ""
	if len(args) > 1 {
		sub = args[1]
	}
	switch sub {
	case "", "list", "recommend", "catalog":
		all := false
		for _, a := range args[2:] {
			all = all || a == "--all"
		}
		printCatalog(all)
		return 0
	case "help", "-h", "--help":
		fmt.Println("usage: llamaherd download list | files <ref> | <ref> [<file>] [--vision <file>|--no-vision] [--mtp <file>|--no-mtp] | status|cancel | token [<t>|--clear]")
		return 0
	case "files":
		return downloadFiles(args[2:])
	case "token":
		return downloadToken(args[2:])
	case "status":
		return downloadStatus()
	case "cancel":
		return downloadCancel()
	default:
		return downloadStart(args[1:])
	}
}

// printCatalog shows the highlighted models for this machine (or, with all, the
// whole catalog). It uses the cached/embedded catalog after a short best-effort
// refresh from the repository; no daemon is needed.
func printCatalog(all bool) {
	hf.LoadCachedCatalog()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	_, _ = hf.RefreshCatalog(ctx)
	cancel()

	vram := control.TotalVramMB()
	ram := totalRAMMB()
	info := hf.Info()
	fmt.Printf("Models for this machine — VRAM %.1f GiB, RAM %.1f GiB   (catalog %s, %s)\n\n",
		float64(vram)/1024, float64(ram)/1024, info.Updated, info.Source)

	line := func(label string, r hf.Rec) {
		e := r.Entry
		mark := map[string]string{hf.FitGPU: "gpu", hf.FitHybrid: "hyb", hf.FitCPU: "cpu", hf.FitNone: " - "}[r.Fit]
		tags := ""
		if e.Vision {
			tags += "  vision"
		}
		if e.MTP {
			tags += "  mtp"
		}
		if label != "" {
			fmt.Printf("  %s\n", label)
		}
		fmt.Printf("  [%s] %-44s %6.1f GiB%s\n", mark, e.Name, float64(e.Bytes)/(1<<30), tags)
		fmt.Printf("        %s\n", r.Why)
		fmt.Printf("        llamaherd download %s %s\n", e.Repo, e.File)
	}

	if all {
		for _, r := range hf.Browse(vram, ram) {
			line("", r)
		}
		return
	}
	recs := hf.Recommend(vram, ram)
	for _, p := range hf.Picks(recs) {
		line(p.Label+":", p.Rec)
		fmt.Println()
	}
	fmt.Printf("  %d of %d models fit this machine. Everything: llamaherd download list --all\n", len(recs), hf.FamilyCount())
	fmt.Println("  Any other repo:  llamaherd download files <owner/name>")
}

// downloadToken shows or stores the Hugging Face token (in-process; no daemon).
func downloadToken(args []string) int {
	if len(args) > 0 && (args[0] == "--clear" || args[0] == "-") {
		if err := settings.SaveHFToken(""); err != nil {
			fmt.Fprintln(os.Stderr, "token: "+err.Error())
			return 1
		}
		fmt.Println("token cleared")
		return 0
	}
	if len(args) > 0 {
		if err := settings.SaveHFToken(args[0]); err != nil {
			fmt.Fprintln(os.Stderr, "token: "+err.Error())
			return 1
		}
		fmt.Println("token saved to " + settings.HFTokenFile())
		return 0
	}
	src := settings.HFTokenSource()
	if src == "" {
		fmt.Println("no Hugging Face token configured (anonymous); set one with `llamaherd download token <token>`")
		return 0
	}
	fmt.Printf("token configured (source: %s)\n", src)
	return 0
}

// downloadStatus prints the current download state.
func downloadStatus() int {
	if err := ensureDaemon(); err != nil {
		fmt.Fprintln(os.Stderr, "download: "+err.Error())
		return 1
	}
	body, _, err := get("/download")
	if err != nil {
		fmt.Fprintln(os.Stderr, "download: "+err.Error())
		return 1
	}
	var st dlState
	if err := json.Unmarshal(body, &st); err != nil {
		fmt.Fprintln(os.Stderr, "download: "+err.Error())
		return 1
	}
	if st.Status == "" || st.Status == "idle" {
		fmt.Println("no download in progress")
		return 0
	}
	fmt.Printf("%s  %s/%s  %.1f%%  %s\n", st.Status, st.Repo, st.File, st.Percent, humanBytes(st.Received)+"/"+humanBytes(st.Total))
	if st.Error != "" {
		fmt.Fprintln(os.Stderr, "error: "+st.Error)
		return 1
	}
	return 0
}

// downloadCancel stops the active download.
func downloadCancel() int {
	if err := ensureDaemon(); err != nil {
		fmt.Fprintln(os.Stderr, "download: "+err.Error())
		return 1
	}
	body, _, err := post("/download/cancel", map[string]any{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "download: "+err.Error())
		return 1
	}
	var r struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	_ = json.Unmarshal(body, &r)
	if !r.OK {
		fmt.Fprintln(os.Stderr, "download: "+strings.TrimSpace(r.Error))
		return 1
	}
	fmt.Println("cancel requested")
	return 0
}

// downloadFiles lists the .gguf files of a Hugging Face repo with their sizes
// (no daemon needed).
func downloadFiles(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: llamaherd download files <owner/name | hf.co URL>")
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	l, err := hf.ListRepo(ctx, args[0], hf.Token())
	if err != nil {
		fmt.Fprintln(os.Stderr, "download: "+err.Error())
		return 1
	}
	fmt.Printf("%s\n\n", l.Repo)
	row := func(f hf.RepoFile, note string) {
		fmt.Printf("  %-14s %9s  %s%s\n", f.Quant, fmtGiB(f.Size), f.Name, note)
	}
	for _, f := range l.Models {
		note := ""
		if f.Parts > 1 {
			note = fmt.Sprintf("   (%d parts — not downloadable yet)", f.Parts)
		}
		row(f, note)
	}
	for _, f := range l.MMProjs {
		row(f, "   [vision projector]")
	}
	for _, f := range l.MTPDrafts {
		row(f, "   [MTP draft]")
	}
	fmt.Printf("\ndownload: llamaherd download %s <file> [--no-vision] [--no-mtp]\n", l.Repo)
	return 0
}

func fmtGiB(n int64) string {
	if n < 1<<30 {
		return fmt.Sprintf("%.0f MiB", float64(n)/(1<<20))
	}
	return fmt.Sprintf("%.2f GiB", float64(n)/(1<<30))
}

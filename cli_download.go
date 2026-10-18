package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
		printCatalog()
		return 0
	case "help", "-h", "--help":
		fmt.Println("usage: llamaherd download list|<ref> [<file>] [--vision <file>]|status|cancel|token [<t>|--clear]")
		return 0
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

// printCatalog ranks the curated models against this machine (no daemon).
func printCatalog() {
	vram := control.TotalVramMB()
	ram := totalRAMMB()
	fmt.Printf("Curated models for this machine — VRAM %.1f GiB, RAM %.1f GiB\n\n",
		float64(vram)/1024, float64(ram)/1024)
	for _, r := range hf.Recommend(vram, ram) {
		e := r.Entry
		mark := map[string]string{"gpu": "gpu", "cpu": "cpu", "too-big": " - "}[r.Fit]
		star := " "
		if e.Starter {
			star = "*"
		}
		vis := ""
		if e.Vision {
			vis += "  vision"
		}
		if e.MTP {
			vis += "  mtp"
		}
		fmt.Printf("  [%s]%s %-44s %6.1f GiB  ~%dk ctx%s\n",
			mark, star, e.Name, float64(e.Bytes)/(1<<30), e.Context/1000, vis)
		fmt.Printf("          %s\n", e.Repo)
		fmt.Printf("          needs ~%.1f GiB VRAM or ~%.1f GiB RAM — %s\n", e.MinVRAMGiB, e.MinRAMGiB, r.Why)
	}
	fmt.Println("\n  * = starter model.  Download with: llamaherd download <repo>")
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

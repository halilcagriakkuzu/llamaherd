package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"llamaherd/api"
)

// dlState mirrors the App's DownloadState for CLI polling.
type dlState struct {
	Active   bool     `json:"active"`
	Repo     string   `json:"repo"`
	File     string   `json:"file"`
	Received int64    `json:"received"`
	Total    int64    `json:"total"`
	Percent  float64  `json:"percent"`
	SpeedBps float64  `json:"speedBps"`
	Status   string   `json:"status"`
	Error    string   `json:"error"`
	Message  string   `json:"message"`
	Files    []string `json:"files"`
}

// ensureDaemon makes sure the control API is up, spawning the headless daemon
// once if needed (same behaviour as `ctl`).
func ensureDaemon() error {
	if _, _, err := get("/status"); err == nil {
		return nil
	}
	fmt.Fprintln(os.Stderr, "download: control API not reachable, starting headless daemon...")
	cmd := exec.Command(daemonSelf(), "daemon")
	cmd.SysProcAttr = daemonSysProcAttr()
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "download: cannot start daemon:", err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if conn, err := net.DialTimeout("tcp", api.Addr(), 300*time.Millisecond); err == nil {
			conn.Close()
			time.Sleep(500 * time.Millisecond)
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("daemon did not come up within 15s")
}

// downloadStart POSTs a download and streams progress until it finishes.
func downloadStart(args []string) int {
	ref, file, vision, mtp := "", "", "", ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--vision":
			if i+1 < len(args) {
				vision = args[i+1]
				i++
			}
		case "--no-vision":
			vision = "none"
		case "--mtp":
			if i+1 < len(args) {
				mtp = args[i+1]
				i++
			}
		case "--no-mtp":
			mtp = "none"
		case "--file":
			if i+1 < len(args) {
				file = args[i+1]
				i++
			}
		default:
			switch {
			case ref == "":
				ref = args[i]
			case file == "":
				file = args[i]
			}
		}
	}
	if ref == "" {
		fmt.Fprintln(os.Stderr, "download: a repo (owner/name) or hf.co URL is required")
		return 2
	}
	if err := ensureDaemon(); err != nil {
		fmt.Fprintln(os.Stderr, "download: "+err.Error())
		return 1
	}
	body, _, err := post("/download", map[string]string{"ref": ref, "file": file, "vision": vision, "mtp": mtp})
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

	last := ""
	for {
		time.Sleep(500 * time.Millisecond)
		b, _, err := get("/download")
		if err != nil {
			fmt.Fprintln(os.Stderr, "download: "+err.Error())
			return 1
		}
		var st dlState
		if json.Unmarshal(b, &st) != nil {
			continue
		}
		switch st.Status {
		case "downloading", "verifying":
			line := fmt.Sprintf("%s %s %s/%s %s/s", st.Status, progressBar(st.Percent),
				humanBytes(st.Received), humanBytes(st.Total), humanBytes(int64(st.SpeedBps)))
			if line != last {
				fmt.Fprintf(os.Stderr, "\r  %-88s", line)
				last = line
			}
		case "done":
			fmt.Fprintf(os.Stderr, "\r  %-88s\n", "done")
			for _, f := range st.Files {
				fmt.Println("  " + f)
			}
			fmt.Println("installed — a preset was written and the model selected")
			return 0
		case "cancelled":
			fmt.Fprintf(os.Stderr, "\r  %-88s\n", "cancelled")
			return 1
		default:
			fmt.Fprintf(os.Stderr, "\r  %-88s\n", "download failed")
			if st.Error != "" {
				fmt.Fprintln(os.Stderr, "error: "+st.Error)
			}
			return 1
		}
	}
}

// progressBar renders a 24-cell bar for a percentage.
func progressBar(pct float64) string {
	const width = 24
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(pct / 100 * width)
	return "[" + strings.Repeat("#", filled) + strings.Repeat(".", width-filled) + "]"
}

// humanBytes formats a byte count with binary units.
func humanBytes(n int64) string {
	if n <= 0 {
		return "0 B"
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

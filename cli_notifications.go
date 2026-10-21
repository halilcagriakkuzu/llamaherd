package main

// cli_notifications.go — read and change the host OS notification toggles.

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"llamaherd/settings"
)

func runNotificationsCLI(args []string) int {
	if err := ensureDaemon(); err != nil {
		fmt.Fprintln(os.Stderr, "notifications: "+err.Error())
		return 1
	}
	body, _, err := get("/notifications")
	if err != nil {
		fmt.Fprintln(os.Stderr, "notifications: "+err.Error())
		return 1
	}
	var s settings.NotificationSettings
	if err := json.Unmarshal(body, &s); err != nil {
		fmt.Fprintln(os.Stderr, "notifications: "+err.Error())
		return 1
	}
	args = args[1:]
	if len(args) == 0 || args[0] == "status" {
		printNotificationSettings(s)
		return 0
	}
	switch args[0] {
	case "on", "off":
		s.Enabled = args[0] == "on"
	case "ready", "crash", "download", "benchmark":
		if len(args) < 2 || (args[1] != "on" && args[1] != "off") {
			fmt.Fprintln(os.Stderr, "usage: llamaherd notifications "+args[0]+" on|off")
			return 2
		}
		on := args[1] == "on"
		switch args[0] {
		case "ready":
			s.ServerReady = on
		case "crash":
			s.ServerCrash = on
		case "download":
			s.Download = on
		case "benchmark":
			s.Benchmark = on
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: llamaherd notifications [status|on|off|ready|crash|download|benchmark on|off]")
		return 2
	}
	reply, _, err := post("/notifications", s)
	if err != nil {
		fmt.Fprintln(os.Stderr, "notifications: "+err.Error())
		return 1
	}
	var r struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	_ = json.Unmarshal(reply, &r)
	if !r.OK {
		fmt.Fprintln(os.Stderr, "notifications: "+strings.TrimSpace(r.Error))
		return 1
	}
	body, _, err = get("/notifications")
	if err == nil {
		_ = json.Unmarshal(body, &s)
	}
	printNotificationSettings(s)
	return 0
}

func printNotificationSettings(s settings.NotificationSettings) {
	onOff := func(b bool) string {
		if b {
			return "on"
		}
		return "off"
	}
	fmt.Printf("notifications: %s\n", onOff(s.Enabled))
	fmt.Printf("  server ready: %s\n  server crash: %s\n  download: %s\n  benchmark: %s\n",
		onOff(s.ServerReady), onOff(s.ServerCrash), onOff(s.Download), onOff(s.Benchmark))
}

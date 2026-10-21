package main

// cli.go — `llamaherdctl`-style subcommands on the llamaherd binary.
// Everything talks to the local control API (api.Addr()); nothing talks to
// llama-server directly, so the GUI and CLI share ONE controller instance.
//
//	if nothing listens on the control port yet, the first ctl command spawns
//	a background HEADLESS daemon (same binary, `llamaherd daemon`) and retries.
//
// Usage:
//	llamaherd list [alias-substring]
//	llamaherd status
//	llamaherd open [alias] [--preset name]
//	llamaherd stop
//	llamaherd save | load
//	llamaherd config add <alias> <name>    [--set basic.port=8090 --set advanced.fit=on]
//	llamaherd config update <alias> <name> [--set basic.port=8090 ...]
//	llamaherd config remove <alias> <name>
//	llamaherd model rename <alias> [--name N] [--alias A]
//	llamaherd model move <alias> [<category/path>]   (no path = remove from category)
//	llamaherd category create|delete <a/b/c>
//	llamaherd category rename <a/b/c> <new-name>
//	llamaherd daemon                       (headless, serves the control API)

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"llamaherd/api"
	"llamaherd/doctor"
	"llamaherd/schema"
	"llamaherd/settings"
)

// version is the LlamaHerd build version. Override at build time with
// -ldflags "-X main.version=v1.2.3".
var version = "dev"

// runCLI dispatches non-GUI invocations. Returns process exit code.
func runCLI(args []string) int {
	if len(args) == 0 {
		return -1 // no args -> GUI
	}
	if len(args) >= 2 && (args[1] == "-h" || args[1] == "--help") {
		if u := subcommandUsage(args[0]); u != "" {
			fmt.Println(u)
			return 0
		}
	}
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Println("llamaherd " + version)
		return 0
	case "daemon":
		return runDaemon()
	case "doctor":
		return runDoctor(args)
	case "download":
		return runDownload(args)
	case "logs":
		return runLogsCLI(args)
	case "notifications":
		return runNotificationsCLI(args)
	case "completion":
		return runCompletion(args)
	case "list", "status", "open", "stop", "restart", "rescan", "save", "load", "bench":
		return ctl(args)
	case "model", "category":
		if len(args) >= 2 {
			return ctl(args)
		}
		fmt.Fprintln(os.Stderr, subcommandUsage(args[0]))
		return 2
	case "config":
		if len(args) >= 2 {
			switch args[1] {
			case "add", "update", "remove", "rename", "reset", "export", "autofit":
				return ctl(args)
			}
		}
		fallthrough
	default:
		fmt.Fprintln(os.Stderr, strings.Join(helpLines(), "\n"))
		if args[0] != "help" && args[0] != "-h" && args[0] != "--help" {
			return 2
		}
		return 0
	}
}

func helpLines() []string {
	return []string{
		"usage: llamaherd <command> [args]",
		"",
		"  (no args)            open the GUI window",
		"  list [--json] [substr]  list models (and their presets); optional alias filter",
		"  status [--summary]   print the full snapshot (JSON) or a one-screen summary",
		"  open [alias] [--preset name]",
		"                       start the real llama-server (alias matches by substring)",
		"  stop                 stop the managed server",
		"  restart              stop and start with the current form settings",
		"  rescan               re-walk the models directory",
		"  logs [-f] [--tail n] show the app/server log (follow with -f)",
		"  notifications [status|on|off|ready|crash|download|benchmark on|off]",
		"                       host OS notifications (ready/crash/download/benchmark)",
		"  bench [list|clear|export]   benchmark the running model/config, list/clear/export results",
		"  save | load          save/load the current selection's configs/<alias>.json",
		"  config add <alias> <name> [--set basic.port=8090 --set advanced.fit=on ...]",
		"	 add a NEW named preset (seeded from the alias's first preset,",
		"	 then --set overrides); bools: true/false, strings quoted",
		"  config update <alias> <name> [--set key=value ...]",
		"                       merge fields into an existing preset (same --set syntax)",
		"  config remove <alias> <name>",
		"                       delete a preset (a saved file survives if needed)",
		"  config rename <alias> <old-name> <new-name>",
		"                       rename a preset in place",
		"  config reset [<name>] | config export",
		"                       reset to the auto-fitted model default / export preset JSON",
		"  config autofit       fit context and GPU layers to this machine",
		"  model rename <alias> [--name N] [--alias A]",
		"                       change a model's display name and/or alias (moves its configs)",
		"  model move <alias> [<a/b/c>]",
		"                       put a model in a category (no path: back to Uncategorized)",
		"  model favorite <alias> [--off] | model delete <alias>",
		"                       pin/unpin a model; delete its .gguf from disk",
		"  model import <file.gguf|dir>... [--mmproj <file.gguf>] [--move] [--allow-duplicate] [--check]",
		"                       import local models (one shared vision projector) into the models dir;",
		"                       files already in the library are skipped; --check only previews",
		"  category create|delete <a/b/c> | category rename <a/b/c> <new-name>",
		"                       manage category folders (deleting keeps the models)",
		"  daemon               headless core serving only the control API",
		"  completion bash|zsh|fish",
		"                       print a shell completion script",
		"  version              print the version and exit",
		"  doctor [--json]      check the environment (binary, GPU, libs, models, port)",
		"  download list        curated models that fit this machine (VRAM/RAM)",
		"  download <ref> [<file>] [--vision <file>]",
		"                       download a HF model (owner/name or hf.co URL; resumable)",
		"  download status|cancel",
		"                       show progress / cancel the active download",
		"  download token [<t>] show or set the Hugging Face token (--clear to remove)",
		"",
		"The control API listens on http://" + api.Addr() + " (loopback only).",
	}
}

// subcommandUsage returns a one-line usage string for a subcommand ("" when
// unknown, so the caller falls through).
func subcommandUsage(sub string) string {
	m := map[string]string{
		"list":          "usage: llamaherd list [alias-substring]",
		"status":        "usage: llamaherd status",
		"open":          "usage: llamaherd open [alias] [--preset name]",
		"stop":          "usage: llamaherd stop",
		"restart":       "usage: llamaherd restart",
		"rescan":        "usage: llamaherd rescan",
		"bench":         "usage: llamaherd bench [list|clear|export [--format csv|json] [--out file]]   # requires a running server",
		"logs":          "usage: llamaherd logs [-f] [--tail n]",
		"notifications": "usage: llamaherd notifications [status|on|off|ready|crash|download|benchmark on|off]",
		"completion":    "usage: llamaherd completion bash|zsh|fish",
		"save":          "usage: llamaherd save",
		"load":          "usage: llamaherd load",
		"daemon":        "usage: llamaherd daemon            # headless: serve only the control API",
		"doctor":        "usage: llamaherd doctor [--json]",
		"download":      "usage: llamaherd download list|<ref> [<file>] [--vision <file>]|status|cancel|token [<t>|--clear]",
		"config":        "usage: llamaherd config add|update|remove|rename <alias> <name> | config reset [<name>] | config export | config autofit",
		"model":         "usage: llamaherd model rename <alias> [--name N] [--alias A] | model move <alias> [<a/b/c>] | model favorite <alias> [--off] | model delete <alias> | model import <file.gguf|dir>... [--mmproj <file>] [--move] [--allow-duplicate] [--check]",
		"category":      "usage: llamaherd category create|delete <a/b/c> | category rename <a/b/c> <new-name>",
		"version":       "usage: llamaherd version          # print the version and exit",
	}
	return m[sub]
}

// ctl connects to the control API, spawning a headless daemon when needed.
func ctl(args []string) int {
	body, status, err := ctlRequest(args)
	if err != nil {
		if derr := ensureDaemon(); derr != nil {
			fmt.Fprintln(os.Stderr, "ctl: "+derr.Error())
			return 1
		}
		body, status, err = ctlRequest(args)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "ctl: "+err.Error())
		return 1
	}
	fmt.Println(string(body))
	if status >= 400 {
		return 1
	}
	return 0
}

// ctlRequest builds the HTTP call for a subcommand; on success it always
// returns a printable body (stdout format).
func ctlRequest(args []string) ([]byte, int, error) {
	sub := args[0]
	switch sub {
	case "list":
		fs := flag.NewFlagSet("list", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		jsonOut := fs.Bool("json", false, "machine-readable output")
		if err := fs.Parse(args[1:]); err != nil {
			return nil, 2, err
		}
		substr := ""
		if rest := fs.Args(); len(rest) > 0 {
			substr = rest[0]
		}
		data, status, err := get("/models")
		if err != nil {
			return nil, status, err
		}
		if status >= 400 {
			return data, status, nil
		}
		var mm struct {
			Models []schema.Model `json:"models"`
		}
		_ = json.Unmarshal(data, &mm) // best effort: an unparsable body yields no aliases
		if *jsonOut {
			return listJSON(mm.Models, substr)
		}
		var out []string
		for _, m := range mm.Models {
			if m.IsMMProj {
				out = append(out, fmt.Sprintf("  [mmproj] %s", m.Display))
				continue
			}
			if substr != "" && !strings.Contains(m.Alias, strings.ToLower(substr)) {
				continue
			}
			pdata, st, err := get("/presets?alias=" + m.Alias)
			if err != nil || st >= 400 {
				pdata = []byte("[]")
			}
			var pp struct {
				Presets []schema.Preset `json:"presets"`
			}
			_ = json.Unmarshal(pdata, &pp) // best effort: no presets listed on parse failure
			names := []string{}
			for _, p := range pp.Presets {
				names = append(names, p.Name)
			}
			out = append(out, fmt.Sprintf("  %s\n    path: %s\n    presets: %s",
				m.Alias, m.Path, strings.Join(names, ", ")))
		}
		return []byte(strings.Join(out, "\n")), 0, nil
	case "status":
		fs := flag.NewFlagSet("status", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		summary := fs.Bool("summary", false, "human-readable one-screen summary")
		_ = fs.Bool("json", false, "JSON output (default)")
		if err := fs.Parse(args[1:]); err != nil {
			return nil, 2, err
		}
		data, status, err := get("/status")
		if err != nil || status >= 400 || !*summary {
			return data, status, err
		}
		txt, err := statusSummary(data)
		if err != nil {
			return nil, 1, err
		}
		return []byte(txt), 0, nil
	case "open":
		fs := flag.NewFlagSet("open", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		preset := fs.String("preset", "", "preset name within the resolved alias")
		if err := fs.Parse(args[1:]); err != nil {
			return nil, 2, err
		}
		alias := ""
		rest := fs.Args()
		if len(rest) > 0 {
			alias = rest[0]
		}
		return post("/open", map[string]string{"alias": alias, "preset": *preset})
	case "stop":
		return post("/stop", map[string]any{})
	case "restart":
		return post("/restart", map[string]any{})
	case "rescan":
		return post("/rescan", map[string]any{})
	case "bench":
		if len(args) > 1 {
			switch args[1] {
			case "export":
				format, out := "csv", ""
				for i := 2; i < len(args); i++ {
					switch args[i] {
					case "--format", "-f":
						if i+1 < len(args) {
							format = args[i+1]
							i++
						}
					case "--out", "-o":
						if i+1 < len(args) {
							out = args[i+1]
							i++
						}
					}
				}
				data, status, err := get("/benchmarks/export?format=" + format)
				if err != nil || status >= 400 {
					return data, status, err
				}
				if out == "" {
					return data, status, nil
				}
				if err := os.WriteFile(out, data, 0o644); err != nil {
					return []byte(err.Error()), 400, nil
				}
				return []byte("wrote " + out), 0, nil
			case "list":
				return get("/benchmarks")
			case "clear":
				return post("/benchmarks/clear", map[string]any{})
			}
		}
		data, status, err := post("/benchmark", map[string]any{})
		if err != nil || status >= 400 {
			return data, status, err
		}
		var r struct {
			OK     bool             `json:"ok"`
			Error  string           `json:"error"`
			Result schema.Benchmark `json:"result"`
		}
		if json.Unmarshal(data, &r) != nil {
			return data, status, nil
		}
		if !r.OK {
			msg := r.Error
			if msg == "" {
				msg = string(data)
			}
			return []byte(msg), 400, nil
		}
		b := r.Result
		return []byte(fmt.Sprintf(
			"%s [%s]  generation %.1f tok/s · prompt %.1f tok/s · ctx %d · -ngl %s · peak VRAM %d MB",
			b.Model, b.Preset, b.GenTokS, b.PromptTokS, b.Context, b.GPULayers, b.PeakVRAMMB)), 0, nil
	case "save":
		return post("/save", map[string]any{})
	case "load":
		return post("/load", map[string]any{})
	case "model":
		return libraryModelRequest(args)
	case "category":
		return libraryCategoryRequest(args)
	case "config":
		if len(args) < 2 {
			return []byte("config add|update|remove|rename|reset|export required"), 400, nil
		}
		action := args[1]
		if action == "export" {
			return get("/config/export")
		}
		if action == "autofit" {
			return post("/config/autofit", map[string]any{})
		}
		if action == "reset" {
			name := ""
			if len(args) >= 3 {
				name = args[2]
			}
			return post("/config/reset", map[string]string{"name": name})
		}
		if len(args) < 3 {
			return []byte("config add|update|remove|rename|reset required"), 400, nil
		}
		var req struct {
			Alias  string         `json:"alias"`
			Name   string         `json:"name"`
			Config *schema.Config `json:"config"`
		}
		var sets []string
		switch action {
		case "add", "update":
			if len(args) < 4 {
				return []byte("usage: config " + action + " <alias> <name> [--set k=v ...]"), 400, nil
			}
			req.Alias = args[2]
			req.Name = args[3]
			fs := flag.NewFlagSet("config-"+action, flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			fs.Var(&setList{&sets}, "set", "--set part.field=value (repeatable)")
			if err := fs.Parse(args[4:]); err != nil {
				return nil, 2, err
			}
			// Fetch the current preset (or first for add) as the base.
			base := schema.Config{}
			bStatus := 0
			{
				pdata, st, err := get("/presets?alias=" + req.Alias)
				if err == nil && st < 400 {
					var pp struct {
						Presets []schema.Preset `json:"presets"`
					}
					_ = json.Unmarshal(pdata, &pp)
					for _, p := range pp.Presets {
						if action == "update" && p.Name == req.Name || action == "add" {
							base = p.AsConfig()
							break
						}
					}
					bStatus = st
				}
			}
			_ = bStatus
			patch := schema.Config{}
			for _, kv := range sets {
				if err := applySet(&patch, kv); err != nil {
					return []byte("bad --set " + kv + ": " + err.Error()), 400, nil
				}
			}
			merged := patch
			if base.Basic.Model != "" {
				merged = mergeConfig(base, patch)
			} else if len(sets) > 0 && action == "add" && needsModel(merged) {
				return []byte("refusing to add a preset without --set basic.model=<path>"), 400, nil
			}
			req.Config = &merged
			return post("/config/"+action, req)
		case "remove":
			if len(args) < 4 {
				return []byte("usage: config remove <alias> <name>"), 400, nil
			}
			req.Alias = args[2]
			req.Name = args[3]
			return post("/config/remove", req)
		case "rename":
			if len(args) < 5 {
				return []byte("usage: config rename <alias> <old-name> <new-name>"), 400, nil
			}
			return post("/config/rename", map[string]string{
				"alias": args[2], "name": args[3], "newName": args[4],
			})
		case "reset":
			name := ""
			if len(args) >= 3 {
				name = args[2]
			}
			return post("/config/reset", map[string]string{"name": name})
		case "export":
			return get("/config/export")
		}
		return []byte("unknown config subcommand"), 400, nil
	}
	return []byte("unknown command " + sub), 400, nil
}

func needsModel(c schema.Config) bool { return c.Basic.Model == "" }

// applySet parses "part.field=value" into a Config patch. Values: bare,
// quoted, int, or bool.
func applySet(cfg *schema.Config, kv string) error {
	eq := strings.Index(kv, "=")
	if eq <= 0 {
		return errors.New("expected key=value")
	}
	key, val := strings.TrimSpace(kv[:eq]), strings.TrimSpace(kv[eq+1:])
	key = strings.ToLower(key)
	val = strings.Trim(val, `"'`)
	var bs, ad *string // routing: which part
	switch {
	case strings.HasPrefix(key, "basic."):
		r := key[len("basic."):]
		_ = bs
		switch r {
		case "model":
			cfg.Basic.Model = val
		case "alias":
			cfg.Basic.Alias = val
		case "host":
			cfg.Basic.Host = val
		case "port":
			n, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			cfg.Basic.Port = n
		case "context":
			n, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			cfg.Basic.Context = n
		case "gpulayers":
			cfg.Basic.GPULayers = val
		case "parallelslots":
			n, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			cfg.Basic.ParallelSlots = n
		case "flashattention":
			cfg.Basic.FlashAttention = val
		case "kcache":
			cfg.Basic.KCache = val
		case "vcache":
			cfg.Basic.VCache = val
		case "jinja":
			cfg.Basic.Jinja = boolVal(val)
		case "reasoningfmt":
			cfg.Basic.ReasoningFmt = val
		case "reasoningbudget":
			n, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			cfg.Basic.ReasoningBudget = n
		default:
			return fmt.Errorf("unknown basic field %q", r)
		}
	case strings.HasPrefix(key, "advanced."):
		r := key[len("advanced."):]
		_ = ad
		switch r {
		case "spectype":
			cfg.Advanced.SpecType = val
		case "draftnmax":
			n, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			cfg.Advanced.DraftNMax = n
		case "draftngl":
			cfg.Advanced.DraftNgl = val
		case "drafttypek":
			cfg.Advanced.DraftTypeK = val
		case "drafttypev":
			cfg.Advanced.DraftTypeV = val
		case "mmproj":
			cfg.Advanced.MMProj = val
		case "nommprojoffload":
			if boolVal(val) {
				cfg.Advanced.NoMMProjOffload = true
			}
		case "fit":
			cfg.Advanced.Fit = val
		case "perf":
			if boolVal(val) {
				cfg.Advanced.Perf = true
			}
		case "sleepidle":
			n, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			cfg.Advanced.SleepIdle = n
		case "cacheram":
			n, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			cfg.Advanced.CacheRAM = n
		case "mlock":
			if boolVal(val) {
				cfg.Advanced.Mlock = true
			}
		case "nommap":
			if boolVal(val) {
				cfg.Advanced.NoMmap = true
			}
		case "cputhreads":
			n, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			cfg.Advanced.CPUThreads = n
		case "ubatch", "batch":
			n, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			if r == "ubatch" {
				cfg.Advanced.UBatch = n
			} else {
				cfg.Advanced.Batch = n
			}
		case "overridetensor":
			cfg.Advanced.OverrideTensor = val
		default:
			return fmt.Errorf("unknown advanced field %q", r)
		}
	default:
		return errors.New("key must start with basic. or advanced")
	}
	return nil
}

func boolVal(s string) bool {
	switch strings.ToLower(s) {
	case "true", "1", "yes", "on":
		return true
	}
	return false
}

// setList is a flag.Value accumulating repeated --set flags.
type setList struct{ v *[]string }

func (s *setList) String() string {
	if s.v == nil {
		return ""
	}
	return strings.Join(*s.v, ",")
}
func (s *setList) Set(kv string) error { *s.v = append(*s.v, kv); return nil }

// get is a thin JSON GET helper (path may carry a query string).
func get(path string) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, "http://"+api.Addr()+path, nil)
	if err != nil {
		return nil, 0, err
	}
	return do(req)
}

func post(path string, body any) ([]byte, int, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequest(http.MethodPost, "http://"+api.Addr()+path, bytes.NewReader(data))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	return do(req)
}

// do sends req, attaching the control-API bearer token when one is configured.
func do(req *http.Request) ([]byte, int, error) {
	if t := settings.APIToken(); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return b, resp.StatusCode, err
}

// runDaemon boots the headless core: init + API (blocking). Used both when
// ctl-spawned and when started manually.
func runDaemon() int {
	a := NewApp()
	a.startupHeadless()
	if err := api.Serve(a); err != nil {
		fmt.Fprintln(os.Stderr, "daemon: "+err.Error())
		return 1
	}
	return 0
}

var doctorMark = map[doctor.Status]string{
	doctor.StatusOK:   " ok ",
	doctor.StatusInfo: "info",
	doctor.StatusWarn: "warn",
	doctor.StatusFail: "FAIL",
	doctor.StatusSkip: "skip",
}

func printDoctor(rep doctor.Report) {
	fmt.Printf("LlamaHerd doctor — %s/%s (%s)\n\n", rep.OS, rep.Arch, rep.Platform)
	for _, c := range rep.Checks {
		mark := doctorMark[c.Status]
		if mark == "" {
			mark = string(c.Status)
		}
		fmt.Printf("[%s] %s\n", mark, c.Title)
		if c.Detail != "" {
			fmt.Printf("         %s\n", c.Detail)
		}
		for _, line := range strings.Split(c.Fix, "\n") {
			if strings.TrimSpace(line) != "" {
				fmt.Printf("         fix: %s\n", line)
			}
		}
	}
	fmt.Printf("\n%s\n", rep.Summary)
}

// runDoctor runs the environment checks and prints them (human or JSON).
// Exits 1 when any check fails.
func runDoctor(args []string) int {
	jsonOut := false
	for _, a := range args[1:] {
		switch a {
		case "--json":
			jsonOut = true
		case "-h", "--help":
			fmt.Println("usage: llamaherd doctor [--json]")
			return 0
		default:
			fmt.Fprintln(os.Stderr, "doctor: unknown argument "+a)
			return 2
		}
	}
	rep := doctor.Run()
	if jsonOut {
		data, _ := json.MarshalIndent(rep, "", "  ")
		fmt.Println(string(data))
	} else {
		printDoctor(rep)
	}
	if !rep.OK {
		return 1
	}
	return 0
}

// daemonSelf returns this executable's path (for spawning the headless
// daemon).
func daemonSelf() string {
	if self, err := os.Executable(); err == nil {
		return self
	}
	return os.Args[0]
}

// splitCategory turns "a/b/c" (or "a › b") into path segments.
func splitCategory(s string) []string {
	var out []string
	for _, p := range strings.FieldsFunc(s, func(r rune) bool { return r == '/' || r == '›' }) {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// libraryModelRequest handles `model rename|move`.
func libraryModelRequest(args []string) ([]byte, int, error) {
	switch args[1] {
	case "rename":
		fs := flag.NewFlagSet("model rename", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		name := fs.String("name", "", "display name (empty: keep the file-derived name)")
		alias := fs.String("alias", "", "new alias (empty: keep the file-derived alias)")
		if len(args) < 3 {
			return nil, 2, errors.New("usage: llamaherd model rename <alias> [--name N] [--alias A]")
		}
		if err := fs.Parse(args[3:]); err != nil {
			return nil, 2, err
		}
		return post("/model/rename", map[string]string{"model": args[2], "name": *name, "alias": *alias})
	case "move":
		if len(args) < 3 {
			return nil, 2, errors.New("usage: llamaherd model move <alias> [<category/path>]")
		}
		path := []string{}
		if len(args) > 3 {
			path = splitCategory(strings.Join(args[3:], " "))
		}
		return post("/model/category", map[string]any{"model": args[2], "category": path})
	case "delete":
		if len(args) < 3 {
			return nil, 2, errors.New("usage: llamaherd model delete <alias>")
		}
		return post("/model/delete", map[string]string{"model": args[2]})
	case "import":
		req, check, err := parseImportArgs(args[2:])
		if err != nil {
			return nil, 2, err
		}
		if check {
			data, status, err := post("/model/import/check", req)
			if err != nil || status >= 400 {
				return data, status, err
			}
			return importCheckSummary(data)
		}
		data, status, err := post("/model/import", req)
		if err != nil || status >= 400 {
			return data, status, err
		}
		return importReportSummary(data)
	case "favorite":
		if len(args) < 3 {
			return nil, 2, errors.New("usage: llamaherd model favorite <alias> [--off]")
		}
		fav := true
		for _, a := range args[3:] {
			if a == "--off" || a == "off" {
				fav = false
			}
		}
		return post("/model/favorite", map[string]any{"model": args[2], "favorite": fav})
	}
	return nil, 2, errors.New("usage: llamaherd model rename|move|favorite|delete <alias> (see llamaherd help)")
}

// libraryCategoryRequest handles `category create|rename|delete`.
func libraryCategoryRequest(args []string) ([]byte, int, error) {
	if len(args) < 3 {
		return nil, 2, errors.New("usage: llamaherd category create|delete <a/b/c> | category rename <a/b/c> <new-name>")
	}
	path := splitCategory(args[2])
	switch args[1] {
	case "create":
		return post("/category/create", map[string]any{"path": path})
	case "delete":
		return post("/category/delete", map[string]any{"path": path})
	case "rename":
		if len(args) < 4 {
			return nil, 2, errors.New("usage: llamaherd category rename <a/b/c> <new-name>")
		}
		return post("/category/rename", map[string]any{"path": path, "name": args[3]})
	}
	return nil, 2, errors.New("usage: llamaherd category create|rename|delete <a/b/c> (see llamaherd help)")
}

// ---- logs ----

type logsResponse struct {
	Lines   []string `json:"lines"`
	Next    int64    `json:"next"`
	Path    string   `json:"path"`
	Enabled bool     `json:"enabled"`
}

func fetchLogs(tail int, after int64) (logsResponse, error) {
	path := "/logs?tail=" + strconv.Itoa(tail)
	if after >= 0 {
		path += "&after=" + strconv.FormatInt(after, 10)
	}
	data, status, err := get(path)
	if err != nil {
		return logsResponse{}, err
	}
	if status >= 400 {
		return logsResponse{}, fmt.Errorf("%s", strings.TrimSpace(string(data)))
	}
	var r logsResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return logsResponse{}, err
	}
	return r, nil
}

// runLogsCLI implements `llamaherd logs [-f] [--tail n]`.
func runLogsCLI(args []string) int {
	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	follow := fs.Bool("f", false, "follow new lines")
	fs.BoolVar(follow, "follow", false, "follow new lines")
	tail := fs.Int("tail", 200, "number of lines to show")
	if err := fs.Parse(args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, subcommandUsage("logs"))
		return 2
	}
	if *tail < 0 {
		fmt.Fprintln(os.Stderr, "logs: --tail must be >= 0")
		return 2
	}
	if *follow && *tail > 20 {
		*tail = 20 // follow mode starts small, then streams
	}
	r, err := fetchLogs(*tail, -1)
	if err != nil {
		if derr := ensureDaemon(); derr != nil {
			fmt.Fprintln(os.Stderr, "logs: "+derr.Error())
			return 1
		}
		if r, err = fetchLogs(*tail, -1); err != nil {
			fmt.Fprintln(os.Stderr, "logs: "+err.Error())
			return 1
		}
	}
	for _, line := range r.Lines {
		fmt.Println(line)
	}
	if !*follow {
		return 0
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	next := r.Next
	for {
		select {
		case <-ctx.Done():
			return 0
		case <-time.After(time.Second):
		}
		r, err := fetchLogs(0, next)
		if err != nil {
			fmt.Fprintln(os.Stderr, "logs: "+err.Error())
			continue
		}
		for _, line := range r.Lines {
			fmt.Println(line)
		}
		next = r.Next
	}
}

// ---- list --json ----

type listJSONRow struct {
	Alias    string   `json:"alias"`
	Display  string   `json:"display"`
	Path     string   `json:"path"`
	IsMMProj bool     `json:"isMMProj"`
	Presets  []string `json:"presets"`
}

func listJSON(models []schema.Model, substr string) ([]byte, int, error) {
	needle := strings.ToLower(strings.TrimSpace(substr))
	rows := make([]listJSONRow, 0, len(models))
	for _, m := range models {
		if needle != "" &&
			!strings.Contains(strings.ToLower(m.Alias), needle) &&
			!strings.Contains(strings.ToLower(m.Display), needle) &&
			!strings.Contains(strings.ToLower(m.Path), needle) {
			continue
		}
		names := []string{}
		if !m.IsMMProj {
			pdata, st, err := get("/presets?alias=" + m.Alias)
			if err == nil && st < 400 {
				var pp struct {
					Presets []schema.Preset `json:"presets"`
				}
				if json.Unmarshal(pdata, &pp) == nil {
					for _, p := range pp.Presets {
						names = append(names, p.Name)
					}
				}
			}
		}
		rows = append(rows, listJSONRow{
			Alias: m.Alias, Display: m.Display, Path: m.Path,
			IsMMProj: m.IsMMProj, Presets: names,
		})
	}
	data, err := json.MarshalIndent(map[string]any{"models": rows}, "", "  ")
	return data, 0, err
}

// ---- status --summary ----

func statusSummary(data []byte) (string, error) {
	var s struct {
		Status         string   `json:"status"`
		Alias          string   `json:"alias"`
		Dirty          bool     `json:"dirty"`
		PresetNames    []string `json:"presetNames"`
		SelectedPreset int      `json:"selectedPreset"`
		Warning        string   `json:"warning"`
		Server         struct {
			Model     string `json:"model"`
			Context   string `json:"context"`
			GPULayers string `json:"gpuLayers"`
			Listen    string `json:"listen"`
			TokPerSec string `json:"tokPerSec"`
			VRAM      string `json:"vram"`
			CPURAM    string `json:"cpuRam"`
			Uptime    string `json:"uptime"`
		} `json:"server"`
		Download struct {
			Active  bool    `json:"active"`
			File    string  `json:"file"`
			Percent float64 `json:"percent"`
		} `json:"download"`
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return "", err
	}
	var b strings.Builder
	status := s.Status
	if status == "" {
		status = "unknown"
	}
	fmt.Fprintf(&b, "status:  %s\n", status)
	if s.Alias != "" {
		preset := ""
		if s.SelectedPreset >= 0 && s.SelectedPreset < len(s.PresetNames) {
			preset = s.PresetNames[s.SelectedPreset]
		}
		dirty := ""
		if s.Dirty {
			dirty = " (unsaved changes)"
		}
		fmt.Fprintf(&b, "model:   %s [%s]%s\n", s.Alias, preset, dirty)
	}
	if s.Server.Listen != "" {
		parts := []string{s.Server.Listen}
		if s.Server.Context != "" && s.Server.Context != "—" {
			parts = append(parts, "ctx "+s.Server.Context)
		}
		if s.Server.GPULayers != "" && s.Server.GPULayers != "—" {
			parts = append(parts, "ngl "+s.Server.GPULayers)
		}
		if s.Server.TokPerSec != "" && s.Server.TokPerSec != "—" {
			parts = append(parts, s.Server.TokPerSec)
		}
		if s.Server.VRAM != "" && s.Server.VRAM != "—" {
			parts = append(parts, "VRAM "+s.Server.VRAM)
		}
		if s.Server.Uptime != "" && s.Server.Uptime != "00:00:00" {
			parts = append(parts, "up "+s.Server.Uptime)
		}
		fmt.Fprintf(&b, "server:  %s\n", strings.Join(parts, " · "))
	}
	if s.Warning != "" {
		fmt.Fprintf(&b, "warning: %s\n", s.Warning)
	}
	if s.Download.Active {
		fmt.Fprintf(&b, "download: %.0f%% %s\n", s.Download.Percent, s.Download.File)
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

// ---- shell completions ----

const cliWords = "list status open stop restart rescan save load logs bench config model category download doctor daemon completion version help"

func runCompletion(args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, subcommandUsage("completion"))
		return 2
	}
	switch args[1] {
	case "bash":
		fmt.Printf(`# bash completion for llamaherd
_llamaherd() {
  local cur="${COMP_WORDS[COMP_CWORD]}"
  if [ "$COMP_CWORD" -eq 1 ]; then
    COMPREPLY=( $(compgen -W "%s" -- "$cur") )
  fi
}
complete -F _llamaherd llamaherd
`, cliWords)
	case "zsh":
		fmt.Printf("#compdef llamaherd\n_arguments '1:command:(%s)'\n", cliWords)
	case "fish":
		for _, w := range strings.Fields(cliWords) {
			fmt.Printf("complete -c llamaherd -f -n '__fish_use_subcommand' -a %s\n", w)
		}
	default:
		fmt.Fprintln(os.Stderr, subcommandUsage("completion"))
		return 2
	}
	return 0
}

const importUsage = "usage: llamaherd model import <file.gguf|dir>... [--mmproj <file.gguf>] [--move] [--allow-duplicate] [--check]"

// parseImportArgs reads `model import` arguments; flags may appear anywhere
// between the paths. Paths are made absolute because the daemon may run in a
// different working directory.
func parseImportArgs(args []string) (req map[string]any, check bool, err error) {
	var paths []string
	var mmproj string
	var move, allowDup bool
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--move":
			move = true
		case arg == "--allow-duplicate":
			allowDup = true
		case arg == "--check":
			check = true
		case arg == "--mmproj":
			if i+1 >= len(args) {
				return nil, false, errors.New(importUsage)
			}
			i++
			mmproj = args[i]
		case strings.HasPrefix(arg, "--mmproj="):
			mmproj = strings.TrimPrefix(arg, "--mmproj=")
		case strings.HasPrefix(arg, "-"):
			return nil, false, fmt.Errorf("unknown flag %s\n%s", arg, importUsage)
		default:
			abs, err := filepath.Abs(arg)
			if err != nil {
				return nil, false, err
			}
			paths = append(paths, abs)
		}
	}
	if len(paths) == 0 {
		return nil, false, errors.New(importUsage)
	}
	if mmproj != "" {
		if mmproj, err = filepath.Abs(mmproj); err != nil {
			return nil, false, err
		}
	}
	return map[string]any{
		"paths": paths, "mmproj": mmproj, "move": move, "allowDuplicate": allowDup,
	}, check, nil
}

// importReportSummary prints one line per imported file; any failure makes the
// command exit non-zero.
func importReportSummary(data []byte) ([]byte, int, error) {
	var r struct {
		OK     bool                `json:"ok"`
		Error  string              `json:"error"`
		Report schema.ImportReport `json:"report"`
	}
	if json.Unmarshal(data, &r) != nil {
		return data, 0, nil
	}
	var b strings.Builder
	failed := 0
	for _, it := range r.Report.Items {
		switch it.Status {
		case "imported":
			fmt.Fprintf(&b, "imported   %s  (%s)\n", it.Alias, it.Source)
		case "duplicate":
			fmt.Fprintf(&b, "skipped    %s: %s\n", it.Source, it.Message)
		default:
			failed++
			fmt.Fprintf(&b, "error      %s: %s\n", it.Source, it.Message)
		}
	}
	if r.Error != "" {
		failed++
		b.WriteString(r.Error + "\n")
	}
	status := 0
	if failed > 0 || !r.OK {
		status = 400
	}
	return []byte(strings.TrimRight(b.String(), "\n")), status, nil
}

// importCheckSummary prints the duplicate preview of `model import --check`.
func importCheckSummary(data []byte) ([]byte, int, error) {
	var r struct {
		Checks []schema.ImportCheck `json:"checks"`
	}
	if json.Unmarshal(data, &r) != nil {
		return data, 0, nil
	}
	var b strings.Builder
	for _, c := range r.Checks {
		switch {
		case c.Error != "":
			fmt.Fprintf(&b, "error      %s: %s\n", c.Path, c.Error)
		case c.Duplicate:
			fmt.Fprintf(&b, "duplicate  %s: already imported as %s\n", c.Path, c.ExistingAlias)
		default:
			fmt.Fprintf(&b, "new        %s\n", c.Path)
		}
	}
	return []byte(strings.TrimRight(b.String(), "\n")), 0, nil
}

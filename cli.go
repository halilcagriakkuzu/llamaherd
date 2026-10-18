package main

// cli.go — `llamaherdctl`-style subcommands on the llamaherd binary (task 13).
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
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
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
	case "list", "status", "open", "stop", "save", "load":
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
			case "add", "update", "remove":
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
		"  list [substr]        list models (and their presets); optional alias filter",
		"  status               print current server snapshot",
		"  open [alias] [--preset name]",
		"                       start the real llama-server (alias matches by substring)",
		"  stop                 stop the managed server",
		"  save | load          save/load the current selection's configs/<alias>.json",
		"  config add <alias> <name> [--set basic.port=8090 --set advanced.fit=on ...]",
		"	 add a NEW named preset (seeded from the alias's first preset,",
		"	 then --set overrides); bools: true/false, strings quoted",
		"  config update <alias> <name> [--set key=value ...]",
		"                       replace an existing preset (same --set syntax)",
		"  config remove <alias> <name>",
		"                       delete a preset (a saved file survives if needed)",
		"  model rename <alias> [--name N] [--alias A]",
		"                       change a model's display name and/or alias (moves its configs)",
		"  model move <alias> [<a/b/c>]",
		"                       put a model in a category (no path: back to Uncategorized)",
		"  category create|delete <a/b/c> | category rename <a/b/c> <new-name>",
		"                       manage category folders (deleting keeps the models)",
		"  daemon               headless core serving only the control API",
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
		"list":     "usage: llamaherd list [alias-substring]",
		"status":   "usage: llamaherd status",
		"open":     "usage: llamaherd open [alias] [--preset name]",
		"stop":     "usage: llamaherd stop",
		"save":     "usage: llamaherd save",
		"load":     "usage: llamaherd load",
		"daemon":   "usage: llamaherd daemon            # headless: serve only the control API",
		"doctor":   "usage: llamaherd doctor [--json]",
		"download": "usage: llamaherd download list|<ref> [<file>] [--vision <file>]|status|cancel|token [<t>|--clear]",
		"config":   "usage: llamaherd config add|update|remove <alias> <name> [--set key=value ...]",
		"model":    "usage: llamaherd model rename <alias> [--name N] [--alias A] | model move <alias> [<a/b/c>]",
		"category": "usage: llamaherd category create|delete <a/b/c> | category rename <a/b/c> <new-name>",
		"version":  "usage: llamaherd version          # print the version and exit",
	}
	return m[sub]
}

// ctl connects to the control API, spawning a headless daemon when needed.
func ctl(args []string) int {
	body, status, err := ctlRequest(args)
	if err != nil {
		// Try spawning the headless daemon once, then retry.
		fmt.Fprintln(os.Stderr, "ctl: control API not reachable, starting headless daemon...")
		cmd := exec.Command(daemonSelf(), "daemon")
		cmd.SysProcAttr = daemonSysProcAttr()
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err != nil {
			fmt.Fprintln(os.Stderr, "ctl: cannot start daemon:", err)
		}
		started := false
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			conn, derr := net.DialTimeout("tcp", api.Addr(), 300*time.Millisecond)
			if derr == nil {
				conn.Close()
				started = true
				break
			}
			time.Sleep(300 * time.Millisecond)
		}
		if !started {
			fmt.Fprintln(os.Stderr, "ctl: daemon did not come up within 15s")
			return 1
		}
		time.Sleep(500 * time.Millisecond)
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
		substr := ""
		if len(args) > 1 {
			substr = args[1]
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
		return get("/status")
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
	case "save":
		return post("/save", map[string]any{})
	case "load":
		return post("/load", map[string]any{})
	case "model":
		return libraryModelRequest(args)
	case "category":
		return libraryCategoryRequest(args)
	case "config":
		if len(args) < 3 {
			return []byte("config add|update|remove required"), 400, nil
		}
		action := args[1]
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

// modelsModel placeholder removed below (unused) —

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
	}
	return nil, 2, errors.New("usage: llamaherd model rename|move <alias> (see llamaherd help)")
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

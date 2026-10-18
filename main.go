package main

import (
	"embed"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Task 13: subcommands on this binary drive the headless core / control
	// API; no args opens the GUI window (wails.Run below).
	if len(os.Args) > 1 {
		os.Exit(runCLI(os.Args[1:]))
	}
	hideConsole()
	app := NewApp()
	err := wails.Run(&options.App{
		Title:            "LlamaHerd — llama.cpp control panel",
		Width:            1100,
		Height:           720,
		DisableResize:    false,
		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 250, G: 250, B: 252, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

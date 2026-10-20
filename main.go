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
	// Subcommands run the CLI/headless core; no args opens the GUI window.
	if len(os.Args) > 1 {
		os.Exit(runCLI(os.Args[1:]))
	}
	hideConsole()
	app := NewApp()
	err := wails.Run(&options.App{
		Title:            "LlamaHerd — llama.cpp control panel",
		Width:            1240,
		Height:           820,
		MinWidth:         960,
		MinHeight:        640,
		DisableResize:    false,
		DragAndDrop:      &options.DragAndDrop{EnableFileDrop: true},
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

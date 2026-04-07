package main

import (
	"log"
	"os"

	"gioui.org/app"
	"github.com/Go-20255/team-project-malloc4.git/internal/ui"
)


func main() {
	go func() {
		window := new(app.Window)
		err := ui.Loop(window)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()

	
	app.Main()
}

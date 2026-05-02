package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"os"
	"sync"

	"gioui.org/app"
	"github.com/Go-20255/team-project-malloc4.git/internal/ui"
)

// TODO: integrate with GUI?
func networking(ctx context.Context) {
	conn, err := tls.Dial("tcp", "smlavine.com:443", nil)
	if err != nil {
		panic("failed to connect: " + err.Error())
	}
	log.Printf("connection: %s\n", conn.RemoteAddr())
	conn.Close()
}

func main() {
	var headless bool
	// TODO server client
	flag.BoolVar(&headless, "headless", false, "launch without GUI")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var networkingTasks sync.WaitGroup
	networkingTasks.Go(func() { networking(ctx) })

	if headless {
		networkingTasks.Wait()
	} else {
		go func() {
			window := new(app.Window)
			err := ui.Loop(window)
			if err != nil {
				log.Fatal(err)
			}
			cancel()
			networkingTasks.Wait()
			os.Exit(0)
		}()
		app.Main()
	}
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/NguyenHien-8/NoteHub/internal/app"
	"github.com/NguyenHien-8/NoteHub/internal/ipc"
)

var version = "0.3.0"

func main() {
	dataDir := flag.String("data-dir", "", "Separate NoteHub data directory")
	printVersion := flag.Bool("version", false, "Print backend version and exit")
	flag.Parse()
	if *printVersion {
		fmt.Println("NoteHub core " + version)
		return
	}
	log.SetOutput(os.Stderr)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	b, err := app.OpenBackend(ctx, app.Config{DataDir: *dataDir, AppName: "NoteHub", Version: version})
	if err != nil {
		log.Printf("Open backend: %v", err)
		os.Exit(1)
	}
	defer b.Close()
	server := ipc.New(b, version)
	defer server.Close()
	// Interrupt a blocked stdin read on OS shutdown. EOF also ends Serve normally.
	go func() { <-ctx.Done(); _ = os.Stdin.Close() }()
	if err := server.Serve(ctx, os.Stdin, os.Stdout); err != nil && ctx.Err() == nil {
		log.Printf("IPC: %v", err)
	}
}

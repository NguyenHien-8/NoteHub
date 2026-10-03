package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/NguyenHien-8/NoteHub/internal/app"
	"github.com/NguyenHien-8/NoteHub/internal/ipc"
)

var version = "0.3.0"

func main() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	dataDir := flag.String("data-dir", "", "Use a separate NoteHub data directory")
	printVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()
	if *printVersion {
		fmt.Println("NoteHub Backend " + version)
		return
	}

	ctx := context.Background()
	backend, err := app.OpenBackend(ctx, app.Config{AppName: "NoteHub", Version: version, DataDir: *dataDir})
	if err != nil {
		_ = ipc.EncodeEvent(os.Stdout, ipc.FatalEvent(err.Error()))
		log.Printf("startup failed: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := backend.Close(); err != nil {
			log.Printf("closing database: %v", err)
		}
	}()

	if err := ipc.EncodeEvent(os.Stdout, ipc.ReadyEvent(backend, version)); err != nil {
		log.Printf("writing ready event: %v", err)
		return
	}
	server := ipc.NewServer(backend, version)
	if err := server.Serve(ctx, os.Stdin, os.Stdout); err != nil {
		log.Printf("IPC stopped: %v", err)
	}
}

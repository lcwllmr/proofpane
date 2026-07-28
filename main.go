package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: proofpane <serve|cursor>")
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "serve":
		runServe()
	case "cursor":
		runCursor()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

func runServe() {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	port := flags.Int("port", 7777, "Port to run the HTTP server on")
	flags.Parse(os.Args[2:])

	proc, err := StartLean()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start lake serve: %v\n", err)
		os.Exit(1)
	}
	defer proc.Kill()

	tracker := NewTracker()
	px := NewProxy(os.Stdin, os.Stdout, proc.Stdin, proc.Stdout)
	px.Start(tracker.HandleNotification)

	// frontend is embedded at the root package
	distFS, _ := fs.Sub(FrontendFS, "frontend")
	srv := NewServer(distFS, px, tracker)

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	go srv.ListenAndServe(addr)

	proc.Wait()
}

func runCursor() {
	flags := flag.NewFlagSet("cursor", flag.ExitOnError)
	line := flags.Int("line", 1, "Line number (1-indexed)")
	col := flags.Int("col", 1, "Column number (1-indexed)")
	port := flags.Int("port", 7777, "Port the server is running on")
	browser := flags.Bool("x", false, "Launch browser sidecar if not running and suppress stdout")
	flags.Parse(os.Args[2:])

	payload := fmt.Sprintf(`{"line": %d, "col": %d, "launch_browser": %t}`, *line, *col, *browser)
	url := fmt.Sprintf("http://127.0.0.1:%d/api/cursor", *port)
	res, err := http.Post(url, "application/json", bytes.NewBufferString(payload))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to connect to Proofpane serve")
		os.Exit(1)
	}
	defer res.Body.Close()

	if !*browser {
		_, _ = io.Copy(os.Stdout, res.Body)
		fmt.Println()
	}
}

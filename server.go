package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Server struct {
	mu         sync.Mutex
	clients    map[chan []byte]bool
	count      atomic.Int32
	frontend   fs.FS
	lastLaunch time.Time
	proxy      *Proxy
	tracker    *Tracker
}

type CursorRequest struct {
	Line          int  `json:"line"`
	Col           int  `json:"col"`
	LaunchBrowser bool `json:"launch_browser"`
}

type CursorResponse struct {
	Goal json.RawMessage `json:"goal"`
	Term json.RawMessage `json:"term"`
}

func NewServer(frontend fs.FS, px *Proxy, tracker *Tracker) *Server {
	return &Server{
		clients:  make(map[chan []byte]bool),
		frontend: frontend,
		proxy:    px,
		tracker:  tracker,
	}
}

func (s *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/cursor", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var ev CursorRequest
		if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		s.mu.Lock()
		if ev.LaunchBrowser && s.count.Load() == 0 && time.Since(s.lastLaunch) > 3*time.Second {
			s.lastLaunch = time.Now()
			// Launch browser asynchronously
			go func() {
				exec.Command("rundll32", "url.dll,FileProtocolHandler", "http://"+r.Host).Start()
			}()
		}
		s.mu.Unlock()

		var resp CursorResponse
		uri := s.tracker.GetActiveURI()
		if uri != "" {
			rawGoal, htmlGoal, rawTerm, htmlTerm := FetchTacticState(s.proxy, uri, ev.Line-1, ev.Col-1)
			resp.Goal = rawGoal
			resp.Term = rawTerm

			out := fmt.Sprintf(`
        <div class="panel">
          <h2>Tactic State</h2>
          <div id="goal-panel" class="code-content">%s</div>
        </div>
        
        <div class="panel">
          <h2>Expected Type</h2>
          <div id="term-panel" class="code-content">%s</div>
        </div>`, htmlGoal, htmlTerm)
			s.Broadcast(out)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		ch := make(chan []byte, 10)
		s.mu.Lock()
		s.clients[ch] = true
		s.count.Add(1)
		s.mu.Unlock()

		defer func() {
			s.mu.Lock()
			delete(s.clients, ch)
			s.count.Add(-1)
			s.mu.Unlock()
		}()

		for {
			select {
			case msg := <-ch:
				lines := strings.Split(string(msg), "\n")
				fmt.Fprintf(w, "event: message\n")
				for _, line := range lines {
					fmt.Fprintf(w, "data: %s\n", line)
				}
				fmt.Fprintf(w, "\n")
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})

	mux.Handle("/", http.FileServer(http.FS(s.frontend)))

	return http.ListenAndServe(addr, mux)
}

func (s *Server) Broadcast(html string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.clients {
		select {
		case ch <- []byte(html):
		default:
		}
	}
}

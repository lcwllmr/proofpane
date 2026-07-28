package main

import (
	"encoding/json"
	"sync"
)

type Tracker struct {
	mu     sync.RWMutex
	active string
}

func NewTracker() *Tracker {
	return &Tracker{}
}

func (t *Tracker) GetActiveURI() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.active
}

func (t *Tracker) HandleNotification(method string, params json.RawMessage) {
	if method != "textDocument/didOpen" && method != "textDocument/didChange" {
		return
	}
	var payload struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if err := json.Unmarshal(params, &payload); err == nil && payload.TextDocument.URI != "" {
		t.mu.Lock()
		t.active = payload.TextDocument.URI
		t.mu.Unlock()
	}
}

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

func ReadMessage(r *bufio.Reader) ([]byte, *Message, error) {
	var length int
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Content-Length:") {
			length, _ = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:")))
		}
	}

	if length == 0 {
		return nil, nil, fmt.Errorf("missing content length")
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, nil, err
	}

	var msg Message
	if err := json.Unmarshal(buf, &msg); err != nil {
		return nil, nil, err
	}

	return buf, &msg, nil
}

func WriteMessage(w io.Writer, payload []byte) error {
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(payload))
	if _, err := w.Write([]byte(header)); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

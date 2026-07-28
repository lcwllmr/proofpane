package main

import (
	"bufio"
	"encoding/json"
	"io"
	"sync"
	"sync/atomic"
)

type Proxy struct {
	clientIn  io.Reader
	clientOut io.Writer
	serverIn  io.WriteCloser
	serverOut io.Reader

	requestID  atomic.Int64
	intercepts sync.Map // int64 ID -> chan *Message
}

func NewProxy(cin io.Reader, cout io.Writer, sin io.WriteCloser, sout io.Reader) *Proxy {
	return &Proxy{
		clientIn:  cin,
		clientOut: cout,
		serverIn:  sin,
		serverOut: sout,
	}
}

// Start spawns two goroutines for bidirectional forwarding.
func (p *Proxy) Start(onNotification func(method string, params json.RawMessage)) {
	go p.forwardClientToServer(onNotification)
	go p.forwardServerToClient()
}

func (p *Proxy) forwardClientToServer(onNotification func(string, json.RawMessage)) {
	r := bufio.NewReader(p.clientIn)
	for {
		payload, msg, err := ReadMessage(r)
		if err != nil {
			return
		}
		if msg.Method != "" && msg.ID == nil {
			onNotification(msg.Method, msg.Params)
		}
		_ = WriteMessage(p.serverIn, payload)
	}
}

func (p *Proxy) forwardServerToClient() {
	r := bufio.NewReader(p.serverOut)
	for {
		payload, msg, err := ReadMessage(r)
		if err != nil {
			return
		}

		var idNum int64
		if msg.ID != nil {
			_ = json.Unmarshal(msg.ID, &idNum)
		}

		if idNum >= 1000000000 {
			if ch, ok := p.intercepts.Load(idNum); ok {
				ch.(chan *Message) <- msg
				p.intercepts.Delete(idNum)
			}
			continue // Intercepted, don't send to clientOut
		}

		_ = WriteMessage(p.clientOut, payload)
	}
}

func (p *Proxy) Inject(method string, params interface{}) (*Message, error) {
	id := int64(1000000000) + p.requestID.Add(1)
	idRaw, _ := json.Marshal(id)
	paramsRaw, _ := json.Marshal(params)

	msg := Message{
		JSONRPC: "2.0",
		ID:      idRaw,
		Method:  method,
		Params:  paramsRaw,
	}

	payload, _ := json.Marshal(msg)

	ch := make(chan *Message, 1)
	p.intercepts.Store(id, ch)

	if err := WriteMessage(p.serverIn, payload); err != nil {
		p.intercepts.Delete(id)
		return nil, err
	}

	return <-ch, nil
}

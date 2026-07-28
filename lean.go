package main

import (
	"io"
	"os"
	"os/exec"
)

type Process struct {
	cmd    *exec.Cmd
	Stdin  io.WriteCloser
	Stdout io.ReadCloser
}

func StartLean() (*Process, error) {
	cmd := exec.Command("lake", "serve")
	cmd.Env = os.Environ()
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &Process{cmd: cmd, Stdin: stdin, Stdout: stdout}, nil
}

func (p *Process) Wait() error { return p.cmd.Wait() }
func (p *Process) Kill() error { return p.cmd.Process.Kill() }

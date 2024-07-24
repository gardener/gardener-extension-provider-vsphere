// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package module

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type Module struct {
	Path    string
	Dir     string
	Version string
	Main    bool
}

type Reader interface {
	Read(data []Module) (int, error)
}

type ReadCloser interface {
	Reader
	io.Closer
}

type readCloser struct {
	mu sync.Mutex

	cmd *exec.Cmd
	dec *json.Decoder

	closed   bool
	closeErr error
}

type OpenGoListOptions struct {
	Dir     string
	Command func() *exec.Cmd
}

func (o *OpenGoListOptions) ApplyToOpenGoList(o2 *OpenGoListOptions) {
	if o.Dir != "" {
		o2.Dir = o.Dir
	}
	if o.Command != nil {
		o2.Command = o.Command
	}
}

func (o *OpenGoListOptions) ApplyOptions(opts []OpenGoListOption) {
	for _, opt := range opts {
		opt.ApplyToOpenGoList(o)
	}
}

type OpenGoListOption interface {
	ApplyToOpenGoList(o *OpenGoListOptions)
}

type InDir string

func (d InDir) ApplyToOpenGoList(o *OpenGoListOptions) {
	o.Dir = string(d)
}

func setOpenGoListDefaults(o *OpenGoListOptions) {
	if o.Dir == "" {
		o.Dir = "."
	}
	if o.Command == nil {
		o.Command = func() *exec.Cmd {
			return exec.Command("go", "list", "-m", "-json", "all")
		}
	}
}

func OpenGoList(opts ...OpenGoListOption) (ReadCloser, error) {
	o := &OpenGoListOptions{}
	o.ApplyOptions(opts)
	setOpenGoListDefaults(o)

	cmd := o.Command()
	cmd.Dir = o.Dir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	dec := json.NewDecoder(stdout)

	return &readCloser{
		cmd: cmd,
		dec: dec,
	}, nil
}

func (r *readCloser) Read(data []Module) (n int, err error) {
	for i := 0; i < len(data); i++ {
		mod := &data[i]
		if err := r.dec.Decode(mod); err != nil {
			return i, err
		}
	}
	return len(data), nil
}

func (r *readCloser) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return r.closeErr
	}

	defer func() { r.closed = true }()

	waitDone := make(chan struct{})
	go func() {
		defer close(waitDone)
		_ = r.cmd.Wait()
	}()

	_ = r.cmd.Process.Signal(syscall.SIGTERM)

	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()

	select {
	case <-timer.C:
		r.closeErr = fmt.Errorf("error waiting for command to be completed")
	case <-waitDone:
	}
	return r.closeErr
}

func ReadAll(r Reader) ([]Module, error) {
	b := make([]Module, 0, 128)
	for {
		if len(b) == cap(b) {
			// Add more capacity (let append pick how much).
			b = append(b, Module{})[:len(b)]
		}
		n, err := r.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = nil
			}
			return b, err
		}
	}
}

func ReadAllGoListModules(opts ...OpenGoListOption) ([]Module, error) {
	rc, err := OpenGoList(opts...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()

	return ReadAll(rc)
}

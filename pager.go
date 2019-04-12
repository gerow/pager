// Copyright 2019 Mike Gerow
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package pager provides functions for setting up and tearing down a pager for
// the stdout and stderr of a Go program running in a unix-like environment. It
// includes the ability to detect non-tty outputs and dumb terminals,
// appropriately skipping opening a pager in such instances.
package pager

import (
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/mattn/go-isatty"
	"golang.org/x/sys/unix"
)

// Open sets up the environment to be paged to a pager found on the system if
// the current stdout/stderr is a non-dumb terminal. It uses the value of the
// environment "PAGER" first. If that isn't set it attempts to use "pager",
// "less", and "more" in that order. If no suitable pager is found Open still
// returns without error but no pager is setup.
//
// If stdout/stderr is a dumb terminal Open does nothing.
//
// After a call to Open subsequent writes to os.Stdout and os.Stderr will be
// redirected to a pager.
//
// Note that Close must be called after an open in order for the pager to be
// closed correctly. This should generally be done using a defer.
func Open() error {
	var err error
	p, err = open()
	return err
}

// Close closes the pager. This call will block until the pager is exited.
func Close() error {
	err := p.close()
	p = nil
	return err
}

type pgr struct {
	proc                       *os.Process
	storedStdout, storedStderr int
}

var p *pgr

func localPager() (name string, args []string) {
	if pager := os.Getenv("PAGER"); pager != "" {
		f := strings.Fields(pager)
		return f[0], f
	}
	return "", nil
}

func (p *pgr) close() error {
	if p == nil {
		return nil
	}

	// Inform pager that we are done.
	// This can fail if the pipe is closed, but that's fine to ignore.
	os.Stdout.Sync()
	if err := unix.Dup2(p.storedStdout, unix.Stdout); err != nil {
		return err
	}
	if err := unix.Close(p.storedStdout); err != nil {
		return err
	}
	os.Stderr.Sync()
	if err := unix.Dup2(p.storedStderr, unix.Stderr); err != nil {
		return err
	}
	if err := unix.Close(p.storedStderr); err != nil {
		return err
	}
	if err := p.proc.Signal(unix.SIGCONT); err != nil {
		return err
	}
	state, err := p.proc.Wait()
	if err != nil {
		return err
	} else if !state.Success() {
		return &exec.ExitError{ProcessState: state}
	}
	return nil
}

func open() (*pgr, error) {
	// no paging if we're not on a tty
	if !isatty.IsTerminal(os.Stdout.Fd()) || !isatty.IsTerminal(os.Stderr.Fd()) {
		return nil, nil
	}
	// no paging on dumb terminals
	if term := os.Getenv("TERM"); term == "" || term == "dumb" {
		return nil, nil
	}

	// add reasonable defaults for less.
	env := append(os.Environ(),
		"LESS=FRSM",
		"LESSCHARSET=utf-8",
	)
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer pr.Close()
	defer pw.Close()
	procAttr := &os.ProcAttr{
		Env:   env,
		Files: []*os.File{pr, os.Stdout, os.Stderr},
	}

	var proc *os.Process
	lp, lpArgs := localPager()
	for _, p := range []struct {
		name string
		args []string
	}{
		{lp, lpArgs},
		// debian provides an alternatives file named "pager"
		{"pager", []string{"pager"}},
		{"less", []string{"less"}},
		{"more", []string{"more"}},
	} {
		// when PAGER isn't set.
		if p.name == "" {
			continue
		}
		path, err := exec.LookPath(p.name)
		if err != nil {
			continue
		}
		p, err := os.StartProcess(path, p.args, procAttr)
		if err != nil {
			continue
		}
		proc = p
		break
	}
	// If we can't find a suitable pager just log an error
	if proc == nil {
		log.Print("Failed to find a suitable pager, continuing without one")
		return nil, nil
	}
	// save stdout and stderr so that we can restore them when we close the pager
	storedStdout, err := unix.Dup(unix.Stdout)
	if err != nil {
		return nil, err
	}
	storedStderr, err := unix.Dup(unix.Stderr)
	if err != nil {
		return nil, err
	}
	if err := unix.Dup2(int(pw.Fd()), unix.Stdout); err != nil {
		return nil, err
	}
	if err := unix.Dup2(int(pw.Fd()), unix.Stderr); err != nil {
		return nil, err
	}

	// Ignore SIGINT, letting our pager handle it if it finds it
	// appropriate. This feels like hacky, but it works, so eh?
	signal.Ignore(os.Interrupt)
	return &pgr{proc, storedStdout, storedStderr}, nil
}

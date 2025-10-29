package shell

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/charmbracelet/hotdiva2000"
)

// BackgroundShell represents a shell running in the background.
type BackgroundShell struct {
	ID         string
	Shell      *Shell
	ctx        context.Context
	cancel     context.CancelFunc
	stdout     *bytes.Buffer
	stderr     *bytes.Buffer
	mu         sync.RWMutex
	done       chan struct{}
	exitErr    error
	workingDir string
}

// BackgroundShellManager manages background shell instances.
type BackgroundShellManager struct {
	shells map[string]*BackgroundShell
	mu     sync.RWMutex
}

var (
	backgroundManager     *BackgroundShellManager
	backgroundManagerOnce sync.Once
)

// GetBackgroundShellManager returns the singleton background shell manager.
func GetBackgroundShellManager() *BackgroundShellManager {
	backgroundManagerOnce.Do(func() {
		backgroundManager = &BackgroundShellManager{
			shells: make(map[string]*BackgroundShell),
		}
	})
	return backgroundManager
}

// Start creates and starts a new background shell with the given command.
func (m *BackgroundShellManager) Start(ctx context.Context, workingDir string, blockFuncs []BlockFunc, command string) (*BackgroundShell, error) {
	id := hotdiva2000.Generate()

	shell := NewShell(&Options{
		WorkingDir: workingDir,
		BlockFuncs: blockFuncs,
	})

	shellCtx, cancel := context.WithCancel(ctx)

	bgShell := &BackgroundShell{
		ID:         id,
		Shell:      shell,
		ctx:        shellCtx,
		cancel:     cancel,
		stdout:     &bytes.Buffer{},
		stderr:     &bytes.Buffer{},
		done:       make(chan struct{}),
		workingDir: workingDir,
	}

	m.mu.Lock()
	m.shells[id] = bgShell
	m.mu.Unlock()

	go func() {
		defer close(bgShell.done)

		stdout, stderr, err := shell.Exec(shellCtx, command)

		bgShell.mu.Lock()
		bgShell.stdout.WriteString(stdout)
		bgShell.stderr.WriteString(stderr)
		bgShell.exitErr = err
		bgShell.mu.Unlock()
	}()

	return bgShell, nil
}

// Get retrieves a background shell by ID.
func (m *BackgroundShellManager) Get(id string) (*BackgroundShell, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	shell, ok := m.shells[id]
	return shell, ok
}

// Kill terminates a background shell by ID.
func (m *BackgroundShellManager) Kill(id string) error {
	m.mu.Lock()
	shell, ok := m.shells[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("background shell not found: %s", id)
	}
	delete(m.shells, id)
	m.mu.Unlock()

	shell.cancel()
	<-shell.done
	return nil
}

// List returns all background shell IDs.
func (m *BackgroundShellManager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.shells))
	for id := range m.shells {
		ids = append(ids, id)
	}
	return ids
}

// GetOutput returns the current output of a background shell.
func (bs *BackgroundShell) GetOutput() (stdout string, stderr string, done bool, err error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	select {
	case <-bs.done:
		return bs.stdout.String(), bs.stderr.String(), true, bs.exitErr
	default:
		return bs.stdout.String(), bs.stderr.String(), false, nil
	}
}

// IsDone checks if the background shell has finished execution.
func (bs *BackgroundShell) IsDone() bool {
	select {
	case <-bs.done:
		return true
	default:
		return false
	}
}

// Wait blocks until the background shell completes.
func (bs *BackgroundShell) Wait() {
	<-bs.done
}

// GetWorkingDir returns the current working directory of the background shell.
func (bs *BackgroundShell) GetWorkingDir() string {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.workingDir
}

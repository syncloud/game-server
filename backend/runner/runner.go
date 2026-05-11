package runner

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type Runner struct {
	mu     sync.Mutex
	procs  map[int64]*exec.Cmd
	logger *log.Logger
}

func New(logger *log.Logger) *Runner {
	return &Runner{
		procs:  map[int64]*exec.Cmd{},
		logger: logger,
	}
}

func (r *Runner) Running(id int64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	cmd, ok := r.procs[id]
	if !ok || cmd.Process == nil {
		return false
	}
	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		return false
	}
	return syscall.Kill(cmd.Process.Pid, 0) == nil
}

func (r *Runner) Start(id int64, startCmd string, workDir string) error {
	if startCmd == "" {
		return fmt.Errorf("start command empty")
	}
	r.mu.Lock()
	if existing, ok := r.procs[id]; ok && existing.Process != nil {
		if existing.ProcessState == nil || !existing.ProcessState.Exited() {
			r.mu.Unlock()
			return fmt.Errorf("already running")
		}
	}
	cmd := exec.Command("sh", "-c", startCmd)
	if workDir != "" {
		if _, err := os.Stat(workDir); err == nil {
			cmd.Dir = workDir
		}
	}
	cmd.Stdout = newLogWriter(r.logger, fmt.Sprintf("server[%d] stdout", id))
	cmd.Stderr = newLogWriter(r.logger, fmt.Sprintf("server[%d] stderr", id))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		r.mu.Unlock()
		return fmt.Errorf("start: %w", err)
	}
	r.procs[id] = cmd
	r.mu.Unlock()

	go func() {
		err := cmd.Wait()
		if err != nil {
			r.logger.Printf("server[%d] exited: %v", id, err)
		} else {
			r.logger.Printf("server[%d] exited cleanly", id)
		}
	}()
	return nil
}

func (r *Runner) Stop(id int64) error {
	r.mu.Lock()
	cmd, ok := r.procs[id]
	r.mu.Unlock()
	if !ok || cmd.Process == nil {
		return fmt.Errorf("not running")
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
	} else {
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}
	done := make(chan struct{})
	go func() {
		_, _ = cmd.Process.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		if pgid > 0 {
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		} else {
			_ = cmd.Process.Kill()
		}
	}
	r.mu.Lock()
	delete(r.procs, id)
	r.mu.Unlock()
	return nil
}

func (r *Runner) Restart(id int64, startCmd string, workDir string) error {
	if r.Running(id) {
		if err := r.Stop(id); err != nil {
			return err
		}
	}
	return r.Start(id, startCmd, workDir)
}

type logWriter struct {
	logger *log.Logger
	prefix string
}

func newLogWriter(l *log.Logger, p string) *logWriter {
	return &logWriter{logger: l, prefix: p}
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.logger.Printf("%s: %s", w.prefix, string(p))
	return len(p), nil
}

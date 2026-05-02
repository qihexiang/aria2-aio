package instance

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type ManagedProcess struct {
	ID       string
	PID      int
	Port     int
	Dir      string
	ConfPath string
	LogFile  *os.File
	Cmd      *exec.Cmd
	Cancel   context.CancelFunc
	done     chan error // closed when cmd.Wait() returns
}

type ProcessSupervisor struct {
	mu         sync.Mutex
	procs      map[string]*ManagedProcess
	aria2cPath string
}

func NewProcessSupervisor(aria2cPath string) *ProcessSupervisor {
	return &ProcessSupervisor{
		procs:      make(map[string]*ManagedProcess),
		aria2cPath: aria2cPath,
	}
}

func (ps *ProcessSupervisor) Start(instanceID string, port int, secret string, confPath string, dir string) (*ManagedProcess, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if _, exists := ps.procs[instanceID]; exists {
		return nil, fmt.Errorf("instance %s already has a managed process", instanceID)
	}

	downloadDir := filepath.Join(dir, "downloads")
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return nil, fmt.Errorf("create download dir: %w", err)
	}

	logPath := filepath.Join(dir, "aria2.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, ps.aria2cPath,
		"--conf-path="+confPath,
		"--dir="+downloadDir,
	)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		cancel()
		logFile.Close()
		return nil, fmt.Errorf("start aria2c: %w", err)
	}

	proc := &ManagedProcess{
		ID:       instanceID,
		PID:      cmd.Process.Pid,
		Port:     port,
		Dir:      dir,
		ConfPath: confPath,
		LogFile:  logFile,
		Cmd:      cmd,
		Cancel:   cancel,
		done:     make(chan error, 1),
	}

	ps.procs[instanceID] = proc

	// Single goroutine calls cmd.Wait() - no other caller should call it
	go func() {
		err := cmd.Wait()
		proc.done <- err

		ps.mu.Lock()
		delete(ps.procs, instanceID)
		ps.mu.Unlock()

		log.Printf("aria2c for instance %s exited: %v", instanceID, err)
	}()

	log.Printf("Started aria2c for instance %s: PID=%d, port=%d", instanceID, proc.PID, port)
	return proc, nil
}

func (ps *ProcessSupervisor) Stop(instanceID string) error {
	ps.mu.Lock()
	proc, exists := ps.procs[instanceID]
	ps.mu.Unlock()

	if !exists {
		// Process already exited via monitor goroutine
		return nil
	}

	// Send SIGTERM for graceful shutdown
	if proc.Cmd.Process != nil {
		proc.Cmd.Process.Signal(syscall.SIGTERM)
	}

	// Wait for process to exit, with timeout
	select {
	case <-proc.done:
		// Process exited cleanly
	case <-time.After(10 * time.Second):
		// Timeout - force kill
		log.Printf("Process %s didn't exit in 10s, sending SIGKILL", instanceID)
		proc.Cmd.Process.Kill()
		// Wait for the done signal from monitor goroutine
		<-proc.done
	}

	proc.Cancel()
	proc.LogFile.Close()

	log.Printf("Stopped aria2c for instance %s", instanceID)
	return nil
}

func (ps *ProcessSupervisor) Kill(instanceID string) error {
	ps.mu.Lock()
	proc, exists := ps.procs[instanceID]
	ps.mu.Unlock()

	if !exists {
		return nil
	}

	proc.Cmd.Process.Kill()
	<-proc.done // wait for monitor goroutine

	proc.Cancel()
	proc.LogFile.Close()

	return nil
}

func (ps *ProcessSupervisor) GetPID(instanceID string) (int, error) {
	ps.mu.Lock()
	proc, exists := ps.procs[instanceID]
	ps.mu.Unlock()

	if !exists {
		return 0, fmt.Errorf("instance %s has no managed process", instanceID)
	}
	return proc.PID, nil
}

func (ps *ProcessSupervisor) IsRunning(instanceID string) bool {
	ps.mu.Lock()
	_, exists := ps.procs[instanceID]
	ps.mu.Unlock()
	return exists
}

func (ps *ProcessSupervisor) WriteAria2Conf(confPath string, port int, secret string, options map[string]string, defaults map[string]string) error {
	dir := filepath.Dir(confPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create conf dir: %w", err)
	}

	sessionPath := filepath.Join(dir, "aria2.session")
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		f, err := os.Create(sessionPath)
		if err != nil {
			return fmt.Errorf("create session file: %w", err)
		}
		f.Close()
	}

	merged := make(map[string]string)
	for k, v := range defaults {
		merged[k] = v
	}
	for k, v := range options {
		merged[k] = v
	}

	merged["enable-rpc"] = "true"
	merged["rpc-listen-port"] = strconv.Itoa(port)
	merged["rpc-secret"] = secret
	merged["rpc-listen-all"] = "true"
	merged["input-file"] = sessionPath
	merged["save-session"] = sessionPath

	var content string
	for k, v := range merged {
		content += fmt.Sprintf("%s=%s\n", k, v)
	}

	return os.WriteFile(confPath, []byte(content), 0644)
}
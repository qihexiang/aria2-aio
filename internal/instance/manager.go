package instance

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/user/aria2-aio/internal/config"
	"github.com/user/aria2-aio/internal/rpc"
	"github.com/user/aria2-aio/internal/store"
)

type InstanceStatus string

const (
	StatusRunning InstanceStatus = "running"
	StatusStopped InstanceStatus = "stopped"
	StatusError   InstanceStatus = "error"
)

type ActiveInstance struct {
	ID        string
	Instance  *store.Instance
	RPCClient *rpc.Aria2Client
	Listener  *rpc.EventListener
}

type InstanceManager struct {
	mu         sync.RWMutex
	active     map[string]*ActiveInstance
	store      *store.DB
	supervisor *ProcessSupervisor
	cfg        *config.Config
}

func NewInstanceManager(store *store.DB, supervisor *ProcessSupervisor, cfg *config.Config) *InstanceManager {
	return &InstanceManager{
		active:     make(map[string]*ActiveInstance),
		store:      store,
		supervisor: supervisor,
		cfg:        cfg,
	}
}

func (m *InstanceManager) Create(name string, dir string, options map[string]string) (*store.Instance, error) {
	// Allocate a port
	port, err := m.allocatePort()
	if err != nil {
		return nil, fmt.Errorf("allocate port: %w", err)
	}

	// Generate RPC secret
	secretBytes := make([]byte, 16)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("generate secret: %w", err)
	}
	secret := hex.EncodeToString(secretBytes)

	id := uuid.New().String()

	if dir == "" {
		dir = m.cfg.InstanceDir(id)
	}

	inst := &store.Instance{
		ID:         id,
		Name:       name,
		RPCPort:    port,
		RPCSecret:  secret,
		Dir:        dir,
		Status:     string(StatusStopped),
		PID:        0,
		ConfigJSON: options,
	}

	// Create instance directory
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create instance dir: %w", err)
	}

	// Write aria2 config
	confPath := filepath.Join(dir, "aria2.conf")
	if err := m.supervisor.WriteAria2Conf(confPath, port, secret, options, m.cfg.Defaults.Aria2Options); err != nil {
		return nil, fmt.Errorf("write aria2 config: %w", err)
	}

	if err := m.store.Instances().Create(inst); err != nil {
		return nil, fmt.Errorf("save instance: %w", err)
	}

	log.Printf("Created instance %s: name=%s, port=%d, dir=%s", id, name, port, dir)
	return inst, nil
}

func (m *InstanceManager) Start(id string) (*store.Instance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.active[id]; exists {
		return nil, fmt.Errorf("instance %s is already running", id)
	}

	inst, err := m.store.Instances().GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("get instance: %w", err)
	}

	if inst.Status == string(StatusRunning) {
		return nil, fmt.Errorf("instance %s is already running", id)
	}

	// Rewrite config file (might have been updated while stopped)
	confPath := filepath.Join(inst.Dir, "aria2.conf")
	if err := m.supervisor.WriteAria2Conf(confPath, inst.RPCPort, inst.RPCSecret, inst.ConfigJSON, m.cfg.Defaults.Aria2Options); err != nil {
		return nil, fmt.Errorf("write aria2 config: %w", err)
	}

	// Start the process
	proc, err := m.supervisor.Start(id, inst.RPCPort, inst.RPCSecret, confPath, inst.Dir)
	if err != nil {
		return nil, fmt.Errorf("start process: %w", err)
	}

	// Create RPC client
	rpcClient := rpc.NewAria2Client(id, inst.RPCPort, inst.RPCSecret)

	// Wait for RPC to become available
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for i := 0; i < 5; i++ {
		_, err := rpcClient.GetVersion(ctx)
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Create event listener
	listener := rpc.NewEventListener(rpcClient)
	listener.Start(context.Background())

	// Update DB
	m.store.Instances().UpdateStatus(id, string(StatusRunning), proc.PID)

	inst.Status = string(StatusRunning)
	inst.PID = proc.PID

	m.active[id] = &ActiveInstance{
		ID:        id,
		Instance:  inst,
		RPCClient: rpcClient,
		Listener:  listener,
	}

	log.Printf("Started instance %s: PID=%d", id, proc.PID)
	return inst, nil
}

func (m *InstanceManager) Stop(id string) (*store.Instance, error) {
	m.mu.Lock()
	active, exists := m.active[id]
	m.mu.Unlock()

	if !exists {
		// Try graceful shutdown via RPC anyway
		inst, err := m.store.Instances().GetByID(id)
		if err != nil {
			return nil, fmt.Errorf("instance %s not found", id)
		}
		if inst.Status != string(StatusRunning) {
			return nil, fmt.Errorf("instance %s is not running", id)
		}
		return nil, fmt.Errorf("instance %s has no active process", id)
	}

	// Try RPC shutdown first
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	err := active.RPCClient.Shutdown(ctx)
	cancel()

	if err != nil {
		log.Printf("RPC shutdown failed for %s: %v, using process termination", id, err)
		m.supervisor.Stop(id)
	} else {
		// Wait a bit for process to exit from RPC shutdown
		time.Sleep(1 * time.Second)
		if m.supervisor.IsRunning(id) {
			m.supervisor.Stop(id)
		}
	}

	// Stop event listener
	active.Listener.Stop()

	m.mu.Lock()
	delete(m.active, id)
	m.mu.Unlock()

	// Update DB
	m.store.Instances().UpdateStatus(id, string(StatusStopped), 0)

	inst := active.Instance
	inst.Status = string(StatusStopped)
	inst.PID = 0

	log.Printf("Stopped instance %s", id)
	return inst, nil
}

func (m *InstanceManager) Restart(id string) (*store.Instance, error) {
	// Stop first (ignore error if not running)
	if m.IsActive(id) {
		if _, err := m.Stop(id); err != nil {
			log.Printf("Stop during restart for %s: %v (continuing)", id, err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	return m.Start(id)
}

func (m *InstanceManager) Delete(id string) error {
	// Stop if running
	if m.IsActive(id) {
		m.Stop(id)
	}

	inst, err := m.store.Instances().GetByID(id)
	if err != nil {
		return fmt.Errorf("instance %s not found", id)
	}

	// Remove from DB (cascades to task_history)
	if err := m.store.Instances().Delete(id); err != nil {
		return fmt.Errorf("delete instance from db: %w", err)
	}

	// Remove data directory
	if inst.Dir != "" && filepath.IsAbs(inst.Dir) {
		os.RemoveAll(inst.Dir)
	}

	log.Printf("Deleted instance %s", id)
	return nil
}

func (m *InstanceManager) Update(id string, name string, dir string, options map[string]string) (*store.Instance, error) {
	inst, err := m.store.Instances().GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("get instance: %w", err)
	}

	if name != "" {
		inst.Name = name
	}
	if dir != "" {
		inst.Dir = dir
	}
	if options != nil {
		inst.ConfigJSON = options
	}

	if err := m.store.Instances().Update(id, inst.Name, inst.Dir); err != nil {
		return nil, fmt.Errorf("update instance: %w", err)
	}

	if options != nil {
		if err := m.store.Instances().UpdateConfig(id, options); err != nil {
			return nil, fmt.Errorf("update instance config: %w", err)
		}
	}

	// Rewrite config file
	confPath := filepath.Join(inst.Dir, "aria2.conf")
	m.supervisor.WriteAria2Conf(confPath, inst.RPCPort, inst.RPCSecret, inst.ConfigJSON, m.cfg.Defaults.Aria2Options)

	// If running, apply live options via RPC
	m.mu.RLock()
	active, exists := m.active[id]
	m.mu.RUnlock()

	if exists {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := active.RPCClient.ChangeGlobalOption(ctx, options); err != nil {
			log.Printf("Failed to apply live options for %s: %v (some options may require restart)", id, err)
		}
	}

	return inst, nil
}

func (m *InstanceManager) GetActive(id string) (*ActiveInstance, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, exists := m.active[id]
	return a, exists
}

func (m *InstanceManager) IsActive(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.active[id] != nil
}

func (m *InstanceManager) ListActive() map[string]*ActiveInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]*ActiveInstance, len(m.active))
	for k, v := range m.active {
		result[k] = v
	}
	return result
}

func (m *InstanceManager) allocatePort() (int, error) {
	usedPorts, err := m.store.Instances().UsedPorts()
	if err != nil {
		return 0, err
	}

	start := m.cfg.Defaults.RPCPortRange.Start
	end := m.cfg.Defaults.RPCPortRange.End

	for p := start; p <= end; p++ {
		if !usedPorts[p] {
			return p, nil
		}
	}
	return 0, fmt.Errorf("no available ports in range %d-%d", start, end)
}

func (m *InstanceManager) RecoverRunningInstances() {
	instances, err := m.store.Instances().GetRunningInstances()
	if err != nil {
		log.Printf("Failed to query running instances for recovery: %v", err)
		return
	}

	for _, inst := range instances {
		// Check if the PID is still alive
		proc, err := os.FindProcess(inst.PID)
		if err != nil || proc == nil {
			// Mark as stopped
			m.store.Instances().UpdateStatus(inst.ID, string(StatusStopped), 0)
			log.Printf("Recovered instance %s: PID %d no longer alive, marking stopped", inst.ID, inst.PID)
			continue
		}

		// Try to connect via RPC to verify it's actually aria2
		rpcClient := rpc.NewAria2Client(inst.ID, inst.RPCPort, inst.RPCSecret)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, err = rpcClient.GetVersion(ctx)
		cancel()

		if err != nil {
			// Not reachable, mark as stopped
			m.store.Instances().UpdateStatus(inst.ID, string(StatusStopped), 0)
			log.Printf("Recovered instance %s: RPC not reachable, marking stopped", inst.ID)
			continue
		}

		// Adopt the running instance
		listener := rpc.NewEventListener(rpcClient)
		listener.Start(context.Background())

		m.mu.Lock()
		m.active[inst.ID] = &ActiveInstance{
			ID:        inst.ID,
			Instance:  inst,
			RPCClient: rpcClient,
			Listener:  listener,
		}
		m.mu.Unlock()

		log.Printf("Recovered running instance %s: PID=%d, port=%d", inst.ID, inst.PID, inst.RPCPort)
	}
}

func (m *InstanceManager) StopAll() {
	m.mu.RLock()
	ids := make([]string, 0, len(m.active))
	for id := range m.active {
		ids = append(ids, id)
	}
	m.mu.RUnlock()

	for _, id := range ids {
		m.Stop(id)
	}
}
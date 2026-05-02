package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type Instance struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	RPCPort    int               `json:"rpc_port"`
	RPCSecret  string            `json:"rpc_secret"`
	Dir        string            `json:"dir"`
	Status     string            `json:"status"`
	PID        int               `json:"pid"`
	ConfigJSON map[string]string `json:"config_json"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

type InstanceRepo struct {
	db *sql.DB
}

func NewInstanceRepo(db *sql.DB) *InstanceRepo {
	return &InstanceRepo{db: db}
}

func (r *InstanceRepo) Create(inst *Instance) error {
	configJSON, err := json.Marshal(inst.ConfigJSON)
	if err != nil {
		return fmt.Errorf("marshal config_json: %w", err)
	}

	_, err = r.db.Exec(
		`INSERT INTO instances (id, name, rpc_port, rpc_secret, dir, status, pid, config_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		inst.ID, inst.Name, inst.RPCPort, inst.RPCSecret, inst.Dir, inst.Status, inst.PID, string(configJSON),
	)
	return err
}

func (r *InstanceRepo) GetByID(id string) (*Instance, error) {
	row := r.db.QueryRow(
		`SELECT id, name, rpc_port, rpc_secret, dir, status, pid, config_json, created_at, updated_at
		 FROM instances WHERE id = ?`, id,
	)
	return scanInstance(row)
}

func (r *InstanceRepo) List() ([]*Instance, error) {
	rows, err := r.db.Query(
		`SELECT id, name, rpc_port, rpc_secret, dir, status, pid, config_json, created_at, updated_at
		 FROM instances ORDER BY created_at`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instances []*Instance
	for rows.Next() {
		inst, err := scanInstanceRows(rows)
		if err != nil {
			return nil, err
		}
		instances = append(instances, inst)
	}
	return instances, rows.Err()
}

func (r *InstanceRepo) UpdateStatus(id string, status string, pid int) error {
	_, err := r.db.Exec(
		`UPDATE instances SET status = ?, pid = ?, updated_at = datetime('now') WHERE id = ?`,
		status, pid, id,
	)
	return err
}

func (r *InstanceRepo) UpdateConfig(id string, configJSON map[string]string) error {
	b, err := json.Marshal(configJSON)
	if err != nil {
		return fmt.Errorf("marshal config_json: %w", err)
	}
	_, err = r.db.Exec(
		`UPDATE instances SET config_json = ?, updated_at = datetime('now') WHERE id = ?`,
		string(b), id,
	)
	return err
}

func (r *InstanceRepo) Update(id string, name string, dir string) error {
	_, err := r.db.Exec(
		`UPDATE instances SET name = ?, dir = ?, updated_at = datetime('now') WHERE id = ?`,
		name, dir, id,
	)
	return err
}

func (r *InstanceRepo) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM instances WHERE id = ?`, id)
	return err
}

func (r *InstanceRepo) GetRunningInstances() ([]*Instance, error) {
	rows, err := r.db.Query(
		`SELECT id, name, rpc_port, rpc_secret, dir, status, pid, config_json, created_at, updated_at
		 FROM instances WHERE status = 'running'`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instances []*Instance
	for rows.Next() {
		inst, err := scanInstanceRows(rows)
		if err != nil {
			return nil, err
		}
		instances = append(instances, inst)
	}
	return instances, rows.Err()
}

func (r *InstanceRepo) UsedPorts() (map[int]bool, error) {
	rows, err := r.db.Query(`SELECT rpc_port FROM instances`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ports := map[int]bool{}
	for rows.Next() {
		var p int
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		ports[p] = true
	}
	return ports, rows.Err()
}

func scanInstance(row *sql.Row) (*Instance, error) {
	inst := &Instance{}
	var configJSONStr string
	var createdAt, updatedAt string

	err := row.Scan(
		&inst.ID, &inst.Name, &inst.RPCPort, &inst.RPCSecret, &inst.Dir,
		&inst.Status, &inst.PID, &configJSONStr, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(configJSONStr), &inst.ConfigJSON); err != nil {
		inst.ConfigJSON = map[string]string{}
	}

	inst.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	inst.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	return inst, nil
}

func scanInstanceRows(rows *sql.Rows) (*Instance, error) {
	inst := &Instance{}
	var configJSONStr string
	var createdAt, updatedAt string

	err := rows.Scan(
		&inst.ID, &inst.Name, &inst.RPCPort, &inst.RPCSecret, &inst.Dir,
		&inst.Status, &inst.PID, &configJSONStr, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(configJSONStr), &inst.ConfigJSON); err != nil {
		inst.ConfigJSON = map[string]string{}
	}

	inst.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	inst.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	return inst, nil
}
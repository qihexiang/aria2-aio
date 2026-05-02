package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
	instances *InstanceRepo
	tasks     *TaskRepo
}

func OpenDB(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	connStr := fmt.Sprintf("file:%s?cache=shared&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&mode=rwc", dbPath)

	db, err := sql.Open("sqlite", connStr)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("migrations: %w", err)
	}

	store := &DB{
		DB:        db,
		instances: NewInstanceRepo(db),
		tasks:     NewTaskRepo(db),
	}

	return store, nil
}

func (d *DB) Instances() *InstanceRepo { return d.instances }
func (d *DB) Tasks() *TaskRepo         { return d.tasks }

func (d *DB) Close() error {
	return d.DB.Close()
}
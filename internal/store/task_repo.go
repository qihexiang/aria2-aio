package store

import (
	"database/sql"
	"fmt"
	"time"
)

type TaskRecord struct {
	ID              int       `json:"id"`
	InstanceID      string    `json:"instance_id"`
	GID             string    `json:"gid"`
	Name            string    `json:"name"`
	URIs            string    `json:"uris"`
	Dir             string    `json:"dir"`
	FilesJSON       string    `json:"files_json"`
	TotalLength     int64     `json:"total_length"`
	CompletedLength int64     `json:"completed_length"`
	DownloadSpeed   int64     `json:"download_speed"`
	UploadLength    int64     `json:"upload_length"`
	Status          string    `json:"status"`
	ErrorCode       int       `json:"error_code"`
	ErrorMessage    string    `json:"error_message"`
	InfoHash        string    `json:"info_hash"`
	CompletedAt     time.Time `json:"completed_at"`
}

type PaginatedResult struct {
	Records []*TaskRecord `json:"records"`
	Total   int           `json:"total"`
	Page    int           `json:"page"`
	PerPage int           `json:"per_page"`
}

type TaskRepo struct {
	db *sql.DB
}

func NewTaskRepo(db *sql.DB) *TaskRepo {
	return &TaskRepo{db: db}
}

func (r *TaskRepo) Create(record *TaskRecord) error {
	_, err := r.db.Exec(
		`INSERT INTO task_history (instance_id, gid, name, uris, dir, files_json,
		 total_length, completed_length, download_speed, upload_length,
		 status, error_code, error_message, info_hash)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.InstanceID, record.GID, record.Name, record.URIs,
		record.Dir, record.FilesJSON,
		record.TotalLength, record.CompletedLength, record.DownloadSpeed,
		record.UploadLength, record.Status, record.ErrorCode, record.ErrorMessage, record.InfoHash,
	)
	return err
}

func (r *TaskRepo) ListByInstance(instanceID string, page int, perPage int) (*PaginatedResult, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	offset := (page - 1) * perPage

	var total int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM task_history WHERE instance_id = ?`, instanceID,
	).Scan(&total)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(
		`SELECT id, instance_id, gid, name, uris, dir, files_json,
		 total_length, completed_length, download_speed, upload_length,
		 status, error_code, error_message, info_hash, completed_at
		 FROM task_history WHERE instance_id = ?
		 ORDER BY completed_at DESC LIMIT ? OFFSET ?`,
		instanceID, perPage, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*TaskRecord
	for rows.Next() {
		rec, err := scanTaskRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &PaginatedResult{
		Records: records,
		Total:   total,
		Page:    page,
		PerPage: perPage,
	}, nil
}

func (r *TaskRepo) GetByGID(instanceID string, gid string) (*TaskRecord, error) {
	row := r.db.QueryRow(
		`SELECT id, instance_id, gid, name, uris, dir, files_json,
		 total_length, completed_length, download_speed, upload_length,
		 status, error_code, error_message, info_hash, completed_at
		 FROM task_history WHERE instance_id = ? AND gid = ?`,
		instanceID, gid,
	)
	return scanTaskRecordRow(row)
}

func (r *TaskRepo) DeleteByGID(instanceID string, gid string) error {
	_, err := r.db.Exec(
		`DELETE FROM task_history WHERE instance_id = ? AND gid = ?`,
		instanceID, gid,
	)
	return err
}

func (r *TaskRepo) ExistsByGID(instanceID string, gid string) (bool, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM task_history WHERE instance_id = ? AND gid = ?`,
		instanceID, gid,
	).Scan(&count)
	return count > 0, err
}

func scanTaskRecord(rows *sql.Rows) (*TaskRecord, error) {
	rec := &TaskRecord{}
	var completedAt string
	err := rows.Scan(
		&rec.ID, &rec.InstanceID, &rec.GID, &rec.Name, &rec.URIs,
		&rec.Dir, &rec.FilesJSON,
		&rec.TotalLength, &rec.CompletedLength, &rec.DownloadSpeed,
		&rec.UploadLength, &rec.Status, &rec.ErrorCode, &rec.ErrorMessage,
		&rec.InfoHash, &completedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan task record: %w", err)
	}
	rec.CompletedAt, _ = time.Parse("2006-01-02 15:04:05", completedAt)
	return rec, nil
}

func scanTaskRecordRow(row *sql.Row) (*TaskRecord, error) {
	rec := &TaskRecord{}
	var completedAt string
	err := row.Scan(
		&rec.ID, &rec.InstanceID, &rec.GID, &rec.Name, &rec.URIs,
		&rec.Dir, &rec.FilesJSON,
		&rec.TotalLength, &rec.CompletedLength, &rec.DownloadSpeed,
		&rec.UploadLength, &rec.Status, &rec.ErrorCode, &rec.ErrorMessage,
		&rec.InfoHash, &completedAt,
	)
	if err != nil {
		return nil, err
	}
	rec.CompletedAt, _ = time.Parse("2006-01-02 15:04:05", completedAt)
	return rec, nil
}
package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"latihan-fiber/app/model"
)

var (
	ErrSupervisorFull = errors.New("kuota dosen pembimbing penuh")
	ErrAlreadyAssigned = errors.New("mahasiswa sudah memiliki pembimbing")
	ErrNotAssigned    = errors.New("mahasiswa belum memiliki pembimbing")
)

type SupervisorRepository interface {
	WithTx(tx DBTX) SupervisorRepository

	FindByID(ctx context.Context, id int) (model.Supervisor, error)
	GetSupervisorByStudent(ctx context.Context, studentID int) (int, error)
	
	// Fitur Penguncian & Aturan Transaksi D.2
	LockStudent(ctx context.Context, studentID int) error
	LockSupervisors(ctx context.Context, ids ...int) error
	IncrementSupervised(ctx context.Context, id int) error
	DecrementSupervised(ctx context.Context, id int) error
	AssignStudent(ctx context.Context, studentID, supervisorID int) error
	SwitchStudent(ctx context.Context, studentID, newSupervisorID int) error
	UpdateQuota(ctx context.Context, id, quota, version int) (model.Supervisor, error)
}

type supervisorPostgresRepo struct{ db DBTX }

func NewSupervisorRepository(db DBTX) SupervisorRepository {
	return &supervisorPostgresRepo{db: db}
}

func (r *supervisorPostgresRepo) WithTx(tx DBTX) SupervisorRepository {
	return &supervisorPostgresRepo{db: tx}
}

func (r *supervisorPostgresRepo) FindByID(ctx context.Context, id int) (model.Supervisor, error) {
	var s model.Supervisor
	err := r.db.QueryRow(ctx, `
		SELECT id, name, quota, supervised_count, version, created_at 
		FROM supervisors WHERE id = $1`, id).
		Scan(&s.ID, &s.Name, &s.Quota, &s.SupervisedCount, &s.Version, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s, ErrNotFound
		}
		return s, err
	}
	return s, nil
}

func (r *supervisorPostgresRepo) GetSupervisorByStudent(ctx context.Context, studentID int) (int, error) {
	var supervisorID int
	err := r.db.QueryRow(ctx, "SELECT supervisor_id FROM supervisions WHERE student_id = $1", studentID).Scan(&supervisorID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotAssigned
		}
		return 0, err
	}
	return supervisorID, nil
}

// Lock Student (Pessimistic Lock) mencegah penetapan serentak untuk mhs yang sama
func (r *supervisorPostgresRepo) LockStudent(ctx context.Context, studentID int) error {
	var id int
	err := r.db.QueryRow(ctx, "SELECT id FROM users WHERE id = $1 FOR UPDATE", studentID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// Lock Supervisors berurutan untuk mencegah deadlock saat switch
func (r *supervisorPostgresRepo) LockSupervisors(ctx context.Context, ids ...int) error {
	rows, err := r.db.Query(ctx, "SELECT id FROM supervisors WHERE id = ANY($1) ORDER BY id FOR UPDATE", ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	found := 0
	for rows.Next() {
		found++
	}
	if found != len(ids) {
		return ErrNotFound
	}
	return nil
}

// Atomic Conditional Update
func (r *supervisorPostgresRepo) IncrementSupervised(ctx context.Context, id int) error {
	tag, err := r.db.Exec(ctx, "UPDATE supervisors SET supervised_count = supervised_count + 1 WHERE id = $1 AND supervised_count < quota", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrSupervisorFull
	}
	return nil
}

func (r *supervisorPostgresRepo) DecrementSupervised(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, "UPDATE supervisors SET supervised_count = supervised_count - 1 WHERE id = $1", id)
	return err
}

func (r *supervisorPostgresRepo) AssignStudent(ctx context.Context, studentID, supervisorID int) error {
	_, err := r.db.Exec(ctx, "INSERT INTO supervisions (student_id, supervisor_id) VALUES ($1, $2)", studentID, supervisorID)
	if err != nil {
		if pgErrorCode(err) == codeUniqueViolation {
			return ErrAlreadyAssigned
		}
		return err
	}
	return nil
}

func (r *supervisorPostgresRepo) SwitchStudent(ctx context.Context, studentID, newSupervisorID int) error {
	tag, err := r.db.Exec(ctx, "UPDATE supervisions SET supervisor_id = $2 WHERE student_id = $1", studentID, newSupervisorID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotAssigned
	}
	return nil
}

// Optimistic Locking
func (r *supervisorPostgresRepo) UpdateQuota(ctx context.Context, id, quota, version int) (model.Supervisor, error) {
	var s model.Supervisor
	err := r.db.QueryRow(ctx, `
		UPDATE supervisors SET quota = $1, version = version + 1
		WHERE id = $2 AND version = $3 AND $1 >= supervised_count
		RETURNING id, name, quota, supervised_count, version, created_at`, 
		quota, id, version).
		Scan(&s.ID, &s.Name, &s.Quota, &s.SupervisedCount, &s.Version, &s.CreatedAt)
	
	if err == nil {
		return s, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return s, err
	}
	
	// Cek kenapa gagal
	var currentVer int
	if err := r.db.QueryRow(ctx, "SELECT version FROM supervisors WHERE id = $1", id).Scan(&currentVer); err == nil {
		if currentVer != version {
			return s, ErrVersionConflict
		}
	}
	return s, ErrQuotaBelowFilled
}
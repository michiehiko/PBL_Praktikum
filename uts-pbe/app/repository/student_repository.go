package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"uts-pbe/app/model"
)

var (
	ErrNotFound  = errors.New("data tidak ditemukan")
	ErrDuplicate = errors.New("data sudah ada")
)

type StudentRepository interface {
	FindAll(ctx context.Context, q model.StudentListQuery) ([]model.Student, int, error)
	FindByID(ctx context.Context, id int) (model.Student, error)
	FindByUserID(ctx context.Context, userID int) (model.Student, error)
	CreateWithUser(ctx context.Context, u model.User, s model.Student) (model.Student, error)
	Update(ctx context.Context, s model.Student) (model.Student, error)
	Delete(ctx context.Context, id int) error
}

type studentPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewStudentRepository(pool *pgxpool.Pool) StudentRepository {
	return &studentPostgresRepository{pool: pool}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func (r *studentPostgresRepository) FindAll(ctx context.Context, q model.StudentListQuery) ([]model.Student, int, error) {
	where := "WHERE deleted_at IS NULL"
	args := []any{}

	if q.Search != "" {
		args = append(args, "%"+q.Search+"%")
		where += fmt.Sprintf(" AND (nama ILIKE $%d OR nim ILIKE $%d)", len(args), len(args))
	}
	if q.Prodi != "" {
		args = append(args, q.Prodi)
		where += fmt.Sprintf(" AND prodi = $%d", len(args))
	}
	if q.Angkatan > 0 {
		args = append(args, q.Angkatan)
		where += fmt.Sprintf(" AND angkatan = $%d", len(args))
	}

	// Menghitung Total
	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM students "+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Sorting
	order := "ORDER BY nama ASC"
	if q.Sort == "-ipk_terakhir" {
		order = "ORDER BY ipk_terakhir DESC"
	} else if q.Sort == "nama" {
		order = "ORDER BY nama ASC"
	}

	args = append(args, q.PerPage, q.Offset())
	sqlText := fmt.Sprintf(
		"SELECT id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, created_at FROM students %s %s LIMIT $%d OFFSET $%d",
		where, order, len(args)-1, len(args),
	)

	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var hasil []model.Student
	for rows.Next() {
		var s model.Student
		if err := rows.Scan(&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan, &s.IpkTerakhir, &s.CreatedAt); err != nil {
			return nil, err
		}
		hasil = append(hasil, s)
	}

	return hasil, total, nil
}

func (r *studentPostgresRepository) FindByID(ctx context.Context, id int) (model.Student, error) {
	var s model.Student
	err := r.pool.QueryRow(ctx,
		"SELECT id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, created_at FROM students WHERE id = $1 AND deleted_at IS NULL", id,
	).Scan(&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan, &s.IpkTerakhir, &s.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		return model.Student{}, err
	}
	return s, nil
}

func (r *studentPostgresRepository) FindByUserID(ctx context.Context, userID int) (model.Student, error) {
	var s model.Student
	err := r.pool.QueryRow(ctx,
		"SELECT id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, created_at FROM students WHERE user_id = $1 AND deleted_at IS NULL", userID,
	).Scan(&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan, &s.IpkTerakhir, &s.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		return model.Student{}, err
	}
	return s, nil
}

// CreateWithUser menggunakan transaction
func (r *studentPostgresRepository) CreateWithUser(ctx context.Context, u model.User, s model.Student) (model.Student, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Student{}, err
	}
	defer tx.Rollback(ctx) // bakal di batalin klo gagal di tengah 

	// Insert User
	err = tx.QueryRow(ctx,
		`INSERT INTO users (email, password, role) VALUES ($1, $2, $3) RETURNING id`,
		u.Email, u.Password, u.Role,
	).Scan(&s.UserID)

	if err != nil {
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, err
	}

	// Insert Student
	err = tx.QueryRow(ctx,
		`INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir) 
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at`,
		s.UserID, s.NIM, s.Nama, s.Prodi, s.Angkatan, s.IpkTerakhir,
	).Scan(&s.ID, &s.CreatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return model.Student{}, ErrDuplicate
		}
		return model.Student{}, err
	}

	// klo smw dah berhasil, commit
	if err = tx.Commit(ctx); err != nil {
		return model.Student{}, err
	}
	return s, nil
}

func (r *studentPostgresRepository) Update(ctx context.Context, s model.Student) (model.Student, error) {
	// NIM tidak diupdate 
	err := r.pool.QueryRow(ctx,
		`UPDATE students SET nama = $1, prodi = $2, angkatan = $3, ipk_terakhir = $4 WHERE id = $5 AND deleted_at IS NULL
		RETURNING id, user_id, nim, nama, prodi, angkatan, ipk_terakhir, created_at`,
		s.Nama, s.Prodi, s.Angkatan, s.IpkTerakhir, s.ID,
	).Scan(&s.ID, &s.UserID, &s.NIM, &s.Nama, &s.Prodi, &s.Angkatan, &s.IpkTerakhir, &s.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Student{}, ErrNotFound
		}
		return model.Student{}, err
	}
	return s, nil
}

func (r *studentPostgresRepository) Delete(ctx context.Context, id int) error {
	// SOFT DELETE
	tag, err := r.pool.Exec(ctx, "UPDATE students SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
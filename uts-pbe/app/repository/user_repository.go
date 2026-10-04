package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgpool"
	"github.com/jackc/pgx/v5/pgxpool"

	"uts-pbe/app/model"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (model.User, error)
	FindByID(ctx context.Context, id int) (model.User, error)
}

type userPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userPostgresRepository{pool: pool}
}

func (r *userPostgresRepository) FindByEmail(ctx context.Context, email string) (model.User, error) {
	var u model.User

	// JOIN ke students untuk mengecek soft delete.
	// Jika role-nya admin (tidak ada di tabel students), dia tetap bisa login.
	err := r.pool.QueryRow(ctx,
		`SELECT u.id, u.email, u.password, u.role, u.created_at
         FROM users u
         LEFT JOIN students s ON u.id = s.user_id
         WHERE LOWER(u.email) = LOWER($1) 
		 AND (s.deleted_at IS NULL OR u.role != 'mahasiswa')`, email,
	).Scan(&u.ID, &u.Email, &u.Password, &u.Role, &u.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, errors.New("data tidak ditemukan") // ErrNotFound
		}
		return model.User{}, fmt.Errorf("mengambil user: %w", err)
	}

	return u, nil
}

func (r *userPostgresRepository) FindByID(ctx context.Context, id int) (model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password, role, created_at FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Password, &u.Role, &u.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.User{}, errors.New("data tidak ditemukan")
		}
		return model.User{}, fmt.Errorf("mengambil user: %w", err)
	}
	return u, nil
}

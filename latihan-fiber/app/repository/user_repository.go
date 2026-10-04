package repository

import (
    "context"
    "errors"
    "fmt"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
    "github.com/jackc/pgx/v5/pgxpool"

    "latihan-fiber/app/model"
)

//var ErrDuplicate = errors.New("data sudah ada")

type UserRepository interface {
    Create(ctx context.Context, u model.User) (model.User, error)
    FindByUsername(ctx context.Context, username string) (model.User, error)
    FindByID(ctx context.Context, id int) (model.User, error)
}

type userPostgresRepository struct {
    pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
    return &userPostgresRepository{pool: pool}
}

func (r *userPostgresRepository) Create(ctx context.Context, u model.User) (model.User, error) {
    err := r.pool.QueryRow(ctx,
        `INSERT INTO users (username, email, password, role, is_active)
         VALUES ($1, $2, $3, $4, $5)
         RETURNING id, created_at`,
        u.Username, u.Email, u.Password, u.Role, u.IsActive,
    ).Scan(&u.ID, &u.CreatedAt)

    if err != nil {
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) && pgErr.Code == "23505" { 
            return model.User{}, ErrDuplicate
        }
        return model.User{}, fmt.Errorf("membuat user: %w", err)
    }

    return u, nil
}

// buat nyocokin usn pas login n case insensitive
func (r *userPostgresRepository) FindByUsername(
    ctx context.Context, username string,
) (model.User, error) {
    var u model.User

    err := r.pool.QueryRow(ctx,
        `SELECT id, username, email, password, role, is_active, created_at
         FROM users WHERE LOWER(username) = LOWER($1)`, username,
    ).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role,
        &u.IsActive, &u.CreatedAt)

    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return model.User{}, ErrNotFound // errnotfound dr student repo
        }
        return model.User{}, fmt.Errorf("mengambil user: %w", err)
    }

    return u, nil
}

func (r *userPostgresRepository) FindByID(
    ctx context.Context, id int,
) (model.User, error) {
    var u model.User

    err := r.pool.QueryRow(ctx,
        `SELECT id, username, email, password, role, is_active, created_at
         FROM users WHERE id = $1`, id,
    ).Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role,
        &u.IsActive, &u.CreatedAt)

    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return model.User{}, ErrNotFound
        }
        return model.User{}, fmt.Errorf("mengambil user: %w", err)
    }

    return u, nil
}
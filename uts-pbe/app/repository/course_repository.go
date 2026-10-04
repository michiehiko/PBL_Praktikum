package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"uts-pbe/app/model"
)

type CourseRepository interface {
	FindAll(ctx context.Context, q model.CourseListQuery) ([]model.Course, error)
}

type coursePostgresRepository struct {
	pool *pgxpool.Pool
}

func NewCourseRepository(pool *pgxpool.Pool) CourseRepository {
	return &coursePostgresRepository{pool: pool}
}

func (r *coursePostgresRepository) FindAll(ctx context.Context, q model.CourseListQuery) ([]model.Course, error) {
	where := "WHERE 1=1"
	args := []any{}
	having := ""

	if q.Search != "" {
		args = append(args, "%"+q.Search+"%")
		where += fmt.Sprintf(" AND (c.kode_mk ILIKE $%d OR c.nama_mk ILIKE $%d)", len(args), len(args))
	}
	if q.Semester > 0 {
		args = append(args, q.Semester)
		where += fmt.Sprintf(" AND c.semester = $%d", len(args))
	}
	if q.Available != nil && *q.Available {
		having = "HAVING (c.kuota - COUNT(e.id)::int) > 0"
	}

	query := fmt.Sprintf(`
		SELECT c.id, c.kode_mk, c.nama_mk, c.sks, c.semester, c.kuota,
		       COUNT(e.id)::int AS terisi, 
			   (c.kuota - COUNT(e.id)::int) AS sisa_kuota, 
			   c.created_at
		FROM courses c
		LEFT JOIN enrollments e ON c.id = e.course_id
		%s
		GROUP BY c.id
		%s
		ORDER BY c.semester ASC, c.kode_mk ASC
	`, where, having)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hasil []model.Course
	for rows.Next() {
		var c model.Course
		if err := rows.Scan(&c.ID, &c.KodeMK, &c.NamaMK, &c.SKS, &c.Semester, &c.Kuota, &c.Terisi, &c.SisaKuota, &c.CreatedAt); err != nil {
			return nil, err
		}
		hasil = append(hasil, c)
	}

	return hasil, nil
}

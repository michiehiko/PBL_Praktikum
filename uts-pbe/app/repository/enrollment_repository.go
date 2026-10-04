package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"uts-pbe/app/model"
)

var (
	ErrQuotaFull = errors.New("kuota mata kuliah sudah penuh")
	ErrSKSLimit  = errors.New("melebihi batas maksimal SKS")
)

type EnrollmentRepository interface {
	EnrollStudent(ctx context.Context, studentID int, courseID int, tahunAkademik string, ipk float64) (model.Enrollment, error)
	Delete(ctx context.Context, id int, studentID int) error
	GetStudentEnrollments(ctx context.Context, studentID int) ([]model.Course, int, error) // Total SKS
}

type enrollmentPostgresRepository struct {
	pool *pgxpool.Pool
}

func NewEnrollmentRepository(pool *pgxpool.Pool) EnrollmentRepository {
	return &enrollmentPostgresRepository{pool: pool}
}

// TRANSACTION & ROW LOCKING
func (r *enrollmentPostgresRepository) EnrollStudent(ctx context.Context, studentID int, courseID int, tahunAkademik string, ipk float64) (model.Enrollment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Enrollment{}, err
	}
	defer tx.Rollback(ctx)

	// Row Locking pada Course (Mencegah Race Condition Kuota)
	var sks, kuota, terisi int
	err = tx.QueryRow(ctx, `
		SELECT sks, kuota, (SELECT COUNT(*) FROM enrollments WHERE course_id = $1) AS terisi 
		FROM courses WHERE id = $1 FOR UPDATE
	`, courseID).Scan(&sks, &kuota, &terisi)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Enrollment{}, ErrNotFound
		}
		return model.Enrollment{}, err
	}

	// Cek Kuota
	if terisi >= kuota {
		return model.Enrollment{}, ErrQuotaFull
	}

	// Cek SKS yang sudah diambil di tahun ini
	var currentSKS int
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(c.sks), 0) FROM enrollments e 
		JOIN courses c ON e.course_id = c.id 
		WHERE e.student_id = $1 AND e.tahun_akademik = $2
	`, studentID, tahunAkademik).Scan(&currentSKS)
	if err != nil {
		return model.Enrollment{}, err
	}

	// Hitung Batas SKS berdasarkan IPK
	batasSKS := 18
	if ipk >= 3.00 {
		batasSKS = 24
	} else if ipk >= 2.50 {
		batasSKS = 21
	}

	if currentSKS+sks > batasSKS {
		return model.Enrollment{}, fmt.Errorf("%w. Sisa SKS Anda: %d", ErrSKSLimit, batasSKS-currentSKS)
	}

	// Insert Enrollment
	var e model.Enrollment
	err = tx.QueryRow(ctx, `
		INSERT INTO enrollments (student_id, course_id, tahun_akademik) 
		VALUES ($1, $2, $3) RETURNING id, student_id, course_id, tahun_akademik, created_at
	`, studentID, courseID, tahunAkademik).Scan(&e.ID, &e.StudentID, &e.CourseID, &e.TahunAkademik, &e.CreatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return model.Enrollment{}, ErrDuplicate // Duplicate = udh ambil
		}
		return model.Enrollment{}, err
	}

	// Commit Transaksi
	if err = tx.Commit(ctx); err != nil {
		return model.Enrollment{}, err
	}
	return e, nil
}

func (r *enrollmentPostgresRepository) Delete(ctx context.Context, id int, studentID int) error {
	// Menambahkan studentID memastikan user hanya bisa menghapus KRS miliknya
	tag, err := r.pool.Exec(ctx, "DELETE FROM enrollments WHERE id = $1 AND student_id = $2", id, studentID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Digunakan di Endpoint 5 (Detail Mahasiswa)
func (r *enrollmentPostgresRepository) GetStudentEnrollments(ctx context.Context, studentID int) ([]model.Course, int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.kode_mk, c.nama_mk, c.sks, c.semester
		FROM enrollments e
		JOIN courses c ON e.course_id = c.id
		WHERE e.student_id = $1
	`, studentID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var courses []model.Course
	totalSKS := 0
	for rows.Next() {
		var c model.Course
		if err := rows.Scan(&c.ID, &c.KodeMK, &c.NamaMK, &c.SKS, &c.Semester); err != nil {
			return nil, 0, err
		}
		totalSKS += c.SKS
		courses = append(courses, c)
	}
	return courses, totalSKS, nil
}

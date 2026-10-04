package model

import "time"

type Student struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id"`
	NIM         string     `json:"nim"`
	Nama        string     `json:"nama"`
	Prodi       string     `json:"prodi"`
	Angkatan    int        `json:"angkatan"`
	IpkTerakhir float64    `json:"ipk_terakhir"`
	CreatedAt   time.Time  `json:"created_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// Request GET /students
type StudentListQuery struct {
	Page     int
	PerPage  int
	Prodi    string
	Angkatan int
	Search   string
	Sort     string
}

func (q StudentListQuery) Offset() int {
	return (q.Page - 1) * q.PerPage
}

// Request POST /students (bikin User dan Student)
type CreateStudentRequest struct {
	NIM         string   `json:"nim" validate:"required,numeric,len=12"`
	Nama        string   `json:"nama" validate:"required"`
	Email       string   `json:"email" validate:"required,email"`
	Prodi       string   `json:"prodi" validate:"required"`
	Angkatan    int      `json:"angkatan" validate:"required,min=2000"`
	IpkTerakhir *float64 `json:"ipk_terakhir,omitempty" validate:"omitempty,min=0,max=4"`
}

// Request PUT /students/:id (NIM ga bole diubah)
type ReplaceStudentRequest struct {
	Nama        string   `json:"nama" validate:"required"`
	Prodi       string   `json:"prodi" validate:"required"`
	Angkatan    int      `json:"angkatan" validate:"required,min=2000"`
	IpkTerakhir *float64 `json:"ipk_terakhir,omitempty" validate:"omitempty,min=0,max=4"`
}

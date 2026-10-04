package model

import "time"

type Course struct {
	ID        int       `json:"id"`
	KodeMK    string    `json:"kode_mk"`
	NamaMK    string    `json:"nama_mk"`
	SKS       int       `json:"sks"`
	Semester  int       `json:"semester"`
	Kuota     int       `json:"kuota"`
	Terisi    int       `json:"terisi"` // Kolom virtual dari perhitungan enrollments
	SisaKuota int       `json:"sisa_kuota"` // Kolom virtual
	CreatedAt time.Time `json:"created_at"`
}

type CourseListQuery struct {
	Semester  int
	Search    string
	Available *bool
}
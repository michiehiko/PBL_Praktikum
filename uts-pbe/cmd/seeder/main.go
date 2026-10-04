package main

import (
	"context"
	"fmt"
	"log"

	"uts-pbe/config"
	"uts-pbe/database"
	"uts-pbe/helper"
)

func main() {
	// Environment & Koneksi Database
	config.LoadEnv()
	pool, err := database.NewPool(context.Background())
	if err != nil {
		log.Fatalf("Gagal terhubung ke database: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	fmt.Println("Memulai proses seeding database...")

	// Seed Roles & Permissions
	_, _ = pool.Exec(ctx, `
		INSERT INTO roles (name, description) VALUES 
		('admin', 'Administrator Akademik'), 
		('mahasiswa', 'Mahasiswa Aktif') 
		ON CONFLICT DO NOTHING;
	`)

	// Seed 1 Admin
	adminPass, _ := helper.HashPassword("admin123")
	var adminID int
	err = pool.QueryRow(ctx, `
		INSERT INTO users (email, password, role) 
		VALUES ('admin@siakad.com', $1, 'admin') 
		ON CONFLICT (email) DO UPDATE SET email=EXCLUDED.email 
		RETURNING id
	`, adminPass).Scan(&adminID)

	if err == nil {
		fmt.Println("Berhasil membuat 1 akun Admin (admin@siakad.com | pass: admin123)")
	}

	// Seed 20 Mahasiswa
	fmt.Println("Membuat 20 Mahasiswa...")
	for i := 1; i <= 20; i++ {
		nim := fmt.Sprintf("12345600%04d", i) // Contoh NIM 12 digit: 123456000001
		email := fmt.Sprintf("mahasiswa%d@siakad.com", i)
		passHash, _ := helper.HashPassword(nim) // Password = NIM

		var userID int
		err := pool.QueryRow(ctx, `
			INSERT INTO users (email, password, role) 
			VALUES ($1, $2, 'mahasiswa') 
			ON CONFLICT (email) DO UPDATE SET email=EXCLUDED.email 
			RETURNING id
		`, email, passHash).Scan(&userID)

		if err == nil {
			_, err = pool.Exec(ctx, `
				INSERT INTO students (user_id, nim, nama, prodi, angkatan, ipk_terakhir) 
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (nim) DO NOTHING
			`, userID, nim, fmt.Sprintf("Mahasiswa Dummy %d", i), "Teknik Informatika", 2024, 3.50)
		}
	}
	fmt.Println("Berhasil membuat 20 Mahasiswa.")

	// Seed 10 Mata Kuliah
	fmt.Println("Membuat 10 Mata Kuliah...")
	for i := 1; i <= 10; i++ {
		kodeMK := fmt.Sprintf("TIK%03d", i) // TIK001, TIK002, dst
		namaMK := fmt.Sprintf("Mata Kuliah Dasar %d", i)

		_, err := pool.Exec(ctx, `
			INSERT INTO courses (kode_mk, nama_mk, sks, semester, kuota) 
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (kode_mk) DO NOTHING
		`, kodeMK, namaMK, 3, 1, 40)

		if err != nil {
			log.Printf("Gagal insert MK %s: %v", kodeMK, err)
		}
	}
	fmt.Println("Berhasil membuat 10 Mata Kuliah.")
	fmt.Println("Seeding selesai!")
}

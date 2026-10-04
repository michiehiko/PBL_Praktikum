# Student Management API - Praktikum Backend Lanjut

Repositori ini berisi REST API untuk manajemen data Mahasiswa (Students) yang dibangun menggunakan **Go (Fiber)** dan **PostgreSQL**. Proyek ini merupakan implementasi dari Modul 3: Database & *Repository Pattern*.

## Persiapan Environment Variabel

Untuk menjalankan aplikasi ini, kamu memerlukan file konfigurasi environment.

**Langkah persiapan:**
1. Salin file `.env.example` yang ada di repositori ini dan ubah namanya menjadi `.env`.
2. Isi nilai pada file `.env` tersebut menyesuaikan dengan konfigurasi komputermu (terutama bagian `DB_PASSWORD`).

Berikut adalah daftar variabel yang harus ada di dalam file `.env`:

| Variabel | Deskripsi 
| :--- | :--- 
| `APP_PORT` | Port server aplikasi Go berjalan 
| `DB_HOST` | Alamat host PostgreSQL 
| `DB_PORT` | Port layanan PostgreSQL 
| `DB_USER` | Username PostgreSQL 
| `DB_PASSWORD` | Kata sandi user PostgreSQL kamu
| `DB_NAME` | Nama database yang akan digunakan 
| `DB_SSLMODE` | Mode SSL database (wajib disable untuk lokal) 
| `DB_MAX_CONNS` | Batas maksimal koneksi di *connection pool* 

---

## Cara Menyiapkan Basis Data 

**1. Masuk ke PostgreSQL**
Masuk ke sistem database menggunakan *user* bawaan postgres:
```bash
psql -U postgres
```
*(Masukkan password postgres milikmu jika diminta)*

**2. Buat Database Baru**
Jalankan perintah SQL berikut untuk membuat database kosong:
```sql
CREATE DATABASE praktikum_backend;
```
Keluar dari prompt psql dengan mengetik \q lalu tekan Enter.

**3. Jalankan Migrasi Tabel**
Arahkan terminalmu ke folder root proyek ini, lalu jalankan file migrasi untuk membuat tabel beserta indeksnya:
```bash
psql -U postgres -d praktikum_backend -f migrations/001_create_students.sql
```
Jika terminal memunculkan pesan CREATE TABLE dan CREATE INDEX, artinya basis data kamu sudah siap digunakan.

## Skema Table
Data mahasiswa disimpan di dalam tabel students. Untuk melihat detail skema pembuatan tabel dan indeks yang digunakan, silakan buka file 001_create_students.sql yang berada di dalam folder migrations/.

Catatan Penting pada Skema:

a) NIM: Menggunakan UNIQUE INDEX langsung dari basis data. Ini mencegah terjadinya duplikasi data secara mutlak jika ada request berbarengan (Race Condition).

b) Pencarian (Search): Menggunakan klausa ILIKE yang didukung oleh indeks pada LOWER(name), sehingga performa pencarian tetap cepat meskipun data sudah mencapai ratusan ribu baris.

## Menjalankan Aplikasi
Setelah .env dan basis data siap, jalankan perintah ini untuk mengunduh modul dan menyalakan server lokal:
```bash
# Merapikan dan mengunduh dependencies
go mod tidy
# Menjalankan aplikasi
go run .
```
Server akan menyala di http://localhost:3000. Kamu bisa verifikasi koneksi database dengan menembak endpoint GET /api/v1/health.

package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
)

// memuat variabel dari .env
func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("peringatan: berkas .env tidak ditemukan, memakai environment sistem")
	}
}

// mengambil nilai string
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

// mengambil nilai integer
func GetEnvInt(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("peringatan: %s bukan angka, memakai bawaan %d", key, fallback)
		return fallback
	}
	return parsed
}

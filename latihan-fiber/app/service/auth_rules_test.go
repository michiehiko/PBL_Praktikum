package service

import (
    "testing"
)

func TestCheckPasswordStrength(t *testing.T) {
    tests := []struct {
        name     string
        password string
        want     string
    }{
        {
            name:     "Terlalu Pendek",
            password: "abc",
            want:     "minimal 8 karakter",
        },
        {
            name:     "Hanya Huruf",
            password: "abcdefghij",
            want:     "harus memuat huruf dan angka",
        },
        {
            name:     "Hanya Angka",
            password: "1234567890",
            want:     "harus memuat huruf dan angka",
        },
        {
            name:     "Password Terlalu Umum",
            password: "password123",
            want:     "password terlalu lemah",
        },
        {
            name:     "Password Kuat",
            password: "SuperSecret99!",
            want:     "", // kosong berarti lolos validasi
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := checkPasswordStrength(tt.password)
            if got != tt.want {
                t.Errorf("checkPasswordStrength(%q) = %q, want %q", tt.password, got, tt.want)
            }
        })
    }
}
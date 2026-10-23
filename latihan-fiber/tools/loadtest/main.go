package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"

	"latihan-fiber/app/model"
	"latihan-fiber/helper"
)

// fire menjalankan pekerjaan secara serentak dan menghitung hasilnya
func fire(n int, job func(i int) string) map[string]int {
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan string, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int, out chan<- string) {
			defer wg.Done()
			<-start
			out <- job(i)
		}(i, results)
	}

	close(start) // Buka gerbang
	wg.Wait()    // Tunggu SEMUA goroutine selesai
	close(results)

	tally := map[string]int{}
	for r := range results {
		tally[r]++
	}
	return tally
}

func main() {
	mode := flag.String("mode", "crowd", "Mode uji beban: crowd, credits, atau swap")
	flag.Parse()

	_ = godotenv.Load(".env") 

	// Ambil secret dan issuer dari .env
	secret := os.Getenv("JWT_SECRET")
	issuer := os.Getenv("JWT_ISSUER")
	if issuer == "" {
		issuer = "latihan-fiber" // Fallback jika kosong
	}
	ttlStr := os.Getenv("JWT_ACCESS_TTL")
	ttl, _ := strconv.Atoi(ttlStr)
	if ttl == 0 {
		ttl = 15 // Fallback default 15 menit
	}

	jwtManager := helper.NewJWTManager(secret, issuer, time.Duration(ttl)*time.Minute)

	baseURL := "http://localhost:3000/api/v1"
	client := &http.Client{Timeout: 5 * time.Second}

	switch *mode {
	case "crowd":
		fmt.Println("60 mahasiswa berebut kelas IF399 secara serentak...")
		start := time.Now()

		tally := fire(60, func(i int) string {
			userID := i + 3 
			
			// Buat dummy user untuk mencetak token[
			u := model.User{
				ID:       userID,
				Username: fmt.Sprintf("mhs%03d", i+1),
				Role:     "user",
			}
			
			// Cetak token asli menggunakan JWTManager 
			token, err := jwtManager.GenerateAccess(u)
			if err != nil {
				return "TOKEN_ERROR"
			}

			// Endpoint IF399 (id=13) beserta payload kosong '{}' dan header Content-Type application/json
			req, _ := http.NewRequest("POST", baseURL+"/courses/13/enroll", bytes.NewBuffer([]byte("{}")))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				return "ERROR: " + err.Error()
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != 201 && resp.StatusCode != 409 {
				return fmt.Sprintf("%d %s", resp.StatusCode, string(body))
			}
			return strconv.Itoa(resp.StatusCode)
		})

		fmt.Printf("waktu total: %v\n", time.Since(start))
		fmt.Println("hasil request:")
		for k, v := range tally {
			fmt.Printf("%s\t: %d\n", k, v)
		}
	default:
		fmt.Println("Mode tidak dikenali. Gunakan -mode crowd")
	}
}
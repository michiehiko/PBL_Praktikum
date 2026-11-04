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

	baseURL := "http://127.0.0.1:3000/api/v1"
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

	case "credits":
		fmt.Println("10 mahasiswa, masing-masing mendaftar ke 12 kelas 3 SKS secara serentak (batas 24 SKS)")
		start := time.Now()

		// 10 mahasiswa x 12 kelas = 120 request serentak
		tally := fire(120, func(i int) string {
			mhsIndex := i % 10             // Hasilnya 0 s.d 9
			courseID := (i / 10) + 1       // Hasilnya 1 s.d 12 (ID kelas IF101 sampai IF310)
			userID := mhsIndex + 3         // ID 3 (mhs001) s.d ID 12 (mhs010)

			u := model.User{
				ID:       userID,
				Username: fmt.Sprintf("mhs%03d", mhsIndex+1),
				Role:     "user",
			}

			token, err := jwtManager.GenerateAccess(u)
			if err != nil {
				return "TOKEN_ERROR"
			}

			// Tembak endpoint berdasarkan courseID (1 sampai 12)
			url := baseURL + "/courses/" + strconv.Itoa(courseID) + "/enroll"
			req, _ := http.NewRequest("POST", url, bytes.NewBuffer([]byte("{}")))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				return "ERROR: " + err.Error()
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			// Hanya tangkap status code-nya untuk dihitung
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
	case "swap":
		fmt.Println("Menyiapkan data awal: Mendaftarkan 20 mhs ke IF101 dan 20 mhs ke IF102")
		for i := 0; i < 40; i++ {
			userID := i + 2
			u := model.User{ID: userID, Username: fmt.Sprintf("mhs%03d", i+1), Role: "user"}
			token, _ := jwtManager.GenerateAccess(u)
			
			// 20 mhs pertama ke IF101 (id: 1), 20 sisanya ke IF102 (id: 2)
			courseID := 1
			if i >= 20 {
				courseID = 2
			}
			
			url := baseURL + "/courses/" + strconv.Itoa(courseID) + "/enroll"
			req, _ := http.NewRequest("POST", url, bytes.NewBuffer([]byte("{}")))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
			}
		}

		fmt.Println("20 pasang mahasiswa pindah kelas berlawanan arah secara serentak")
		start := time.Now()

		tally := fire(40, func(i int) string {
			userID := i + 3
			u := model.User{ID: userID, Username: fmt.Sprintf("mhs%03d", i+1), Role: "user"}
			token, _ := jwtManager.GenerateAccess(u)

			fromCourse := 1
			toCourse := 2
			if i >= 20 {
				fromCourse = 2
				toCourse = 1
			}

			payload := fmt.Sprintf(`{"from_course_id": %d, "to_course_id": %d}`, fromCourse, toCourse)
			req, _ := http.NewRequest("POST", baseURL+"/enrollments/swap", bytes.NewBuffer([]byte(payload)))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				return "ERROR: " + err.Error()
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != 200 && resp.StatusCode != 409 {
				return fmt.Sprintf("%d %s", resp.StatusCode, string(body))
			}
			return strconv.Itoa(resp.StatusCode)
		})

		fmt.Printf("waktu total: %v\n", time.Since(start))
		fmt.Println("hasil request:")
		for k, v := range tally {
			fmt.Printf("%s\t: %d\n", k, v)
		}
	case "assign_ta":
		fmt.Println("30 penetapan serentak ke seorang pembimbing berkuota 3 (Dr. Cheng Xiaoshi)")
		start := time.Now()

		tally := fire(30, func(i int) string {
			// bertindak sebagai ADMIN (id: 1, role: admin) karena butuh permission supervision:manage
			adminUser := model.User{
				ID:       1,
				Username: "admin",
				Role:     "admin",
			}
			token, _ := jwtManager.GenerateAccess(adminUser)

			studentID := i + 2 // Mahasiswa ID 2 (mhs001) sampai 31 (mhs030)
			
			// Payload: {"student_id": X} ke dosen ID 1
			payload := fmt.Sprintf(`{"student_id": %d}`, studentID)
			
			req, _ := http.NewRequest("POST", baseURL+"/supervisors/1/assign", bytes.NewBuffer([]byte(payload)))
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
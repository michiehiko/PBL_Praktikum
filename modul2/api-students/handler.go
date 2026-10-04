package main

import (
	"sort"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

//in memory database
var students = []Student{}
var nextID = 1

//mencari index student berdasarkan id
func findStudentIndex(id int) int {
	for i := range students {
		if students[i].ID == id {
			return i
		}
	}
	return -1
}

//memeriksan keyword search di nama (case insensitive)
func matchStudentSearch(s Student, keyword string) bool {
	keyword = strings.ToLower(keyword)
	return strings.Contains(strings.ToLower(s.Nama), keyword)
}

//mengambil param id dari url dan memastikan angka positif
func paramID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, false
	}	
	return id, true
}

// GET /api/v1/students
func listStudents(c *fiber.Ctx) error {
	q := parseListQuery(c)

	// 1. Saring (Filter & Search)
	var result []Student
	for _, s := range students {
		if q.IsActive != nil && s.IsActive != *q.IsActive {
			continue
		}
		if q.Search != "" && !matchStudentSearch(s, q.Search) {
			continue
		}
		result = append(result, s)
	}

	// 2. Urutkan (Sort)
	sort.SliceStable(result, func(i, j int) bool {
		var isLess bool
		switch q.Sort {
		case "nim":
			isLess = result[i].NIM < result[j].NIM
		case "name":
			isLess = result[i].Nama < result[j].Nama
		case "IPK":
			isLess = result[i].IPK < result[j].IPK
		default: // default ke "id"
			isLess = result[i].ID < result[j].ID
		}
		if q.Order == "desc" {
			return !isLess
		}
		return isLess
	})

	// 3. Potong sesuai Halaman (Pagination)
	total := len(result)
	totalPages := (total + q.Limit - 1) / q.Limit
	start := (q.Page - 1) * q.Limit
	if start > total {
		start = total
	}
	end := start + q.Limit
	if end > total {
		end = total
	}

	return okList(c, "daftar mahasiswa berhasil diambil", result[start:end], &Meta{
		Page: q.Page, Limit: q.Limit, Total: total, TotalPages: totalPages,
	})
}

// GET /api/v1/students/:id
func getStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid {
		return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}

	i := findStudentIndex(id)
	if i == -1 {
		return fail(c, fiber.StatusNotFound, "mahasiswa tidak ditemukan") //404
	}

	return ok(c, "mahasiswa ditemukan", students[i]) //200
}

// POST /api/v1/students
func createStudent(c *fiber.Ctx) error {
	var req CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid") 
	}

	// Validasi Input
	errs := map[string]string{}
	req.NIM = strings.TrimSpace(req.NIM)
	req.Nama = strings.TrimSpace(req.Nama)

	if req.NIM == "" { errs["nim"] = "wajib diisi" }
	if req.Nama == "" { errs["nama"] = "wajib diisi" }
	if len(errs) > 0 {
		return failValidation(c, errs) //422
	}

	// Cek Duplikasi NIM (409)
	for _, s := range students {
		if strings.EqualFold(s.NIM, req.NIM) {
			return fail(c, fiber.StatusConflict, "NIM sudah terdaftar") //409
		}
	}

	baru := Student{
		ID:       nextID,
		NIM:      req.NIM,
		Nama:     req.Nama,
		IPK:    req.IPK,
		IsActive: req.IsActive,
	}
	students = append(students, baru)
	nextID++

	return created(c, "mahasiswa berhasil ditambahkan", baru, "/api/v1/students/"+strconv.Itoa(baru.ID)) // Status 201 + Header Location
}

// PUT /api/v1/students/:id (ganti seluruh isi)
func replaceStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid { return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif") }

	i := findStudentIndex(id)
	if i == -1 { return fail(c, fiber.StatusNotFound, "mahasiswa tidak ditemukan") }

	var req ReplaceStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	// Validasi PUT: semua field diwajibkan ada
	errs := map[string]string{}
	if strings.TrimSpace(req.NIM) == "" { errs["nim"] = "wajib diisi pada PUT" }
	if strings.TrimSpace(req.Nama) == "" { errs["nama"] = "wajib diisi pada PUT" }
	if len(errs) > 0 { return failValidation(c, errs) }

	// Cek NIM ganda (kecuali milik sendiri)
	for idx, s := range students {
		if idx != i && strings.EqualFold(s.NIM, req.NIM) {
			return fail(c, fiber.StatusConflict, "NIM sudah dipakai mahasiswa lain")
		}
	}

	// Timpa seluruh data
	students[i].NIM = req.NIM
	students[i].Nama = req.Nama
	students[i].IPK = req.IPK
	students[i].IsActive = req.IsActive

	return ok(c, "data mahasiswa berhasil diganti seluruhnya", students[i])
}

// PATCH /api/v1/students/:id (ubah sebagian isi)
func patchStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid { return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif") }

	i := findStudentIndex(id)
	if i == -1 { return fail(c, fiber.StatusNotFound, "mahasiswa tidak ditemukan") }

	var req PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}

	if req.NIM == nil && req.Nama == nil && req.IPK == nil && req.IsActive == nil {
		return fail(c, fiber.StatusBadRequest, "tidak ada field yang diubah")
	}

	// Update field hanya jika nilainya dikirim (tidak nil)
	if req.NIM != nil {
		if strings.TrimSpace(*req.NIM) == "" {
			return failValidation(c, map[string]string{"nim": "tidak boleh kosong"})
		}
		// Cek duplikasi NIM
		for idx, s := range students {
			if idx != i && strings.EqualFold(s.NIM, *req.NIM) {
				return fail(c, fiber.StatusConflict, "NIM sudah dipakai mahasiswa lain")
			}
		}
		students[i].NIM = *req.NIM
	}
	if req.Nama != nil {
		if strings.TrimSpace(*req.Nama) == "" {
			return failValidation(c, map[string]string{"nama": "tidak boleh kosong"})
		}
		students[i].Nama = *req.Nama
	}
	if req.IPK != nil {
		students[i].IPK = *req.IPK
	}
	if req.IsActive != nil {
		students[i].IsActive = *req.IsActive
	}

	return ok(c, "data mahasiswa diperbarui sebagian", students[i])
}

// DELETE /api/v1/students/:id
func deleteStudent(c *fiber.Ctx) error {
	id, valid := paramID(c)
	if !valid { return fail(c, fiber.StatusBadRequest, "id harus berupa angka positif") }

	i := findStudentIndex(id)
	if i == -1 { return fail(c, fiber.StatusNotFound, "mahasiswa tidak ditemukan") }

	// Hapus elemen dari slice (array)
	students = append(students[:i], students[i+1:]...)

	return noContent(c) //204
}

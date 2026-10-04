package service

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"uts-pbe/app/model"
	"uts-pbe/app/repository"
	"uts-pbe/helper"
)

type StudentService struct {
	repo           repository.StudentRepository
	enrollmentRepo repository.EnrollmentRepository
	perms          *helper.PermissionSet
}

func NewStudentService(repo repository.StudentRepository, enrollRepo repository.EnrollmentRepository, perms *helper.PermissionSet) *StudentService {
	return &StudentService{repo: repo, enrollmentRepo: enrollRepo, perms: perms}
}

func translateStudentError(err error, entity string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound(entity + " tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Conflict(entity + " sudah dipakai/terdaftar")
	default:
		return helper.Internal(err)
	}
}

func (s *StudentService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	// Parse Offset Pagination
	page := c.QueryInt("page", 1)
	perPage := c.QueryInt("per_page", 10)
	if perPage > 50 {
		perPage = 50
	}

	q := model.StudentListQuery{
		Page:     page,
		PerPage:  perPage,
		Prodi:    c.Query("prodi"),
		Angkatan: c.QueryInt("angkatan", 0),
		Search:   c.Query("search"),
		Sort:     c.Query("sort", "nama"), // default nama
	}

	rows, total, err := s.repo.FindAll(ctx, q)
	if err != nil {
		return helper.Internal(err)
	}

	lastPage := int(math.Ceil(float64(total) / float64(perPage)))
	if lastPage == 0 {
		lastPage = 1
	}

	meta := &model.Meta{
		CurrentPage: page,
		PerPage:     perPage,
		Total:       total,
		LastPage:    lastPage,
	}

	return helper.SuccessList(c, "Data mahasiswa berhasil diambil", rows, meta)
}

func (s *StudentService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id tidak valid")
	}

	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateStudentError(err, "mahasiswa")
	}

	// Cek Otorisasi (Hanya Admin atau Pemilik Data)
	if current.Role != "admin" && current.UserID != student.UserID {
		return helper.Forbidden("tidak berhak mengakses data mahasiswa ini")
	}

	// Mengambil data KRS
	courses, totalSKS, err := s.enrollmentRepo.GetStudentEnrollments(ctx, student.ID)
	if err != nil {
		return helper.Internal(err)
	}

	batasSKS := 18
	if student.IpkTerakhir >= 3.00 {
		batasSKS = 24
	} else if student.IpkTerakhir >= 2.50 {
		batasSKS = 21
	}

	return helper.Success(c, fiber.StatusOK, "detail mahasiswa berhasil diambil", fiber.Map{
		"student":   student,
		"courses":   courses,
		"total_sks": totalSKS,
		"batas_sks": batasSKS,
	})
}

func (s *StudentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	// Password default = NIM yang di-hash
	hashedPassword, err := helper.HashPassword(req.NIM)
	if err != nil {
		return helper.Internal(err)
	}

	user := model.User{
		Email:    req.Email,
		Password: hashedPassword,
		Role:     "mahasiswa",
	}

	ipk := 0.00
	if req.IpkTerakhir != nil {
		ipk = *req.IpkTerakhir
	}

	student := model.Student{
		NIM:         req.NIM,
		Nama:        req.Nama,
		Prodi:       req.Prodi,
		Angkatan:    req.Angkatan,
		IpkTerakhir: ipk,
	}

	baru, err := s.repo.CreateWithUser(ctx, user, student)
	if err != nil {
		// Pengecekan pesan error karena ada 2 tabel yang dicek duplikatnya
		if errors.Is(err, repository.ErrDuplicate) {
			return helper.Validation(map[string]string{"nim/email": "NIM atau Email sudah terdaftar"})
		}
		return translateStudentError(err, "mahasiswa")
	}

	return helper.Created(c, "mahasiswa berhasil dibuat", baru, "/api/v1/students/"+strconv.Itoa(baru.ID))
}

func (s *StudentService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id tidak valid")
	}

	var req model.ReplaceStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	saatIni, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateStudentError(err, "mahasiswa")
	}

	ipk := saatIni.IpkTerakhir
	if req.IpkTerakhir != nil {
		ipk = *req.IpkTerakhir
	}

	hasil, err := s.repo.Update(ctx, model.Student{
		ID:          id,
		Nama:        req.Nama,
		Prodi:       req.Prodi,
		Angkatan:    req.Angkatan,
		IpkTerakhir: ipk,
	})

	if err != nil {
		return translateStudentError(err, "mahasiswa")
	}

	return helper.Success(c, fiber.StatusOK, "data mahasiswa berhasil diperbarui", hasil)
}

func (s *StudentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id tidak valid")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return translateStudentError(err, "mahasiswa")
	}
	return helper.NoContent(c)
}
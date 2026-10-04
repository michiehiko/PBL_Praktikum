package service

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"uts-pbe/app/model"
	"uts-pbe/app/repository"
	"uts-pbe/helper"
)

type EnrollmentService struct {
	repo        repository.EnrollmentRepository
	studentRepo repository.StudentRepository
}

func NewEnrollmentService(repo repository.EnrollmentRepository, studentRepo repository.StudentRepository) *EnrollmentService {
	return &EnrollmentService{repo: repo, studentRepo: studentRepo}
}

func (s *EnrollmentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	if current.Role != "mahasiswa" {
		return helper.Forbidden("hanya mahasiswa yang dapat mengambil KRS")
	}

	var req model.CreateEnrollmentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	// Mengambil profil mahasiswa untuk mendapatkan student_id dan IPK
	student, err := s.studentRepo.FindByUserID(ctx, current.UserID)
	if err != nil {
		return helper.Internal(err)
	}

	enrollment, err := s.repo.EnrollStudent(ctx, student.ID, req.CourseID, req.TahunAkademik, student.IpkTerakhir)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return helper.Conflict("Mata kuliah ini sudah Anda ambil pada tahun akademik yang sama")
		}
		if errors.Is(err, repository.ErrQuotaFull) || errors.Is(err, repository.ErrSKSLimit) {
			// Status 422 untuk pelanggaran kuota atau batas SKS
			return &helper.AppError{
				Status:  fiber.StatusUnprocessableEntity,
				Code:    "UNPROCESSABLE_ENTITY",
				Message: err.Error(),
			}
		}
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("Mata kuliah tidak ditemukan")
		}
		return helper.Internal(err)
	}

	return helper.Created(c, "Mata kuliah berhasil ditambahkan ke KRS", enrollment, "")
}

func (s *EnrollmentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	if current.Role != "mahasiswa" {
		return helper.Forbidden("hanya mahasiswa yang dapat membatalkan KRS")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id tidak valid")
	}

	student, err := s.studentRepo.FindByUserID(ctx, current.UserID)
	if err != nil {
		return helper.Internal(err)
	}

	if err := s.repo.Delete(ctx, id, student.ID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("KRS tidak ditemukan atau bukan milik Anda")
		}
		return helper.Internal(err)
	}

	return helper.NoContent(c)
}

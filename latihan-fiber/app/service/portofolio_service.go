package service

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

type PortfolioService struct {
	portfolios repository.PortfolioRepository
	students   repository.StudentRepository
}

func NewPortfolioService(p repository.PortfolioRepository, s repository.StudentRepository) *PortfolioService {
	return &PortfolioService{portfolios: p, students: s}
}

func (s *PortfolioService) GetPortfolio(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	studentID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id tidak valid")
	}

	portfolio, err := s.portfolios.GetByStudentID(ctx, studentID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("portofolio belum dibuat")
		}
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "portofolio ditemukan", portfolio)
}

func (s *PortfolioService) PutPortfolio(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id tidak valid")
	}
	// mengecek apakah mhs ada di db (psql)
	if _, err := s.students.FindByID(ctx, studentID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("mahasiswa tidak ditemukan di database")
		}
		return helper.Internal(err)
	}

	var req model.PutPortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}
	if req.Certificates == nil {
		req.Certificates = []model.Certificate{}
	}

	portfolio, err := s.portfolios.Upsert(ctx, studentID, req)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "portofolio berhasil disimpan", portfolio)
}

func (s *PortfolioService) AddProject(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	studentID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id tidak valid")
	}

	var req model.AddProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}
	// Normalisasi tech tags
	req.Technologies = NormalizeTags(req.Technologies)

	portfolio, err := s.portfolios.AddProject(ctx, studentID, model.Project{
		Title:        req.Title,
		Technologies: req.Technologies,
		Link:         req.Link,
		Year:         req.Year,
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("portofolio tidak ditemukan, buat dulu")
		}
		if errors.Is(err, repository.ErrProjectLimit) {
			return helper.NewError(fiber.StatusConflict, "PROJECT_LIMIT", "jumlah proyek sudah mencapai batas (20)")
		}
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusCreated, "proyek ditambahkan", portfolio)
}

func (s *PortfolioService) RemoveProject(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	studentID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id tidak valid")
	}
	projectID := c.Params("projectid")

	portfolio, err := s.portfolios.DeleteProject(ctx, studentID, projectID)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidID) {
			return helper.BadRequest("id proyek tidak valid")
		}
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("portofolio tidak ditemukan")
		}
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "proyek dihapus", portfolio)
}

func (s *PortfolioService) StatsTechnologies(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	stats, err := s.portfolios.StatsTechnologies(ctx)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "statistik teknologi diambil", stats)
}
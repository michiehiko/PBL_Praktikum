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

// GetPortfolio godoc
//
// @Summary      Portofolio seorang mahasiswa
// @Tags         students
// @Produce      json
// @Param        id   path      int  true  "ID mahasiswa"
// @Success      200  {object}  model.SuccessResponse{data=model.Portfolio}
// @Failure      400  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse  "Portofolio belum dibuat"
// @Security     BearerAuth
// @Router       /students/{id}/portfolio [get]
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

// PutPortfolio godoc
//
// @Summary      Membuat atau mengganti portofolio
// @Description  Upsert: bila portofolio belum ada, dibuat; bila sudah ada, diganti seluruhnya. Daftar proyek tidak tersentuh.
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        id    path      int                        true  "ID mahasiswa"
// @Param        body  body      model.PutPortfolioRequest  true  "Headline, bio, dan sertifikat"
// @Success      200   {object}  model.SuccessResponse{data=model.Portfolio}
// @Failure      400   {object}  model.ErrorResponse
// @Failure      404   {object}  model.ErrorResponse  "Mahasiswa tidak ditemukan"
// @Failure      422   {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /students/{id}/portfolio [put]
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

// AddProject godoc
//
// @Summary      Menambah proyek ke portofolio
// @Description  Maksimal 20 proyek per portofolio. Teknologi dinormalisasi (huruf kecil, tanpa duplikat).
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        id    path      int                     true  "ID mahasiswa"
// @Param        body  body      model.AddProjectRequest  true  "Data proyek"
// @Success      201   {object}  model.SuccessResponse{data=model.Portfolio}
// @Failure      400   {object}  model.ErrorResponse
// @Failure      404   {object}  model.ErrorResponse  "Portofolio belum dibuat"
// @Failure      409   {object}  model.ErrorResponse  "PROJECT_LIMIT"
// @Failure      422   {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /students/{id}/portfolio/projects [post]
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

// RemoveProject godoc
//
// @Summary      Menghapus satu proyek dari portofolio
// @Tags         students
// @Produce      json
// @Param        id         path      int     true  "ID mahasiswa"
// @Param        projectid  path      string  true  "ID proyek"
// @Success      200        {object}  model.SuccessResponse{data=model.Portfolio}
// @Failure      400        {object}  model.ErrorResponse
// @Failure      404        {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /students/{id}/portfolio/projects/{projectid} [delete]
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

// StatsTechnologies godoc
//
// @Summary      Statistik pemakaian teknologi
// @Description  Menghitung berapa kali setiap teknologi muncul pada seluruh proyek portofolio.
// @Tags         portfolios
// @Produce      json
// @Success      200  {object}  model.SuccessResponse{data=[]model.TechStat}
// @Failure      401  {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /portfolios/stats/technologies [get]
func (s *PortfolioService) StatsTechnologies(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	stats, err := s.portfolios.StatsTechnologies(ctx)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "statistik teknologi diambil", stats)
}
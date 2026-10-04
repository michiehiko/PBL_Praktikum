package service

import (
	"github.com/gofiber/fiber/v2"

	"uts-pbe/app/model"
	"uts-pbe/app/repository"
	"uts-pbe/helper"
)

type CourseService struct {
	repo repository.CourseRepository
}

func NewCourseService(repo repository.CourseRepository) *CourseService {
	return &CourseService{repo: repo}
}

func (s *CourseService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var available *bool
	if c.Query("available") == "true" {
		t := true
		available = &t
	}

	q := model.CourseListQuery{
		Semester:  c.QueryInt("semester", 0),
		Search:    c.Query("search"),
		Available: available,
	}

	rows, err := s.repo.FindAll(ctx, q)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "Daftar mata kuliah berhasil diambil", rows)
}

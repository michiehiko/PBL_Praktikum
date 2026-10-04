package service

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

type StudentService struct {
	repo  repository.StudentRepository
	perms *helper.PermissionSet
}

func NewStudentService(repo repository.StudentRepository, perms *helper.PermissionSet) *StudentService {
	return &StudentService{repo: repo, perms: perms}
}

// bdenerin bug 7: Menghapus c *fiber.Ctx, dan me-return helper.Internal di blok default
func translateError(err error, entity string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound(entity + " tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Conflict("identitas " + entity + " sudah dipakai")
	default:
		return helper.Internal(err)
	}
}

func (s *StudentService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	// Content Negotiation (CSV vs JSON)
	format, err := helper.Negotiate(c, helper.FormatJSON, helper.FormatCSV)
	if err != nil {
		return err
	}

	q, err := helper.ParseCursorQuery(c)
	if err != nil {
		return err
	}

	rows, err := s.repo.FindAfterCursor(ctx, q)
	if err != nil {
		return helper.Internal(err)
	}

	hasMore := len(rows) > q.Limit
	if hasMore {
		rows = rows[:q.Limit] // buang data ekstra yg diambil
	}

	if format == helper.FormatCSV {
		return helper.WriteStudentsCSV(c, rows)
	}

	meta := &model.CursorMeta{Limit: q.Limit, HasMore: hasMore}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		meta.NextCursor = helper.EncodeCursor(last.CreatedAt, last.ID)
	}

	return helper.SuccessCursor(c, "daftar mahasiswa berhasil diambil", rows, meta)
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
		return helper.BadRequest("id harus berupa angka positif")
	}

	student, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err, "mahasiswa")
	}

	if !CanAccessStudent(current, student.OwnerID, s.perms, "student:read:any") {
		return helper.Forbidden("tidak berhak mengakses data mahasiswa ini")
	}

	return helper.Success(c, fiber.StatusOK, "mahasiswa ditemukan", student)
}

func (s *StudentService) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	var req model.CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	// validasi deklaratif: Menggunakan go-playground/validator
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	baru, err := s.repo.Create(ctx, model.Student{
		NIM:      strings.TrimSpace(req.NIM),
		Name:     strings.TrimSpace(req.Name),
		Grade:    strings.TrimSpace(req.Grade),
		IsActive: req.IsActive,
		OwnerID:  current.UserID,
	})

	if err != nil {
		return translateError(err, "mahasiswa")
	}

	return helper.Created(c, "mahasiswa berhasil dibuat", baru, "/api/v1/students/"+strconv.Itoa(baru.ID))
}

func (s *StudentService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.ReplaceStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	// validasi deklaratif
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	saatIni, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err, "mahasiswa")
	}

	if !CanAccessStudent(current, saatIni.OwnerID, s.perms, "student:update:any") {
		return helper.Forbidden("tidak berhak mengubah data mahasiswa ini")
	}

	hasil, err := s.repo.Update(ctx, model.Student{
		ID:       id,
		NIM:      strings.TrimSpace(req.NIM),
		Name:     strings.TrimSpace(req.Name),
		Grade:    strings.TrimSpace(req.Grade),
		IsActive: req.IsActive,
		OwnerID:  saatIni.OwnerID,
	})

	if err != nil {
		return translateError(err, "mahasiswa")
	}

	return helper.Success(c, fiber.StatusOK, "data mahasiswa berhasil diganti seluruhnya", hasil)
}

func (s *StudentService) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}

	if IsEmptyPatch(req) {
		return helper.BadRequest("tidak ada field yang diubah")
	}

	// validasi deklaratif
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	saatIni, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return translateError(err, "mahasiswa")
	}

	if !CanAccessStudent(current, saatIni.OwnerID, s.perms, "student:update:any") {
		return helper.Forbidden("tidak berhak mengubah data mahasiswa ini")
	}

	// ApplyPatch ga nampilin error karena struct dijamin sudah valid oleh tag validator
	updated := ApplyPatch(saatIni, req)
	
	hasil, err := s.repo.Update(ctx, updated)
	if err != nil {
		return translateError(err, "mahasiswa")
	}

	return helper.Success(c, fiber.StatusOK, "data mahasiswa diperbarui sebagian", hasil)
}

func (s *StudentService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id tidak valid")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return translateError(err, "mahasiswa")
	}
	return helper.NoContent(c)
}
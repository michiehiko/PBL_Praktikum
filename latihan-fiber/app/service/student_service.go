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

// List godoc
//
// @Summary      Daftar mahasiswa dengan cursor pagination
// @Description  Cursor diambil dari meta.next_cursor. Bila klien mengirim Accept: text/csv, response berupa CSV, bukan JSON.
// @Tags         students
// @Produce      json
// @Param        limit      query     int     false  "Jumlah per halaman, 1 sampai 100"  default(10)
// @Param        search     query     string  false  "Cari pada NIM atau nama"
// @Param        is_active  query     bool    false  "Saring status aktif"
// @Param        cursor     query     string  false  "Cursor dari meta.next_cursor"
// @Success      200        {object}  model.CursorPage{data=[]model.Student}
// @Failure      401        {object}  model.ErrorResponse
// @Failure      403        {object}  model.ErrorResponse  "Butuh permission student:list"
// @Security     BearerAuth
// @Router       /students [get]
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

	// Memanggil FindAfterCursor 
	rows, err := s.repo.FindAfterCursor(ctx, q)
	if err != nil {
		return helper.Internal(err)
	}

	hasMore := len(rows) > q.Limit
	if hasMore {
		rows = rows[:q.Limit] // Buang data ekstra 
	}

	if format == helper.FormatCSV {
		return helper.WriteStudentsCSV(c, rows)
	}

	meta := &model.CursorMeta{Limit: q.Limit, HasMore: hasMore}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		meta.NextCursor = helper.EncodeCursor(last.CreatedAt, last.ID)
	}

	// Memanggil SuccessCursor 
	return helper.SuccessCursor(c, "daftar mahasiswa berhasil diambil", rows, meta)
}

// Get godoc
//
// @Summary      Detail satu mahasiswa
// @Description  Pemilik data atau pemegang permission student:read:any boleh mengakses.
// @Tags         students
// @Produce      json
// @Param        id   path      int  true  "ID mahasiswa"
// @Success      200  {object}  model.SuccessResponse{data=model.Student}
// @Failure      400  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /students/{id} [get]
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

// Create godoc
//
// @Summary      Menambah mahasiswa
// @Description  Butuh permission student:create. Header Location berisi URL data yang baru dibuat.
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        body  body      model.CreateStudentRequest  true  "Data mahasiswa"
// @Success      201   {object}  model.SuccessResponse{data=model.Student}
// @Failure      403   {object}  model.ErrorResponse
// @Failure      409   {object}  model.ErrorResponse  "NIM sudah dipakai"
// @Failure      422   {object}  model.ErrorResponse  "Validasi gagal; detail per field ada di fields"
// @Security     BearerAuth
// @Router       /students [post]
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

// Replace godoc
//
// @Summary      Mengganti seluruh data mahasiswa (PUT)
// @Description  Semua field wajib diisi; field yang tidak dikirim akan menjadi nilai nol.
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        id    path      int                         true  "ID mahasiswa"
// @Param        body  body      model.ReplaceStudentRequest  true  "Data mahasiswa lengkap"
// @Success      200   {object}  model.SuccessResponse{data=model.Student}
// @Failure      400   {object}  model.ErrorResponse
// @Failure      403   {object}  model.ErrorResponse
// @Failure      404   {object}  model.ErrorResponse
// @Failure      409   {object}  model.ErrorResponse
// @Failure      422   {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /students/{id} [put]
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

// Patch godoc
//
// @Summary      Mengubah sebagian data mahasiswa (PATCH)
// @Description  Hanya field yang dikirim yang diubah. Body kosong ditolak dengan 400.
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        id    path      int                       true  "ID mahasiswa"
// @Param        body  body      model.PatchStudentRequest  true  "Field yang ingin diubah"
// @Success      200   {object}  model.SuccessResponse{data=model.Student}
// @Failure      400   {object}  model.ErrorResponse  "Tidak ada field yang diubah"
// @Failure      403   {object}  model.ErrorResponse
// @Failure      404   {object}  model.ErrorResponse
// @Failure      422   {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /students/{id} [patch]
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

// Delete godoc
//
// @Summary      Menghapus mahasiswa
// @Description  Butuh permission student:delete. Portofolio milik mahasiswa ikut terhapus.
// @Tags         students
// @Produce      json
// @Param        id   path  int  true  "ID mahasiswa"
// @Success      204  "Tidak ada isi"
// @Failure      400  {object}  model.ErrorResponse
// @Failure      403  {object}  model.ErrorResponse
// @Failure      404  {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /students/{id} [delete]
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
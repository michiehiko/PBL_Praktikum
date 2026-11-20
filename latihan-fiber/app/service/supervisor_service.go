package service

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

type SupervisorService struct {
	repo repository.SupervisorRepository
	tx   *repository.TxManager
}

func NewSupervisorService(repo repository.SupervisorRepository, tx *repository.TxManager) *SupervisorService {
	return &SupervisorService{repo: repo, tx: tx}
}

// POST /supervisors/:id/assign
// Assign godoc
//
// @Summary      Menetapkan dosen pembimbing
// @Description  Butuh permission supervision:manage. Baris mahasiswa dikunci lebih dulu agar dua permintaan serentak tidak melewati kuota.
// @Tags         supervisors
// @Accept       json
// @Produce      json
// @Param        id    path      int                              true  "ID dosen pembimbing"
// @Param        body  body      model.AssignSupervisorRequest     true  "Mahasiswa yang dibimbing"
// @Success      201   {object}  model.SuccessResponse
// @Failure      400   {object}  model.ErrorResponse
// @Failure      403   {object}  model.ErrorResponse
// @Failure      404   {object}  model.ErrorResponse
// @Failure      409   {object}  model.ErrorResponse  "SUPERVISOR_FULL atau ALREADY_ASSIGNED"
// @Failure      422   {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /supervisors/{id}/assign [post]
func (s *SupervisorService) Assign(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	supervisorID, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id dosen harus berupa angka positif")
	}

	var req model.AssignSupervisorRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.repo.WithTx(tx)

		// Kunci baris mahasiswa untuk cegah pendaftaran serentak
		if err := repo.LockStudent(ctx, req.StudentID); err != nil {
			return err
		}

		// Ambil kuota atomik
		if err := repo.IncrementSupervised(ctx, supervisorID); err != nil {
			return err
		}

		// Daftarkan
		return repo.AssignStudent(ctx, req.StudentID, supervisorID)
	})

	if err != nil {
		return translateSupervisorError(err)
	}
	return helper.Success(c, fiber.StatusCreated, "berhasil menetapkan dosen pembimbing", nil)
}

// POST /supervisions/switch
// Switch godoc
//
// @Summary      Memindahkan mahasiswa ke dosen pembimbing lain
// @Description  Kedua dosen dikunci berurutan agar bebas deadlock; kuota dosen lama berkurang dan dosen baru bertambah dalam satu transaksi.
// @Tags         supervisions
// @Accept       json
// @Produce      json
// @Param        body  body      model.SwitchSupervisorRequest  true  "Mahasiswa dan dosen tujuan"
// @Success      200   {object}  model.SuccessResponse
// @Failure      400   {object}  model.ErrorResponse
// @Failure      403   {object}  model.ErrorResponse
// @Failure      404   {object}  model.ErrorResponse
// @Failure      409   {object}  model.ErrorResponse  "SAME_SUPERVISOR, SUPERVISOR_FULL, atau NOT_ASSIGNED"
// @Failure      422   {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /supervisions/switch [post]
func (s *SupervisorService) Switch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.SwitchSupervisorRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.repo.WithTx(tx)

		// Kunci mahasiswa
		if err := repo.LockStudent(ctx, req.StudentID); err != nil {
			return err
		}

		// Cek dosen pembimbing saat ini
		currentSupervisorID, err := repo.GetSupervisorByStudent(ctx, req.StudentID)
		if err != nil {
			return err
		}

		if currentSupervisorID == req.NewSupervisorID {
			return errors.New("dosen baru tidak boleh sama dengan dosen lama")
		}

		// Kunci kedua dosen secara berurutan agar BEBAS DEADLOCK[cite: 31]
		if err := repo.LockSupervisors(ctx, currentSupervisorID, req.NewSupervisorID); err != nil {
			return err
		}

		// Kurangi dosen lama, tambah dosen baru (atomik)
		if err := repo.DecrementSupervised(ctx, currentSupervisorID); err != nil {
			return err
		}
		if err := repo.IncrementSupervised(ctx, req.NewSupervisorID); err != nil {
			return err
		}

		// Update data
		return repo.SwitchStudent(ctx, req.StudentID, req.NewSupervisorID)
	})

	if err != nil {
		if err.Error() == "dosen baru tidak boleh sama dengan dosen lama" {
			return helper.NewError(fiber.StatusConflict, "SAME_SUPERVISOR", err.Error())
		}
		return translateSupervisorError(err)
	}
	return helper.Success(c, fiber.StatusOK, "berhasil pindah pembimbing", nil)
}

// PATCH /supervisors/:id
// UpdateQuota godoc
//
// @Summary      Mengubah kuota dosen pembimbing
// @Description  Optimistic locking: kirim version yang terakhir dilihat. Kuota tidak boleh di bawah jumlah bimbingan saat ini.
// @Tags         supervisors
// @Accept       json
// @Produce      json
// @Param        id    path      int                                      true  "ID dosen pembimbing"
// @Param        body  body      model.UpdateSupervisorQuotaRequest       true  "Kuota baru dan version"
// @Success      200   {object}  model.SuccessResponse{data=model.Supervisor}
// @Failure      400   {object}  model.ErrorResponse
// @Failure      403   {object}  model.ErrorResponse
// @Failure      404   {object}  model.ErrorResponse
// @Failure      409   {object}  model.ErrorResponse  "VERSION_CONFLICT atau QUOTA_BELOW_ENROLLED"
// @Failure      422   {object}  model.ErrorResponse
// @Security     BearerAuth
// @Router       /supervisors/{id} [patch]
func (s *SupervisorService) UpdateQuota(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}

	var req model.UpdateSupervisorQuotaRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	// Optimistic locking (satu query, tidak butuh transaksi penuh)
	updated, err := s.repo.UpdateQuota(ctx, id, req.Quota, req.Version)
	if err != nil {
		return translateSupervisorError(err)
	}
	return helper.Success(c, fiber.StatusOK, "kuota berhasil diubah", updated)
}

func translateSupervisorError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("data tidak ditemukan")
	case errors.Is(err, repository.ErrSupervisorFull):
		return helper.NewError(fiber.StatusConflict, "SUPERVISOR_FULL", "kuota pembimbing penuh")
	case errors.Is(err, repository.ErrAlreadyAssigned):
		return helper.NewError(fiber.StatusConflict, "ALREADY_ASSIGNED", "mahasiswa ini sudah memiliki pembimbing")
	case errors.Is(err, repository.ErrNotAssigned):
		return helper.NewError(fiber.StatusConflict, "NOT_ASSIGNED", "mahasiswa ini belum memiliki pembimbing")
	case errors.Is(err, repository.ErrVersionConflict):
		return helper.NewError(fiber.StatusConflict, "VERSION_CONFLICT", "data sudah diubah pihak lain, muat ulang lalu coba lagi")
	case errors.Is(err, repository.ErrQuotaBelowFilled):
		return helper.NewError(fiber.StatusConflict, "QUOTA_BELOW_ENROLLED", "kuota tidak boleh lebih kecil dari beban bimbingan saat ini")
	default:
		return helper.Internal(err)
	}
}
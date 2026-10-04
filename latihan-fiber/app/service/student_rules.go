package service

import (
	"strings"
	"latihan-fiber/app/model" 
)
// memeriksa isi permintaan POST (pembuatan student).
func ValidateCreate(req model.CreateStudentRequest) map[string]string {
	errs := map[string]string{}
	if strings.TrimSpace(req.NIM) == "" {
		errs["nim"] = "wajib diisi"
	}
	if strings.TrimSpace(req.Name) == "" {
		errs["name"] = "wajib diisi"
	}
	if strings.TrimSpace(req.Grade) == "" {
		errs["grade"] = "wajib diisi"
	}
	return errs
}

// memeriksa isi permintaan PUT.
func ValidateReplace(req model.ReplaceStudentRequest) map[string]string {
	errs := map[string]string{}
	if strings.TrimSpace(req.NIM) == "" {
		errs["nim"] = "wajib diisi pada PUT"
	}
	if strings.TrimSpace(req.Name) == "" {
		errs["name"] = "wajib diisi pada PUT"
	}
	if strings.TrimSpace(req.Grade) == "" {
		errs["grade"] = "wajib diisi pada PUT"
	}
	return errs
}

// menyalin field yang dikirim ke data yang sudah ada (PATCH). membiarkan field nil apa adanya
func ApplyPatch(current model.Student, req model.PatchStudentRequest) (model.Student, map[string]string) {
	errs := map[string]string{}

	if req.NIM != nil {
		if strings.TrimSpace(*req.NIM) == "" {
			errs["nim"] = "tidak boleh kosong"
		} else {
			current.NIM = *req.NIM
		}
	}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			errs["name"] = "tidak boleh kosong"
		} else {
			current.Name = *req.Name
		}
	}
	if req.Grade != nil {
		if strings.TrimSpace(*req.Grade) == "" {
			errs["grade"] = "tidak boleh kosong"
		} else {
			current.Grade = *req.Grade
		}
	}
	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}

	return current, errs
}

// menandai permintaan PATCH yang tidak mengubah apa pun.
func IsEmptyPatch(req model.PatchStudentRequest) bool {
	return req.NIM == nil && req.Name == nil && req.Grade == nil && req.IsActive == nil
}
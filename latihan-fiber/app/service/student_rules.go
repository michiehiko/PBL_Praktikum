package service

import (
	"strings"
	"latihan-fiber/app/model" 
)

// menyalin field yang dikirim ke data yang sudah ada (PATCH). membiarkan field nil apa adanya
func ApplyPatch(current model.Student, req model.PatchStudentRequest) model.Student {
	if req.NIM != nil {
		current.NIM = strings.TrimSpace(*req.NIM)
	}
	if req.Name != nil {
		current.Name = strings.TrimSpace(*req.Name)
	}
	if req.Grade != nil {
		current.Grade = strings.TrimSpace(*req.Grade)
	}
	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}
	return current
}

// menandai permintaan PATCH yang tidak mengubah apa pun.
func IsEmptyPatch(req model.PatchStudentRequest) bool {
	return req.NIM == nil && req.Name == nil && req.Grade == nil && req.IsActive == nil
}
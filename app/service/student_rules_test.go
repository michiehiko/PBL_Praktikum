package service

import (
	"testing"
	"latihan-fiber/app/model" 
)

func TestValidateCreate(t *testing.T) {
	// isi nim dan grade, tapi name kosong
	req := model.CreateStudentRequest{NIM: "152019", Name: "", Grade: "A"}
	errs := ValidateCreate(req)
	
	if len(errs) == 0 || errs["name"] == "" {
		t.Error("Seharusnya ada error validasi karena Name kosong")
	}
}

func TestApplyPatch(t *testing.T) {
	// data saat ini ada di DB
	initial := model.Student{ID: 1, NIM: "152019", Name: "Andi", Grade: "B", IsActive: true}
	
	// user hanya mengirim perubahan untuk Grade
	newGrade := "A"
	req := model.PatchStudentRequest{Grade: &newGrade} 
	
	result, errs := ApplyPatch(initial, req)
	
	if len(errs) != 0 {
		t.Fatalf("Tidak seharusnya ada error: %v", errs)
	}
	if result.Grade != "A" {
		t.Errorf("Grade seharusnya berubah menjadi A, dapat %s", result.Grade)
	}
	if result.Name != "Andi" {
		t.Error("Field Name yang tidak dikirim seharusnya tidak berubah")
	}
}

func TestValidateReplace(t *testing.T) {
	// Skenario: Mengirim Name dan Grade, tetapi NIM kosong pada PUT
	req := model.ReplaceStudentRequest{NIM: "", Name: "Budi", Grade: "B"}
	errs := ValidateReplace(req)
	
	if len(errs) == 0 || errs["nim"] == "" {
		t.Error("Seharusnya ada error validasi karena NIM kosong pada proses PUT")
	}
}
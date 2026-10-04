package main

type Student struct {
	ID       int    `json:"id"`
    NIM      string `json:"nim"` 
    Nama     string `json:"nama"`
    IPK    string 	`json:"ipk"`
    IsActive bool   `json:"is_active"`
}

//id diatur server
type CreateStudentRequest struct {
    NIM      string `json:"nim"`
    Nama     string `json:"nama"`
    IPK    string 	`json:"ipk"`
    IsActive bool   `json:"is_active"`
}

//mengganti isi
type ReplaceStudentRequest struct {
    NIM      string `json:"nim"`
    Nama     string `json:"nama"`
    IPK    string	 `json:"ipk"`
    IsActive bool   `json:"is_active"`
}

//mengubah sebagian
type PatchStudentRequest struct {
    NIM      *string `json:"nim,omitempty"`
    Nama     *string `json:"nama,omitempty"`
    IPK    *string	 `json:"ipk,omitempty"`
    IsActive *bool   `json:"is_active,omitempty"`
}

//menerima semua respons (berhasil maupun gagal)
type WebResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message"`
    Data    any    `json:"data,omitempty"`
    Meta    *Meta  `json:"meta,omitempty"`
    Errors  any    `json:"errors,omitempty"`
}

type Meta struct {
	Page       int `json:"page"`
    Limit      int `json:"limit"`
    Total      int `json:"total"`
    TotalPages int `json:"total_pages"`
}

type ListQuery struct {
    Page     int
    Limit    int
    Search   string
    Sort     string
    Order    string
    IsActive *bool
}
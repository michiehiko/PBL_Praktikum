package model

import "time"

type Student struct {
    ID        int       `json:"id"`
    NIM       string    `json:"nim"`
    Name      string    `json:"name"`
    Grade     string    `json:"grade"`
    IsActive  bool      `json:"is_active"`
	OwnerID   int       `json:"owner_id"`
    CreatedAt time.Time `json:"created_at"`
}

// paginasi, search, filter, dan sorting
type ListQuery struct {
    Page     int
    Limit    int
    Search   string
    Sort     string
    Order    string
    IsActive *bool
}

// menghitung offset untuk query
func (q ListQuery) Offset() int {
    return (q.Page - 1) * q.Limit
}

type CreateStudentRequest struct {
	NIM      string `json:"nim" validate:"required,nim,min=5,max=15"`
	Name     string `json:"name" validate:"required,min=3,max=50"`
	Grade    string `json:"grade" validate:"required,oneof=A B C D E"`
	IsActive bool   `json:"is_active"` // Boolean tidak usah pakai required (bisa bentrok dengan nilai false)
}

type ReplaceStudentRequest struct {
	NIM      string `json:"nim" validate:"required,nim,min=5,max=15"`
	Name     string `json:"name" validate:"required,min=3,max=50"`
	Grade    string `json:"grade" validate:"required,oneof=A B C D E"`
	IsActive bool   `json:"is_active"`
}

type PatchStudentRequest struct {
	NIM      *string `json:"nim,omitempty" validate:"omitnil,nim,min=5,max=15"`
	Name     *string `json:"name,omitempty" validate:"omitnil,min=3,max=50"`
	Grade    *string `json:"grade,omitempty" validate:"omitnil,oneof=A B C D E"`
	IsActive *bool   `json:"is_active,omitempty"`
}

type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type WebResponse struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Data    any               `json:"data,omitempty"`
	Meta    *Meta             `json:"meta,omitempty"`
	Errors  map[string]string `json:"errors,omitempty"`
}

type ErrorResponse struct {
	Success   bool              `json:"success"`
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`     // bug 2 dah dibenerin
	RequestID string            `json:"request_id,omitempty"` // bug 2 dah dibenerin
}

type Cursor struct {
	CreatedAt time.Time
	ID        int
}

type CursorQuery struct {
	Limit    int
	Search   string
	IsActive *bool
	After    *Cursor
}

type CursorMeta struct {
	Limit      int    `json:"limit"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}
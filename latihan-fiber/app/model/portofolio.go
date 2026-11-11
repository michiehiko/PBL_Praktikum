package model

type Portfolio struct {
	ID           string        `json:"id"`
	StudentID    int           `json:"student_id"`
	Headline     string        `json:"headline"`
	Bio          string        `json:"bio"`
	Projects     []Project     `json:"projects"`
	Certificates []Certificate `json:"certificates"`
}

type Project struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Technologies []string `json:"technologies"`
	Link         string   `json:"link"`
	Year         int      `json:"year"`
}

type Certificate struct {
	Name   string `json:"name"`
	Issuer string `json:"issuer"`
	Date   string `json:"date"`
}

type PutPortfolioRequest struct {
	Headline     string        `json:"headline" validate:"required,max=100"`
	Bio          string        `json:"bio" validate:"required,max=500"`
	Certificates []Certificate `json:"certificates" validate:"dive"`
}

type AddProjectRequest struct {
	Title        string   `json:"title" validate:"required,max=100"`
	Technologies []string `json:"technologies" validate:"required,min=1,max=10,dive,max=30"`
	Link         string   `json:"link" validate:"omitempty,url"`
	Year         int      `json:"year" validate:"required,min=2000,max=2100"`
}

type TechStat struct {
	Technology string `json:"technology"`
	Count      int    `json:"count"`
}
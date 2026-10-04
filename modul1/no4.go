package main

import "fmt"

// membuat struct
type Student struct {
	ID       string
	Name     string
	Grade    float64
	IsActive bool
}

//method value receiver
func (s Student) GetInfo() string {
	return fmt.Sprintf("ID: %s, Name: %s, Grade: %.2f, Active: %t", s.ID, s.Name, s.Grade, s.IsActive)
}

//method pointer receiver
func (s *Student) UpdateGrade(newGrade float64) {
	s.Grade = newGrade
}

func (s *Student) Activate() {
	s.IsActive = true
}

func (s *Student) Deactivate() {
	s.IsActive = false
}

func main() {
	//membuat data baru dari struct
	mhs := Student{
		ID:       "123123",
		Name:     "Kamonohashi Ron",
		Grade:    75.5,
		IsActive: false,
	}

	fmt.Println("Informasi awal")
	fmt.Println(mhs.GetInfo())

	// mengubah data (update nilai dan aktifkan status)
	fmt.Println("\n Setelah Update Nilai & Status")
	mhs.UpdateGrade(92.0) 
	mhs.Activate()       
	fmt.Println(mhs.GetInfo())

	// Menonaktifkan status
	fmt.Println("\n Setelah Status Dinonaktifkan")
	mhs.Deactivate() 
	fmt.Println(mhs.GetInfo())
}
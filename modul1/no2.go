package main

import "fmt"

func main() {
	// 5 variabel
	fmt.Println("1. variable n slice")

	var nama string = "Saskeh"
	var umur int = 20
	var ipk float64 = 3.5
	var isStudent bool = true
	var mataKuliah = []string{"Basis Data", "Algoritma", "Pemrograman"}

	fmt.Println("Nama:", nama, "Umur:", umur, "IPK:", ipk, "Is Student:", isStudent, "Mata Kuliah:", mataKuliah)

	// operasi & map
	fmt.Println("2. operasi map")

	nilai := make(map[string]int)

	fmt.Println("- Menambahkan data ke map")
	nilai["Saskeh"] = 90
	nilai["Naruto"] = 80
	nilai["Gaara"] = 70

	fmt.Println("cek nilai saskeh dan sakura")

	if nilaiSaskeh, ok := nilai["Saskeh"]; ok {
		fmt.Println("Nilai Saskeh:", nilaiSaskeh)
	} else {
		fmt.Println("Nilai Saskeh tidak ditemukan")
	}

	if nilaiSakura, ok := nilai["Sakura"]; ok {
		fmt.Println("Nilai Sakura:", nilaiSakura)
	} else {
		fmt.Println("Nilai Sakura tidak ditemukan")
	}

	fmt.Println("Menampilkan semua data yang ada di map")
	for key, value := range nilai {
		fmt.Println("Nama:", key, "Nilai:", value)
	}

	fmt.Println("hapus data (Naruto) dari map")
	delete(nilai, "Naruto")

	fmt.Println("looping seluruh isi map")
	for key, value := range nilai {
		fmt.Println("Nama:", key, "Nilai:", value)
	}
}

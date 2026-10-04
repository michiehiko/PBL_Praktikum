package main

import "fmt"

//pass by value
func swapValue(a, b int) {
	temp := a
	a = b
	b = temp
}

//pass by pointer
func swap(a, b *int) {
	temp := *a
	*a = *b
	*b = temp
}

//update slice dengan pointer
func updateSlice(s *[]string, newItem string) {
	*s = append(*s, newItem)
}

func main() {
	fmt.Println("1. Perbandingan pass by value vs pass by pointer")
	x := 10
	y := 20

	fmt.Printf("Nilai awal : x = %d, y = %d\n", x, y)

	// menukar nilai dengan Pass by Value
	swapValue(x, y)
	fmt.Printf("Setelah swapValue (Value) : x = %d, y = %d (TIDAK BERUBAH)\n", x, y)

	// menukar nilai dengan Pass by Pointer
	swap(&x, &y)
	fmt.Printf("Setelah swap (Pointer) : x = %d, y = %d (BERHASIL DITUKAR)\n", x, y)

	fmt.Println("\n 2. Update slice dengan pointer")
	daftarDonghua := []string{"LOTM,", "Link Click,"}
	fmt.Printf("Slice awal	: %v\n", daftarDonghua)

	updateSlice(&daftarDonghua, "The King's Avatar")
	fmt.Printf("Slice setelah di-update : %v\n", daftarDonghua)
}

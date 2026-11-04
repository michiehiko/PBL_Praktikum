package service

// batas SKS yang boleh diambil dalam satu KRS.
const MaxCreditsPerSemester = 24

// aturan tidak menyentuh database dan tidak menyentuh Fiber, sehingga dapat diuji hanya dengan memanggilnya.
func WithinCreditLimit(taken, adding int) bool {
	return taken+adding <= MaxCreditsPerSemester
}

// menghitung SKS setelah pindah kelas.
func CreditsAfterSwap(taken, leaving, joining int) int {
	return taken - leaving + joining
}
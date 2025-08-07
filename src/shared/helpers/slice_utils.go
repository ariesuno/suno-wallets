package helpers

// Contains verifica se um slice contém um valor específico
func Contains[T comparable](slice []T, item T) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ContainsRune verifica se um slice de runes contém um rune específico
func ContainsRune(slice []rune, item rune) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

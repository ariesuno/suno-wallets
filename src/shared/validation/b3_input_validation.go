package validation

import (
	"errors"
	"regexp"
	"time"
)

// Comentários em pt-BR: validações de entrada para endpoints B3

var (
	cpfRe  = regexp.MustCompile(`^\d{11}$`)
	dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// ValidateCPF valida CPF com 11 dígitos (apenas números)
func ValidateCPF(cpf string) error {
	if !cpfRe.MatchString(cpf) {
		return errors.New("cpf must be 11 digits")
	}
	return nil
}

// ValidateDateYMD valida datas no formato YYYY-MM-DD
func ValidateDateYMD(s string) error {
	if !dateRe.MatchString(s) {
		return errors.New("date must be in YYYY-MM-DD format")
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return errors.New("invalid date")
	}
	return nil
}

// ValidatePage valida página >= 1
func ValidatePage(page int) error {
	if page < 1 {
		return errors.New("page must be >= 1")
	}
	return nil
}

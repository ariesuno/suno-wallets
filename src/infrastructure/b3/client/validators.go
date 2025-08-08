package client

import (
	"errors"
	"regexp"
	"time"
)

var (
	reDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// Comentários em pt-BR: validações de datas e paginação
func validateDate(s string) error {
	if !reDate.MatchString(s) {
		return errors.New("date must be in YYYY-MM-DD format")
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return errors.New("invalid date")
	}
	return nil
}

func validatePage(page int) error {
	if page < 1 {
		return errors.New("page must be >= 1")
	}
	return nil
}

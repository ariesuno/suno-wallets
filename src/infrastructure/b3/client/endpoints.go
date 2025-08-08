package client

import (
	"context"
	"net/http"
)

// Comentários em pt-BR: métodos typed para contratos v3/v2
func (c *B3OfficialClient) GetPositionsV3(ctx context.Context, cpf string, startDate, endDate string, page int) (*http.Response, error) {
	if err := validateDate(startDate); err != nil {
		return nil, err
	}
	if err := validateDate(endDate); err != nil {
		return nil, err
	}
	if err := validatePage(page); err != nil {
		return nil, err
	}
	q := map[string]string{
		"referenceStartDate": startDate,
		"referenceEndDate":   endDate,
		"page":               string(rune(page)),
	}
	return c.MakeRequest(ctx, http.MethodGet, "/position/v3/equities/investors/"+cpf, q, cpf, true)
}

func (c *B3OfficialClient) GetTransactionsV2(ctx context.Context, cpf string, startDate, endDate string, page int) (*http.Response, error) {
	if err := validateDate(startDate); err != nil {
		return nil, err
	}
	if err := validateDate(endDate); err != nil {
		return nil, err
	}
	if err := validatePage(page); err != nil {
		return nil, err
	}
	q := map[string]string{
		"referenceStartDate": startDate,
		"referenceEndDate":   endDate,
		"page":               string(rune(page)),
	}
	return c.MakeRequest(ctx, http.MethodGet, "/assets-trading/v2/equity/"+cpf, q, cpf, true)
}

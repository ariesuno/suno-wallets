package client

import (
	"context"
	"fmt"
	"net/http"

	"suno-wallets/src/domain/enums"
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
		"page":               fmt.Sprintf("%d", page),
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
		"page":               fmt.Sprintf("%d", page),
	}
	return c.MakeRequest(ctx, http.MethodGet, "/assets-trading/v2/investors/"+cpf, q, cpf, true)
}

// Comentários em pt-BR: novos métodos para endpoints específicos por tipo de ativo B3

// GetTransactionsByAssetType busca transações por tipo específico de ativo B3
func (c *B3OfficialClient) GetTransactionsByAssetType(ctx context.Context, cpf string, assetType enums.B3AssetType, startDate, endDate string, page int) (*http.Response, error) {
	if err := validateDate(startDate); err != nil {
		return nil, err
	}
	if err := validateDate(endDate); err != nil {
		return nil, err
	}
	if err := validatePage(page); err != nil {
		return nil, err
	}
	if !assetType.IsValid() {
		return nil, fmt.Errorf("invalid asset type: %s", assetType)
	}

	q := map[string]string{
		"referenceStartDate": startDate,
		"referenceEndDate":   endDate,
		"page":               fmt.Sprintf("%d", page),
	}

	endpoint := assetType.GetAPIEndpoint() + "/" + cpf
	return c.MakeRequest(ctx, http.MethodGet, endpoint, q, cpf, true)
}

// GetPositionsByAssetType busca posições por tipo específico de ativo B3
func (c *B3OfficialClient) GetPositionsByAssetType(ctx context.Context, cpf string, assetType enums.B3AssetType, startDate, endDate string, page int) (*http.Response, error) {
	if err := validateDate(startDate); err != nil {
		return nil, err
	}
	if err := validateDate(endDate); err != nil {
		return nil, err
	}
	if err := validatePage(page); err != nil {
		return nil, err
	}
	if !assetType.IsValid() {
		return nil, fmt.Errorf("invalid asset type: %s", assetType)
	}
	if !assetType.IsPositionSupported() {
		return nil, fmt.Errorf("positions not supported for asset type: %s", assetType)
	}

	q := map[string]string{
		"referenceStartDate": startDate,
		"referenceEndDate":   endDate,
		"page":               fmt.Sprintf("%d", page),
	}

	endpoint := assetType.GetPositionsEndpoint() + "/" + cpf
	return c.MakeRequest(ctx, http.MethodGet, endpoint, q, cpf, true)
}

// GetEquitiesTransactions busca transações de ações (compatibilidade)
func (c *B3OfficialClient) GetEquitiesTransactions(ctx context.Context, cpf string, startDate, endDate string, page int) (*http.Response, error) {
	return c.GetTransactionsByAssetType(ctx, cpf, enums.B3AssetTypeEquities, startDate, endDate, page)
}

// GetFixedIncomeTransactions busca transações de renda fixa
func (c *B3OfficialClient) GetFixedIncomeTransactions(ctx context.Context, cpf string, startDate, endDate string, page int) (*http.Response, error) {
	return c.GetTransactionsByAssetType(ctx, cpf, enums.B3AssetTypeFixedIncome, startDate, endDate, page)
}

// GetTreasuryBondsTransactions busca transações de títulos do tesouro
func (c *B3OfficialClient) GetTreasuryBondsTransactions(ctx context.Context, cpf string, startDate, endDate string, page int) (*http.Response, error) {
	return c.GetTransactionsByAssetType(ctx, cpf, enums.B3AssetTypeTreasuryBonds, startDate, endDate, page)
}

// GetDerivativesTransactions busca transações de derivativos
func (c *B3OfficialClient) GetDerivativesTransactions(ctx context.Context, cpf string, startDate, endDate string, page int) (*http.Response, error) {
	return c.GetTransactionsByAssetType(ctx, cpf, enums.B3AssetTypeDerivatives, startDate, endDate, page)
}

// GetSecuritiesLendingTransactions busca transações de empréstimos de valores mobiliários
func (c *B3OfficialClient) GetSecuritiesLendingTransactions(ctx context.Context, cpf string, startDate, endDate string, page int) (*http.Response, error) {
	return c.GetTransactionsByAssetType(ctx, cpf, enums.B3AssetTypeSecuritiesLending, startDate, endDate, page)
}

// GetEquitiesPositions busca posições de ações (mantém compatibilidade com v3)
func (c *B3OfficialClient) GetEquitiesPositions(ctx context.Context, cpf string, startDate, endDate string, page int) (*http.Response, error) {
	return c.GetPositionsByAssetType(ctx, cpf, enums.B3AssetTypeEquities, startDate, endDate, page)
}

// GetFixedIncomePositions busca posições de renda fixa
func (c *B3OfficialClient) GetFixedIncomePositions(ctx context.Context, cpf string, startDate, endDate string, page int) (*http.Response, error) {
	return c.GetPositionsByAssetType(ctx, cpf, enums.B3AssetTypeFixedIncome, startDate, endDate, page)
}

// GetTreasuryBondsPositions busca posições de títulos do tesouro
func (c *B3OfficialClient) GetTreasuryBondsPositions(ctx context.Context, cpf string, startDate, endDate string, page int) (*http.Response, error) {
	return c.GetPositionsByAssetType(ctx, cpf, enums.B3AssetTypeTreasuryBonds, startDate, endDate, page)
}

// GetDerivativesPositions busca posições de derivativos
func (c *B3OfficialClient) GetDerivativesPositions(ctx context.Context, cpf string, startDate, endDate string, page int) (*http.Response, error) {
	return c.GetPositionsByAssetType(ctx, cpf, enums.B3AssetTypeDerivatives, startDate, endDate, page)
}

// GetSecuritiesLendingPositions busca posições de empréstimos de valores mobiliários
func (c *B3OfficialClient) GetSecuritiesLendingPositions(ctx context.Context, cpf string, startDate, endDate string, page int) (*http.Response, error) {
	return c.GetPositionsByAssetType(ctx, cpf, enums.B3AssetTypeSecuritiesLending, startDate, endDate, page)
}

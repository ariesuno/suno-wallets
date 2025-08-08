package dto

// Comentários em pt-BR: DTOs para preview de transações v2

type TransactionsPreviewRequest struct {
	CPF           string `form:"cpf" json:"cpf"`
	Start         string `form:"start" json:"start"`
	End           string `form:"end" json:"end"`
	AssetType     string `form:"assetType" json:"assetType"`
	Page          int    `form:"page" json:"page"`
	FetchAllPages bool   `form:"fetchAllPages" json:"fetchAllPages"`
}

type TransactionsPreviewResponse struct {
	AssetType string            `json:"asset_type"`
	Start     string            `json:"start"`
	End       string            `json:"end"`
    Pages     int               `json:"pages"`
    Payloads  []interface{}     `json:"payloads"`
}

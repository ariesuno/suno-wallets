package dtos

import (
    "encoding/json"
    "time"

    "suno-wallets/src/domain/entities"

    "github.com/google/uuid"
)

// CreateWalletRequest DTO para criação de carteira
type CreateWalletRequest struct {
	TenantID     uuid.UUID           `json:"tenant_id" validate:"required"`
	Name         string              `json:"name" validate:"required,min=1,max=255"`
	Description  string              `json:"description,omitempty"`
	Type         entities.WalletType `json:"type" validate:"required,oneof=personal business corporate"`
	Currency     string              `json:"currency" validate:"required,len=3"`
	IsDefault    bool                `json:"is_default"`
	OwnerID      uuid.UUID           `json:"owner_id" validate:"required"`
	OwnerType    string              `json:"owner_type" validate:"required"`
	DailyLimit   *int64              `json:"daily_limit,omitempty"`
	MonthlyLimit *int64              `json:"monthly_limit,omitempty"`
	AllowDebit   bool                `json:"allow_debit"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedBy    uuid.UUID           `json:"created_by" validate:"required"`
}

// Validate valida os dados da requisição
func (r *CreateWalletRequest) Validate() error {
	if r.TenantID == uuid.Nil {
		return entities.NewValidationError("tenant_id é obrigatório")
	}

	if r.Name == "" {
		return entities.NewValidationError("nome é obrigatório")
	}

	if len(r.Name) > 255 {
		return entities.NewValidationError("nome não pode ter mais de 255 caracteres")
	}

	if r.OwnerID == uuid.Nil {
		return entities.NewValidationError("owner_id é obrigatório")
	}

	if r.CreatedBy == uuid.Nil {
		return entities.NewValidationError("created_by é obrigatório")
	}

	if r.Currency == "" {
		r.Currency = "BRL"
	}

	if len(r.Currency) != 3 {
		return entities.NewValidationError("moeda deve ter exatamente 3 caracteres")
	}

	// Validar limites
	if r.DailyLimit != nil && *r.DailyLimit < 0 {
		return entities.NewValidationError("limite diário deve ser maior ou igual a zero")
	}

	if r.MonthlyLimit != nil && *r.MonthlyLimit < 0 {
		return entities.NewValidationError("limite mensal deve ser maior ou igual a zero")
	}

	return nil
}

// UpdateWalletRequest DTO para atualização de carteira
type UpdateWalletRequest struct {
	ID           uuid.UUID              `json:"id" validate:"required"`
	TenantID     uuid.UUID              `json:"tenant_id" validate:"required"`
	Name         *string                `json:"name,omitempty"`
	Description  *string                `json:"description,omitempty"`
	Status       *entities.WalletStatus `json:"status,omitempty"`
	IsDefault    *bool                  `json:"is_default,omitempty"`
	DailyLimit   *int64                 `json:"daily_limit,omitempty"`
	MonthlyLimit *int64                 `json:"monthly_limit,omitempty"`
	AllowDebit   *bool                  `json:"allow_debit,omitempty"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
	UpdatedBy    *uuid.UUID             `json:"updated_by,omitempty"`
}

// Validate valida os dados da requisição
func (r *UpdateWalletRequest) Validate() error {
	if r.ID == uuid.Nil {
		return entities.NewValidationError("id é obrigatório")
	}

	if r.TenantID == uuid.Nil {
		return entities.NewValidationError("tenant_id é obrigatório")
	}

	if r.Name != nil && *r.Name == "" {
		return entities.NewValidationError("nome não pode ser vazio")
	}

	if r.Name != nil && len(*r.Name) > 255 {
		return entities.NewValidationError("nome não pode ter mais de 255 caracteres")
	}

	// Validar limites
	if r.DailyLimit != nil && *r.DailyLimit < 0 {
		return entities.NewValidationError("limite diário deve ser maior ou igual a zero")
	}

	if r.MonthlyLimit != nil && *r.MonthlyLimit < 0 {
		return entities.NewValidationError("limite mensal deve ser maior ou igual a zero")
	}

	return nil
}

// ListWalletsRequest DTO para listagem de carteiras
type ListWalletsRequest struct {
	TenantID uuid.UUID              `json:"tenant_id" validate:"required"`
	OwnerID  *uuid.UUID             `json:"owner_id,omitempty"`
	Status   *entities.WalletStatus `json:"status,omitempty"`
	Type     *entities.WalletType   `json:"type,omitempty"`
	Page     int                    `json:"page" validate:"min=1"`
	Limit    int                    `json:"limit" validate:"min=1,max=100"`
	Search   string                 `json:"search,omitempty"`
	SortBy   string                 `json:"sort_by,omitempty"`
	SortDesc bool                   `json:"sort_desc"`
}

// Validate valida os dados da requisição
func (r *ListWalletsRequest) Validate() error {
	if r.TenantID == uuid.Nil {
		return entities.NewValidationError("tenant_id é obrigatório")
	}

	if r.Page <= 0 {
		r.Page = 1
	}

	if r.Limit <= 0 {
		r.Limit = 10
	}

	if r.Limit > 100 {
		r.Limit = 100
	}

	return nil
}

// WalletResponse DTO para resposta de carteira
type WalletResponse struct {
	ID                uuid.UUID             `json:"id"`
	TenantID          uuid.UUID             `json:"tenant_id"`
	Name              string                `json:"name"`
	Description       string                `json:"description"`
	Type              entities.WalletType   `json:"type"`
	Status            entities.WalletStatus `json:"status"`
	Balance           int64                 `json:"balance"`
	BalanceInReais    float64               `json:"balance_in_reais"`
	Currency          string                `json:"currency"`
	IsDefault         bool                  `json:"is_default"`
	OwnerID           uuid.UUID             `json:"owner_id"`
	OwnerType         string                `json:"owner_type"`
	DailyLimit        *int64                `json:"daily_limit,omitempty"`
	DailyLimitReais   *float64              `json:"daily_limit_reais,omitempty"`
	MonthlyLimit      *int64                `json:"monthly_limit,omitempty"`
	MonthlyLimitReais *float64              `json:"monthly_limit_reais,omitempty"`
	AllowDebit        bool                  `json:"allow_debit"`
    Metadata          map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt         time.Time             `json:"created_at"`
	UpdatedAt         time.Time             `json:"updated_at"`
	CreatedBy         uuid.UUID             `json:"created_by"`
	UpdatedBy         *uuid.UUID            `json:"updated_by,omitempty"`
}

// ListWalletsResponse DTO para resposta de listagem de carteiras
type ListWalletsResponse struct {
	Data []*WalletResponse `json:"data"`
	Meta PaginationMeta    `json:"meta"`
}

// PaginationMeta metadados de paginação
type PaginationMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// WalletToResponse converte entidade para DTO de resposta
func WalletToResponse(wallet *entities.Wallet) *WalletResponse {
	if wallet == nil {
		return nil
	}

    // Converter metadata JSON (datatypes.JSON) para map para o DTO
    var metadata map[string]interface{}
    if len(wallet.Metadata) > 0 {
        _ = json.Unmarshal(wallet.Metadata, &metadata)
    }

	response := &WalletResponse{
		ID:             wallet.ID,
		TenantID:       wallet.TenantID,
		Name:           wallet.Name,
		Description:    wallet.Description,
		Type:           wallet.Type,
		Status:         wallet.Status,
		Balance:        wallet.Balance,
		BalanceInReais: wallet.GetBalanceInReais(),
		Currency:       wallet.Currency,
		IsDefault:      wallet.IsDefault,
		OwnerID:        wallet.OwnerID,
		OwnerType:      wallet.OwnerType,
		AllowDebit:     wallet.AllowDebit,
        Metadata:       metadata,
		CreatedAt:      wallet.CreatedAt,
		UpdatedAt:      wallet.UpdatedAt,
		CreatedBy:      wallet.CreatedBy,
		UpdatedBy:      wallet.UpdatedBy,
	}

	// Converter limites para reais se existirem
	if wallet.DailyLimit != nil {
		response.DailyLimit = wallet.DailyLimit
		response.DailyLimitReais = wallet.GetDailyLimitInReais()
	}

	if wallet.MonthlyLimit != nil {
		response.MonthlyLimit = wallet.MonthlyLimit
		response.MonthlyLimitReais = wallet.GetMonthlyLimitInReais()
	}

	return response
}

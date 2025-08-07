package usecases

import (
	"context"

	"suno-wallets/src/application/dtos"
	"suno-wallets/src/domain/entities"
	"suno-wallets/src/domain/repositories"
	"suno-wallets/src/shared/helpers"

	"github.com/google/uuid"
)

// WalletUseCase interface para casos de uso de carteiras
type WalletUseCase interface {
	CreateWallet(ctx context.Context, req *dtos.CreateWalletRequest) (*dtos.WalletResponse, error)
	GetWalletByID(ctx context.Context, id, tenantID uuid.UUID) (*dtos.WalletResponse, error)
	GetWalletsByOwner(ctx context.Context, ownerID, tenantID uuid.UUID) ([]*dtos.WalletResponse, error)
	UpdateWallet(ctx context.Context, req *dtos.UpdateWalletRequest) (*dtos.WalletResponse, error)
	DeleteWallet(ctx context.Context, id, tenantID uuid.UUID) error
	ListWallets(ctx context.Context, req *dtos.ListWalletsRequest) (*dtos.ListWalletsResponse, error)
}

// walletUseCaseImpl implementação dos casos de uso de carteiras
type walletUseCaseImpl struct {
	walletRepo repositories.WalletRepository
}

// NewWalletUseCase cria uma nova instância do caso de uso de carteiras
func NewWalletUseCase(walletRepo repositories.WalletRepository) WalletUseCase {
	return &walletUseCaseImpl{
		walletRepo: walletRepo,
	}
}

// CreateWallet cria uma nova carteira
func (uc *walletUseCaseImpl) CreateWallet(ctx context.Context, req *dtos.CreateWalletRequest) (*dtos.WalletResponse, error) {
	// Validar entrada
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Converter DTO para entidade
	wallet := &entities.Wallet{
		TenantID:     req.TenantID,
		Name:         req.Name,
		Description:  req.Description,
		Type:         req.Type,
		Status:       entities.WalletStatusActive,
		Balance:      0, // Sempre inicia com saldo zero
		Currency:     req.Currency,
		IsDefault:    req.IsDefault,
		OwnerID:      req.OwnerID,
		OwnerType:    req.OwnerType,
		DailyLimit:   req.DailyLimit,
		MonthlyLimit: req.MonthlyLimit,
		AllowDebit:   req.AllowDebit,
		Metadata:     req.Metadata,
		CreatedBy:    req.CreatedBy,
	}

	// Se for a primeira carteira do usuário, definir como padrão
	if !req.IsDefault {
		existingWallets, err := uc.walletRepo.GetByOwner(ctx, req.OwnerID, req.TenantID)
		if err != nil {
			return nil, err
		}
		if len(existingWallets) == 0 {
			wallet.IsDefault = true
		}
	}

	// Criar carteira
	if err := uc.walletRepo.Create(ctx, wallet); err != nil {
		helpers.LogError("Falha ao criar carteira via use case", err, map[string]interface{}{
			"tenant_id": req.TenantID,
			"owner_id":  req.OwnerID,
		})
		return nil, err
	}

	helpers.LogInfo("Carteira criada com sucesso", map[string]interface{}{
		"wallet_id": wallet.ID,
		"tenant_id": wallet.TenantID,
		"owner_id":  wallet.OwnerID,
	})

	return dtos.WalletToResponse(wallet), nil
}

// GetWalletByID busca uma carteira por ID
func (uc *walletUseCaseImpl) GetWalletByID(ctx context.Context, id, tenantID uuid.UUID) (*dtos.WalletResponse, error) {
	wallet, err := uc.walletRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	return dtos.WalletToResponse(wallet), nil
}

// GetWalletsByOwner busca carteiras por proprietário
func (uc *walletUseCaseImpl) GetWalletsByOwner(ctx context.Context, ownerID, tenantID uuid.UUID) ([]*dtos.WalletResponse, error) {
	wallets, err := uc.walletRepo.GetByOwner(ctx, ownerID, tenantID)
	if err != nil {
		return nil, err
	}

	var responses []*dtos.WalletResponse
	for _, wallet := range wallets {
		responses = append(responses, dtos.WalletToResponse(wallet))
	}

	return responses, nil
}

// UpdateWallet atualiza uma carteira existente
func (uc *walletUseCaseImpl) UpdateWallet(ctx context.Context, req *dtos.UpdateWalletRequest) (*dtos.WalletResponse, error) {
	// Validar entrada
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Buscar carteira existente
	wallet, err := uc.walletRepo.GetByID(ctx, req.ID, req.TenantID)
	if err != nil {
		return nil, err
	}

	// Atualizar campos
	if req.Name != nil {
		wallet.Name = *req.Name
	}
	if req.Description != nil {
		wallet.Description = *req.Description
	}
	if req.Status != nil {
		wallet.Status = *req.Status
	}
	if req.IsDefault != nil {
		wallet.IsDefault = *req.IsDefault
	}
	if req.DailyLimit != nil {
		wallet.DailyLimit = req.DailyLimit
	}
	if req.MonthlyLimit != nil {
		wallet.MonthlyLimit = req.MonthlyLimit
	}
	if req.AllowDebit != nil {
		wallet.AllowDebit = *req.AllowDebit
	}
	if req.Metadata != nil {
		wallet.Metadata = req.Metadata
	}
	if req.UpdatedBy != nil {
		wallet.UpdatedBy = req.UpdatedBy
	}

	// Atualizar carteira
	if err := uc.walletRepo.Update(ctx, wallet); err != nil {
		helpers.LogError("Falha ao atualizar carteira via use case", err, map[string]interface{}{
			"wallet_id": req.ID,
			"tenant_id": req.TenantID,
		})
		return nil, err
	}

	helpers.LogInfo("Carteira atualizada com sucesso", map[string]interface{}{
		"wallet_id": wallet.ID,
		"tenant_id": wallet.TenantID,
	})

	return dtos.WalletToResponse(wallet), nil
}

// DeleteWallet remove uma carteira
func (uc *walletUseCaseImpl) DeleteWallet(ctx context.Context, id, tenantID uuid.UUID) error {
	// Verificar se a carteira existe
	wallet, err := uc.walletRepo.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}

	// Não permitir deletar carteira padrão se houver outras carteiras
	if wallet.IsDefault {
		wallets, err := uc.walletRepo.GetByOwner(ctx, wallet.OwnerID, tenantID)
		if err != nil {
			return err
		}
		if len(wallets) > 1 {
			return entities.NewBusinessError("CANNOT_DELETE_DEFAULT_WALLET", "Não é possível deletar a carteira padrão enquanto houver outras carteiras")
		}
	}

	// Deletar carteira
	if err := uc.walletRepo.Delete(ctx, id, tenantID); err != nil {
		helpers.LogError("Falha ao deletar carteira via use case", err, map[string]interface{}{
			"wallet_id": id,
			"tenant_id": tenantID,
		})
		return err
	}

	helpers.LogInfo("Carteira deletada com sucesso", map[string]interface{}{
		"wallet_id": id,
		"tenant_id": tenantID,
	})

	return nil
}

// ListWallets lista carteiras com filtros e paginação
func (uc *walletUseCaseImpl) ListWallets(ctx context.Context, req *dtos.ListWalletsRequest) (*dtos.ListWalletsResponse, error) {
	// Validar entrada
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Converter para parâmetros do repositório
	params := repositories.ListWalletParams{
		TenantID: req.TenantID,
		OwnerID:  req.OwnerID,
		Status:   req.Status,
		Type:     req.Type,
		Page:     req.Page,
		Limit:    req.Limit,
		Search:   req.Search,
		SortBy:   req.SortBy,
		SortDesc: req.SortDesc,
	}

	// Buscar carteiras
	wallets, total, err := uc.walletRepo.List(ctx, params)
	if err != nil {
		helpers.LogError("Falha ao listar carteiras via use case", err, map[string]interface{}{
			"tenant_id": req.TenantID,
		})
		return nil, err
	}

	// Converter para DTOs de resposta
	var responses []*dtos.WalletResponse
	for _, wallet := range wallets {
		responses = append(responses, dtos.WalletToResponse(wallet))
	}

	// Calcular metadados de paginação
	totalPages := int((total + int64(req.Limit) - 1) / int64(req.Limit))
	if req.Limit == 0 {
		totalPages = 1
	}

	return &dtos.ListWalletsResponse{
		Data: responses,
		Meta: dtos.PaginationMeta{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      int(total),
			TotalPages: totalPages,
		},
	}, nil
}

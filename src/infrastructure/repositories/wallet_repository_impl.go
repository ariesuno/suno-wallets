package repositories

import (
	"context"
	"fmt"

	"suno-wallets/src/domain/entities"
	domaininterfaces "suno-wallets/src/domain/interfaces"
	"suno-wallets/src/shared/helpers"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// walletRepositoryImpl implementação do repositório de carteiras
type walletRepositoryImpl struct {
	db *gorm.DB
}

// NewWalletRepository cria uma nova instância do repositório de carteiras
func NewWalletRepository(db *gorm.DB) domaininterfaces.WalletRepository {
	return &walletRepositoryImpl{
		db: db,
	}
}

// Create cria uma nova carteira
func (r *walletRepositoryImpl) Create(ctx context.Context, wallet *entities.Wallet) error {
	if err := wallet.Validate(); err != nil {
		return err
	}

	// Verificar se já existe carteira padrão para o proprietário
	if wallet.IsDefault {
		exists, err := r.ExistsDefaultWallet(ctx, wallet.OwnerID, wallet.TenantID, nil)
		if err != nil {
			return err
		}
		if exists {
			return entities.ErrDuplicateDefaultWallet
		}
	}

	if err := r.db.WithContext(ctx).Create(wallet).Error; err != nil {
		helpers.LogError("Falha ao criar carteira", err, map[string]interface{}{
			"wallet_id": wallet.ID,
			"tenant_id": wallet.TenantID,
			"owner_id":  wallet.OwnerID,
		})
		return err
	}

	helpers.LogAudit("wallet_created", wallet.TenantID.String(), wallet.CreatedBy.String(), map[string]interface{}{
		"wallet_id":   wallet.ID,
		"wallet_name": wallet.Name,
		"owner_id":    wallet.OwnerID,
	})

	return nil
}

// GetByID busca uma carteira por ID e tenant
func (r *walletRepositoryImpl) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*entities.Wallet, error) {
	var wallet entities.Wallet

	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		First(&wallet).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entities.NewNotFoundError("wallet", id.String())
		}
		helpers.LogError("Falha ao buscar carteira por ID", err, map[string]interface{}{
			"wallet_id": id,
			"tenant_id": tenantID,
		})
		return nil, err
	}

	return &wallet, nil
}

// GetByOwner busca carteiras por proprietário e tenant
func (r *walletRepositoryImpl) GetByOwner(ctx context.Context, ownerID, tenantID uuid.UUID) ([]*entities.Wallet, error) {
	var wallets []*entities.Wallet

	err := r.db.WithContext(ctx).
		Where("owner_id = ? AND tenant_id = ?", ownerID, tenantID).
		Order("is_default DESC, created_at DESC").
		Find(&wallets).Error

	if err != nil {
		helpers.LogError("Falha ao buscar carteiras por proprietário", err, map[string]interface{}{
			"owner_id":  ownerID,
			"tenant_id": tenantID,
		})
		return nil, err
	}

	return wallets, nil
}

// GetDefaultByOwner busca a carteira padrão do proprietário
func (r *walletRepositoryImpl) GetDefaultByOwner(ctx context.Context, ownerID, tenantID uuid.UUID) (*entities.Wallet, error) {
	var wallet entities.Wallet

	err := r.db.WithContext(ctx).
		Where("owner_id = ? AND tenant_id = ? AND is_default = true", ownerID, tenantID).
		First(&wallet).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entities.NewNotFoundError("default_wallet", ownerID.String())
		}
		helpers.LogError("Falha ao buscar carteira padrão", err, map[string]interface{}{
			"owner_id":  ownerID,
			"tenant_id": tenantID,
		})
		return nil, err
	}

	return &wallet, nil
}

// Update atualiza uma carteira existente
func (r *walletRepositoryImpl) Update(ctx context.Context, wallet *entities.Wallet) error {
	if err := wallet.Validate(); err != nil {
		return err
	}

	// Verificar se já existe carteira padrão para o proprietário (excluindo a atual)
	if wallet.IsDefault {
		exists, err := r.ExistsDefaultWallet(ctx, wallet.OwnerID, wallet.TenantID, &wallet.ID)
		if err != nil {
			return err
		}
		if exists {
			return entities.ErrDuplicateDefaultWallet
		}
	}

	result := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", wallet.ID, wallet.TenantID).
		Updates(wallet)

	if result.Error != nil {
		helpers.LogError("Falha ao atualizar carteira", result.Error, map[string]interface{}{
			"wallet_id": wallet.ID,
			"tenant_id": wallet.TenantID,
		})
		return result.Error
	}

	if result.RowsAffected == 0 {
		return entities.NewNotFoundError("wallet", wallet.ID.String())
	}

	helpers.LogAudit("wallet_updated", wallet.TenantID.String(), wallet.UpdatedBy.String(), map[string]interface{}{
		"wallet_id": wallet.ID,
	})

	return nil
}

// Delete remove uma carteira (soft delete)
func (r *walletRepositoryImpl) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		Delete(&entities.Wallet{})

	if result.Error != nil {
		helpers.LogError("Falha ao deletar carteira", result.Error, map[string]interface{}{
			"wallet_id": id,
			"tenant_id": tenantID,
		})
		return result.Error
	}

	if result.RowsAffected == 0 {
		return entities.NewNotFoundError("wallet", id.String())
	}

	helpers.LogAudit("wallet_deleted", tenantID.String(), "", map[string]interface{}{
		"wallet_id": id,
	})

	return nil
}

// List lista carteiras com paginação e filtros
func (r *walletRepositoryImpl) List(ctx context.Context, params domaininterfaces.ListWalletParams) ([]*entities.Wallet, int64, error) {
	var wallets []*entities.Wallet
	var total int64

	query := r.db.WithContext(ctx).Model(&entities.Wallet{}).
		Where("tenant_id = ?", params.TenantID)

	// Aplicar filtros
	if params.OwnerID != nil {
		query = query.Where("owner_id = ?", *params.OwnerID)
	}

	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	if params.Type != nil {
		query = query.Where("type = ?", *params.Type)
	}

	if params.Search != "" {
		query = query.Where("name ILIKE ?", "%"+params.Search+"%")
	}

	// Contar total de registros
	if err := query.Count(&total).Error; err != nil {
		helpers.LogError("Falha ao contar carteiras", err, map[string]interface{}{
			"tenant_id": params.TenantID,
		})
		return nil, 0, err
	}

	// Aplicar ordenação
	orderBy := "created_at DESC"
	if params.SortBy != "" {
		direction := "ASC"
		if params.SortDesc {
			direction = "DESC"
		}
		orderBy = fmt.Sprintf("%s %s", params.SortBy, direction)
	}
	query = query.Order(orderBy)

	// Aplicar paginação
	if params.Limit > 0 {
		offset := (params.Page - 1) * params.Limit
		query = query.Offset(offset).Limit(params.Limit)
	}

	// Buscar registros
	if err := query.Find(&wallets).Error; err != nil {
		helpers.LogError("Falha ao listar carteiras", err, map[string]interface{}{
			"tenant_id": params.TenantID,
		})
		return nil, 0, err
	}

	return wallets, total, nil
}

// UpdateBalance atualiza o saldo de uma carteira
func (r *walletRepositoryImpl) UpdateBalance(ctx context.Context, id, tenantID uuid.UUID, newBalance int64) error {
	result := r.db.WithContext(ctx).
		Model(&entities.Wallet{}).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		Update("balance", newBalance)

	if result.Error != nil {
		helpers.LogError("Falha ao atualizar saldo", result.Error, map[string]interface{}{
			"wallet_id":   id,
			"tenant_id":   tenantID,
			"new_balance": newBalance,
		})
		return result.Error
	}

	if result.RowsAffected == 0 {
		return entities.NewNotFoundError("wallet", id.String())
	}

	helpers.LogAudit("wallet_balance_updated", tenantID.String(), "", map[string]interface{}{
		"wallet_id":   id,
		"new_balance": newBalance,
	})

	return nil
}

// ExistsDefaultWallet verifica se já existe carteira padrão para o proprietário
func (r *walletRepositoryImpl) ExistsDefaultWallet(ctx context.Context, ownerID, tenantID uuid.UUID, excludeID *uuid.UUID) (bool, error) {
	query := r.db.WithContext(ctx).Model(&entities.Wallet{}).
		Where("owner_id = ? AND tenant_id = ? AND is_default = true", ownerID, tenantID)

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		helpers.LogError("Falha ao verificar carteira padrão existente", err, map[string]interface{}{
			"owner_id":  ownerID,
			"tenant_id": tenantID,
		})
		return false, err
	}

	return count > 0, nil
}

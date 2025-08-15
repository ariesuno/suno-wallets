package services

import (
	"context"
	"fmt"
	"time"

	"suno-wallets/src/domain/entities"
	"suno-wallets/src/domain/interfaces"
	"suno-wallets/src/domain/valueobjects"
)

// NormalizationService implementa regras de negócio de normalização (Domain Service)
type NormalizationService struct {
	repository interfaces.NormalizationRepository
}

// NewNormalizationService cria nova instância do serviço de domínio
func NewNormalizationService(repository interfaces.NormalizationRepository) *NormalizationService {
	return &NormalizationService{
		repository: repository,
	}
}

// ValidateNormalizationParams implementa validação de domínio
func (s *NormalizationService) ValidateNormalizationParams(ctx context.Context, params *interfaces.NormalizationParams) error {
	// Validar CPF
	_, err := valueobjects.NewCPF(params.CPF)
	if err != nil {
		return fmt.Errorf("CPF inválido: %w", err)
	}

	// Validar Data Type
	_, err = valueobjects.NewDataType(params.DataType)
	if err != nil {
		return fmt.Errorf("tipo de dados inválido: %w", err)
	}

	// Validar Asset Type
	if params.AssetType == "" {
		return fmt.Errorf("tipo de ativo é obrigatório")
	}

	// Validar datas
	if params.Start.IsZero() {
		return fmt.Errorf("data de início é obrigatória")
	}
	if params.End.IsZero() {
		return fmt.Errorf("data de fim é obrigatória")
	}
	if params.Start.After(params.End) {
		return fmt.Errorf("data de início não pode ser posterior à data de fim")
	}

	// Validar tenant
	if params.TenantID == "" {
		return fmt.Errorf("tenant ID é obrigatório")
	}

	// Regras de negócio específicas
	maxDateRange := 365 * 24 * time.Hour // 1 ano
	if params.End.Sub(params.Start) > maxDateRange {
		return fmt.Errorf("intervalo de datas não pode exceder 1 ano")
	}

	return nil
}

// ExecuteNormalization implementa o processo completo de normalização
func (s *NormalizationService) ExecuteNormalization(ctx context.Context, params *interfaces.NormalizationParams) (*interfaces.NormalizationResult, error) {
	result := &interfaces.NormalizationResult{
		StartedAt: time.Now(),
	}

	// 1. Validar parâmetros
	if err := s.ValidateNormalizationParams(ctx, params); err != nil {
		return nil, fmt.Errorf("parâmetros inválidos: %w", err)
	}

	// 2. Buscar dados RAW
	rawRecords, err := s.repository.FindPendingRawData(ctx, params.TenantID, params.CPF, params.DataType, params.AssetType, params.Start, params.End, params.Force)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar dados RAW: %w", err)
	}

	result.TotalProcessed = len(rawRecords)

	// 3. Processar cada registro RAW
	for _, rawRecord := range rawRecords {
		if err := s.processRawRecord(ctx, params, rawRecord, result); err != nil {
			result.Errors = append(result.Errors, err)
			result.ErrorsCount++
		}
	}

	result.CompletedAt = time.Now()
	return result, nil
}

// processRawRecord processa um registro RAW individual
func (s *NormalizationService) processRawRecord(ctx context.Context, params *interfaces.NormalizationParams, rawRecord interfaces.RawDataRecord, result *interfaces.NormalizationResult) error {
	cpf, err := valueobjects.NewCPF(params.CPF)
	if err != nil {
		return fmt.Errorf("erro ao criar CPF: %w", err)
	}

	dataType, err := valueobjects.NewDataType(params.DataType)
	if err != nil {
		return fmt.Errorf("erro ao criar DataType: %w", err)
	}

	// Aplicar regras de negócio específicas por tipo
	if dataType.IsTransactions() {
		return s.processTransactions(ctx, params, rawRecord, cpf, result)
	} else if dataType.IsPositions() {
		return s.processPositions(ctx, params, rawRecord, cpf, result)
	}

	return fmt.Errorf("tipo de dados não suportado: %s", dataType.Value())
}

// processTransactions processa transações
func (s *NormalizationService) processTransactions(ctx context.Context, params *interfaces.NormalizationParams, rawRecord interfaces.RawDataRecord, cpf *valueobjects.CPF, result *interfaces.NormalizationResult) error {
	// TODO: Implementar parsing e validação de transações
	// Por agora, criar entidade básica para demonstrar estrutura

	transaction, err := entities.NewNormalizedTransaction(params.TenantID, cpf, params.AssetType, rawRecord.GetID())
	if err != nil {
		return fmt.Errorf("erro ao criar transação normalizada: %w", err)
	}

	if err := transaction.Validate(); err != nil {
		return fmt.Errorf("transação inválida: %w", err)
	}

	if !params.DryRun {
		if err := s.repository.SaveNormalizedTransactions(ctx, []*entities.NormalizedTransaction{transaction}); err != nil {
			return fmt.Errorf("erro ao salvar transação: %w", err)
		}

		if err := s.repository.MarkRawAsProcessed(ctx, rawRecord.GetID(), 1); err != nil {
			return fmt.Errorf("erro ao marcar RAW como processado: %w", err)
		}
	}

	result.Transactions = append(result.Transactions, transaction)
	result.TransactionsCount++

	return nil
}

// processPositions processa posições
func (s *NormalizationService) processPositions(ctx context.Context, params *interfaces.NormalizationParams, rawRecord interfaces.RawDataRecord, cpf *valueobjects.CPF, result *interfaces.NormalizationResult) error {
	// TODO: Implementar parsing e validação de posições
	// Por agora, criar entidade básica para demonstrar estrutura

	position, err := entities.NewNormalizedPosition(params.TenantID, cpf, params.AssetType, rawRecord.GetID())
	if err != nil {
		return fmt.Errorf("erro ao criar posição normalizada: %w", err)
	}

	if err := position.Validate(); err != nil {
		return fmt.Errorf("posição inválida: %w", err)
	}

	if !params.DryRun {
		if err := s.repository.SaveNormalizedPositions(ctx, []*entities.NormalizedPosition{position}); err != nil {
			return fmt.Errorf("erro ao salvar posição: %w", err)
		}

		if err := s.repository.MarkRawAsProcessed(ctx, rawRecord.GetID(), 1); err != nil {
			return fmt.Errorf("erro ao marcar RAW como processado: %w", err)
		}
	}

	result.Positions = append(result.Positions, position)
	result.PositionsCount++

	return nil
}

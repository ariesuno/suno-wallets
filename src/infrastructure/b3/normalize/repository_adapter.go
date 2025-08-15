package normalize

import (
	"context"
	"time"

	"suno-wallets/src/domain/entities"
	"suno-wallets/src/domain/interfaces"
	"suno-wallets/src/infrastructure/b3/persistence"

	"github.com/google/uuid"
)

// RawRecordAdapter adapta persistence.RawRecord para interfaces.RawDataRecord (DDD)
type RawRecordAdapter struct {
	record persistence.RawRecord
}

func NewRawRecordAdapter(record persistence.RawRecord) *RawRecordAdapter {
	return &RawRecordAdapter{record: record}
}

func (r *RawRecordAdapter) GetID() uuid.UUID {
	return r.record.ID
}

func (r *RawRecordAdapter) GetTenantID() string {
	return r.record.TenantID
}

func (r *RawRecordAdapter) GetCPF() string {
	return r.record.CPF
}

func (r *RawRecordAdapter) GetDataType() string {
	return r.record.DataType
}

func (r *RawRecordAdapter) GetAssetType() string {
	return r.record.AssetType
}

func (r *RawRecordAdapter) GetPayload() []byte {
	return r.record.PayloadJSON
}

func (r *RawRecordAdapter) GetPeriodStart() time.Time {
	return r.record.PeriodStart
}

func (r *RawRecordAdapter) GetPeriodEnd() time.Time {
	return r.record.PeriodEnd
}

// NormalizationRepositoryAdapter adapta NormalizedRepository para interfaces de domínio
type NormalizationRepositoryAdapter struct {
	repo NormalizedRepository
}

func NewNormalizationRepositoryAdapter(repo NormalizedRepository) *NormalizationRepositoryAdapter {
	return &NormalizationRepositoryAdapter{repo: repo}
}

// FindPendingRawData implementa interface de domínio
func (r *NormalizationRepositoryAdapter) FindPendingRawData(ctx context.Context, tenantID string, cpf, dataType, assetType string, start, end time.Time, force bool) ([]interfaces.RawDataRecord, error) {
	rawRecords, err := r.repo.SelectPendingRaw(ctx, tenantID, cpf, dataType, assetType, start, end, force)
	if err != nil {
		return nil, err
	}

	result := make([]interfaces.RawDataRecord, len(rawRecords))
	for i, record := range rawRecords {
		result[i] = NewRawRecordAdapter(record)
	}

	return result, nil
}

// SaveNormalizedTransactions implementa interface de domínio
func (r *NormalizationRepositoryAdapter) SaveNormalizedTransactions(ctx context.Context, transactions []*entities.NormalizedTransaction) error {
	// Converter entidades de domínio para estruturas de infraestrutura
	infraTransactions := make([]NormalizedTransaction, len(transactions))
	for i, domainTx := range transactions {
		// Converter campos de domínio para infraestrutura
		var price string
		if domainTx.Price != nil {
			price = *domainTx.Price
		}

		infraTransactions[i] = NormalizedTransaction{
			ID:             domainTx.ID,
			TenantID:       domainTx.TenantID,
			CPF:            domainTx.CPF.Value(),
			AssetType:      domainTx.AssetType,
			SourceVersion:  domainTx.SourceVersion,
			RawID:          domainTx.RawID,
			SequenceInRaw:  domainTx.SequenceInRaw,
			TradeDate:      domainTx.ReferenceDate, // Mapear ReferenceDate para TradeDate
			Ticker:         domainTx.Ticker,
			ISIN:           domainTx.ISIN,
			Side:           domainTx.Side,
			Quantity:       domainTx.Quantity,
			Price:          price,
			GrossValue:     domainTx.Amount, // Mapear Amount para GrossValue
			Currency:       domainTx.Currency,
			NormalizedHash: domainTx.NormalizedHash,
			NormalizedAt:   domainTx.NormalizedAt,
		}
	}

	return r.repo.UpsertTransactions(ctx, infraTransactions)
}

// SaveNormalizedPositions implementa interface de domínio
func (r *NormalizationRepositoryAdapter) SaveNormalizedPositions(ctx context.Context, positions []*entities.NormalizedPosition) error {
	// Converter entidades de domínio para estruturas de infraestrutura
	infraPositions := make([]NormalizedPosition, len(positions))
	for i, domainPos := range positions {
		infraPositions[i] = NormalizedPosition{
			ID:             domainPos.ID,
			TenantID:       domainPos.TenantID,
			CPF:            domainPos.CPF.Value(),
			AssetType:      domainPos.AssetType,
			SourceVersion:  domainPos.SourceVersion,
			RawID:          domainPos.RawID,
			SequenceInRaw:  domainPos.SequenceInRaw,
			ReferenceDate:  domainPos.ReferenceDate,
			Ticker:         domainPos.Ticker,
			ISIN:           domainPos.ISIN,
			Quantity:       domainPos.Quantity,
			AvgPrice:       domainPos.AvgPrice,
			PositionValue:  domainPos.PositionValue,
			Currency:       domainPos.Currency,
			NormalizedHash: domainPos.NormalizedHash,
			NormalizedAt:   domainPos.NormalizedAt,
		}
	}

	return r.repo.UpsertPositions(ctx, infraPositions)
}

// MarkRawAsProcessed implementa interface de domínio
func (r *NormalizationRepositoryAdapter) MarkRawAsProcessed(ctx context.Context, rawID uuid.UUID, count int) error {
	return r.repo.MarkRawNormalized(ctx, rawID, count)
}

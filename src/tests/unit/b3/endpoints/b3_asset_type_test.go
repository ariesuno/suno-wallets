package endpoints

import (
	"testing"

	"suno-wallets/src/domain/enums"

	"github.com/stretchr/testify/assert"
)

// TestB3AssetType testa o enum de tipos de ativo B3
func TestB3AssetType(t *testing.T) {
	t.Run("IsValid", func(t *testing.T) {
		assert.True(t, enums.B3AssetTypeEquities.IsValid())
		assert.True(t, enums.B3AssetTypeFixedIncome.IsValid())
		assert.True(t, enums.B3AssetTypeTreasuryBonds.IsValid())
		assert.True(t, enums.B3AssetTypeDerivatives.IsValid())
		assert.True(t, enums.B3AssetTypeSecuritiesLending.IsValid())

		invalidType := enums.B3AssetType("invalid")
		assert.False(t, invalidType.IsValid())
	})

	t.Run("GetAPIEndpoint", func(t *testing.T) {
		assert.Equal(t, "/equities/investors", enums.B3AssetTypeEquities.GetAPIEndpoint())
		assert.Equal(t, "/fixed-income/investors", enums.B3AssetTypeFixedIncome.GetAPIEndpoint())
		assert.Equal(t, "/treasury-bonds/investors", enums.B3AssetTypeTreasuryBonds.GetAPIEndpoint())
		assert.Equal(t, "/derivatives/investors", enums.B3AssetTypeDerivatives.GetAPIEndpoint())
		assert.Equal(t, "/securities-lending/investors", enums.B3AssetTypeSecuritiesLending.GetAPIEndpoint())
	})

	t.Run("GetPositionsEndpoint", func(t *testing.T) {
		// Equities usa endpoint v3 específico
		assert.Equal(t, "/position/v3/equities/investors", enums.B3AssetTypeEquities.GetPositionsEndpoint())

		// Outros tipos usam endpoint base
		assert.Equal(t, "/fixed-income/investors", enums.B3AssetTypeFixedIncome.GetPositionsEndpoint())
		assert.Equal(t, "/treasury-bonds/investors", enums.B3AssetTypeTreasuryBonds.GetPositionsEndpoint())
	})

	t.Run("GetDisplayName", func(t *testing.T) {
		assert.Equal(t, "Ações", enums.B3AssetTypeEquities.GetDisplayName())
		assert.Equal(t, "Renda Fixa", enums.B3AssetTypeFixedIncome.GetDisplayName())
		assert.Equal(t, "Títulos do Tesouro", enums.B3AssetTypeTreasuryBonds.GetDisplayName())
		assert.Equal(t, "Derivativos", enums.B3AssetTypeDerivatives.GetDisplayName())
		assert.Equal(t, "Empréstimos de Valores Mobiliários", enums.B3AssetTypeSecuritiesLending.GetDisplayName())
	})

	t.Run("IsTransactionSupported", func(t *testing.T) {
		// Todos os tipos devem suportar transações
		allTypes := enums.GetAllB3AssetTypes()
		for _, assetType := range allTypes {
			assert.True(t, assetType.IsTransactionSupported(),
				"Asset type %s should support transactions", assetType)
		}
	})

	t.Run("IsPositionSupported", func(t *testing.T) {
		// Todos os tipos devem suportar posições conforme implementação atual
		allTypes := enums.GetAllB3AssetTypes()
		for _, assetType := range allTypes {
			assert.True(t, assetType.IsPositionSupported(),
				"Asset type %s should support positions", assetType)
		}
	})

	t.Run("ParseB3AssetType", func(t *testing.T) {
		validCases := map[string]enums.B3AssetType{
			"equities":           enums.B3AssetTypeEquities,
			"fixed-income":       enums.B3AssetTypeFixedIncome,
			"treasury-bonds":     enums.B3AssetTypeTreasuryBonds,
			"derivatives":        enums.B3AssetTypeDerivatives,
			"securities-lending": enums.B3AssetTypeSecuritiesLending,
		}

		for input, expected := range validCases {
			result, valid := enums.ParseB3AssetType(input)
			assert.True(t, valid, "Should parse %s as valid", input)
			assert.Equal(t, expected, result, "Should parse %s correctly", input)
		}

		// Caso inválido
		_, valid := enums.ParseB3AssetType("invalid-type")
		assert.False(t, valid, "Should reject invalid type")
	})

	t.Run("MapFromGenericAssetType", func(t *testing.T) {
		stockMapping := enums.MapFromGenericAssetType(enums.AssetTypeStock)
		assert.Contains(t, stockMapping, enums.B3AssetTypeEquities)

		bondMapping := enums.MapFromGenericAssetType(enums.AssetTypeBond)
		assert.Contains(t, bondMapping, enums.B3AssetTypeFixedIncome)
		assert.Contains(t, bondMapping, enums.B3AssetTypeTreasuryBonds)

		derivativeMapping := enums.MapFromGenericAssetType(enums.AssetTypeDerivative)
		assert.Contains(t, derivativeMapping, enums.B3AssetTypeDerivatives)

		// Teste fallback
		unknownMapping := enums.MapFromGenericAssetType(enums.AssetTypeCash)
		assert.Contains(t, unknownMapping, enums.B3AssetTypeEquities) // fallback
	})

	t.Run("GetAllB3AssetTypes", func(t *testing.T) {
		allTypes := enums.GetAllB3AssetTypes()
		assert.Len(t, allTypes, 5)

		expectedTypes := []enums.B3AssetType{
			enums.B3AssetTypeEquities,
			enums.B3AssetTypeFixedIncome,
			enums.B3AssetTypeTreasuryBonds,
			enums.B3AssetTypeDerivatives,
			enums.B3AssetTypeSecuritiesLending,
		}

		for _, expectedType := range expectedTypes {
			assert.Contains(t, allTypes, expectedType)
		}
	})
}

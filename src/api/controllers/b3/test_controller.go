package b3

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"suno-wallets/src/application/b3/diagnostics"
	"suno-wallets/src/shared/validation"
)

// Comentários em pt-BR: controller para teste da conexão com B3

type TestController struct {
	svc *diagnostics.Service
}

func NewTestController(svc *diagnostics.Service) *TestController {
	return &TestController{svc: svc}
}

// GET /b3/test/connection - Testa conexão e retorna relatório básico
func (c *TestController) TestConnection(ctx *gin.Context) {
	tenantIDStr := ctx.GetHeader("X-Tenant-ID")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_TENANT_ID"})
		return
	}

	cpf := ctx.Query("cpf")
	yearMonth := ctx.Query("yearMonth") // Formato: 2024-01

	// Validação do CPF
	if err := validation.ValidateCPF(cpf); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validação e parse do yearMonth
	if yearMonth == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "yearMonth é obrigatório (formato: YYYY-MM)"})
		return
	}

	date, err := time.Parse("2006-01", yearMonth)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "formato inválido para yearMonth (use YYYY-MM)"})
		return
	}

	// Calcula início e fim do mês
	startOfMonth := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, -1) // Último dia do mês

	// Executa o teste
	result, err := c.svc.TestConnectionAndReport(ctx, diagnostics.TestParams{
		TenantID:  tenantID,
		CPF:       cpf,
		StartDate: startOfMonth,
		EndDate:   endOfMonth,
	})

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "TESTE_CONEXAO_FALHOU",
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success":          true,
		"connectionStatus": "OK",
		"period": gin.H{
			"yearMonth": yearMonth,
			"start":     startOfMonth.Format("2006-01-02"),
			"end":       endOfMonth.Format("2006-01-02"),
		},
		"cpf":    cpf,
		"report": result,
	})
}

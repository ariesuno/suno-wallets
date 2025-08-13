package b3

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"suno-wallets/src/api/middlewares"
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

// TestConnection godoc
// @Summary Teste de conexão com a B3
// @Description Testa a conectividade com a API da B3 usando credenciais do cliente e retorna relatório detalhado de status. Este endpoint é essencial para validar se as credenciais estão funcionando antes de executar operações de sincronização.
// @Tags B3 Test
// @Produce json
// @Param X-Tenant-ID header string true "ID do inquilino (tenant)" example(status_invest)
// @Param cpf query string true "CPF do cliente (11 dígitos, apenas números)" example(12345678901)
// @Param yearMonth query string true "Período para teste no formato YYYY-MM" example(2024-01)
// @Success 200 {object} map[string]interface{} "Conexão bem-sucedida"
// @Failure 400 {object} map[string]string "Parâmetros inválidos (CPF ou yearMonth)"
// @Failure 500 {object} map[string]string "Falha na conexão com B3"
// @Router /b3/test/connection [get]
func (c *TestController) TestConnection(ctx *gin.Context) {
	// Usar o middleware para obter o tenant name (já validado)
	tenantID := middlewares.MustGetTenantName(ctx)

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

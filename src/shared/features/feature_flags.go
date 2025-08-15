package features

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"suno-wallets/src/shared/helpers"

	"github.com/redis/go-redis/v9"
)

// FeatureFlag representa uma feature flag
type FeatureFlag struct {
	Name        string                 `json:"name"`
	Enabled     bool                   `json:"enabled"`
	Description string                 `json:"description"`
	Rollout     int                    `json:"rollout"`    // Porcentagem 0-100
	Config      map[string]interface{} `json:"config"`     // Configurações específicas
	Conditions  []Condition            `json:"conditions"` // Condições para ativação
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CreatedBy   string                 `json:"created_by"`
}

// Condition representa uma condição para ativação de feature
type Condition struct {
	Type     ConditionType `json:"type"`
	Property string        `json:"property"`
	Operator string        `json:"operator"` // eq, ne, in, gt, lt, contains
	Value    interface{}   `json:"value"`
}

// ConditionType define tipos de condições
type ConditionType string

const (
	ConditionTypeTenant     ConditionType = "tenant"
	ConditionTypeUser       ConditionType = "user"
	ConditionTypePercentage ConditionType = "percentage"
	ConditionTypeDate       ConditionType = "date"
	ConditionTypeCustom     ConditionType = "custom"
)

// EvaluationContext contém dados para avaliação de flags
type EvaluationContext struct {
	TenantID string                 `json:"tenant_id"`
	UserID   string                 `json:"user_id"`
	CPF      string                 `json:"cpf,omitempty"`
	Custom   map[string]interface{} `json:"custom"`
	Hash     string                 `json:"hash"` // Para distribuição percentual consistente
}

// FeatureFlagManager gerencia feature flags
type FeatureFlagManager struct {
	redisClient *redis.Client
	cache       map[string]*FeatureFlag
	cacheMutex  sync.RWMutex
	keyPrefix   string
	cacheTTL    time.Duration
}

// NewFeatureFlagManager cria uma nova instância do gerenciador
func NewFeatureFlagManager(redisClient *redis.Client) *FeatureFlagManager {
	return &FeatureFlagManager{
		redisClient: redisClient,
		cache:       make(map[string]*FeatureFlag),
		keyPrefix:   "suno:feature_flags:",
		cacheTTL:    5 * time.Minute, // Cache local por 5 minutos
	}
}

// IsEnabled verifica se uma feature está habilitada
func (ffm *FeatureFlagManager) IsEnabled(ctx context.Context, flagName string, evalCtx *EvaluationContext) bool {
	flag, err := ffm.GetFlag(ctx, flagName)
	if err != nil {
		helpers.LogError("failed to get feature flag", err, map[string]interface{}{
			"flag_name": flagName,
			"tenant_id": evalCtx.TenantID,
		})
		return false // Safe default
	}

	if flag == nil {
		return false // Flag não existe
	}

	return ffm.evaluateFlag(flag, evalCtx)
}

// GetConfig retorna configuração de uma feature
func (ffm *FeatureFlagManager) GetConfig(ctx context.Context, flagName string, evalCtx *EvaluationContext) map[string]interface{} {
	flag, err := ffm.GetFlag(ctx, flagName)
	if err != nil || flag == nil || !ffm.evaluateFlag(flag, evalCtx) {
		return nil
	}

	return flag.Config
}

// GetFlag recupera uma feature flag (com cache)
func (ffm *FeatureFlagManager) GetFlag(ctx context.Context, flagName string) (*FeatureFlag, error) {
	// Verificar cache local primeiro
	ffm.cacheMutex.RLock()
	if cached, exists := ffm.cache[flagName]; exists {
		ffm.cacheMutex.RUnlock()
		return cached, nil
	}
	ffm.cacheMutex.RUnlock()

	// Buscar no Redis
	key := ffm.keyPrefix + flagName
	result, err := ffm.redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Flag não existe
		}
		return nil, fmt.Errorf("failed to get flag from Redis: %w", err)
	}

	var flag FeatureFlag
	if err := json.Unmarshal([]byte(result), &flag); err != nil {
		return nil, fmt.Errorf("failed to unmarshal flag: %w", err)
	}

	// Atualizar cache local
	ffm.cacheMutex.Lock()
	ffm.cache[flagName] = &flag
	ffm.cacheMutex.Unlock()

	// Configurar limpeza de cache
	go func() {
		time.Sleep(ffm.cacheTTL)
		ffm.cacheMutex.Lock()
		delete(ffm.cache, flagName)
		ffm.cacheMutex.Unlock()
	}()

	return &flag, nil
}

// SetFlag cria ou atualiza uma feature flag
func (ffm *FeatureFlagManager) SetFlag(ctx context.Context, flag *FeatureFlag) error {
	now := time.Now()
	if flag.CreatedAt.IsZero() {
		flag.CreatedAt = now
	}
	flag.UpdatedAt = now

	data, err := json.Marshal(flag)
	if err != nil {
		return fmt.Errorf("failed to marshal flag: %w", err)
	}

	key := ffm.keyPrefix + flag.Name
	if err := ffm.redisClient.Set(ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to save flag to Redis: %w", err)
	}

	// Invalidar cache local
	ffm.cacheMutex.Lock()
	delete(ffm.cache, flag.Name)
	ffm.cacheMutex.Unlock()

	helpers.LogInfo("feature flag updated", map[string]interface{}{
		"flag_name":  flag.Name,
		"enabled":    flag.Enabled,
		"rollout":    flag.Rollout,
		"updated_by": flag.CreatedBy,
	})

	return nil
}

// ListFlags lista todas as feature flags
func (ffm *FeatureFlagManager) ListFlags(ctx context.Context) ([]*FeatureFlag, error) {
	keys, err := ffm.redisClient.Keys(ctx, ffm.keyPrefix+"*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list flag keys: %w", err)
	}

	var flags []*FeatureFlag
	for _, key := range keys {
		result, err := ffm.redisClient.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var flag FeatureFlag
		if err := json.Unmarshal([]byte(result), &flag); err != nil {
			continue
		}

		flags = append(flags, &flag)
	}

	return flags, nil
}

// DeleteFlag remove uma feature flag
func (ffm *FeatureFlagManager) DeleteFlag(ctx context.Context, flagName string) error {
	key := ffm.keyPrefix + flagName
	if err := ffm.redisClient.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete flag: %w", err)
	}

	// Invalidar cache local
	ffm.cacheMutex.Lock()
	delete(ffm.cache, flagName)
	ffm.cacheMutex.Unlock()

	helpers.LogInfo("feature flag deleted", map[string]interface{}{
		"flag_name": flagName,
	})

	return nil
}

// evaluateFlag avalia se uma flag deve estar habilitada para o contexto
func (ffm *FeatureFlagManager) evaluateFlag(flag *FeatureFlag, evalCtx *EvaluationContext) bool {
	if !flag.Enabled {
		return false
	}

	// Avaliar condições
	for _, condition := range flag.Conditions {
		if !ffm.evaluateCondition(&condition, evalCtx) {
			return false
		}
	}

	// Avaliar rollout percentual
	if flag.Rollout < 100 {
		hash := evalCtx.Hash
		if hash == "" {
			hash = fmt.Sprintf("%s:%s", evalCtx.TenantID, evalCtx.UserID)
		}

		// Usar hash simples para distribuição consistente
		hashNum := 0
		for _, char := range hash {
			hashNum += int(char)
		}

		percentage := hashNum % 100
		if percentage >= flag.Rollout {
			return false
		}
	}

	return true
}

// evaluateCondition avalia uma condição específica
func (ffm *FeatureFlagManager) evaluateCondition(condition *Condition, evalCtx *EvaluationContext) bool {
	var contextValue interface{}

	switch condition.Type {
	case ConditionTypeTenant:
		contextValue = evalCtx.TenantID
	case ConditionTypeUser:
		contextValue = evalCtx.UserID
	case ConditionTypeCustom:
		contextValue = evalCtx.Custom[condition.Property]
	default:
		return false
	}

	return ffm.compareValues(contextValue, condition.Operator, condition.Value)
}

// compareValues compara valores baseado no operador
func (ffm *FeatureFlagManager) compareValues(contextValue interface{}, operator string, targetValue interface{}) bool {
	switch operator {
	case "eq":
		return contextValue == targetValue
	case "ne":
		return contextValue != targetValue
	case "in":
		if slice, ok := targetValue.([]interface{}); ok {
			for _, item := range slice {
				if contextValue == item {
					return true
				}
			}
		}
		return false
	case "contains":
		if contextStr, ok := contextValue.(string); ok {
			if targetStr, ok := targetValue.(string); ok {
				return strings.Contains(contextStr, targetStr)
			}
		}
		return false
	default:
		return false
	}
}

// Pre-defined feature flags
const (
	FeatureFlagNewWindowLogic        = "new_b3_window_logic"
	FeatureFlagBatchProcessing       = "batch_processing"
	FeatureFlagAdvancedRetries       = "advanced_retries"
	FeatureFlagEnhancedLogging       = "enhanced_logging"
	FeatureFlagPriorityQueues        = "priority_queues"
	FeatureFlagIdempotencyKeys       = "idempotency_keys"
	FeatureFlagRateLimitOptimization = "rate_limit_optimization"
	FeatureFlagDeadLetterQueue       = "dead_letter_queue"
)

// InitializeDefaultFlags configura flags padrão do sistema
func (ffm *FeatureFlagManager) InitializeDefaultFlags(ctx context.Context) error {
	defaultFlags := []*FeatureFlag{
		{
			Name:        FeatureFlagNewWindowLogic,
			Enabled:     true,
			Description: "Nova lógica de geração de janelas B3 com timezone América/São Paulo",
			Rollout:     100,
			Config: map[string]interface{}{
				"min_start_date": "2019-11-01T00:00:00-03:00",
				"timezone":       "America/Sao_Paulo",
			},
			CreatedBy: "system",
		},
		{
			Name:        FeatureFlagBatchProcessing,
			Enabled:     true,
			Description: "Processamento em lote para melhor performance",
			Rollout:     100,
			Config: map[string]interface{}{
				"default_batch_size":  1000,
				"max_batch_size":      5000,
				"batch_delay_seconds": 30,
			},
			CreatedBy: "system",
		},
		{
			Name:        FeatureFlagAdvancedRetries,
			Enabled:     true,
			Description: "Sistema avançado de retry com backoff exponencial",
			Rollout:     100,
			Config: map[string]interface{}{
				"max_retries":        3,
				"initial_delay_ms":   200,
				"max_delay_ms":       2000,
				"backoff_multiplier": 2.0,
			},
			CreatedBy: "system",
		},
		{
			Name:        FeatureFlagDeadLetterQueue,
			Enabled:     true,
			Description: "Dead letter queue para jobs que falharam permanentemente",
			Rollout:     100,
			Config: map[string]interface{}{
				"ttl_days":                30,
				"auto_retry_retriable":    true,
				"manual_review_threshold": 5,
			},
			CreatedBy: "system",
		},
	}

	for _, flag := range defaultFlags {
		// Verificar se já existe
		existing, _ := ffm.GetFlag(ctx, flag.Name)
		if existing == nil {
			if err := ffm.SetFlag(ctx, flag); err != nil {
				helpers.LogError("failed to initialize default flag", err, map[string]interface{}{
					"flag_name": flag.Name,
				})
			}
		}
	}

	return nil
}

package transactions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"suno-wallets/src/infrastructure/observability"
	"suno-wallets/src/shared/dto"
	"suno-wallets/src/shared/helpers"
	"suno-wallets/src/shared/validation"
)

// Comentários em pt-BR: serviço de orquestração para preview de transações v2

// TransactionsService define as operações de preview de transações
type TransactionsService interface {
	PreviewTransactions(ctx context.Context, req *dto.TransactionsPreviewRequest) (*dto.TransactionsPreviewResponse, error)
}

type transactionsServiceImpl struct {
	client B3Client
}

// NewTransactionsService cria uma nova instância do serviço
func NewTransactionsService(client B3Client) TransactionsService {
	return &transactionsServiceImpl{client: client}
}

// B3Client descreve as operações necessárias do cliente B3 (para facilitar testes/mocks)
type B3Client interface {
	MakeRequest(ctx context.Context, method, path string, query map[string]string, cpf string, needsAuth bool) (*http.Response, error)
	Paginate(ctx context.Context, method, path string, baseQuery map[string]string, cpf string, needsAuth bool, fetch func(*http.Response) (hasNext bool, nextPage int, err error)) error
}

func (s *transactionsServiceImpl) PreviewTransactions(ctx context.Context, req *dto.TransactionsPreviewRequest) (*dto.TransactionsPreviewResponse, error) {
	// Validar entradas
	if err := validation.ValidateCPF(req.CPF); err != nil {
		return nil, err
	}
	if err := validation.ValidateDateYMD(req.Start); err != nil {
		return nil, err
	}
	if err := validation.ValidateDateYMD(req.End); err != nil {
		return nil, err
	}
	if req.Page == 0 {
		req.Page = 1
	}
	if err := validation.ValidatePage(req.Page); err != nil {
		return nil, err
	}

	assetType := strings.ToLower(strings.TrimSpace(req.AssetType))
	if assetType == "" {
		assetType = "equity"
	}
	switch assetType {
	case "equity":
		// suportado
	default:
		return nil, errors.New("assetType not implemented")
	}

	path := fmt.Sprintf("/assets-trading/v2/%s/%s", assetType, req.CPF)
	baseQuery := map[string]string{
		"referenceStartDate": req.Start,
		"referenceEndDate":   req.End,
	}

	resp := &dto.TransactionsPreviewResponse{AssetType: assetType, Start: req.Start, End: req.End}

	// Função para coletar payload e decidir próxima página
	fetch := func(httpResp *http.Response) (hasNext bool, nextPage int, err error) {
		body, e := io.ReadAll(httpResp.Body)
		if e != nil {
			return false, 0, e
		}
		// anexar payload deserializado genericamente
		var generic interface{}
		_ = json.Unmarshal(body, &generic)
		if generic == nil {
			generic = map[string]interface{}{"raw": string(body)}
		}
		resp.Payloads = append(resp.Payloads, generic)

		// heurística: procurar por campo nextPage (inteiro) ou links
		var meta struct {
			NextPage *int  `json:"nextPage"`
			Page     *int  `json:"page"`
			HasNext  *bool `json:"hasNext"`
			DataLen  *int  `json:"dataLength"`
		}
		_ = json.Unmarshal(body, &meta)
		if meta.HasNext != nil && *meta.HasNext {
			if meta.NextPage != nil && *meta.NextPage > 0 {
				return true, *meta.NextPage, nil
			}
			if meta.Page != nil {
				return true, *meta.Page + 1, nil
			}
			return true, 0, nil // fallback: deixa Paginate incrementar
		}
		if meta.NextPage != nil && meta.Page != nil && *meta.NextPage > *meta.Page {
			return true, *meta.Page + 1, nil
		}
		return false, 0, nil
	}

	if req.FetchAllPages {
		// Paginar usando helper do cliente
		err := s.client.Paginate(ctx, http.MethodGet, path, baseQuery, req.CPF, true, fetch)
		if err != nil {
			observability.ObserveTransactionsPreview(assetType, "error", 0)
			return nil, err
		}
		resp.Pages = len(resp.Payloads)
		observability.ObserveTransactionsPreview(assetType, "success", resp.Pages)
		return resp, nil
	}

	// Buscar apenas a página solicitada
	q := map[string]string{
		"referenceStartDate": req.Start,
		"referenceEndDate":   req.End,
		"page":               fmt.Sprintf("%d", req.Page),
	}
	httpResp, err := s.client.MakeRequest(ctx, http.MethodGet, path, q, req.CPF, true)
	if err != nil {
		observability.ObserveTransactionsPreview(assetType, "error", 0)
		return nil, err
	}
	defer httpResp.Body.Close()
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	var generic interface{}
	_ = json.Unmarshal(body, &generic)
	if generic == nil {
		generic = map[string]interface{}{"raw": string(body)}
	}
	resp.Payloads = append(resp.Payloads, generic)
	resp.Pages = 1
	observability.ObserveTransactionsPreview(assetType, "success", resp.Pages)

	helpers.LogInfo("B3 transactions preview fetched", map[string]interface{}{
		"asset_type": assetType,
		"tenant_id":  "", // opcional: middleware injeta no contexto
		"cpf_masked": "*********" + req.CPF[9:],
		"fetch_all":  req.FetchAllPages,
	})

	return resp, nil
}

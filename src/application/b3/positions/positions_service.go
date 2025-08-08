package positions

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

// Comentários em pt-BR: serviço de preview de posições v3 (equities)

type B3Client interface {
	MakeRequest(ctx context.Context, method, path string, query map[string]string, cpf string, needsAuth bool) (*http.Response, error)
	Paginate(ctx context.Context, method, path string, baseQuery map[string]string, cpf string, needsAuth bool, fetch func(*http.Response) (hasNext bool, nextPage int, err error)) error
}

type Service interface {
	PreviewPositions(ctx context.Context, req *dto.PositionsPreviewRequest) (*dto.PositionsPreviewResponse, error)
}

type serviceImpl struct{ client B3Client }

func NewPositionsService(client B3Client) Service { return &serviceImpl{client: client} }

func (s *serviceImpl) PreviewPositions(ctx context.Context, req *dto.PositionsPreviewRequest) (*dto.PositionsPreviewResponse, error) {
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
	default:
		return nil, errors.New("assetType not implemented")
	}

	// v3 equities path
	path := fmt.Sprintf("/position/v3/equities/investors/%s", req.CPF)
	baseQuery := map[string]string{"referenceStartDate": req.Start, "referenceEndDate": req.End}
	resp := &dto.PositionsPreviewResponse{AssetType: assetType, Start: req.Start, End: req.End}

	fetch := func(httpResp *http.Response) (bool, int, error) {
		body, e := io.ReadAll(httpResp.Body)
		if e != nil {
			return false, 0, e
		}
		var generic interface{}
		_ = json.Unmarshal(body, &generic)
		if generic == nil {
			generic = map[string]interface{}{"raw": string(body)}
		}
		resp.Payloads = append(resp.Payloads, generic)
		// heurística simples para nextPage
		var meta struct {
			NextPage *int  `json:"nextPage"`
			Page     *int  `json:"page"`
			HasNext  *bool `json:"hasNext"`
		}
		_ = json.Unmarshal(body, &meta)
		if meta.HasNext != nil && *meta.HasNext {
			if meta.NextPage != nil && *meta.NextPage > 0 {
				return true, *meta.NextPage, nil
			}
			if meta.Page != nil {
				return true, *meta.Page + 1, nil
			}
			return true, 0, nil
		}
		if meta.NextPage != nil && meta.Page != nil && *meta.NextPage > *meta.Page {
			return true, *meta.Page + 1, nil
		}
		return false, 0, nil
	}

	if req.FetchAllPages {
		if err := s.client.Paginate(ctx, http.MethodGet, path, baseQuery, req.CPF, true, fetch); err != nil {
			observability.ObservePositionsPreview(assetType, "error", 0)
			return nil, err
		}
		resp.Pages = len(resp.Payloads)
		observability.ObservePositionsPreview(assetType, "success", resp.Pages)
		return resp, nil
	}

	q := map[string]string{"referenceStartDate": req.Start, "referenceEndDate": req.End, "page": fmt.Sprintf("%d", req.Page)}
    httpResp, err := s.client.MakeRequest(ctx, http.MethodGet, path, q, req.CPF, true)
	if err != nil {
		observability.ObservePositionsPreview(assetType, "error", 0)
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
	observability.ObservePositionsPreview(assetType, "success", resp.Pages)

	helpers.LogInfo("B3 positions preview fetched", map[string]interface{}{
		"asset_type": assetType,
		"tenant_id":  "",
		"cpf_masked": "*********" + req.CPF[9:],
		"fetch_all":  req.FetchAllPages,
	})

	return resp, nil
}

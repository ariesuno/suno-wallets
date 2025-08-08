package client

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Comentários em pt-BR: helper genérico de paginação
func (c *B3OfficialClient) Paginate(
	ctx context.Context,
	method, path string,
	baseQuery map[string]string,
	cpf string,
	needsAuth bool,
	fetch func(*http.Response) (hasNext bool, nextPage int, err error),
) error {
	page := 1
	for {
		q := make(map[string]string, len(baseQuery)+1)
		for k, v := range baseQuery {
			q[k] = v
		}
		q["page"] = fmt.Sprintf("%d", page)

		resp, err := c.MakeRequest(ctx, method, path, q, cpf, needsAuth)
		if err != nil {
			return err
		}
		hasNext, next, err := fetch(resp)
		_ = resp.Body.Close()
		if err != nil {
			return err
		}
		if !hasNext {
			return nil
		}
		page = next
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}

package b3_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"

	svc "suno-wallets/src/application/b3/transactions"
	"suno-wallets/src/shared/dto"
	"suno-wallets/src/shared/validation"

	"github.com/stretchr/testify/require"
)

// Comentários em pt-BR: testes unitários do TransactionsService com mock de cliente B3

type mockB3Client struct {
	lastQuery map[string]string
}

func (m *mockB3Client) MakeRequest(ctx context.Context, method, path string, query map[string]string, cpf string, needsAuth bool) (*http.Response, error) {
	m.lastQuery = query
	body := bytes.NewBufferString(`{"page":1,"data":[]}`)
	return &http.Response{StatusCode: 200, Body: ioutil.NopCloser(body)}, nil
}

func (m *mockB3Client) Paginate(ctx context.Context, method, path string, baseQuery map[string]string, cpf string, needsAuth bool, fetch func(*http.Response) (bool, int, error)) error {
	// Simula duas páginas
	for i := 0; i < 2; i++ {
		body := bytes.NewBufferString(`{"page":1,"hasNext":` + map[bool]string{true: "true", false: "false"}[i == 0] + `,"data":[]}`)
		resp := &http.Response{StatusCode: 200, Body: ioutil.NopCloser(body)}
		hasNext, _, err := fetch(resp)
		if err != nil {
			return err
		}
		if !hasNext {
			break
		}
	}
	return nil
}

func TestB3_TransactionsService_Validations(t *testing.T) {
	m := &mockB3Client{}
	s := svc.NewTransactionsService(m)

	// CPF inválido
	_, err := s.PreviewTransactions(context.Background(), &dto.TransactionsPreviewRequest{CPF: "123", Start: "2024-01-01", End: "2024-01-02", Page: 1})
	require.Error(t, err)

	// Datas inválidas
	_, err = s.PreviewTransactions(context.Background(), &dto.TransactionsPreviewRequest{CPF: "12345678901", Start: "2024-01-32", End: "2024-01-02", Page: 1})
	require.Error(t, err)

	// Página inválida (service normaliza para 1 e só então valida)
	_, err = s.PreviewTransactions(context.Background(), &dto.TransactionsPreviewRequest{CPF: "12345678901", Start: "2024-01-01", End: "2024-01-02", Page: 0})
	require.NoError(t, err)
}

func TestB3_TransactionsService_SinglePage(t *testing.T) {
	m := &mockB3Client{}
	s := svc.NewTransactionsService(m)
	req := &dto.TransactionsPreviewRequest{CPF: "12345678901", Start: "2024-01-01", End: "2024-01-02", Page: 1}
	require.NoError(t, validation.ValidateCPF(req.CPF))
	require.NoError(t, validation.ValidateDateYMD(req.Start))
	require.NoError(t, validation.ValidateDateYMD(req.End))

	resp, err := s.PreviewTransactions(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, 1, resp.Pages)
	require.Len(t, resp.Payloads, 1)
}

func TestB3_TransactionsService_FetchAllPages(t *testing.T) {
	m := &mockB3Client{}
	s := svc.NewTransactionsService(m)
	req := &dto.TransactionsPreviewRequest{CPF: "12345678901", Start: "2024-01-01", End: "2024-01-10", FetchAllPages: true}
	require.NoError(t, validation.ValidateCPF(req.CPF))
	require.NoError(t, validation.ValidateDateYMD(req.Start))
	require.NoError(t, validation.ValidateDateYMD(req.End))

	resp, err := s.PreviewTransactions(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, 2, resp.Pages)
	require.Len(t, resp.Payloads, 2)
}

package accrual

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestClient_GetOrder(t *testing.T) {
	tests := []struct {
		name                 string
		orderNumber          string
		mockResponse         interface{}
		mockStatusCode       int
		wantOrder            *Order
		wantErr              bool
		wantNonOkErrorStatus int
	}{
		{
			name:        "successful_response",
			orderNumber: "346436439",
			mockResponse: Order{
				Order:   "346436439",
				Status:  "PROCESSED",
				Accrual: 10.5,
			},
			mockStatusCode: http.StatusOK,
			wantOrder: &Order{
				Order:   "346436439",
				Status:  "PROCESSED",
				Accrual: 10.5,
			},
			wantErr: false,
		},
		{
			name:                 "no_content",
			orderNumber:          "346436439",
			mockResponse:         nil,
			mockStatusCode:       http.StatusNoContent,
			wantOrder:            nil,
			wantErr:              true,
			wantNonOkErrorStatus: http.StatusNoContent,
		},
		{
			name:        "invalid_json_response",
			orderNumber: "346436439",
			mockResponse: map[string]interface{}{
				"order":   "346436439",
				"status":  "PROCESSED",
				"accrual": "invalid", // не число
			},
			mockStatusCode: http.StatusOK,
			wantOrder:      nil,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем mock сервер
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatusCode)
				if tt.mockResponse != nil {
					json.NewEncoder(w).Encode(tt.mockResponse)
				}
			}))
			defer server.Close()

			// Создаем клиент с адресом mock сервера
			client := &Client{
				baseURL:    server.URL,
				httpClient: server.Client(),
			}

			// Вызываем тестируемый метод
			gotOrder, err := client.GetOrder(tt.orderNumber)

			// Проверяем ошибки
			if tt.wantErr {
				require.Error(t, err, "expected error when calling the client.GetOrder")
			}

			// Проверяем код статуса в ошибке NonOkError.
			if tt.wantNonOkErrorStatus != 0 {
				var nonOkErr *NonOkError
				require.ErrorAs(t, err, &nonOkErr, "expected NonOkError, got %T", nonOkErr, err)
				assert.Equal(t, tt.wantNonOkErrorStatus, nonOkErr.Code, "expected NonOkErrorCode = %v, got = %v", tt.wantNonOkErrorStatus, nonOkErr.Code)
			}

			// Проверяем результат
			if tt.wantOrder != nil {
				assert.Equal(t, gotOrder, tt.wantOrder, "GetOrder() = %v, want %v", gotOrder, tt.wantOrder)
			}
		})
	}
}

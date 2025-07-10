package accrual

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/eduardtungatarov/gofermart/internal/config"
)

// NonOkError в случае, если сбербанк ответил кодом ошибки != 200 (OK).
type NonOkError struct {
	Msg  string
	Code int
}

// Error текст ошибки.
func (e *NonOkError) Error() string {
	return fmt.Sprintf("unexpected status code: %d, status message: %s", e.Code, e.Msg)
}

type Order struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(cfg config.Config) *Client {
	return &Client{
		baseURL:    cfg.AccrualADDR,
		httpClient: &http.Client{},
	}
}

func (c *Client) GetOrder(orderNumber string) (*Order, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/orders/" + orderNumber)
	if err != nil {
		return nil, fmt.Errorf("httpClient.Get net err: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &NonOkError{Msg: resp.Status, Code: resp.StatusCode}
	}

	var order *Order
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("io.ReadAll: %w", err)
	}
	defer resp.Body.Close()

	err = json.Unmarshal(body, &order)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}

	return order, nil
}

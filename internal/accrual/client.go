package accrual

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/eduardtungatarov/gofermart/internal/config"
)

type Order struct {
	Order   string `json:"order"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual"`
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
		return nil, fmt.Errorf("httpClient.Get: %w", err)
	}

	var order *Order

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("io.ReadAll: %w", err)
	}

	err = json.Unmarshal(body, &order)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}

	return order, nil
}

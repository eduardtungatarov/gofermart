package handlers

type OrderResp struct {
	Number     string `json:"number"`
	Status     string `json:"status"`
	Accrual    int    `json:"accrual,omitempty"`
	UploadedAt string `json:"uploaded_at"`
}

type BalanceResp struct {
	Current   int `json:"current"`
	Withdrawn int `json:"withdrawn"`
}

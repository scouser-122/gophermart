package models

// WithdrawBalanceRequest defines request structure for withdraw user's loyalty points from balance for order
type WithdrawBalanceRequest struct {
	// Order order ID
	Order string `json:"order"`
	// Sum sum of loyalty points to withdraw from balance
	Sum float32 `json:"sum"`
}

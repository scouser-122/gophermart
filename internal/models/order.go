package models

import "time"

// OrderStatus defines values for order status enum
type OrderStatus string

const (
	// NewOrder new order
	NewOrder OrderStatus = "NEW"

	// ProcessingOrder order is processing
	ProcessingOrder OrderStatus = "PROCESSING"

	// Processing order is invalid
	InvalidOrder OrderStatus = "INVALID"

	// ProcessedOrder order is already processed
	ProcessedOrder OrderStatus = "PROCESSED"
)

// Order model
type Order struct {
	// ID order identifier
	ID string `db:"id"`

	// Status order status
	Status OrderStatus `db:"status"`

	// UploadedAt time when order was uploaded (saved in DB)
	UploadedAt time.Time `db:"uploaded_at"`

	// Order accrual
	Accrual *int64 `db:"accrual"`

	// UserLogin login of user who made this order
	UserLogin string `db:"user_login"`
}

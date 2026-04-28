package models

import "time"

// OrderStatus defines values for order status enum
type OrderStatus string

const (
	// New new order
	New OrderStatus = "NEW"

	// Processing order is processing
	Processing OrderStatus = "PROCESSING"

	// Processing order is invalid
	Invalid OrderStatus = "INVALID"

	// Processed order is already processed
	Processed OrderStatus = "PROCESSED"
)

// Order model
type Order struct {
	// ID order identifier
	ID string

	// Status order status
	Status OrderStatus

	// UploadedAt time when order was uploaded (saved in DB)
	UploadedAt time.Time

	// Order accrual
	Accrual *int64

	// UserLogin login of user who made this order
	UserLogin string
}

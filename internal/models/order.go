package models

import (
	"encoding/json"
	"time"
)

// OrderStatus defines values for order status enum
type OrderStatus string

const (
	// NewOrder new order
	NewOrder OrderStatus = "NEW"

	// RegisteredOrder new registered order
	RegisteredOrder OrderStatus = "REGISTERED"

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
	ID string `json:"number" db:"id"`

	// Status order status
	Status OrderStatus `json:"status" db:"status"`

	// UploadedAt time when order was uploaded (saved in DB)
	UploadedAt time.Time `json:"uploaded_at" db:"uploaded_at"`

	// Accrual for order
	Accrual *float32 `json:"accrual,omitempty" db:"accrual"`

	// Withdrawn sum for order
	Withdrawn *float32 `json:"-" db:"withdrawn"`

	// ProcessedAt time when order was processed
	ProcessedAt *time.Time `json:"processed_at" db:"processed_at"`

	// UserLogin login of user who made this order
	UserLogin string `json:"-" db:"user_login"`
}

// MarshalJSON кастомная сериализация всей структуры
func (o Order) MarshalJSON() ([]byte, error) {
	var processedAt string
	if o.ProcessedAt != nil {
		processedAt = o.ProcessedAt.Format("2006-01-02T15:04:05-07:00")
	} else {
		processedAt = ""
	}
	return json.Marshal(&struct {
		ID          string   `json:"number"`
		Status      string   `json:"status"`
		UploadedAt  string   `json:"uploaded_at"`
		Accrual     *float32 `json:"accrual,omitempty"`
		ProcessedAt string   `json:"processed_at,omitempty"`
	}{
		ID:          o.ID,
		Status:      string(o.Status),
		UploadedAt:  o.UploadedAt.Format("2006-01-02T15:04:05-07:00"),
		Accrual:     o.Accrual,
		ProcessedAt: processedAt,
	})
}

// UnmarshalJSON кастомная десериализация всей структуры
func (o *Order) UnmarshalJSON(data []byte) error {
	type Alias Order
	var aux Alias

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// пробуем распарсить с учётом возможных имён
	var temp map[string]interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// проверяем возможные имена полей
	if val, ok := temp["id"]; ok {
		o.ID = val.(string)
	} else if val, ok := temp["number"]; ok {
		o.ID = val.(string)
	} else if val, ok := temp["order"]; ok {
		o.ID = val.(string)
	}
	o.Status = aux.Status
	o.UploadedAt = aux.UploadedAt
	o.Accrual = aux.Accrual
	o.ProcessedAt = aux.ProcessedAt

	return nil
}

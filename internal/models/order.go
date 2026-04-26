package models

import "time"

type OrderStatus string

const (
	New        OrderStatus = "NEW"
	Processing OrderStatus = "PROCESSING"
	Invalid    OrderStatus = "INVALID"
	Processed  OrderStatus = "PROCESSED"
)

type Order struct {
	ID         string
	Status     OrderStatus
	UploadedAt time.Time
	Accrual    *int64
	UserLogin  string
}

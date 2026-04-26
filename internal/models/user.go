package models

import "time"

type User struct {
	Login     string    `json:"login" db:"login" table:"users"`
	Password  string    `json:"password" db:"password"`
	Balance   float64   `db:"balance"`
	CreatedAt time.Time `db:"created_at"`
}

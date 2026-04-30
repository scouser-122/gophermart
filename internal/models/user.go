package models

import "time"

// User model
type User struct {
	// Login for user
	Login string `json:"login" db:"login" table:"users"`

	// Password for user
	Password string `json:"password" db:"password"`

	// Balance contains user's balance
	Balance float64 `db:"balance"`

	// CreatedAt defines user's registration date and time
	CreatedAt time.Time `db:"created_at"`
}

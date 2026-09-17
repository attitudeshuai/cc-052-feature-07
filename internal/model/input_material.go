package model

import "time"

type InputMaterial struct {
	ID                int64     `db:"id" json:"id"`
	Name              string    `db:"name" json:"name"`
	Type              string    `db:"type" json:"type"`
	RegistrationNo    string    `db:"registration_no" json:"registration_no"`
	SafeIntervalDays  int       `db:"safe_interval_days" json:"safe_interval_days"`
	ActiveIngredient  string    `db:"active_ingredient" json:"active_ingredient"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
}
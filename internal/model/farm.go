package model

import "time"

type Farm struct {
	ID         int64     `db:"id" json:"id"`
	Name       string    `db:"name" json:"name"`
	RegionCode string    `db:"region_code" json:"region_code"`
	ContactRef string    `db:"contact_ref" json:"contact_ref"`
	CertNo     string    `db:"cert_no" json:"cert_no"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

type CreateFarmRequest struct {
	Name       string `json:"name" binding:"required"`
	RegionCode string `json:"region_code" binding:"required"`
	ContactRef string `json:"contact_ref"`
	CertNo     string `json:"cert_no"`
}
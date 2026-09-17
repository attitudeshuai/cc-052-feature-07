package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type ActivityKind string

const (
	ActivityFertilize  ActivityKind = "fertilize"
	ActivityPesticide  ActivityKind = "pesticide"
	ActivityIrrigation ActivityKind = "irrigation"
	ActivityWeed       ActivityKind = "weed"
)

type StringMap map[string]interface{}

func (m StringMap) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m *StringMap) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, m)
}

type Activity struct {
	ID         int64        `db:"id" json:"id"`
	BatchID    int64        `db:"batch_id" json:"batch_id"`
	ClientUUID string       `db:"client_uuid" json:"client_uuid"`
	Kind       ActivityKind `db:"kind" json:"kind"`
	HappenedAt time.Time    `db:"happened_at" json:"happened_at"`
	InputID    *int64       `db:"input_id" json:"input_id,omitempty"`
	Dose       *float64     `db:"dose" json:"dose,omitempty"`
	DoseUnit   *string      `db:"dose_unit" json:"dose_unit,omitempty"`
	Operator   string       `db:"operator" json:"operator"`
	Photos     StringMap    `db:"photos" json:"photos,omitempty"`
	Geo        *string      `db:"geo" json:"geo,omitempty"`
	CreatedAt  time.Time    `db:"created_at" json:"created_at"`
}

type CreateActivityRequest struct {
	ClientUUID string       `json:"client_uuid" binding:"required"`
	Kind       ActivityKind `json:"kind" binding:"required"`
	HappenedAt string       `json:"happened_at" binding:"required"`
	InputID    *int64       `json:"input_id"`
	Dose       *float64     `json:"dose"`
	DoseUnit   *string      `json:"dose_unit"`
	Operator   string       `json:"operator" binding:"required"`
	Photos     StringMap    `json:"photos"`
	Geo        *string      `json:"geo"`
}

type BatchCreateActivitiesRequest struct {
	Activities []CreateActivityRequest `json:"activities" binding:"required,min=1,dive"`
}
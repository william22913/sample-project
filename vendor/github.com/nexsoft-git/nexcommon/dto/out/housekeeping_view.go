package out

import (
	"time"
)

type GetListHousekeeping struct {
	ID             int64
	JobID          string
	Group          string
	Type           string
	Name           string
	Counter        int
	Total          int
	Status         string
	ProcessingTime float64
	CreatedBy      int64
	CreatedClient  string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

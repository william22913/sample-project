package model

import (
	"database/sql"
	"time"
)

type ActivityModel struct {
	ID                sql.NullInt64
	UUID              sql.NullString
	TimeActivity      sql.NullTime
	IsMenuAdmin       sql.NullBool
	ResponseStatus    sql.NullInt64
	MenuCode          sql.NullString
	IPMac             sql.NullString
	Activity          sql.NullString
	Header            sql.NullString
	Body              sql.NullString
	CreatedBy         sql.NullInt64
	CreatedClient     sql.NullString
	CreatedAt         sql.NullTime
	ResponseTimestamp sql.NullTime
	RequestTimestamp  sql.NullTime
}

type ActivityDTO struct {
	ID                int64                  `json:"id"`
	UUID              string                 `json:"uuid"`
	TimeActivity      time.Time              `json:"time_activity"`
	IsMenuAdmin       bool                   `json:"is_menu_admin"`
	MenuCode          string                 `json:"menu_code"`
	IPMac             string                 `json:"ip_mac"`
	Header            string                 `json:"header"`
	Body              string                 `json:"body"`
	ResponseStatus    int                    `json:"response_status"`
	Activity          map[string]interface{} `json:"activity"`
	CreatedBy         int64                  `json:"created_by"`
	CreatedClient     string                 `json:"created_client"`
	CreatedAt         time.Time              `json:"created_at"`
	RequestTimestamp  time.Time              `json:"request_timestamp"`
	ResponseTimestamp time.Time              `json:"response_timestamp"`
}

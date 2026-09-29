package out

import (
	"time"
)

type ListJobProcess struct {
	Level          int       `json:"level"`
	JobID          string    `json:"job_id"`
	Group          string    `json:"group"`
	Type           string    `json:"type"`
	Name           string    `json:"name"`
	Counter        int       `json:"counter"`
	Total          int       `json:"total"`
	Status         string    `json:"status"`
	URLIn          string    `json:"url_in"`
	FileNameIn     string    `json:"file_name_in"`
	ContentDataIn  string    `json:"content_data_in"`
	URLOut         string    `json:"url_out"`
	FileNameOut    string    `json:"file_name_out"`
	ContentDataOut string    `json:"content_data_out"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ParentJobProcess struct {
	JobId     string    `json:"job_id"`
	Group     string    `json:"group"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	Counter   int       `json:"counter"`
	Total     int       `json:"total"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Level     int       `json:"level"`
}

type DetailListJobProcessItemData struct {
	Level          int       `json:"level"`
	JobID          string    `json:"job_id"`
	Group          string    `json:"group"`
	Type           string    `json:"type"`
	Name           string    `json:"name"`
	Counter        int       `json:"counter"`
	Total          int       `json:"total"`
	Status         string    `json:"status"`
	URLIn          string    `json:"url_in"`
	FileNameIn     string    `json:"file_name_in"`
	ContentDataIn  string    `json:"content_data_in"`
	URLOut         string    `json:"url_out"`
	FileNameOut    string    `json:"file_name_out"`
	ContentDataOut string    `json:"content_data_out"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type DetailListJobProcess struct {
	ParentJobData ParentJobProcess               `json:"parent_job_data"`
	Childs        []DetailListJobProcessItemData `json:"childs"`
}

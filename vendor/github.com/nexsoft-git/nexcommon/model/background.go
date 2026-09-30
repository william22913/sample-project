package model

import "database/sql"

type BackgroundServiceModel struct {
	SearchParam   []SearchParam
	IsCheckStatus bool
	CreatedBy     int64
	Data          interface{}
}

type ChildTask struct {
	Group             string
	Type              string
	Name              string
	Data              BackgroundServiceModel
	GetCountData      func(interface{}) (int, error)
	DoJob             func(interface{}, *JobProcessModel) (interface{}, error)
	GetCountParameter interface{}
	DoJobParameter    interface{}
	AdditionalTask    interface{}
}

type JobProcessModel struct {
	ID               sql.NullInt64
	UUIDKey          sql.NullString
	ParentJobID      sql.NullString
	Level            sql.NullInt32
	JobID            sql.NullString
	Group            sql.NullString
	Type             sql.NullString
	Name             sql.NullString
	Counter          sql.NullInt32
	Total            sql.NullInt32
	Status           sql.NullString
	Message          sql.NullString
	AlertId          sql.NullString
	AlertContentData sql.NullString
	Parameter        sql.NullString
	URLIn            sql.NullString
	FileNameIn       sql.NullString
	ContentDataIn    sql.NullString
	URLOut           sql.NullString
	FileNameOut      sql.NullString
	ContentDataOut   sql.NullString
	CreatedBy        sql.NullInt64
	CreatedAt        sql.NullTime
	CreatedClient    sql.NullString
	UpdatedAt        sql.NullTime
	EmptyTotal       bool
}

type JobProcessModelWithChildModel struct {
	ID               sql.NullInt64
	UUIDKey          sql.NullString
	ParentJobData    JobProcessModel
	Level            sql.NullInt32
	JobID            sql.NullString
	Group            sql.NullString
	Type             sql.NullString
	Name             sql.NullString
	Counter          sql.NullInt32
	Total            sql.NullInt32
	Status           sql.NullString
	Message          sql.NullString
	AlertId          sql.NullString
	AlertContentData sql.NullString
	Parameter        sql.NullString
	URLIn            sql.NullString
	FileNameIn       sql.NullString
	ContentDataIn    sql.NullString
	URLOut           sql.NullString
	FileNameOut      sql.NullString
	ContentDataOut   sql.NullString
	CreatedBy        sql.NullInt64
	CreatedAt        sql.NullTime
	CreatedClient    sql.NullString
	UpdatedAt        sql.NullTime
	EmptyTotal       bool
}

type SchedulerChildParameter struct {
	ChildJobModel *JobProcessModel
	Parameter     interface{}
}

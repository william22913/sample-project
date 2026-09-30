package multi_database

import (
	"database/sql"
)

type MultiDatabase struct {
	dbNode                map[string]*sql.DB
	mappingTableName      string
	databaseNameFieldName string
	schemaFieldName       string
	customIDFields        string
	defaultDBKey          string
}

type MultiDatabaseParam struct {
	dbNodeParam  map[string]*sql.DB
	defaultDBKey string
}

type DBMappingNode struct {
	ID           interface{}
	BusinessKey  interface{}
	DatabaseName string
	SchemaName   string
	DB           *sql.DB
}

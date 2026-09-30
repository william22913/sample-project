package multi_database

import (
	"database/sql"
	"errors"
	"fmt"

	internalCtx "github.com/nexsoft-git/nexcommon/context"
)

const defaultCustomIDFields = "id"

func NewMultiDatabaseParam() *MultiDatabaseParam {
	return &MultiDatabaseParam{
		dbNodeParam: make(map[string]*sql.DB),
	}
}

func NewMultiDatabase(
	param *MultiDatabaseParam,
	mappingTableName string, // nexchief_account
	databaseNameFieldName string, // db_name
	schemaFieldName string, // schema `select db_name, schema from nexchief_account where id = $1`
) *MultiDatabase {
	return &MultiDatabase{
		dbNode:                param.dbNodeParam,
		mappingTableName:      mappingTableName,
		databaseNameFieldName: databaseNameFieldName,
		schemaFieldName:       schemaFieldName,
		customIDFields:        defaultCustomIDFields,
		defaultDBKey:          param.defaultDBKey,
	}
}

func (p *MultiDatabaseParam) addDBNode(key string, db *sql.DB) *MultiDatabaseParam {
	if p.dbNodeParam == nil {
		p.dbNodeParam = make(map[string]*sql.DB)
	}

	p.dbNodeParam[key] = db
	return p
}

func (p *MultiDatabaseParam) AddDBNode(key string, db *sql.DB) *MultiDatabaseParam {
	return p.addDBNode(key, db)
}

func (p *MultiDatabaseParam) SetDefaultDBNode(key string) *MultiDatabaseParam {
	p.defaultDBKey = key
	return p
}

func (m *MultiDatabase) GetDBNode(key string) (*sql.DB, bool) {
	db, found := m.dbNode[key]
	return db, found
}

func (m *MultiDatabase) GetDefaultDBNode() (*sql.DB, bool) {
	if m.dbNode == nil {
		return nil, false
	}

	db, found := m.dbNode[m.defaultDBKey]
	return db, found
}

func (m *MultiDatabase) SetCustomIDFields(customIDFields string) *MultiDatabase {
	if customIDFields == "" {
		customIDFields = defaultCustomIDFields
	}

	m.customIDFields = customIDFields
	return m
}

func (m *MultiDatabase) GetDBConnectionByContext(
	ctx *internalCtx.ContextModel,
) (
	*sql.DB,
	error,
) {
	mappingNode, err := m.CheckDBMappingNode(ctx.AuthAccessTokenModel.DBName)
	if err != nil {
		return nil, err
	}

	return mappingNode.DB, nil

}

func (m *MultiDatabase) Close() {
	for _, db := range m.dbNode {
		if db != nil {
			_ = db.Close()
		}
	}
}

func (m *MultiDatabase) CheckDBMappingNode(key interface{}) (*DBMappingNode, error) {
	defaultDB, found := m.GetDefaultDBNode()
	if !found || defaultDB == nil {
		return nil, errors.New("default db node is not connected")
	}

	if key == "" {
		return &DBMappingNode{
			DatabaseName: m.defaultDBKey,
			DB:           defaultDB,
		}, nil
	}

	if m.mappingTableName == "" ||
		m.databaseNameFieldName == "" ||
		m.schemaFieldName == "" {
		return nil, errors.New("mapping fields are not fully configured")
	}

	customIDFields := m.customIDFields
	if customIDFields == "" {
		customIDFields = defaultCustomIDFields
	}

	query := fmt.Sprintf(
		"select p.%s, p.%s from %s p where p.%s = $1",
		m.databaseNameFieldName,
		m.schemaFieldName,
		m.mappingTableName,
		customIDFields,
	)

	var (
		id           interface{}
		businessKey  interface{}
		databaseName string
		schemaName   sql.NullString
	)

	err := defaultDB.QueryRow(query, key).Scan(&databaseName, &schemaName)
	if err != nil {
		return nil, err
	}

	mappedDB, ok := m.dbNode[databaseName]
	if !ok || mappedDB == nil {
		return nil, errors.New("mapped database node is not registered")
	}

	return &DBMappingNode{
		ID:           id,
		BusinessKey:  businessKey,
		DatabaseName: databaseName,
		SchemaName:   schemaName.String,
		DB:           mappedDB,
	}, nil
}

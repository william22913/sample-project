package util

import (
	"context"
	"errors"

	"github.com/olivere/elastic/v7"
)

func NewBulkUpsertDataParam() bulkUpsertDataParam {
	return make(map[string]interface{})
}

type bulkUpsertDataParam map[string]interface{}

func (b bulkUpsertDataParam) AddData(
	id string,
	data interface{},
) bulkUpsertDataParam {
	b[id] = data
	return b
}

func NewElasticSearchConnection(
	param *elasticsearchParam,
) (
	ElasticSearchUtil,
	error,
) {

	var opts []elastic.ClientOptionFunc

	opts = append(opts, elastic.SetURL(param.url...))
	opts = append(opts, elastic.SetSniff(false))

	if param.username != "" {
		opts = append(opts, elastic.SetBasicAuth(param.username, param.password))
	}

	es, err := elastic.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	return &elasticSearchUtil{
		client: es,
	}, nil

}

type ElasticSearchUtil interface {
	GetClient() *elastic.Client

	BulkDeleteData(
		index string,
		ids []string,
	) (
		listErr []ElasticErrorResponse,
		err error,
	)

	BulkUpsertData(
		index string,
		param bulkUpsertDataParam,
	) (
		listErr []ElasticErrorResponse,
		err error,
	)
}

func NewElasticsearchParam(
	url []string,
) *elasticsearchParam {
	return &elasticsearchParam{
		url: url,
	}
}

type elasticsearchParam struct {
	url                []string
	username           string
	password           string
	maxRetries         int
	isSecureConnection bool
}

func (e *elasticsearchParam) Username(username string) *elasticsearchParam {
	e.username = username
	return e
}

func (e *elasticsearchParam) Password(password string) *elasticsearchParam {
	e.password = password
	return e
}

func (e *elasticsearchParam) MaxRetries(max int) *elasticsearchParam {
	e.maxRetries = max
	return e
}

func (e *elasticsearchParam) IsSecureConnection(isSecure bool) *elasticsearchParam {
	e.isSecureConnection = isSecure
	return e
}

type elasticSearchUtil struct {
	client *elastic.Client
}

func (e *elasticSearchUtil) GetClient() *elastic.Client {
	return e.client
}

type ElasticErrorResponse struct {
	ID    string
	Error string
}

func (e *elasticSearchUtil) BulkUpsertData(
	index string,
	param bulkUpsertDataParam,
) (
	listErr []ElasticErrorResponse,
	err error,
) {

	ctx := context.Background()
	bulkRequest := e.client.Bulk()

	for key := range param {
		upsert := elastic.NewBulkUpdateRequest().
			Index(index).
			Id(key).
			Doc(map[string]interface{}{"content": param[key]}).
			DocAsUpsert(true)

		bulkRequest = bulkRequest.Add(upsert)
	}

	res, err := bulkRequest.Do(ctx)

	if err != nil {
		return nil, err
	}

	if res.Errors {
		for _, item := range res.Failed() {
			listErr = append(listErr, ElasticErrorResponse{
				ID:    item.Id,
				Error: item.Error.Reason,
			})
		}

		err = errors.New("bulk upsert data failed")
	}

	return listErr, err
}

func (e *elasticSearchUtil) BulkDeleteData(
	index string,
	ids []string,
) (
	listErr []ElasticErrorResponse,
	err error,
) {

	ctx := context.Background()
	bulkRequest := e.client.Bulk()

	for i := 0; i < len(ids); i++ {
		delete := elastic.NewBulkDeleteRequest().
			Index(index).
			Id(ids[i])
		bulkRequest = bulkRequest.Add(delete)
	}

	res, err := bulkRequest.Do(ctx)

	if err != nil {
		return nil, err
	}

	if res.Errors {
		for _, item := range res.Failed() {
			listErr = append(listErr, ElasticErrorResponse{
				ID:    item.Id,
				Error: item.Error.Reason,
			})
		}

		err = errors.New("bulk upsert data failed")
	}

	return listErr, err
}

// func (e *elasticSearchUtil) buildUpsertSyntax(
// 	b strings.Builder,
// 	index string,
// 	param bulkUpsertDataParam,
// 	docAsUpsert bool,
// ) strings.Builder {
// 	for keys := range param {
// 		values := text.StructToJSON(param[keys])

// 		b.WriteString(fmt.Sprintf(`{ "update": { "_id": "%s", "_index": "%s" } }`, keys, index) + "\n")
// 		b.WriteString(fmt.Sprintf(`{ "doc": %s, "doc_as_upsert": %v }`, values, docAsUpsert) + "\n")
// 	}

// 	return b

// }

// func (e *elasticSearchUtil) buildDeleteSyntax(
// 	b strings.Builder,
// 	index string,
// 	ids []string,
// ) strings.Builder {
// 	for i := 0; i < len(ids); i++ {
// 		b.WriteString(fmt.Sprintf(`{ "delete": { "_id": "%s", "_index": "%s" } }`, ids[i], index) + "\n")
// 	}
// 	return b
// }

package client

import (
	"context"
	"time"

	"github.com/nexsoft-git/nexcommon/model"
)

type APIConnector interface {
	HitAPI(
		ctx context.Context,
		method string,
		host string,
		path string,
		header map[string]string,
		body interface{},
		result AbsHTTPClientDTO,
	) (
		int,
		error,
	)

	HitAPIMultipart(
		ctx context.Context,
		method string,
		host string,
		path string,
		header map[string]string,
		body model.MultipartRequest,
		result AbsHTTPClientDTO,
	) (
		int,
		error,
	)

	HitAPIExternal(
		ctx context.Context,
		method string,
		host string,
		path string,
		header map[string]string,
		body interface{},
		result AbsHTTPClientDTO,
		externalParam ExternalAPIParam,
	) (
		int,
		error,
	)

	EnableExternalActivity(
		resourceID string,
		externalActivityHost string,
		externalActivityPort int,
		fixedToken string,
	)

	DisableApiConnectorLog()

	ParseURL(
		url string,
		urlParam interface{},
	) (
		parsedURL string,
	)

	ParseHeader(
		headerParam interface{},
	) map[string]string

	ParseMultipartStruct(
		multipartParam interface{},
	) model.MultipartRequest
}

type ExternalActivityDTO struct {
	Err            bool              `json:"-"`
	AccountKey     string            `json:"account_key"`
	Source         string            `json:"source"`
	Destination    string            `json:"destination"`
	URL            string            `json:"url"`
	URLParam       map[string]string `json:"url_param"`
	IPAddress      string            `json:"ip_address"`
	MethodRequest  string            `json:"method_request"`
	HeaderRequest  map[string]string `json:"header_request"`
	BodyRequest    interface{}       `json:"body_request"`
	RequestID      string            `json:"request_id"`
	RequestAt      string            `json:"request_at"`
	StatusResponse int               `json:"status_response"`
	HeaderResponse map[string]string `json:"header_response"`
	BodyResponse   string            `json:"body_response"`
	ResponseAt     string            `json:"response_at"`
}

type ExternalActivityDTOOut struct {
	HTTPClientDTO

	Success bool `json:"success"`
	Header  struct {
		RequestID string    `json:"request_id"`
		Version   string    `json:"version"`
		Timestamp time.Time `json:"timestamp"`
	} `json:"header"`
	Payload interface{} `json:"payload"`
}

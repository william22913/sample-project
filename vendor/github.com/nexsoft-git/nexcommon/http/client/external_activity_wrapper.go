package client

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/nexsoft-git/nexcommon/constanta"
	"github.com/nexsoft-git/nexcommon/util/text"
	"github.com/nexsoft-git/nexlogger/log"
)

type ExternalAPIParam struct {
	DestinationName string
	IPAddress       string
}

func (a apiConnector) HitAPIExternal(
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
) {
	if header == nil {
		header = make(map[string]string)
	}

	request_id := text.GetUUID()
	if header[constanta.RequestIDConstanta] != "" {
		request_id = header[constanta.RequestIDConstanta]
	} else {
		header[constanta.RequestIDConstanta] = request_id
	}

	externalDTO := ExternalActivityDTO{
		Source:        a.resourceID,
		Destination:   externalParam.DestinationName,
		IPAddress:     externalParam.IPAddress,
		RequestID:     request_id,
		MethodRequest: method,
		URL:           path,
		HeaderRequest: header,
		BodyRequest:   body,
		RequestAt:     time.Now().Format(constanta.DefaultTimeFormat),
	}

	http_code, err := a.HitAPI(ctx, method, host, path, header, body, result)
	if err != nil {
		log.Error().
			Err(err).
			Caller().
			Msg("Error Happened when do hit api External with Activity")
		return 0, err
	}

	externalDTO.HeaderResponse = getHeadersAsMapString(result.GetHeader())
	externalDTO.StatusResponse = http_code
	externalDTO.ResponseAt = time.Now().Format(constanta.DefaultTimeFormat)

	var interfaceResponse interface{}
	if http_code != 200 {
		interfaceResponse = result.Unsuccessfully()
	} else {
		interfaceResponse = result
	}

	byteData, err := json.Marshal(interfaceResponse)
	if err != nil {
		log.Error().
			Err(err).
			Caller().
			Msg("Error Happened when parsing message body")
		return http_code, err
	}

	externalDTO.BodyResponse = string(byteData)
	externalActivityHeader := make(map[string]string)
	externalActivityHeader[constanta.AuthorizationHeaderConstanta] = a.fixedToken
	externalActivityHeader[constanta.RequestIDConstanta] = request_id
	externalResult := ExternalActivityDTOOut{}
	externalcode, externalErr := a.HitAPI(ctx, http.MethodPost, a.externalActivityPath, a.outgoingPath, externalActivityHeader, &externalDTO, &externalResult)
	if externalErr != nil {
		log.Error().
			Err(externalErr).
			Caller().
			Msg("Error found when hit external Activity")
	}

	if externalcode != 200 {
		log.Info().
			Interface("ext_result", externalResult).
			Caller().
			Msg("Error found when hit external Activity")
	}

	return http_code, nil
}

func getHeadersAsMapString(header http.Header) map[string]string {
	result := make(map[string]string)
	for key, values := range header {
		if len(values) > 0 {
			result[key] = values[0] // use only the first value
		}
	}
	return result
}

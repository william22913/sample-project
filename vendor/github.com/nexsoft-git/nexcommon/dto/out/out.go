package out

type DefaultResponse struct {
	DefaultMessage
	Payload interface{} `json:"payload"`
}

type DefaultErrorResponse struct {
	DefaultMessage
	Payload DefaultError `json:"payload"`
}

type DefaultError struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type DefaultMessage struct {
	Success bool   `json:"success"`
	Header  Header `json:"header"`
}

type Header struct {
	RequestID string `json:"request_id"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
}

type ErrorPayload struct {
	ErrorCode int    `json:"error_code"`
	Message   string `json:"message"`
}

type DefaultResponsePayloadMessage struct {
	Status DefaultResponsePayloadStatus `json:"status"`
	Data   interface{}                  `json:"data"`
}

type DefaultResponsePayloadStatus struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

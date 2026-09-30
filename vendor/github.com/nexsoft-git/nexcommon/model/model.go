package model

type DefaultOperator struct {
	DataType string   `json:"data_type"`
	Operator []string `json:"operator"`
}

type URLParam struct {
	Query map[string]string
	Path  map[string]string
}

type JSONReportValue struct {
	Value string `json:"value"`
	Span  int    `json:"span"`
	Spans Span   `json:"spans"`
}

type Span struct {
	Col int `json:"col"`
	Row int `json:"row"`
}

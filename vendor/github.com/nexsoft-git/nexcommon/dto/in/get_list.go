package in

import "encoding/json"

type GetListRequest struct {
	Page               int             `json:"page"`
	Limit              int             `json:"limit"`
	Order              string          `json:"order"`
	Filter             string          `json:"filter"`
	Connector          string          `json:"connector"`
	OrderIDs           string          `json:"order_ids"`
	OrderIDOperator    string          `json:"order_id_opt"`
	EnableTokenChecker bool            `json:"-"`
	Other              json.RawMessage `json:"other"` //Used only for POST getlist
}

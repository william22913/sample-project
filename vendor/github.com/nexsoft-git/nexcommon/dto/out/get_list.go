package out

import "github.com/nexsoft-git/nexcommon/model"

type InitiateGetListDataDTOOut struct {
	ValidOrderBy  []string                         `json:"valid_order_by"`
	ValidSearchBy []string                         `json:"valid_search_by"`
	ValidLimit    []int                            `json:"valid_limit"`
	ValidOperator map[string]model.DefaultOperator `json:"valid_operator"`
	EnumData      interface{}                      `json:"enum_data"`
	CountData     int                              `json:"count_data"`
}

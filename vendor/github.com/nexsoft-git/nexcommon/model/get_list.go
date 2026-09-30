package model

type SearchParam struct {
	NegationQuery       bool
	SearchKey           string
	DataType            string
	SearchOperator      string
	SearchValue         interface{}
	FormatedSearchValue interface{}
}

package services

import (
	"reflect"

	"github.com/nexsoft-git/nexcommon/dto/out"
	"github.com/nexsoft-git/nexcommon/model"
)

type Services interface {
	GetListScope() []string
	GetDTO() interface{}
	GetMultipartDTO() interface{}
}

type DefaultOperators map[string]model.DefaultOperator

func (d DefaultOperators) GetListValidSearch() []string {
	var result []string

	for key := range d {
		result = append(result, key)
	}

	return result
}

type ServicesWithGetListData interface {
	GetListScope() []string
	GetListValidLimit() []int
	GetListValidOrderBy() []string
	GetListValidSearch() []string
	GetDefaultOperator() DefaultOperators
}

func ToInitiateGetListData(
	d ServicesWithGetListData,
) out.InitiateGetListDataDTOOut {

	st := reflect.TypeOf(d)
	vt := reflect.ValueOf(d)

	m, ok := st.MethodByName("GetListEnum")
	var enum map[string][]string
	if ok {
		val := m.Func.Call([]reflect.Value{vt})
		if len(val) > 0 {
			enum, _ = val[0].Interface().(map[string][]string)
		}
	}

	return out.InitiateGetListDataDTOOut{
		ValidOrderBy:  d.GetListValidOrderBy(),
		ValidSearchBy: d.GetListValidSearch(),
		ValidLimit:    d.GetListValidLimit(),
		ValidOperator: d.GetDefaultOperator(),
		EnumData:      enum,
		CountData:     0,
	}
}

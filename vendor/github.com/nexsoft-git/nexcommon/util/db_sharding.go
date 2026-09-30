package util

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/nexsoft-git/nexcommon/constanta"
	"github.com/nexsoft-git/nexcommon/dao"
	"github.com/nexsoft-git/nexcommon/util/text"
	"github.com/nexsoft-git/nexlogger/log"
)

const TIME_FORMAT_FOR_SHARDING_KEY = "02-01-2006"

type DBShard interface {
	GetListData(
		date RangeDate,
		getListParam *dao.GetListDataParam,
	) (
		[]interface{},
		error,
	)

	CountData(
		date RangeDate,
		getListParam *dao.GetListDataParam,
	) (
		int,
		error,
	)

	GetDBConnByRangeDate(
		rng RangeDate,
	) []*sql.DB

	GetKey() (
		string,
		error,
	)

	OpenKey(
		item string,
	) (
		time.Time,
		error,
	)

	GetDBByKey(
		item string,
	) (
		[]*sql.DB,
		error,
	)

	GetDefaultDBConn() *sql.DB

	Close()
}

type ShardConfig struct {
	Name  string
	Param *dBAddressParam
}

func NewDBShard(
	param []ShardConfig,
	defaultName string,
	strategics ShardingStrategics,
	dao dao.GetListDataDAO,
	shardKey string,
) (
	DBShard,
	error,
) {

	var dbDefault *sql.DB
	var encryptionHelper text.EncryptionHelper
	var err error

	dbList := make(map[string]*sql.DB)

	if defaultName == "" {
		return nil, errors.New("please specify one of your shard db name")
	}

	if shardKey != "" {
		encryptionHelper, err = text.NewEncryptionHelper(shardKey)

		if err != nil {
			return nil, err
		}

		encryptionHelper.SetForURLEncryption()

	}

	for i := 0; i < len(param); i++ {
		dbConn := GetDbConnection(param[i].Param)
		dbList[param[i].Name] = dbConn

		if defaultName == param[i].Name {
			dbDefault = dbConn
		}
	}

	if dbDefault == nil {
		return nil, errors.New("no db name is match to your specified default db")
	}

	return &dbShard{
		dbList:             dbList,
		defaultDB:          dbDefault,
		shardingStrategics: strategics,
		dao:                dao,
		encryptionHelper:   encryptionHelper,
	}, nil

}

type RangeDate struct {
	DateStart time.Time
	DateEnd   time.Time
}

type dbList map[string]*sql.DB
type ShardingStrategics func(RangeDate) []string

type dbShard struct {
	encryptionHelper   text.EncryptionHelper
	dbList             dbList
	defaultDB          *sql.DB
	shardingStrategics ShardingStrategics
	dao                dao.GetListDataDAO
}

func (d *dbShard) GetKey() (
	string,
	error,
) {
	key := fmt.Sprintf("%s:%s", time.Now().Format(TIME_FORMAT_FOR_SHARDING_KEY), text.GetUUID())
	return d.encryptionHelper.EncryptMessage(key)
}

func (d *dbShard) GetDBByKey(
	item string,
) (
	[]*sql.DB,
	error,
) {

	timeNow, err := d.OpenKey(item)

	if err != nil {
		return nil, err
	}

	return d.GetDBConnByRangeDate(
		RangeDate{
			DateStart: timeNow,
			DateEnd:   timeNow,
		},
	), nil
}

func (d *dbShard) OpenKey(
	item string,
) (
	time.Time,
	error,
) {

	key, err := d.encryptionHelper.DecryptMessage(item)

	if err != nil {
		return time.Time{}, errors.New("invalid key")
	}

	items := strings.Split(key, ":")

	if len(items) < 2 {
		return time.Time{}, errors.New("invalid key")
	}

	date, err := time.Parse(TIME_FORMAT_FOR_SHARDING_KEY, items[0])

	if err != nil {
		return time.Time{}, err
	}

	return date, nil
}

func (d *dbShard) GetListData(
	date RangeDate,
	getListParam *dao.GetListDataParam,
) (
	[]interface{},
	error,
) {

	dbNameList := d.shardingStrategics(date)
	var result []interface{}
	var sortBy []sortField
	var err error

	if len(dbNameList) > 1 {
		getListParam.DTO.Limit = -1
		for i := 0; i < len(dbNameList); i++ {

			if dbNameList[i] == "" {
				return nil, errors.New("no database matches your sharding strategics")
			}

			getListParam = getListParam.DB(d.dbList[dbNameList[i]])
			tempResult, err := d.dao.GetListDataWithDefaultMustCheck(getListParam)
			if err != nil {
				return nil, err
			}

			result = append(result, tempResult...)
		}

		orderSplit := strings.Split(getListParam.DTO.Order, ",")
		for i := 0; i < len(orderSplit); i++ {
			splitSpace := strings.Split(strings.Trim(orderSplit[i], " "), " ")

			if len(splitSpace) == 1 {
				sortBy = append(sortBy, sortField{
					field:       splitSpace[0],
					isAscending: true,
				})
			} else {
				sortBy = append(sortBy, sortField{
					field:       splitSpace[0],
					isAscending: strings.ToUpper(splitSpace[len(splitSpace)-1]) == "ASC",
				})
			}
		}

		result = d.combineData(result, sortBy)
	} else {
		getListParam = getListParam.DB(d.dbList[dbNameList[0]])
		result, err = d.dao.GetListDataWithDefaultMustCheck(getListParam)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (d *dbShard) CountData(
	date RangeDate,
	getListParam *dao.GetListDataParam,
) (
	int,
	error,
) {

	dbNameList := d.shardingStrategics(date)
	var result int
	var err error

	if len(dbNameList) > 1 {
		getListParam.DTO.Limit = -1
		for i := 0; i < len(dbNameList); i++ {
			if dbNameList[i] == "" {
				return 0, errors.New("no database matches your sharding strategics")
			}

			getListParam = getListParam.DB(d.dbList[dbNameList[i]])
			tempResult, err := d.dao.GetCountDataWithDefaultMustCheck(getListParam)
			if err != nil {
				return 0, err
			}

			result += tempResult
		}

	} else {
		getListParam = getListParam.DB(d.dbList[dbNameList[0]])
		result, err = d.dao.GetCountDataWithDefaultMustCheck(getListParam)
		if err != nil {
			return 0, err
		}
	}

	return result, nil
}

type sortField struct {
	field       string
	isAscending bool
}

func (d *dbShard) GetDefaultDBConn() *sql.DB {
	return d.defaultDB
}

func (d *dbShard) GetDBConnByRangeDate(
	rng RangeDate,
) []*sql.DB {
	var result []*sql.DB
	dbName := d.shardingStrategics(rng)

	for i := 0; i < len(dbName); i++ {
		db, found := d.dbList[dbName[i]]
		if found {
			result = append(result, db)
		}
	}

	return result
}

func (d *dbShard) Close() {
	for keys := range d.dbList {
		d.dbList[keys].Close()
	}
}

func (d *dbShard) combineData(
	data []interface{},
	sortBy []sortField,
) []interface{} {

	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Caller().
				Interface("panic", r).
				Str("stack", string(debug.Stack())).
				Msg("Panic recovered")
		}
	}()

	sort.Slice(data, func(i, j int) bool {
		itemI := reflect.ValueOf(data[i])
		itemJ := reflect.ValueOf(data[j])
		result := true

		for i := 0; i < len(sortBy); i++ {
			attribute := strings.Title(strings.ReplaceAll(sortBy[i].field, "_", " "))
			field := itemI.FieldByName(attribute)
			field2 := itemJ.FieldByName(attribute)

			switch field.Type().Name() {
			case reflect.String.String(), constanta.StringKind:
				if field.Type().Name() == constanta.StringKind {
					field = reflect.ValueOf(field.Interface().(sql.NullString).String)
					field2 = reflect.ValueOf(field2.Interface().(sql.NullString).String)
				}

				if !sortBy[i].isAscending {
					if field.String() != field2.String() {
						return field.String() > field2.String()
					}
				} else {
					if field.String() != field2.String() {
						return field.String() < field2.String()
					}
				}

			case reflect.Float32.String(), reflect.Float64.String(), constanta.Float64Kind:
				if field.Type().Name() == constanta.Float64Kind {
					field = reflect.ValueOf(field.Interface().(sql.NullFloat64).Float64)
					field2 = reflect.ValueOf(field2.Interface().(sql.NullFloat64).Float64)
				}

				fieldVal := field.Float()
				field2Val := field2.Float()

				if !sortBy[i].isAscending {
					if fieldVal != field2Val {
						return fieldVal > field2Val
					}
				} else {
					if fieldVal != field2Val {
						return fieldVal < field2Val
					}
				}

			case reflect.Bool.String(), constanta.BoolKind:
				if field.Type().Name() == constanta.BoolKind {
					field = reflect.ValueOf(field.Interface().(sql.NullBool).Bool)
					field2 = reflect.ValueOf(field2.Interface().(sql.NullBool).Bool)
				}

				if !sortBy[i].isAscending {
					return field.Bool() && !field2.Bool()
				} else {
					return !field.Bool() && field2.Bool()
				}

			case reflect.Int.String(), reflect.Int16.String(), reflect.Int32.String(), reflect.Int64.String(), constanta.Int64Kind, constanta.Int32Kind, constanta.Int16Kind:
				if field.Type().Name() == constanta.Int16Kind {
					field = reflect.ValueOf(int64(field.Interface().(sql.NullInt16).Int16))
					field2 = reflect.ValueOf(int64(field2.Interface().(sql.NullInt16).Int16))
				} else if field.Type().Name() == constanta.Int32Kind {
					field = reflect.ValueOf(int64(field.Interface().(sql.NullInt32).Int32))
					field2 = reflect.ValueOf(int64(field2.Interface().(sql.NullInt32).Int32))
				} else if field.Type().Name() == constanta.Int64Kind {
					field = reflect.ValueOf(field.Interface().(sql.NullInt64).Int64)
					field2 = reflect.ValueOf(field2.Interface().(sql.NullInt64).Int64)
				}

				fieldVal := field.Int()
				field2Val := field2.Int()

				if !sortBy[i].isAscending {
					if fieldVal != field2Val {
						return fieldVal > field2Val
					}
				} else {
					if fieldVal != field2Val {
						return fieldVal < field2Val
					}
				}

			case constanta.TimeKind, constanta.NullTimeKind:
				if field.Type().Name() == constanta.NullTimeKind {
					field = reflect.ValueOf(field.Interface().(sql.NullTime).Time)
					field2 = reflect.ValueOf(field2.Interface().(sql.NullTime).Time)
				}

				fieldVal := field.Interface().(time.Time)
				field2Val := field2.Interface().(time.Time)

				if !sortBy[i].isAscending {
					if fieldVal.Unix() != field2Val.Unix() {
						return fieldVal.Unix() > field2Val.Unix()
					}
				} else {
					if fieldVal.Unix() != field2Val.Unix() {
						return fieldVal.Unix() < field2Val.Unix()
					}
				}
			}
		}

		return result
	})

	return data
}

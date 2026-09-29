package multipart_validator

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	errors "github.com/nexsoft-git/nexcommon/error"
	"github.com/nexsoft-git/nexcommon/server"
	"github.com/nexsoft-git/nexcommon/util/validator/basic_validator"
	"github.com/nexsoft-git/nexcommon/util/validator/tag_validator"
)

func NewMultipartValidator(
	reader server.MultipartReader,
	basicValidator basic_validator.BasicValidator,
	tagValidator tag_validator.TagValidator,
) MultipartValidator {
	return &multipartValidator{
		reader:         reader,
		basicValidator: basicValidator,
		tagValidator:   tagValidator,
	}
}

type multipartValidator struct {
	reader         server.MultipartReader
	basicValidator basic_validator.BasicValidator
	tagValidator   tag_validator.TagValidator
}

func (m multipartValidator) ValidateMultipart(
	param *validateMultipartParam,
) (
	interface{},
	error,
) {

	reflectType := reflect.TypeOf(param.dto).Elem()
	reflectValue := reflect.ValueOf(param.dto).Elem()

	for i := 0; i < reflectType.NumField(); i++ {

		currentField := reflectType.Field(i)
		currentValue := reflectValue.FieldByName(currentField.Name)
		multipart := currentField.Tag.Get("multipart")
		jsonField := currentField.Tag.Get("json")

		required := currentField.Tag.Get("required")
		requiredArray := strings.Split(required, ",")
		empty := currentField.Tag.Get("empty")

		switch multipart {
		case "file", "file_multiple":
			if !m.basicValidator.ValidateStringContainInStringArray(requiredArray, param.menu) {
				continue
			}

			file, err := m.reader.ReadMultipartFile(
				param.r,
				jsonField,
			)

			if err != nil || len(file) == 0 {
				if empty != "allowed" {
					return nil, errors.ErrEmptyField.Param(jsonField).ContextModel(param.ctx)
				}

				continue
			}

			min, _ := strconv.Atoi(currentField.Tag.Get("min"))
			max, _ := strconv.Atoi(currentField.Tag.Get("max"))
			maxFile, _ := strconv.Atoi(currentField.Tag.Get("max_file"))
			if maxFile == 0 {
				maxFile = 1
			}

			if len(file) > maxFile {
				return nil, errors.ErrFormatFieldRule.Param(jsonField, "NEED_LESS_THAN_EQUAL", strconv.Itoa(maxFile)).ContextModel(param.ctx)
			}

			var totalData int64

			for i := 0; i < len(file); i++ {
				totalData += file[i].Header.Size

				if min > 0 && int(totalData) < min {
					return nil, errors.ErrFormatFieldRule.Param(jsonField, "NEED_MORE_THAN", strconv.Itoa(min)).ContextModel(param.ctx)
				}

				if max > 0 && int(totalData) > max {
					return nil, errors.ErrFormatFieldRule.Param(jsonField, "NEED_LESS_THAN", strconv.Itoa(max)).ContextModel(param.ctx)
				}

				param.ctx.ClientAccess.Logger.ByteIn += int(file[i].Header.Size)

				extensions := strings.Split(currentField.Tag.Get("ext"), ",")
				extensionfile := strings.Split(file[i].Header.Filename, ".")

				if !m.basicValidator.ValidateStringContainInStringArray(
					extensions,
					extensionfile[len(extensionfile)-1],
				) {
					return nil, errors.ErrUnknownData.Param(jsonField).ContextModel(param.ctx)
				}

				if multipart == "file" {
					currentValue.Set(reflect.ValueOf(reflect.ValueOf(file[i]).Interface()))
					continue
				} else {
					currentValue.Set(reflect.ValueOf(reflect.ValueOf(file).Interface()))
				}
			}
		case "json":
			value := param.r.FormValue(jsonField)
			if !m.basicValidator.ValidateStringContainInStringArray(requiredArray, param.menu) {
				continue
			}

			if value == "" && empty == "allowed" {
				continue
			}

			param.ctx.ClientAccess.Logger.ByteIn += len(value)

			if value == "" {
				return nil, errors.ErrEmptyField.Param(jsonField).ContextModel(param.ctx)
			}

			temp := reflect.New(currentField.Type)
			ptr := temp.Interface()
			err := json.Unmarshal([]byte(value), ptr)

			if err != nil {
				return nil, errors.ErrReadBody
			}

			currentValue.Set(reflect.ValueOf(ptr).Elem())
		default:
			value := param.r.FormValue(jsonField)
			param.ctx.ClientAccess.Logger.ByteIn += len(value)

			switch currentValue.Kind() {
			case reflect.String:
				currentValue.SetString(value)
			case reflect.Int:
				if val, err := strconv.Atoi(value); err == nil {
					currentValue.SetInt(int64(val))
				} else {
					return nil, errors.ErrFormatField.Param(jsonField).ContextModel(param.ctx)
				}
			case reflect.Float32:
				if val, err := strconv.ParseFloat(value, 32); err == nil {
					currentValue.SetFloat(val)
				} else {
					return nil, errors.ErrFormatField.Param(jsonField).ContextModel(param.ctx)
				}
			case reflect.Float64:
				if val, err := strconv.ParseFloat(value, 64); err == nil {
					currentValue.SetFloat(val)
				} else {
					return nil, errors.ErrFormatField.Param(jsonField).ContextModel(param.ctx)
				}
			}
		}
	}

	err := m.tagValidator.ValidateByTag(param.ctx, param.dto, param.functionValidator, param.menu)

	if err != nil {
		return nil, err
	}

	return param.dto, nil
}

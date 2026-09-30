package swagger

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/nexsoft-git/nexcommon/util/validator/basic_validator"
	"github.com/nexsoft-git/nexcommon/util/validator/tag_validator"
)

type SwaggerGenerator struct {
	TagValidator   tag_validator.TagValidator
	BasicValidator basic_validator.BasicValidator
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func (s *SwaggerGenerator) ReadDTO(
	isMultipart bool,
	isURLEncoded bool,
	dto interface{},
	menu string,
) SwaggerControllerService {
	return s.readTag(isMultipart, isURLEncoded, dto, menu)
}

func (s *SwaggerGenerator) DiscoverKnownAuthorization() {
	for i := 0; i < len(KnownAuth); i++ {
		SwaggerContainer.SwaggerComponents.SwaggerSecurity[RenameSwaggerSecurityName(KnownAuth[i])] = SwaggerSecurity{
			Type: "apiKey",
			In:   "header",
			Name: "Authorization",
		}
	}
}

func (s *SwaggerGenerator) readTag(
	isMultipart bool,
	isURLEncoded bool,
	dto interface{},
	menu string,
) (
	swaggerService SwaggerControllerService,
) {

	if dto == nil {
		return
	}

	reflectType := reflect.TypeOf(dto).Elem()
	reflectValue := reflect.ValueOf(dto).Elem()

	contentData := make(map[string]interface{})
	contentType := "application/json"
	content := make(map[string]SwaggerContent)

	max := 0
	min := 0
	isMinFound := false
	isMaxFound := false

	for i := 0; i < reflectType.NumField(); i++ {

		contentField := make(map[string]interface{})
		currentField := reflectType.Field(i)
		currentValue := reflectValue.FieldByName(currentField.Name)

		jsonField := currentField.Tag.Get("json")

		if currentField.Type.Kind() == reflect.Struct {
			contentField["type"] = "object"

			newDTO := currentValue.Addr().Interface()
			prop := s.readTag(false, false, newDTO, menu)

			requestBody, valid := prop.RequestBody.(map[string]SwaggerContent)
			if valid {
				for key := range requestBody {
					contentField["properties"] = requestBody[key].Schema.Properties
					break
				}
			}
		}

		autoFix := ""

		if currentField.Type.Kind() == reflect.String {
			autoFix = currentField.Tag.Get("auto_fix")
		}

		reservedValues := currentField.Tag.Get("reserved")

		required := currentField.Tag.Get("required")
		requiredArray := strings.Split(required, ",")
		var allowEmptyValue bool

		empty := currentField.Tag.Get("empty")
		if empty == "allowed" {
			allowEmptyValue = true
		}

		if menu != "" {

			if s.BasicValidator.ValidateStringContainInStringArray(requiredArray, menu) {
				description := ""

				if isMultipart {
					contentType = "multipart/form-data"
				} else if isURLEncoded {
					contentType = "application/x-www-form-urlencoded"
				}

				contentField["allowEmptyValue"] = allowEmptyValue

				if autoFix != "" {
					description += fmt.Sprintf("- auto_fix : %s \n", autoFix)
				}

				if reservedValues != "" {
					description += fmt.Sprintf("- reserved : %v \n", reservedValues)
				}

				defaultValue := currentField.Tag.Get("default")
				if defaultValue != "" {
					description += fmt.Sprintf("- default : %v \n", defaultValue)
				}

				min, isMinFound, max, isMaxFound = s.TagValidator.GetMinMaxValue(currentField)

				if currentValue.Addr().Type().String() == "*in.MultipartFile" ||
					currentValue.Addr().Type().String() == "*in.MultipleMultipartFile" {

					if currentValue.Addr().Type().String() == "*in.MultipartFile" {
						contentField["type"] = "string"
						contentField["format"] = "binary"
					} else {
						contentItem := make(map[string]string)
						contentItem["type"] = "string"
						contentItem["format"] = "binary"
						contentField["type"] = "array"
						contentField["items"] = contentItem
					}

					extension := currentField.Tag.Get("ext")
					maxFileTag := currentField.Tag.Get("max_file")
					maxFile := 1
					if maxFileTag != "" {
						maxFile, _ = strconv.Atoi(maxFileTag)
					}

					description += fmt.Sprintf("- extension : %s \n", extension)

					if isMinFound {
						description += fmt.Sprintf("- min : %v kilobyte(s) \n", float64(min)/1000.0)
					}

					if isMaxFound {
						description += fmt.Sprintf("- max : %v kilobyte(s) \n", float64(max)/1000.0)
					}

					description += fmt.Sprintf("- maxFile : %d file(s) \n", maxFile)

					contentField["description"] = strings.Trim(description, "\n")
					contentData[jsonField] = contentField

					continue
				}

				if !s.validatePrimitiveType(
					contentField, currentField.Type, currentField.Tag, isMinFound, isMaxFound, min, max,
				) {
					if currentValue.Kind() == reflect.Slice {
						contentField["type"] = "array"

						if isMinFound {
							contentField["minLength"] = min
						}

						if isMaxFound {
							contentField["maxLength"] = max
						}

						if currentValue.Len() == 0 {
							dto := currentValue.Type().Elem()
							itemsType := make(map[string]interface{})

							if !s.validatePrimitiveType(itemsType, dto, currentField.Tag, isMinFound, isMaxFound, min, max) {
								newDTO := reflect.New(dto).Interface()
								x := s.readTag(false, false, newDTO, menu)
								itemsType["type"] = "object"

								requestBody, valid := x.RequestBody.(RequestBodyContent)
								if valid {
									itemsType["properties"] = requestBody.Content["application/json"].Schema.Properties
								}
							}

							contentField["items"] = itemsType
						} else {
							for i := 0; i < currentValue.Len(); i++ {
								temp := currentValue.Index(i)

								if temp.Type().String() == reflect.Struct.String() || temp.Type().Kind().String() == reflect.Struct.String() {
									newDTO := currentValue.Index(i).Addr().Interface()
									x := s.readTag(false, false, newDTO, menu)
									contentField["items"] = x
								}

								if temp.Type().Kind() == reflect.String {
									items := make(map[string]interface{})
									items["type"] = "string"

									enumField := currentField.Tag.Get("enum")
									if enumField != "" {

										enum := s.TagValidator.GetEnum(enumField)
										if enum != nil {
											items["enum"] = enum
										}
									}

									contentField["items"] = items
								}
							}
						}
					} else if currentValue.Kind() == reflect.Map {
						// items := map[string]interface{}{"type": "object"}

						if currentField.Type.Elem().Kind() != reflect.Interface {
							additionalProperties := make(map[string]interface{})

							dto := currentValue.Type().Elem()
							newDTO := reflect.New(dto).Interface()
							x := s.readTag(false, false, newDTO, menu)
							additionalProperties["type"] = "object"

							requestBody, valid := x.RequestBody.(RequestBodyContent)
							if valid {
								additionalProperties["properties"] = requestBody.Content["application/json"].Schema.Properties
								contentField["additionalProperties"] = additionalProperties
							}
						}
						contentField["type"] = "object"
					}
				}

				if jsonField == "" {
					jsonField = ToSnakeCase(currentField.Name)
				}

				contentField["description"] = strings.Trim(description, "\n")
				contentData[jsonField] = contentField
			}
		}
	}

	temp := SwaggerContent{}
	temp.Schema.Properties = contentData
	if isURLEncoded || isMultipart {
		temp.Schema.Type = "object"
	}
	content[contentType] = temp
	swaggerService.RequestBody = content

	return
}

func (s *SwaggerGenerator) validatePrimitiveType(
	contentField map[string]interface{},
	currentField reflect.Type,
	tag reflect.StructTag,
	isMinFound, isMaxFound bool,
	min, max int,
) bool {

	if s.BasicValidator.ValidateStringContainInStringArray(tag_validator.IntType, currentField.String()) {
		contentField["type"] = "integer"
		contentField["format"] = currentField.String()

		if isMinFound {
			contentField["minimum"] = min
		}

		if isMaxFound {
			contentField["maximum"] = max
		}

		return true
	} else if s.BasicValidator.ValidateStringContainInStringArray(tag_validator.FloatType, currentField.String()) {
		contentField["type"] = "number"
		contentField["format"] = "float"

		if isMinFound {
			contentField["minimum"] = min
		}

		if isMaxFound {
			contentField["maximum"] = max
		}

		return true
	} else if reflect.String.String() == currentField.String() {
		contentField["type"] = "string"

		if isMinFound {
			contentField["minLength"] = min
		}

		if isMaxFound {
			contentField["maxLength"] = max
		}

		enumField := tag.Get("enum")
		if enumField != "" {

			enum := s.TagValidator.GetEnum(enumField)
			if enum != nil {
				contentField["enum"] = enum
			}
		}

		regexField := tag.Get("regex")
		if regexField != "" {
			if s.TagValidator.GetRegex(regexField).Regex != "" {
				contentField["pattern"] = s.TagValidator.GetRegex(regexField).Regex
			}
		}
		return true

	} else if reflect.Bool.String() == currentField.String() {
		contentField["type"] = "boolean"
		return true
	} else if currentField.String() == "time.Time" {
		contentField["type"] = "string"
		contentField["format"] = "date-time"

		dateFormatTag := tag.Get("dateFormat")

		if dateFormatTag == "" {
			contentField["x-time-format"] = s.TagValidator.GetDateFormat("default")
			return true
		}

		dateSplit := strings.Split(dateFormatTag, ",")
		var validFormat []string

		for i := 0; i < len(dateSplit); i++ {
			if s.TagValidator.GetDateFormat(dateSplit[i]) != "" {
				validFormat = append(validFormat, s.TagValidator.GetDateFormat(dateSplit[i]))
			}
		}

		contentField["x-time-format"] = fmt.Sprintf("%v", validFormat)
		return true
	}

	return false
}

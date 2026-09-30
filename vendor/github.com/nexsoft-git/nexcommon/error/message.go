package error

import "errors"

var ErrUnauthorized = NewUnBundledErrorMessages(401, errors.New("E-1-CMD-AUT-001"), nil)
var ErrExpiredToken = NewUnBundledErrorMessages(401, errors.New("E-1-CMD-AUT-002"), nil)
var ErrJWTInvalidMethod = NewUnBundledErrorMessages(401, errors.New("E-1-CMD-AUT-003"), nil)
var ErrUserNonActive = NewUnBundledErrorMessages(401, errors.New("E-1-CMD-AUT-004"), nil)
var ErrPermissionUpdated = NewUnBundledErrorMessages(401, errors.New("E-1-CMD-AUT-005"), nil)

var ErrEmptyField = NewUnBundledErrorMessages(400, errors.New("E-4-CMD-DTO-001"), errFieldNameConverter)
var ErrFormatField = NewUnBundledErrorMessages(400, errors.New("E-4-CMD-DTO-002"), errFieldNameConverter)
var ErrFormatFieldRule = NewUnBundledErrorMessages(400, errors.New("E-4-CMD-DTO-003"), errFieldRuleConverter)
var ErrUnknownData = NewUnBundledErrorMessages(400, errors.New("E-4-CMD-DTO-004"), errFieldNameConverter)
var ErrDataAlreadyUsed = NewUnBundledErrorMessages(400, errors.New("E-4-CMD-DTO-005"), errFieldNameConverter)
var ErrReservedValueString = NewUnBundledErrorMessages(400, errors.New("E-4-CMD-DTO-006"), errFieldNameConverter)

var ErrDataLocked = NewUnBundledErrorMessages(400, errors.New("E-4-CMD-DTO-007"), errFieldNameConverter)
var ErrJWTForbiddenByResource = NewUnBundledErrorMessages(403, errors.New("E-3-CMD-AUT-001"), nil)
var ErrJWTForbiddenByPermission = NewUnBundledErrorMessages(403, errors.New("E-3-CMD-AUT-002"), nil)
var ErrJWTForbiddenByScope = NewUnBundledErrorMessages(403, errors.New("E-3-CMD-AUT-003"), nil)
var ErrInvalidIPAddress = NewUnBundledErrorMessages(403, errors.New("E-3-CMD-AUT-004"), nil)
var ErrInvalidURLWhitelist = NewUnBundledErrorMessages(403, errors.New("E-3-CMD-AUT-005"), nil)

var ErrReadBody = NewUnBundledErrorMessages(400, errors.New("E-4-CMD-BDY-001"), nil)
var ErrMarshalingBody = NewUnBundledErrorMessages(400, errors.New("E-4-CMD-BDY-002"), nil)
var ErrInvalidSignature = NewUnBundledErrorMessages(400, errors.New("E-4-CMD-BDY-003"), nil)
var ErrAuditIsDisabled = NewUnBundledErrorMessages(404, errors.New("E-4-CMD-ADT-001"), nil)

var ErrUnimplemted = NewUnBundledErrorMessages(404, errors.New("E-4-CMD-SRV-001"), nil)

var errFieldNameConverter = func(value ...interface{}) map[string]ErrorParam {
	result := make(map[string]ErrorParam)
	result["FieldName"] = ErrorParam{value[0], true}
	return result
}

var errFieldRuleConverter = func(value ...interface{}) map[string]ErrorParam {
	result := make(map[string]ErrorParam)
	result["FieldName"] = ErrorParam{value[0], true}
	result["RuleName"] = ErrorParam{value[1], true}
	result["Other"] = ErrorParam{value[2], false}
	return result
}

package controller

import "context"

const FIXED_TOKEN_VALIDATOR = "${FIXED_TOKEN}"
const USER_TOKEN_VALIDATOR = "${USER_TOKEN}"
const INTERNAL_TOKEN_VALIDATOR = "${INTERNAL_TOKEN}"
const EXTERNAL_TOKEN_VALIDATOR = "${EXTERNAL_TOKEN}"
const WHITELIST_TOKEN_VALIDATOR = "${WHITELIST}"

type ServerAccessValidatorFunction func(ctx context.Context, header map[string]string) error

type ServerAccessValidator struct {
	Func ServerAccessValidatorFunction
	Name string
}
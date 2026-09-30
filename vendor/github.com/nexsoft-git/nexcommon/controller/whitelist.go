package controller

import (
	"context"
	"strings"

	"github.com/nexsoft-git/nexcommon/constanta"
	errors "github.com/nexsoft-git/nexcommon/error"
)

func (cv ControllerValidator) WhitelistValidator() ServerAccessValidator {
	return ServerAccessValidator{
		Name: WHITELIST_TOKEN_VALIDATOR,
		Func: func(ctx context.Context, header map[string]string) error {
			return nil
		},
	}
}

func (cv ControllerValidator) FixedTokenValidator() ServerAccessValidator {
	return ServerAccessValidator{
		Name: FIXED_TOKEN_VALIDATOR,
		Func: func(ctx context.Context, header map[string]string) error {

			err := cv.readAuthorization(header)

			if err != nil {
				return err
			}

			if header[constanta.AuthorizationHeaderConstanta] != cv.fixedToken {
				return errors.ErrUnauthorized
			}

			return nil
		},
	}

}

func (cv ControllerValidator) readAuthorization(header map[string]string) error {
	if header[constanta.AuthorizationHeaderConstanta] == "" {
		return errors.ErrUnauthorized
	}

	header[constanta.AuthorizationHeaderConstanta] = strings.ReplaceAll(header[constanta.AuthorizationHeaderConstanta], "Bearer ", "")

	return nil
}

func (cv ControllerValidator) FixedTokenValidatorWithKong() ServerAccessValidator {
	return ServerAccessValidator{
		Name: FIXED_TOKEN_VALIDATOR,
		Func: func(ctx context.Context, header map[string]string) error {
			result := header[constanta.NexAuthResultKey]

			if result != constanta.NexAuthFixedTokenResult {
				return errors.ErrUnauthorized
			}

			return nil
		},
	}

}

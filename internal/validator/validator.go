package validator

import (
	"errors"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

type ErrorResponse struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func NewValidator() (*validator.Validate, ut.Translator, error) {
	v := validator.New()

	english := en.New()
	uni := ut.New(english, english)
	trans, found := uni.GetTranslator("en")
	if !found {
		return nil, nil, errors.New("failed to get english translator")
	}

	if err := en_translations.RegisterDefaultTranslations(v, trans); err != nil {
		return nil, nil, errors.New("failed to register default translations: " + err.Error())
	}
	if err := v.RegisterValidation("eth_addr", func(fl validator.FieldLevel) bool {
		field := fl.Field()

		if field.Kind() == reflect.String {
			return common.IsHexAddress(field.String())
		}

		return false
	}); err != nil {
		return nil, nil, errors.New("failed to register eth_addr translation: " + err.Error())
	}

	if err := v.RegisterValidation("uint_gt_zero", func(fl validator.FieldLevel) bool {
		field := fl.Field()
		if field.Kind() != reflect.String {
			return false
		}

		raw := strings.TrimSpace(field.String())
		if raw == "" {
			return false
		}

		n := new(big.Int)
		if _, ok := n.SetString(raw, 10); !ok {
			return false
		}

		return n.Sign() > 0
	}); err != nil {
		return nil, nil, errors.New("failed to register uint_gt_zero translation: " + err.Error())
	}

	if err := v.RegisterValidation("uint_gte_zero", func(fl validator.FieldLevel) bool {
		field := fl.Field()
		if field.Kind() != reflect.String {
			return false
		}

		raw := strings.TrimSpace(field.String())
		if raw == "" {
			return false
		}

		n := new(big.Int)
		if _, ok := n.SetString(raw, 10); !ok {
			return false
		}

		return n.Sign() >= 0
	}); err != nil {
		return nil, nil, errors.New("failed to register uint_gte_zero translation: " + err.Error())
	}

	if err := v.RegisterTranslation("required", trans, func(ut ut.Translator) error {
		return ut.Add("required", "{0} is a required field", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("required", fe.Field())
		return t
	}); err != nil {
		return nil, nil, errors.New("failed to register required translation: " + err.Error())
	}

	if err := v.RegisterTranslation("email", trans, func(ut ut.Translator) error {
		return ut.Add("email", "{0} must be a valid email address", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("email", fe.Field())
		return t
	}); err != nil {
		return nil, nil, errors.New("failed to register email translation: " + err.Error())
	}

	if err := v.RegisterTranslation("min", trans, func(ut ut.Translator) error {
		return ut.Add("min", "{0} must be at least {1} characters long", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("min", fe.Field(), fe.Param())
		return t
	}); err != nil {
		return nil, nil, errors.New("failed to register min translation: " + err.Error())
	}

	if err := v.RegisterTranslation("eth_addr", trans, func(ut ut.Translator) error {
		return ut.Add("eth_addr", "{0} must be a valid evm address", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("eth_addr", fe.Field())
		return t
	}); err != nil {
		return nil, nil, errors.New("failed to register eth_addr translation: " + err.Error())
	}

	if err := v.RegisterTranslation("uint_gt_zero", trans, func(ut ut.Translator) error {
		return ut.Add("uint_gt_zero", "{0} must be greater than 0", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("uint_gt_zero", fe.Field())
		return t
	}); err != nil {
		return nil, nil, errors.New("failed to register uint_gt_zero translation: " + err.Error())
	}

	if err := v.RegisterTranslation("uint_gte_zero", trans, func(ut ut.Translator) error {
		return ut.Add("uint_gte_zero", "{0} must be greater than or equal to 0", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("uint_gte_zero", fe.Field())
		return t
	}); err != nil {
		return nil, nil, errors.New("failed to register uint_gte_zero translation: " + err.Error())
	}

	return v, trans, nil
}

func TranslateError(err error, trans ut.Translator) []ErrorResponse {
	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		var errs []ErrorResponse

		for _, e := range validationErrs {
			errs = append(errs, ErrorResponse{
				Field:   e.Field(),
				Message: e.Translate(trans),
			})
		}

		return errs
	}

	return []ErrorResponse{{
		Field:   "unknown",
		Message: "An unknown error occurred",
	}}
}

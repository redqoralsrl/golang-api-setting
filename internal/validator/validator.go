package validator

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"slices"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

type ErrorResponse struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// FieldLevel Rule.Fn 이 받는 값. 다른 패키지가 go-playground 를 직접 import 하지 않게 별칭을 둔다.
type FieldLevel = validator.FieldLevel

// Rule 커스텀 validate 태그. Message 의 {0} 은 필드 이름으로 바뀐다.
//
//	validator.Rule{Tag: "eth_addr", Message: "{0} must be a valid evm address", Fn: ...}
type Rule struct {
	Tag     string
	Message string
	Fn      func(fl FieldLevel) bool
}

// baseRules 모든 프로젝트에서 쓰는 공용 규칙. 외부 라이브러리 의존이 없는 것만 둔다.
// 도메인 특화 규칙(예: 블록체인 주소)은 해당 패키지에서 Rule 로 만들어 NewValidationService 에 넘긴다.
var baseRules = []Rule{
	{Tag: "uint_gt_zero", Message: "{0} must be greater than 0", Fn: bigIntSign(func(sign int) bool { return sign > 0 })},
	{Tag: "uint_gte_zero", Message: "{0} must be greater than or equal to 0", Fn: bigIntSign(func(sign int) bool { return sign >= 0 })},
}

func NewValidator(extra ...Rule) (*validator.Validate, ut.Translator, error) {
	v := validator.New(validator.WithRequiredStructEnabled())

	// 에러의 field 를 Go 필드명 대신 json 태그 이름으로 돌려준다 (Email → email)
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" {
			return ""
		}
		if name == "" {
			return f.Name
		}
		return name
	})

	english := en.New()
	trans, found := ut.New(english, english).GetTranslator("en")
	if !found {
		return nil, nil, errors.New("failed to get english translator")
	}
	if err := en_translations.RegisterDefaultTranslations(v, trans); err != nil {
		return nil, nil, fmt.Errorf("register default translations: %w", err)
	}

	for _, r := range slices.Concat(baseRules, extra) {
		if err := register(v, trans, r); err != nil {
			return nil, nil, err
		}
	}
	return v, trans, nil
}

func register(v *validator.Validate, trans ut.Translator, r Rule) error {
	if err := v.RegisterValidation(r.Tag, r.Fn); err != nil {
		return fmt.Errorf("register %s validation: %w", r.Tag, err)
	}
	err := v.RegisterTranslation(r.Tag, trans, func(t ut.Translator) error {
		return t.Add(r.Tag, r.Message, true)
	}, func(t ut.Translator, fe validator.FieldError) string {
		msg, _ := t.T(r.Tag, fe.Field())
		return msg
	})
	if err != nil {
		return fmt.Errorf("register %s translation: %w", r.Tag, err)
	}
	return nil
}

// bigIntSign 10진수 문자열을 big.Int 로 읽어 부호를 검사한다 (토큰 수량처럼 int64 를 넘는 값)
func bigIntSign(ok func(sign int) bool) validator.Func {
	return func(fl validator.FieldLevel) bool {
		if fl.Field().Kind() != reflect.String {
			return false
		}
		n, valid := new(big.Int).SetString(strings.TrimSpace(fl.Field().String()), 10)
		return valid && ok(n.Sign())
	}
}

func TranslateError(err error, trans ut.Translator) []ErrorResponse {
	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
		return []ErrorResponse{{Field: "unknown", Message: "An unknown error occurred"}}
	}

	errs := make([]ErrorResponse, 0, len(validationErrs))
	for _, e := range validationErrs {
		errs = append(errs, ErrorResponse{Field: e.Field(), Message: e.Translate(trans)})
	}
	return errs
}

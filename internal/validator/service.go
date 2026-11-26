package validator

import (
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

type ValidationService interface {
	Validate(any) []ErrorResponse
}

type validationService struct {
	validator *validator.Validate
	trans     ut.Translator
}

// NewValidationService extra 로 선택 모듈의 규칙을 더한다.
//
//	validator.NewValidationService(blockchain.ValidationRules()...)
func NewValidationService(extra ...Rule) (ValidationService, error) {
	v, trans, err := NewValidator(extra...)
	if err != nil {
		return nil, err
	}
	return &validationService{validator: v, trans: trans}, nil
}

func (s *validationService) Validate(data any) []ErrorResponse {
	if err := s.validator.Struct(data); err != nil {
		return TranslateError(err, s.trans)
	}
	return nil
}

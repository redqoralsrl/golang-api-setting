package validator

import (
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

type ValidationService interface {
	Validate(interface{}) []ErrorResponse
}

type validationService struct {
	validator *validator.Validate
	trans     ut.Translator
}

func NewValidationService() (ValidationService, error) {
	v, trans, err := NewValidator()
	if err != nil {
		return nil, err
	}
	return &validationService{
		validator: v,
		trans:     trans,
	}, nil
}

func (s *validationService) Validate(data interface{}) []ErrorResponse {
	if err := s.validator.Struct(data); err != nil {
		return TranslateError(err, s.trans)
	}
	return nil
}

package validator

import (
	"reflect"
	"testing"

	"github.com/go-playground/validator/v10"
)

type sample struct {
	Email  string `json:"email" validate:"required,email"`
	Amount string `json:"amount" validate:"uint_gt_zero"`
	Code   string `json:"code" validate:"is_abc"`
}

func TestValidateReportsJSONFieldNames(t *testing.T) {
	isABC := Rule{Tag: "is_abc", Message: "{0} must be abc", Fn: func(fl validator.FieldLevel) bool {
		return fl.Field().Kind() == reflect.String && fl.Field().String() == "abc"
	}}
	s, err := NewValidationService(isABC)
	if err != nil {
		t.Fatal(err)
	}

	if errs := s.Validate(sample{Email: "a@b.co", Amount: "10", Code: "abc"}); errs != nil {
		t.Fatalf("expected valid, got %v", errs)
	}

	errs := s.Validate(sample{Email: "nope", Amount: "0", Code: "x"})
	got := map[string]string{}
	for _, e := range errs {
		got[e.Field] = e.Message
	}
	want := map[string]string{
		"email":  "email must be a valid email address",
		"amount": "amount must be greater than 0",
		"code":   "code must be abc",
	}
	for field, msg := range want {
		if got[field] != msg {
			t.Errorf("%s: got %q, want %q", field, got[field], msg)
		}
	}
}

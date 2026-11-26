package blockchain

import (
	"reflect"

	"go-template/internal/validator"

	"github.com/ethereum/go-ethereum/common"
)

// ValidationRules 블록체인용 validate 태그. 쓰는 프로젝트만 main.go 에서 켠다.
//
//	v, err := validator.NewValidationService(blockchain.ValidationRules()...)
//
// 블록체인을 안 쓰는 프로젝트는 이 줄이 없으니 go-ethereum 이 빌드에 들어가지 않는다.
func ValidationRules() []validator.Rule {
	return []validator.Rule{
		{
			Tag:     "eth_addr",
			Message: "{0} must be a valid evm address",
			Fn: func(fl validator.FieldLevel) bool {
				return fl.Field().Kind() == reflect.String && common.IsHexAddress(fl.Field().String())
			},
		},
	}
}

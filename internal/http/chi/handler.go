package chi

import (
	"encoding/json"
	"errors"
	"net/http"

	"go-template/domain/errorlog"
	"go-template/domain/user"
	response "go-template/internal/http"
	"go-template/internal/validator"

	"github.com/go-chi/chi/v5"
)

type Services struct {
	UserService     user.UseCase
	ErrorLogService errorlog.UseCase
}

// Routes 핸들러가 라우트를 등록할 그룹. body 한도가 그룹마다 다르다.
// 한도는 바깥 그룹에서 걸면 안쪽에서 늘릴 수 없으므로 그룹(라우트)마다 따로 건다.
type Routes struct {
	Public    chi.Router // /api/v1        인증 없음, 1MB
	App       chi.Router // /api/v1/app    사용자 JWT, 1MB
	AppUpload chi.Router // /api/v1/app    사용자 JWT, 파일 업로드용 20MB
	Admin     chi.Router // /admin/v1      예약. 관리자 인증 미들웨어를 붙이기 전에는 라우트를 등록하지 않는다
}

// Handler 각 도메인 핸들러가 라우트를 등록할 때 받는 공통 의존성
type Handler struct {
	Routes Routes
	v      validator.ValidationService
}

// bindJSON JSON 바디를 dst 로 읽고 validate 태그를 검사한다. 실패하면 응답을 쓰고 false 를 돌려준다.
//
//	var req CreateUserRequest
//	if !bindJSON(w, r, h.validator, &req) {
//		return
//	}
func bindJSON(w http.ResponseWriter, r *http.Request, v validator.ValidationService, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		if maxErr := (*http.MaxBytesError)(nil); errors.As(err, &maxErr) {
			response.WriteError(w, http.StatusRequestEntityTooLarge, response.PayloadTooLarge, errors.New("request body too large"))
			return false
		}
		response.WriteError(w, http.StatusBadRequest, response.InvalidInput, errors.New("invalid JSON body"))
		return false
	}
	if errs := v.Validate(dst); errs != nil {
		response.WriteValidationError(w, errs)
		return false
	}
	return true
}

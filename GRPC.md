# gRPC 적용 가이드

이 문서는 현재 `user` 도메인에 적용된 gRPC 구조를 기준으로, 다른 도메인도 같은 방식으로 추가하는 순서를 정리한다.

## 현재 구조

gRPC는 HTTP와 동일하게 외부 요청을 받는 어댑터 계층이다.

```text
gRPC Client
  -> proto generated server interface
  -> internal/grpc/<domain>/server.go
  -> domain/<domain>.UseCase
  -> domain/<domain>/postgresql.Repository
  -> PostgreSQL
```

핵심 원칙:

- gRPC handler에는 비즈니스 로직을 넣지 않는다.
- gRPC handler는 `domain/<domain>.UseCase`만 호출한다.
- DB 접근은 기존 repository/sqlc 구조를 그대로 사용한다.
- HTTP handler와 gRPC handler는 같은 service를 공유한다.

## 1. proto 파일 작성

도메인별 proto 파일을 추가한다.

예시:

```text
proto/user/v1/user.proto
```

```proto
syntax = "proto3";

package user.v1;

option go_package = "go-template/proto/user/v1;userv1";

service UserService {
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
}

message CreateUserRequest {
  string email = 1;
  string password = 2;
}

message CreateUserResponse {
  int64 id = 1;
  string email = 2;
  string access_token = 3;
}
```

주의:

- `package user.v1`은 gRPC method path에 사용된다.
- `service UserService`는 Go에서 `RegisterUserServiceServer`로 생성된다.
- `go_package`는 실제 import 경로와 맞아야 한다.
- 현재 코드에서는 `go-template/proto/user/v1`로 import하고 있으므로 `go_package`도 이 경로와 맞추는 편이 단순하다.

## 2. proto 코드 생성

`Makefile`의 `proto` 명령을 사용한다.

```bash
make proto
```

현재 명령:

```makefile
proto: ## Generate protobuf files
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/user/v1/user.proto
```

생성 결과:

```text
proto/user/v1/user.pb.go
proto/user/v1/user_grpc.pb.go
```

새 도메인을 추가하면 `Makefile`의 `proto` 명령에 proto 파일을 추가한다.

예:

```makefile
proto: ## Generate protobuf files
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/user/v1/user.proto \
		proto/order/v1/order.proto
```

## 3. gRPC server 구현

도메인별 gRPC handler를 만든다.

예시:

```text
internal/grpc/user/server.go
```

```go
package usergrpc

import (
	"context"

	"go-template/domain/user"
	userv1 "go-template/proto/user/v1"
)

type Server struct {
	userv1.UnimplementedUserServiceServer
	userService user.UseCase
}

func NewServer(userService user.UseCase) *Server {
	return &Server{userService: userService}
}

func (s *Server) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	createdUser, err := s.userService.Create(ctx, req.Email, req.Password)
	if err != nil {
		return nil, err
	}

	return &userv1.CreateUserResponse{
		Id:          int64(createdUser.ID),
		Email:       createdUser.Email,
		AccessToken: createdUser.AccessToken,
	}, nil
}
```

작성 규칙:

- request DTO를 domain service 입력값으로 변환한다.
- domain model을 response DTO로 변환한다.
- DB transaction, bcrypt, JWT 발급 같은 로직은 service에 둔다.
- request validation은 최소한 gRPC handler 초입에서 처리한다.

예:

```go
if req.Email == "" {
	return nil, status.Error(codes.InvalidArgument, "email is required")
}
```

## 4. gRPC 서버 실행부 연결

gRPC 서버 실행 함수는 다음 위치에 둔다.

```text
internal/api/grpc.go
```

현재 구조:

```go
func StartGRPC(l logger.Logger, port string, register func(*grpc.Server)) error {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	server := grpc.NewServer()
	register(server)

	l.Info(fmt.Sprintf("gRPC listening on port %s", port))

	return server.Serve(listener)
}
```

이 함수는 gRPC 서버 생성과 listen만 담당한다.

서비스 등록은 `cmd/api/main.go`에서 처리한다.

## 5. main.go에 서비스 등록

`cmd/api/main.go`에서 기존 HTTP service와 같은 instance를 gRPC에도 연결한다.

예:

```go
go func() {
	err := api.StartGRPC(l, conf.GRPCPort, func(grpcServer *grpc.Server) {
		userv1.RegisterUserServiceServer(
			grpcServer,
			usergrpc.NewServer(userService),
		)
	})
	if err != nil {
		l.Fatal("Failed to start gRPC server", logger.Field{
			Key:   "error",
			Value: err.Error(),
		})
	}
}()
```

새 도메인을 추가하면 같은 register callback 안에 추가한다.

예:

```go
err := api.StartGRPC(l, conf.GRPCPort, func(grpcServer *grpc.Server) {
	userv1.RegisterUserServiceServer(
		grpcServer,
		usergrpc.NewServer(userService),
	)

	orderv1.RegisterOrderServiceServer(
		grpcServer,
		ordergrpc.NewServer(orderService),
	)
})
```

## 6. config와 docker port 확인

gRPC port는 env로 관리한다.

```text
.env.local
.env.template
```

예:

```env
GRPC_PORT=8888
```

Docker Compose도 같은 포트를 열어야 한다.

```yaml
ports:
  - "8080:8080"
  - "8888:8888"
```

## 7. grpcurl로 호출 테스트

서버 실행 후 `grpcurl`로 테스트한다.

```bash
grpcurl \
  -plaintext \
  -import-path . \
  -proto proto/user/v1/user.proto \
  -d '{"email":"test@example.com","password":"password123"}' \
  localhost:8888 \
  user.v1.UserService/CreateUser
```

정상 응답 예:

```json
{
  "id": "1",
  "email": "test@example.com",
  "accessToken": "..."
}
```

## 8. 보안 적용 순서

처음부터 TLS, JWT, role, rate limit을 한 번에 넣지 말고 순서대로 적용한다.

권장 순서:

1. gRPC unary interceptor 구조 추가
2. panic recovery interceptor 추가
3. request logging interceptor 추가
4. JWT auth interceptor 추가
5. public method allowlist 추가
6. role 기반 권한 체크 추가
7. 운영 환경 TLS 적용

예상 위치:

```text
internal/grpc/interceptor/recovery.go
internal/grpc/interceptor/logger.go
internal/grpc/interceptor/auth.go
```

`CreateUser` 같은 회원가입 RPC는 인증 없이 접근 가능해야 하므로 allowlist에 넣는다.

예:

```go
var publicMethods = map[string]bool{
	userv1.UserService_CreateUser_FullMethodName: true,
}
```

JWT가 필요한 RPC는 metadata에서 token을 읽는다.

```text
authorization: Bearer <access_token>
```

`grpcurl` 예:

```bash
grpcurl \
  -plaintext \
  -H "authorization: Bearer ${ACCESS_TOKEN}" \
  localhost:8888 \
  user.v1.UserService/GetMe
```

## 9. 새 도메인 추가 체크리스트

예를 들어 `order` 도메인을 gRPC로 추가할 때:

- `proto/order/v1/order.proto` 작성
- `Makefile`의 `proto` 명령에 `proto/order/v1/order.proto` 추가
- `make proto` 실행
- `internal/grpc/order/server.go` 작성
- `ordergrpc.NewServer(orderService)` 생성자 작성
- `cmd/api/main.go`에서 `orderv1.RegisterOrderServiceServer` 등록
- `grpcurl`로 RPC 호출 확인
- 필요한 경우 auth allowlist 또는 role rule 추가

## 10. 설계 기준

gRPC를 추가해도 Clean Architecture 의존성 방향은 유지한다.

허용:

```text
internal/grpc/user -> domain/user
```

금지:

```text
domain/user -> internal/grpc/user
domain/user -> proto/user/v1
```

domain 계층은 gRPC, HTTP, protobuf를 몰라야 한다.


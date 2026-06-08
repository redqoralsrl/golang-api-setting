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

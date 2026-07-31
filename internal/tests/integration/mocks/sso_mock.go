package mocks

import (
	"context"

	authv1 "github.com/Mas4trt/protos/gen/go/auth/v1"
)

type MockSSO struct {
	authv1.UnimplementedAuthServer
}

func (m *MockSSO) Register(
	ctx context.Context,
	req *authv1.RegisterRequest,
) (*authv1.RegisterResponse, error) {
	return &authv1.RegisterResponse{
		UserId: 1,
	}, nil
}

func (m *MockSSO) Authenticate(
	ctx context.Context,
	req *authv1.LoginRequest,
) (*authv1.LoginResponse, error) {
	return &authv1.LoginResponse{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
	}, nil
}

func (m *MockSSO) RefreshTokens(
	ctx context.Context,
	req *authv1.RefreshTokensRequest,
) (*authv1.LoginResponse, error) {
	return &authv1.LoginResponse{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
	}, nil
}

func (m *MockSSO) Logout(
	ctx context.Context,
	req *authv1.LogoutRequest,
) (*authv1.LogoutResponse, error) {
	return &authv1.LogoutResponse{
		Success: true,
	}, nil
}

func (m *MockSSO) GetRole(
	ctx context.Context,
	req *authv1.GetRoleRequest,
) (*authv1.GetRoleResponse, error) {
	return &authv1.GetRoleResponse{
		Role: authv1.Role_ROLE_USER,
	}, nil
}

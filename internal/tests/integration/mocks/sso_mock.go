package mocks

import (
	"context"
	"time"

	authv1 "github.com/Mas4trt/protos/gen/go/auth/v1"

	"github.com/golang-jwt/jwt/v5"
)

// mockAccessTokenTTL — срок жизни токена, который выписывает мок.
// Значение не принципиально (тесты короткие), важно лишь что токен
// валиден на момент проверки.
const mockAccessTokenTTL = time.Hour

// MockSSO подменяет реальный sso в интеграционных тестах. Authenticate и
// RefreshTokens подписывают настоящий HS256-токен тем же секретом и
// application_id, что использует authn.Verifier под тестом — иначе мок
// имитирует только форму ответа, а не контракт: любой тест, доходящий до
// /auth/login -> /url по полному флоу, получил бы 401 от верификатора на
// строке-заглушке вместо токена.
type MockSSO struct {
	authv1.UnimplementedAuthServer

	// Secret и ApplicationID должны совпадать с тем, чем сконфигурирован
	// authn.Verifier в тестируемом приложении (см. testAppSecret и
	// config.SSOConfig.ApplicationID в containers_test.go).
	Secret        string
	ApplicationID uint64
}

func (m *MockSSO) issueAccessToken(userID uint64, email string) (string, error) {
	claims := jwt.MapClaims{
		"app_id": float64(m.ApplicationID),
		"uid":    float64(userID),
		"email":  email,
		"exp":    time.Now().Add(mockAccessTokenTTL).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.Secret))
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
	access, err := m.issueAccessToken(1, req.GetEmail())
	if err != nil {
		return nil, err
	}

	return &authv1.LoginResponse{
		AccessToken:  access,
		RefreshToken: "test-refresh-token",
	}, nil
}

func (m *MockSSO) RefreshTokens(
	ctx context.Context,
	req *authv1.RefreshTokensRequest,
) (*authv1.LoginResponse, error) {
	access, err := m.issueAccessToken(1, "integration-test@example.com")
	if err != nil {
		return nil, err
	}

	return &authv1.LoginResponse{
		AccessToken:  access,
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

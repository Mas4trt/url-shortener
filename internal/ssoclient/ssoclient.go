package ssoclient

import (
	"context"
	"errors"
	"fmt"
	"time"

	authv1 "github.com/Mas4trt/protos/gen/go/auth/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidCredentials = errors.New("sso: invalid email or password")
	ErrUserExists         = errors.New("sso: user already exists")
	ErrRefreshInvalid     = errors.New("sso: refresh token invalid or expired")
)

type Options struct {
	// Addr is host:port of the sso gRPC server, e.g. "sso:44044".
	Addr string
	// ApplicationID is this app's row in sso's `apps` table.
	ApplicationID uint64
	// DialTimeout bounds the initial connection attempt.
	DialTimeout time.Duration
	// TLS, if nil, connects insecurely — fine on a private network/service
	// mesh, not fine over the public internet.
	TLS credentials.TransportCredentials
}

type Client struct {
	conn  *grpc.ClientConn
	api   authv1.AuthClient
	appID uint64
}

func New(ctx context.Context, opts Options) (*Client, error) {
	creds := opts.TLS

	if creds == nil {
		creds = insecure.NewCredentials()
	}

	conn, err := grpc.NewClient(
		opts.Addr,
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true,
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("ssoclient: create client: %w", err)
	}

	return &Client{
		conn:  conn,
		api:   authv1.NewAuthClient(conn),
		appID: opts.ApplicationID,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Register(ctx context.Context, email, password string) (userID uint64, err error) {
	resp, err := c.api.Register(ctx, &authv1.RegisterRequest{Email: email, Password: password})
	if err != nil {
		return 0, mapErr(err)
	}
	return resp.GetUserId(), nil
}

func (c *Client) Login(ctx context.Context, email, password string) (access, refresh string, err error) {
	resp, err := c.api.Authenticate(ctx, &authv1.LoginRequest{
		Email:         email,
		Password:      password,
		ApplicationId: c.appID,
	})
	if err != nil {
		return "", "", mapErr(err)
	}
	return resp.GetAccessToken(), resp.GetRefreshToken(), nil
}

func (c *Client) RefreshTokens(ctx context.Context, refreshToken string) (access, refresh string, err error) {
	resp, err := c.api.RefreshTokens(ctx, &authv1.RefreshTokensRequest{
		RefreshToken:  refreshToken,
		ApplicationId: c.appID,
	})
	if err != nil {
		return "", "", mapErr(err)
	}
	return resp.GetAccessToken(), resp.GetRefreshToken(), nil
}

func (c *Client) Logout(ctx context.Context, refreshToken string) error {
	_, err := c.api.Logout(ctx, &authv1.LogoutRequest{RefreshToken: refreshToken})
	return mapErr(err)
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	switch st.Code() {
	case codes.InvalidArgument:
		return fmt.Errorf("%w: %s", ErrInvalidCredentials, st.Message())
	case codes.AlreadyExists:
		return fmt.Errorf("%w: %s", ErrUserExists, st.Message())
	case codes.Unauthenticated:
		return fmt.Errorf("%w: %s", ErrRefreshInvalid, st.Message())
	default:
		return err
	}
}

package forest

import (
	"errors"
	"fmt"
	"os"

	"github.com/webx-top/echo"
	"github.com/webx-top/echo/middleware"
)

var (
	ErrPasswordInvalid = errors.New("密码不正确")
)

func NewAPIAuth(admName, admPassword, jwtKey string) *APIAuth {
	if len(admPassword) == 0 {
		admPassword = os.Getenv("FOREST_ADMIN_PASSWORD")
	}
	if len(admName) == 0 {
		admName = os.Getenv("FOREST_ADMIN_NAME")
	}
	if len(admName) == 0 {
		admName = `admin`
	}
	auth := &APIAuth{
		Auth: func(user *InputLogin) error {
			if user.Username != admName {
				return fmt.Errorf("用户名不正确: %s", user.Username)
			}
			if user.Password != admPassword {
				return ErrPasswordInvalid
			}
			return nil
		},
		JWTKey: jwtKey,
	}
	return auth
}

func APIServiceAuth() echo.MiddlewareFuncd {
	return middleware.KeyAuth(func(token string, ctx echo.Context) (bool, error) {
		apiToken := os.Getenv("FOREST_API_TOKEN")
		if len(apiToken) == 0 {
			return false, ErrApiTokenEnvVarNotSet
		}
		return apiToken == token, nil
	})
}

type APIAuth struct {
	Auth   func(*InputLogin) error
	JWTKey string
}

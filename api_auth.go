package forest

import (
	"errors"
	"fmt"
	"os"
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

type APIAuth struct {
	Auth   func(*InputLogin) error
	JWTKey string
}

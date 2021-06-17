package forest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/andistributed/forest/crypto"
	"github.com/webx-top/echo"
)

var (
	ErrPasswordInvalid = errors.New("密码不正确")
	ErrInvalidPostBody = errors.New("无效的post数据")
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

func APIServiceAuth(recvNew func() interface{}) echo.MiddlewareFuncd {
	return func(h echo.Handler) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			secret := os.Getenv("FOREST_API_SECRET")
			if len(secret) == 0 {
				return crypto.ErrApiSecretEnvVarNotSet
			}
			defer ctx.Request().Body().Close()
			b, err := ioutil.ReadAll(ctx.Request().Body())
			if err != nil {
				return err
			}
			crypto.DecryptBytes([]byte(secret), &b)
			if len(b) == 0 {
				return ErrInvalidPostBody
			}
			if recvNew != nil {
				recv := recvNew()
				err = json.Unmarshal(b, recv)
				if err != nil {
					return err
				}
				ctx.Internal().Set(`recv`, recv)
			} else {
				ctx.Internal().Set(`body`, b)
			}
			return h.Handle(ctx)
		}
	}
}

type APIAuth struct {
	Auth   func(*InputLogin) error
	JWTKey string
}

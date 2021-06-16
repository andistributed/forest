package crypto

import (
	"encoding/json"
	"errors"
	"os"
)

var ErrApiSecretEnvVarNotSet = errors.New("FOREST_API_SECRET environment variable is not set")

func EncryptBody(body interface{}) ([]byte, error) {
	secret := os.Getenv("FOREST_API_SECRET")
	if len(secret) == 0 {
		return nil, ErrApiSecretEnvVarNotSet
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	EncryptBytes([]byte(secret), &b)
	return b, nil
}

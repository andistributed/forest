package crypto

import (
	"github.com/webx-top/codec"
	"github.com/webx-top/com"
)

// Decrypt 数据解密
func Decrypt(secret string, datas ...*string) {
	crypto := codec.NewAesECBCrypto(`AES-256`)
	for _, data := range datas {
		if len(*data) == 0 {
			continue
		}
		*data = crypto.Decode(*data, secret)
	}
}

// DecryptBytes 数据解密
func DecryptBytes(secret []byte, datas ...*[]byte) {
	crypto := codec.NewAesECBCrypto(`AES-256`)
	for _, data := range datas {
		if len(*data) == 0 {
			continue
		}
		*data = crypto.DecodeBytes(*data, secret)
	}
}

// Encrypt 数据加密
func Encrypt(secret string, datas ...*string) {
	crypto := codec.NewAesECBCrypto(`AES-256`)
	for _, data := range datas {
		if len(*data) == 0 {
			continue
		}
		*data = crypto.Encode(*data, secret)
	}
}

// EncryptBytes 数据加密
func EncryptBytes(secret []byte, datas ...*[]byte) {
	crypto := codec.NewAesECBCrypto(`AES-256`)
	for _, data := range datas {
		if len(*data) == 0 {
			continue
		}
		*data = crypto.EncodeBytes(*data, secret)
	}
}

// GenSecret 生成随机密钥
func GenSecret(sizes ...uint) string {
	var size uint
	if len(sizes) > 0 {
		size = sizes[0]
	}
	if size < 1 {
		size = 32
	}
	return com.RandomAlphanumeric(size)
}

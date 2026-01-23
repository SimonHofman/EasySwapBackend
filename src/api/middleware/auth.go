package middleware

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/SimonHofman/EasySwapBase/errcode"
	"github.com/SimonHofman/EasySwapBase/stores/xkv"
	"github.com/SimonHofman/EasySwapBase/xhttp"
)

const CR_LOGIN_MSG_KEY string = "cache:es:login:msg"
const CR_LOGIN_KEY string = "cache:login:address:data"
const CR_LOGIN_SALT string = "es_login_salt&$%"

func AuthMiddleWare(ctx *xkv.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		values := c.Request.Header.Get("session_id")
		if values == "" {
			c.Next()
			return
		}

		sessionIDs := strings.Split(values, ",")
		for _, sessionID := range sessionIDs {
			encryptCode, err := hex.DecodeString(sessionID)
			if err != nil {
				xhttp.Error(c, errcode.ErrTokenVerify)
				c.Abort()
				return
			}

			decrptCode, err := AesDecryptOFB(encryptCode, []byte(CR_LOGIN_SALT))
			if err != nil {
				xhttp.Error(c, errcode.ErrTokenExpire)
				c.Abort()
				return
			}

			result, err := ctx.Get(string(decrptCode))
			if result == "" || err != nil {
				xhttp.Error(c, errcode.ErrTokenExpire)
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

func GetAuthUserAddress(c *gin.Context, ctx *xkv.Store) ([]string, error) {
	values := c.Request.Header.Get("session_id")
	if values == "" {
		return nil, errors.New("fails on get token")
	}

	sessionIDs := strings.Split(values, ",")
	var addrs []string
	for _, sessionID := range sessionIDs {
		encryptCode, err := hex.DecodeString(sessionID)
		if err != nil {
			return nil, errors.New("fails on decode cookie")
		}

		decrptCode, err := AesDecryptOFB(encryptCode, []byte(CR_LOGIN_SALT))
		if err != nil {
			return nil, errors.Wrap(err, "invalid cookie")
		}
		result, err := ctx.Get(string(decrptCode))
		if result == "" || err != nil {
			return nil, errors.Wrap(err, "failed on read cookie from cache")
		}

		arr := strings.Split(string(decrptCode), CR_LOGIN_KEY+":")
		if len(arr) != 2 {
			return nil, errors.New("user cache info format err")
		}

		if arr[1] == "" {
			return nil, errors.New("invalid user address")
		}
		addrs = append(addrs, arr[1])
	}
	return addrs, nil
}

func AesDecryptOFB(data []byte, key []byte) ([]byte, error) {
	block, _ := aes.NewCipher([]byte(key))
	iv := data[:aes.BlockSize]
	data = data[aes.BlockSize:]
	if len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("data is not a multiple of the block size")
	}

	out := make([]byte, len(data))
	mode := cipher.NewOFB(block, iv)
	mode.XORKeyStream(out, data)

	out = PKCS7UnPadding(out)
	return out, nil
}

func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

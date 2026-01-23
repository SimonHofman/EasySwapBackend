package middleware

import (
	"bytes"

	"github.com/gin-gonic/gin"
)

type BodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

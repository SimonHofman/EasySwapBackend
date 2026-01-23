package middleware

import (
	"bytes"
	"crypto/sha512"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/SimonHofman/EasySwapBase/errcode"
	"github.com/SimonHofman/EasySwapBase/stores/xkv"
	"github.com/SimonHofman/EasySwapBase/xhttp"
	"github.com/gin-gonic/gin"
)

const CacheApiPrefix = "apicache:"

type responseCache struct {
	Status int
	Header http.Header
	Data   []byte
}

func CacheApi(store *xkv.Store, expireSeconds int) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data xhttp.Response
		bodyLogWriter := &BodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = bodyLogWriter

		cacheKey := CreateKey(c)
		if cacheKey == "" {
			xhttp.Error(c, errcode.NewCustomErr("cache error: no cache"))
			c.Abort()
		}

		cacheData, err := (*store).Get(cacheKey)
		if err == nil && cacheData != "" {
			cache := unserialize(cacheData)
			if cache != nil {
				bodyLogWriter.ResponseWriter.WriteHeader(cache.Status)
				for k, vals := range cache.Header {
					for _, v := range vals {
						bodyLogWriter.ResponseWriter.Header().Set(k, v)
					}
				}

				if err := json.Unmarshal(cache.Data, &data); err == nil {
					if data.Code == http.StatusOK {
						bodyLogWriter.ResponseWriter.Write(cache.Data)
						c.Abort()
					}
				}
			}
		}

		c.Next()

		responseBody := bodyLogWriter.body.Bytes()

		if err := json.Unmarshal(responseBody, &data); err == nil {
			if data.Code == http.StatusOK {
				storeCache := responseCache{
					Header: bodyLogWriter.Header().Clone(),
					Status: bodyLogWriter.ResponseWriter.Status(),
					Data:   responseBody,
				}
				store.SetnxEx(cacheKey, serialize(storeCache), expireSeconds)
			}
		}
	}
}

func CreateKey(c *gin.Context) string {
	var buf bytes.Buffer
	tee := io.TeeReader(c.Request.Body, &buf)
	requestBody, _ := ioutil.ReadAll(tee)
	c.Request.Body = ioutil.NopCloser(&buf)

	path := c.Request.URL.Path
	query := c.Request.URL.RawQuery

	cacheKey := path + "," + query + string(requestBody)

	if len(cacheKey) > 128 {
		hash := sha512.New()
		hash.Write([]byte(cacheKey))
		cacheKey = string(hash.Sum([]byte("")))
		cacheKey = fmt.Sprintf("%x", cacheKey)
	}

	cacheKey = CacheApiPrefix + cacheKey
	return cacheKey
}

func serialize(cache responseCache) string {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(cache); err != nil {
		return ""
	} else {
		return buf.String()
	}
}

func unserialize(data string) *responseCache {
	var g1 = responseCache{}
	dec := gob.NewDecoder(bytes.NewBuffer([]byte(data)))
	if err := dec.Decode(&g1); err != nil {
		return nil
	} else {
		return &g1
	}
}

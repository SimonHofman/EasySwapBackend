package middleware

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"runtime"

	"github.com/SimonHofman/EasySwapBase/errcode"
	"github.com/SimonHofman/EasySwapBase/logger/xzap"
	"github.com/SimonHofman/EasySwapBase/xhttp"
	"github.com/gin-gonic/gin"
)

var (
	dunno     = []byte("???")
	centerDot = []byte(".")
	dot       = []byte(".")
	slash     = []byte("/")
)

func RecoverMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if cause := recover(); cause != nil {
				xzap.WithContext(c.Request.Context()).Errorf("[Recovery] panic recovered, request:%s%v [## statck]:\n%s", dumpRequest(c.Request), cause, dumpStack(3))
				xhttp.Error(c, errcode.ErrUnexpected)
			}
		}()

		c.Next()
	}
}

func dumpRequest(req *http.Request) string {
	var dup io.ReadCloser
	req.Body, dup = dupReadCloser(req.Body)

	var b bytes.Buffer
	var err error

	reqURI := req.RequestURI
	if reqURI == "" {
		reqURI = req.URL.RequestURI()
	}

	_, _ = fmt.Fprintf(&b, "%s %s HTTP/%d.%d\n", req.Method, reqURI, req.ProtoMajor, req.ProtoMinor)
	chunked := len(req.TransferEncoding) > 0 && req.TransferEncoding[0] == "chunked"
	if req.Body != nil {
		var n int64
		var dest io.Writer = &b
		if chunked {
			dest = httputil.NewChunkedWriter(dest)
		}

		n, err = io.Copy(dest, req.Body)
		if chunked {
			dest.(io.Closer).Close()
		}
		if n > 0 {
			_, _ = io.WriteString(&b, "\n")
		}
	}

	req.Body = dup
	if err != nil {
		return err.Error()
	}

	return b.String()
}

func dupReadCloser(reader io.ReadCloser) (io.ReadCloser, io.ReadCloser) {
	var buf bytes.Buffer
	tee := io.TeeReader(reader, &buf)
	return ioutil.NopCloser(tee), ioutil.NopCloser(&buf)
}

func dumpStack(skip int) []byte {
	buf := new(bytes.Buffer)
	var lines [][]byte
	var lastFile string
	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		_, _ = fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		if file != lastFile {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}

		_, _ = fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
	}
	return buf.Bytes()
}

func source(lines [][]byte, n int) []byte {
	n--
	if n < 0 || n >= len(lines) {
		return dunno
	}
	return bytes.TrimSpace(lines[n])
}

func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return dunno
	}

	name := []byte(fn.Name())
	if lastslash := bytes.LastIndex(name, slash); lastslash >= 0 {
		name = name[lastslash+1:]
	}
	if period := bytes.Index(name, dot); period >= 0 {
		name = name[period+1:]
	}
	name = bytes.Replace(name, centerDot, dot, -1)
	return name
}

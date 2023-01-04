package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/go-playground/validator"
	"github.com/google/uuid"
	"github.com/vaish1707/golang-logging-instrumentation/logger"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"
)

type errResponse struct {
	Message string `json:"message"`
}

type metadata struct {
	ReqId       string
	UserId      string
	UserAgent   string
	ReqMethod   string
	ReqPath     string
	Host        string
	ServiceName string
	MethodName  string
}

func GetExtraFields(r *http.Request, userId string, serviceName string, methodName string) []zap.Field {
	hostname, _ := os.Hostname()
	userAgent := r.UserAgent()
	fields := buildFields(metadata{ReqId: r.Header.Get("requestId"), UserId: userId, UserAgent: userAgent, ReqMethod: r.Method, ReqPath: r.URL.Path, Host: hostname, ServiceName: serviceName, MethodName: methodName})
	return fields
}

func shortPath(path string) string {
	dirs := strings.Split(path, "/")
	if len(dirs) > 1 {
		return strings.Join(dirs[len(dirs)-2:], "/")
	}
	return path
}

func buildLine() zap.Field {
	_, fn, line, _ := runtime.Caller(3) // reach back into callstack for log.* call
	return zap.String("line", fmt.Sprintf("%s:%d", shortPath(fn), line))
}

func buildFields(m metadata) []zap.Field {
	requestId := zap.String("requestId", m.ReqId)
	userId := zap.String("userId", m.UserId)
	userAgent := zap.String("userAgent", m.UserAgent)
	reqMethod := zap.String("requestMethod", m.ReqMethod)
	reqPath := zap.String("requestPath", m.ReqPath)
	host := zap.String("hostname", m.Host)
	serviceName := zap.String("serviceName", m.ServiceName)
	methodName := zap.String("methodName", m.MethodName)
	fields := []zap.Field{buildLine(), requestId, userId, userAgent, reqMethod, reqPath, host, serviceName, methodName}
	return fields
}

func ReadBody(w http.ResponseWriter, r *http.Request, obj interface{}) error {
	// read body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, err)
		return fmt.Errorf("read body error: %w", err)
	}

	// unmarshal into object
	if err := json.Unmarshal(body, obj); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, err)
		return fmt.Errorf("json unmarshal error: %w", err)
	}

	// validate object
	if err := validator.New().Struct(obj); err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, err)
		return fmt.Errorf("validate object error: %w", err)
	}

	return nil
}

func WriteErrorResponse(w http.ResponseWriter, statusCode int, err error) {
	WriteResponse(w, statusCode, errResponse{err.Error()})
}

func WriteResponse(w http.ResponseWriter, statusCode int, response interface{}) {
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Printf("encode response error: %v", err)
	}
}

func SendRequest(ctx context.Context, method string, url string, data []byte) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("create request error: %w", err)
	}

	client := http.Client{
		// Wrap the Transport with one that starts a span and injects the span context
		// into the outbound request headers.
		Transport: otelhttp.NewTransport(http.DefaultTransport),
		Timeout:   10 * time.Second,
	}

	return client.Do(request)
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	// WriteHeader() is not claled if our response implicitly returns 200 OK, so
	// we default to that status code
	return &responseWriter{w, http.StatusOK}
}

func LoggingMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// wrap the response writer to capture the response
		rw := newResponseWriter(w)
		// Once the body is read, it cannot be re-read. Hence, use the TeeReader
		// to write the r.Body to buf as it is being read.
		// This buf is later used for logging.
		var buf bytes.Buffer
		tee := io.TeeReader(r.Body, &buf)
		r.Body = ioutil.NopCloser(tee)
		next.ServeHTTP(rw, r)
		duration := time.Since(start)
		statusCode := zap.Int("statusCode", rw.statusCode)
		reqbody := zap.String("requestBody", buf.String())
		timeTaken := zap.Int64("duration", duration.Milliseconds())
		fields := []zap.Field{statusCode, reqbody, timeTaken}
		logger.Ctx(r.Context()).Info("Request completed",
			fields...)
	})
}

func LogRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New()
		r.Header.Set("requestId", id.String())
		next.ServeHTTP(w, r)
	})
}

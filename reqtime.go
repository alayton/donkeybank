package main

import (
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	green   = string([]byte{27, 91, 57, 55, 59, 52, 50, 109})
	white   = string([]byte{27, 91, 57, 48, 59, 52, 55, 109})
	yellow  = string([]byte{27, 91, 57, 55, 59, 52, 51, 109})
	red     = string([]byte{27, 91, 57, 55, 59, 52, 49, 109})
	blue    = string([]byte{27, 91, 57, 55, 59, 52, 52, 109})
	magenta = string([]byte{27, 91, 57, 55, 59, 52, 53, 109})
	cyan    = string([]byte{27, 91, 57, 55, 59, 52, 54, 109})
	reset   = string([]byte{27, 91, 48, 109})
)

func reqTime(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		writer := newWriter(w)
		h.ServeHTTP(writer, r)
		duration := time.Since(start)

		clientIP := r.Context().Value("IP")
		method := r.Method
		path := r.URL.Path
		statusCode := writer.Status()
		statusColor := colorForStatus(statusCode)
		methodColor := colorForMethod(method)

		log.Infof("%v |%s %3d %s| %13v | %s |%s  %s %-7s %s",
			start.Format("2006/01/02 - 15:04:05"),
			statusColor, statusCode, reset,
			duration,
			clientIP,
			methodColor, reset, method,
			path,
		)
	})
}

func colorForStatus(code int) string {
	switch {
	case code >= 200 && code < 300:
		return green
	case code >= 300 && code < 400:
		return white
	case code >= 400 && code < 500:
		return yellow
	default:
		return red
	}
}

func colorForMethod(method string) string {
	switch method {
	case "GET":
		return blue
	case "POST":
		return cyan
	case "PUT":
		return yellow
	case "DELETE":
		return red
	case "PATCH":
		return green
	case "HEAD":
		return magenta
	case "OPTIONS":
		return white
	default:
		return reset
	}
}

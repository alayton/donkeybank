package main

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

type responseWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

func newWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
	}
}

func (w *responseWriter) WriteHeader(status int) {
	if w.written {
		log.Warnf("Header already written. Tried to override original status %d with %d", w.status, status)
		return
	}

	w.status = status
}

func (w *responseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}

	w.written = true
	w.ResponseWriter.WriteHeader(w.status)
	return w.ResponseWriter.Write(data)
}

func (w *responseWriter) Status() int {
	return w.status
}

func (w *responseWriter) Written() bool {
	return w.written
}

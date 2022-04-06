package respond

import (
	"encoding/json"
	"net/http"
)

func writeContentType(w http.ResponseWriter, value []string) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = value
	}
}

var jsonContentType = []string{"application/json; charset=utf-8"}

// JSON writes the object to the Writer as a JSON encoded string
func JSON(w http.ResponseWriter, statusCode int, obj interface{}) error {
	w.WriteHeader(statusCode)
	writeContentType(w, jsonContentType)
	return json.NewEncoder(w).Encode(obj)
}

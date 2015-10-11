package webhookproxy

import "net/http"

type CompatMux interface {
	HandleFunc(path string, handler func(w http.ResponseWriter, r *http.Request))
}

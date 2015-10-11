package webhookproxy

import "github.com/twinj/uuid"
import "net/http"
import "time"

type Session struct {
	id        string
	w         *http.ResponseWriter
	r         *http.Request
	c         chan *http.Request
	errorChan chan error
	okChan    chan bool
	started   time.Time
}

func NewSession(w *http.ResponseWriter, r *http.Request) *Session {
	id := uuid.NewV4().String()
	resultChannel := make(chan *http.Request)
	errorChannel := make(chan error)
	okChannel := make(chan bool, 1)
	started := time.Now()
	return &Session{id, w, r, resultChannel, errorChannel, okChannel, started}
}

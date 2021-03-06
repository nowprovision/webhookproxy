package webhookproxy

import "time"
import "bytes"
import "fmt"
import "log"
import "io/ioutil"
import "net/http"

type WebHookHandlers struct {
	HookHandler  func(http.ResponseWriter, *http.Request)
	PollHandler  func(http.ResponseWriter, *http.Request)
	ReplyHandler func(http.ResponseWriter, *http.Request)
}

func RegisterHandlers(config *Config, mux CompatMux) {

	handlers := BuildHandlers(config)

	hostname := config.Hostname
	secret := config.Secret
	mux.HandleFunc(hostname+"/webhook/"+secret, handlers.HookHandler)
	mux.HandleFunc(hostname+"/poll/"+secret, handlers.PollHandler)
	mux.HandleFunc(hostname+"/reply"+secret, handlers.ReplyHandler)

}

func BuildHandlers(config *Config) *WebHookHandlers {

	incomingQueue := make(chan *Session, config.BackQueueSize)
	sessionMap := make(map[string]*Session)

	handlerWebhook := Protect(config.FilteringEnabled, config.WebhookFilters, func(w http.ResponseWriter, r *http.Request) {

		log.Printf("Web hook call received from %s to %s", r.RemoteAddr, r.URL)
		session := NewSession(&w, r)

		sessionMap[session.id] = session

		if len(sessionMap) > 50 {
			w.WriteHeader(config.TryLaterStatusCode)
			if config.ShowDebugInfo {
				fmt.Fprintf(w, "Too busy")
			}
			return
		}

		defer delete(sessionMap, session.id)
		log.Printf("Web hook call assigned session id %s", session.id)

		incomingQueue <- session
		log.Printf("Web hook call added to incoming queue, waiting for reply")

		// reply with empty if auto reply has been sent
		if config.Autoreply {

			bytesBuffer := bytes.NewBuffer([]byte{})
			_, err := CopyMax(config.MaxPayloadSize, bytesBuffer, session.r.Body)
			if err != nil {
				session.errorChan <- err
			} else {
				session.r.Body = ioutil.NopCloser(bytesBuffer)
				req, _ := http.NewRequest("POST", "/", NewStringPayload(""))
				go func() { session.c <- req }()
			}
		}

		select {
		case reply := <-session.c:
			log.Printf("Reply received for %s, copying body", session.id)
			written, err := CopyMax(config.MaxPayloadSize, w, reply.Body)
			if err != nil {
				log.Printf("Error copying body for web hook call reply for %s. %s", session.id, err)
				session.okChan <- false
				// We have to panic here so the default mux handler wrapper
				// terminates the http conn prematurely and client gets an
				// aborted http stream, as we have already sent status ok
				// in future consider hijack and more graceful handling
				panic(err)
			}
			log.Printf("Completed reply to /webook for %s with %d bytes", session.id, written)
			session.okChan <- true
		case err := <-session.errorChan:
			log.Printf("Error during processing for %s. Error %s", session.id, err)
			w.WriteHeader(config.TryLaterStatusCode)
			if config.ShowDebugInfo {
				fmt.Fprintf(w, "Error: %s", err.Error())
			}
			log.Printf("Sent status code %d to %s", config.TryLaterStatusCode, session.id)
			session.okChan <- false
		case <-time.After(config.LongPollWait):
			log.Printf("Time out waiting for reply to return to web hook callee for %s", session.id)
			w.WriteHeader(config.TryLaterStatusCode)
			if config.ShowDebugInfo {
				fmt.Fprintf(w, "%s", "Timed out")
			}
			log.Printf("Sent status code %d to %s", config.TryLaterStatusCode, session.id)
			session.okChan <- false
		}
	})

	handlerPoll := Protect(config.FilteringEnabled, config.PollReplyFilters, func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Poll client connected from %s to %s. Waiting for a web hook", r.RemoteAddr, r.URL)
		notify := w.(http.CloseNotifier).CloseNotify()
		select {
		case payload := <-incomingQueue:
			delay := time.Now().Sub(payload.started).Seconds()
			log.Printf("Proxying payload %s from web hook callee to poll client", payload.id)
			log.Printf("Proxying pickup latency %.5fs for %s", delay, payload.id)
			w.Header().Add("X-ReplyId", payload.id)
			for headerKey, headerValues := range payload.r.Header {
				for _, headerValue := range headerValues {
					prefix := "x-whheader-"
					if headerKey == "Content-Type" || headerKey == "Content-Encoding" {
						prefix = ""
					}
					w.Header().Add(prefix+headerKey, headerValue)
				}
			}
			w.Header().Add("X-WhFrom", r.RemoteAddr)
			w.Header().Add("X-WhDelaySecs", fmt.Sprintf("%.5f", delay))

			written, err := CopyMax(config.MaxPayloadSize, w, payload.r.Body)
			log.Printf("Proxied %d bytes from web hook to poll client for %s", written, payload.id)

			if err != nil {
				log.Printf("Error proxying request %s : %s", payload.id, err)
				if !(config.Autoreply) {
					payload.errorChan <- err
				}
				return
			}
		case <-notify:
			log.Printf("Poll client disconnected from %s to %s. Waiting for a web hook", r.RemoteAddr, r.URL)
		case <-time.After(config.LongPollWait):
			log.Println("No web hook waiting after %s duration", config.LongPollWait)
			w.WriteHeader(http.StatusNoContent)
		}
	})

	handlerReply := Protect(config.FilteringEnabled, config.PollReplyFilters, func(w http.ResponseWriter, r *http.Request) {

		log.Printf("Call received to /reply from %s", r.RemoteAddr)

		replyTo := r.Header.Get("X-InReplyTo")

		replyToLength := len(replyTo)

		if replyToLength == 0 {
			log.Printf("Missing/empty X-InReplyTo header received from %s", r.RemoteAddr)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Bad Request: %s", "Non-empty X-InReplyTo HTTP header required")
			return
		}

		if replyToLength != 36 {
			log.Printf("Incorrect lengeth X-InReplyTo header received from %s, length %d",
				r.RemoteAddr, replyToLength)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Bad Request: %s", "Incorrect length X-InReplyTo HTTP header received")
			return
		}

		log.Printf("Looking up %s X-InReplyTo key", replyTo)

		session, found := sessionMap[replyTo]
		if !found {
			log.Printf("InReplyTo %s received from %s not found, perhaps web hook callee has left already",
				replyTo, r.RemoteAddr)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Bad Request: %s %s", replyTo, "X-InReplyTo not found")
			return
		}
		session.c <- r
		allGood := <-session.okChan
		if !allGood {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Unable to send reply successfully")
		}
	})

	return &WebHookHandlers{handlerWebhook, handlerPoll, handlerReply}
}

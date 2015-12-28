package webhookproxy

import "io"
import "io/ioutil"
import "fmt"

import "time"
import "bytes"
import "testing"
import "net"
import "net/http"
import "net/http/httptest"

func TestFullCycle(t *testing.T) {

	config := Config{BackQueueSize: 1,
		TryLaterStatusCode: 503,
		MaxPayloadSize:     30,
		LongPollWait:       time.Second * 10}

	_, localNetwork, _ := net.ParseCIDR("127.0.0.1/24")
	config.WebhookWhiteList = []*net.IPNet{localNetwork}
	config.PollReplyWhiteList = []*net.IPNet{localNetwork}

	mux := http.NewServeMux()
	RegisterHandlers(config, mux)

	done := make(chan bool, 1)
	errorChan := make(chan error)

	testPayload := "webhoooks suck"
	testReply := "we agree, clients not servers"

	go func() {
		req, err := LocalRequest("POST", "/webhook", bytes.NewBuffer([]byte(testPayload)))

		if err != nil {
			errorChan <- fmt.Errorf("Unable to create POST request for /webhook")
			return
		}

		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			errorChan <- fmt.Errorf("Bad status code for  /webhook, %d", resp.Code)
			return
		}
		reply, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			errorChan <- err
			return
		}

		if string(reply) != testReply {
			errorChan <- fmt.Errorf("Expected %s got %s", testReply, reply)
			return
		}

		done <- true
	}()

	go func() {
		req, _ := LocalRequest("GET", "/poll", nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			errorChan <- fmt.Errorf("Unexpected response from /poll of %d", resp.Code)
			return
		}

		respBody, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			errorChan <- fmt.Errorf("Reading  response from /poll error: %s ", err)
			return
		}

		if string(respBody) != testPayload {
			errorChan <- fmt.Errorf("Response payload not matched %s != %s", string(respBody), testPayload)
			return
		}

		replyId := resp.Header().Get("X-ReplyId")

		resp = httptest.NewRecorder()
		replyReq, err := LocalRequest("POST", "/reply", NewStringPayload(testReply))

		if err != nil {
			errorChan <- fmt.Errorf("Unable to create POST request for /reply")
			return
		}
		replyReq.Header.Add("X-InReplyTo", replyId)
		mux.ServeHTTP(resp, replyReq)

		if resp.Code != http.StatusOK {
			errorChan <- fmt.Errorf("Unexpected response from /reply of %d", resp.Code)
			return
		}

		ioutil.ReadAll(resp.Body)

		if err != nil {
			errorChan <- err
			return
		}

		done <- true
	}()

	allDone := make(chan bool)

	go func() {
		<-done
		<-done
		allDone <- true
	}()

	select {
	case <-allDone:
		return
	case err := <-errorChan:
		t.Errorf("Error: %s", err)
	case <-time.After(10 * time.Second):
		t.Errorf("Timeout waiting for handlers to complete")
	}
}

func TestTimeoutWhenNooneListening(t *testing.T) {

	/* send a post request to the webhook endpoint, because
	   we have no clients long polling within the 2sec timeout period
	   we should return a not available status code to originating
	   webhook callee. verify timeout is respected and status code
	   matches config*/
	resp := httptest.NewRecorder()

	uri := "/webhook"

	req, err := LocalRequest("POST", uri, nil)

	if err != nil {
		t.Fatal(err)
	}

	config := Config{BackQueueSize: 1,
		MaxPayloadSize:     0,
		TryLaterStatusCode: 503,
		LongPollWait:       2 * time.Second}

	_, localnetwork, _ := net.ParseCIDR("127.0.0.1/24")
	config.WebhookWhiteList = []*net.IPNet{localnetwork}
	config.PollReplyWhiteList = []*net.IPNet{localnetwork}

	mux := http.NewServeMux()
	RegisterHandlers(config, mux)

	started := time.Now()

	mux.ServeHTTP(resp, req)

	if time.Since(started) < config.LongPollWait {
		t.Errorf("timeout not respected, elapsed %f, expected min of %d", time.Since(started),
			config.LongPollWait)
	}
	if resp.Code != config.TryLaterStatusCode {
		t.Errorf("expected %d got %d - incorrect resp status code",
			config.TryLaterStatusCode, resp.Code)
	}
}

func LocalRequest(method string, uri string, payload io.Reader) (req *http.Request, err error) {
	req, err = http.NewRequest(method, uri, payload)

	if err != nil {
		return
	}
	req.RemoteAddr = "127.0.0.1:53000"
	return
}

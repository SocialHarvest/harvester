package harvester

import (
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// From https://gist.github.com/seantalts/11266762
// Tests for the http.Client wrapper.

func TestHttpTimeout(t *testing.T) {
	http.HandleFunc("/normal", func(w http.ResponseWriter, req *http.Request) {
		// Empirically, timeouts less than these seem to be flaky
		time.Sleep(100 * time.Millisecond)
		io.WriteString(w, "ok")
	})
	http.HandleFunc("/timeout", func(w http.ResponseWriter, req *http.Request) {
		time.Sleep(250 * time.Millisecond)
		io.WriteString(w, "ok")
	})
	ts := httptest.NewServer(http.DefaultServeMux)
	defer ts.Close()

	numDials := 0

	client := &http.Client{
		Transport: &TimeoutTransport{
			Transport: http.Transport{
				Dial: func(netw, addr string) (net.Conn, error) {
					t.Logf("dial to %s://%s", netw, addr)
					numDials++                  // For testing only.
					return net.Dial(netw, addr) // Regular ass dial.
				},
			},
			RoundTripTimeout: time.Millisecond * 200,
		},
	}

	addr := ts.URL

	SendTestRequest(t, client, "1st", addr, "normal")
	if numDials != 1 {
		t.Fatalf("Should only have 1 dial at this point.")
	}
	SendTestRequest(t, client, "2st", addr, "normal")
	if numDials != 1 {
		t.Fatalf("Should only have 1 dial at this point.")
	}
	SendTestRequest(t, client, "3st", addr, "timeout")
	if numDials != 1 {
		t.Fatalf("Should only have 1 dial at this point.")
	}
	SendTestRequest(t, client, "4st", addr, "normal")
	if numDials != 2 {
		t.Fatalf("Should have our 2nd dial.")
	}

	time.Sleep(time.Millisecond * 700)

	SendTestRequest(t, client, "5st", addr, "normal")
	if numDials != 2 {
		t.Fatalf("Should still only have 2 dials.")
	}
}

func SendTestRequest(t *testing.T, client *http.Client, id, addr, path string) {
	req, err := http.NewRequest("GET", addr+"/"+path, nil)

	if err != nil {
		t.Fatalf("new request failed - %s", err)
	}

	req.Header.Add("Connection", "keep-alive")

	switch path {
	case "normal":
		if resp, err := client.Do(req); err != nil {
			t.Fatalf("%s request failed - %s", id, err)
		} else {
			result, err2 := ioutil.ReadAll(resp.Body)
			if err2 != nil {
				t.Fatalf("%s response read failed - %s", id, err2)
			}
			resp.Body.Close()
			t.Logf("%s request - %s", id, result)
		}
	case "timeout":
		if _, err := client.Do(req); err == nil {
			t.Fatalf("%s request not timeout", id)
		} else {
			t.Logf("%s request - %s", id, err)
		}
	}
}

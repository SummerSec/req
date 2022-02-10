package req

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func tr() *Request {
	return tc().R()
}

func tc() *Client {
	return C().SetBaseURL(getTestServerURL())
}

var testDataPath string

func init() {
	pwd, _ := os.Getwd()
	testDataPath = filepath.Join(pwd, ".testdata")
}

var testServerMu sync.Mutex
var testServer *httptest.Server

func getTestServerURL() string {
	if testServer != nil {
		return testServer.URL
	}
	testServerMu.Lock()
	defer testServerMu.Unlock()
	testServer = createTestServer()
	return testServer.URL
}

type echo struct {
	Header http.Header `json:"header"`
	Body   string      `json:"body"`
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		w.Write([]byte("TestPost: text response"))
	case "/echo":
		b, _ := ioutil.ReadAll(r.Body)
		e := echo{
			Header: r.Header,
			Body:   string(b),
		}
		result, _ := json.Marshal(&e)
		w.Header().Set(hdrContentTypeKey, jsonContentType)
		w.Write(result)
	}
}

func handleGetUserProfile(w http.ResponseWriter, r *http.Request) {
	user := strings.TrimLeft(r.URL.Path, "/user")
	user = strings.TrimSuffix(user, "/profile")
	w.Write([]byte(fmt.Sprintf("%s's profile", user)))
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		w.Write([]byte("TestGet: text response"))
	case "/bad-request":
		w.WriteHeader(http.StatusBadRequest)
	case "/host-header":
		w.Write([]byte(r.Host))
	case "/header":
		b, _ := json.Marshal(r.Header)
		w.Header().Set(hdrContentTypeKey, jsonContentType)
		w.Write(b)
	case "/user-agent":
		w.Write([]byte(r.Header.Get(hdrUserAgentKey)))
	case "/content-type":
		w.Write([]byte(r.Header.Get(hdrContentTypeKey)))
	case "/query-parameter":
		w.Write([]byte(r.URL.RawQuery))
	default:
		if strings.HasPrefix(r.URL.Path, "/user") {
			handleGetUserProfile(w, r)
		}
	}
}

func createTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGet(w, r)
		case http.MethodPost:
			handlePost(w, r)
		}
	}))
}

func assertStatus(t *testing.T, resp *Response, err error, statusCode int, status string) {
	assertError(t, err)
	assertNotNil(t, resp)
	assertNotNil(t, resp.Body)
	assertEqual(t, statusCode, resp.StatusCode)
	assertEqual(t, status, resp.Status)
}

func assertSucess(t *testing.T, resp *Response, err error) {
	assertError(t, err)
	assertNotNil(t, resp)
	assertNotNil(t, resp.Body)
	assertEqual(t, http.StatusOK, resp.StatusCode)
	assertEqual(t, "200 OK", resp.Status)
	assertEqual(t, "HTTP/1.1", resp.Proto)
}

func assertNil(t *testing.T, v interface{}) {
	if !isNil(v) {
		t.Errorf("[%v] was expected to be nil", v)
	}
}

func assertNotNil(t *testing.T, v interface{}) {
	if isNil(v) {
		t.Errorf("[%v] was expected to be non-nil", v)
	}
}

func assertType(t *testing.T, typ, v interface{}) {
	if reflect.DeepEqual(reflect.TypeOf(typ), reflect.TypeOf(v)) {
		t.Errorf("Expected type %t, got %t", typ, v)
	}
}

func assertContains(t *testing.T, s, substr string, shouldContain bool) {
	isContain := strings.Contains(s, substr)
	if shouldContain {
		if !isContain {
			t.Errorf("%q is not included in %s", substr, s)
		}
	} else {
		if isContain {
			t.Errorf("%q is included in %s", substr, s)
		}
	}
}

func assertError(t *testing.T, err error) {
	if err != nil {
		t.Errorf("Error occurred [%v]", err)
	}
}

func assertEqual(t *testing.T, e, g interface{}) {
	if !equal(e, g) {
		t.Errorf("Expected [%v], got [%v]", e, g)
	}
	return
}

func removeEmptyString(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return s
}

func assertNotEqual(t *testing.T, e, g interface{}) (r bool) {
	if equal(e, g) {
		t.Errorf("Expected [%v], got [%v]", e, g)
	} else {
		r = true
	}
	return
}

func equal(expected, got interface{}) bool {
	return reflect.DeepEqual(expected, got)
}

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	kind := rv.Kind()
	if kind >= reflect.Chan && kind <= reflect.Slice && rv.IsNil() {
		return true
	}
	return false
}

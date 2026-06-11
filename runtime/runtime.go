package main

/*
#include <stdlib.h>
#include <string.h>
*/
import "C"
import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"unsafe"
)

type httpRequest struct {
	method      string
	path        string
	body        string
	contentType string
	headers     map[string]string
	respC       chan *httpResponse
}

type httpResponse struct {
	status      int
	body        string
	contentType string
	headers     map[string]string
}

var (
	requestCh      = make(chan *httpRequest, 1)
	handlerDoneCh  = make(chan struct{}, 1)
	currentReq     *httpRequest
	listenSrv      *http.Server
	pendingHeaders map[string]string
)

func httpClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
}

func copyToBuf(response *byte, responseCap int64, responseLen *int64, data []byte) {
	if response == nil || responseCap <= 0 {
		return
	}
	n := len(data)
	if int64(n) > responseCap {
		n = int(responseCap)
	}
	dest := unsafe.Slice(response, n)
	copy(dest, data[:n])
	*responseLen = int64(n)
}

func doRequest(req *http.Request, response *byte, responseCap int64, responseLen *int64, statusCode *int32) int {
	client := httpClient()
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "HTTP error: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	if statusCode != nil {
		*statusCode = int32(resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read error: %v\n", err)
		return 1
	}

	copyToBuf(response, responseCap, responseLen, body)
	return 0
}

//export cob_http_get
func cob_http_get(url *C.char, response *C.char, responseCap C.long, responseLen *C.long, statusCode *C.int) C.int {
	req, err := http.NewRequest("GET", C.GoString(url), nil)
	if err != nil {
		return 1
	}
	return C.int(doRequest(req, (*byte)(unsafe.Pointer(response)), int64(responseCap),
		(*int64)(unsafe.Pointer(responseLen)), (*int32)(unsafe.Pointer(statusCode))))
}

//export cob_http_post
func cob_http_post(url *C.char, body *C.char, response *C.char, responseCap C.long, responseLen *C.long, statusCode *C.int) C.int {
	req, err := http.NewRequest("POST", C.GoString(url), toReader(body))
	if err != nil {
		return 1
	}
	return C.int(doRequest(req, (*byte)(unsafe.Pointer(response)), int64(responseCap),
		(*int64)(unsafe.Pointer(responseLen)), (*int32)(unsafe.Pointer(statusCode))))
}

//export cob_http_put
func cob_http_put(url *C.char, body *C.char, response *C.char, responseCap C.long, responseLen *C.long, statusCode *C.int) C.int {
	req, err := http.NewRequest("PUT", C.GoString(url), toReader(body))
	if err != nil {
		return 1
	}
	return C.int(doRequest(req, (*byte)(unsafe.Pointer(response)), int64(responseCap),
		(*int64)(unsafe.Pointer(responseLen)), (*int32)(unsafe.Pointer(statusCode))))
}

//export cob_http_patch
func cob_http_patch(url *C.char, body *C.char, response *C.char, responseCap C.long, responseLen *C.long, statusCode *C.int) C.int {
	req, err := http.NewRequest("PATCH", C.GoString(url), toReader(body))
	if err != nil {
		return 1
	}
	return C.int(doRequest(req, (*byte)(unsafe.Pointer(response)), int64(responseCap),
		(*int64)(unsafe.Pointer(responseLen)), (*int32)(unsafe.Pointer(statusCode))))
}

//export cob_http_delete
func cob_http_delete(url *C.char, response *C.char, responseCap C.long, responseLen *C.long, statusCode *C.int) C.int {
	req, err := http.NewRequest("DELETE", C.GoString(url), nil)
	if err != nil {
		return 1
	}
	return C.int(doRequest(req, (*byte)(unsafe.Pointer(response)), int64(responseCap),
		(*int64)(unsafe.Pointer(responseLen)), (*int32)(unsafe.Pointer(statusCode))))
}

func toReader(body *C.char) io.Reader {
	if body == nil {
		return nil
	}
	s := C.GoString(body)
	return &stringReader{s: s}
}

// --- JSON field extraction ---

func jsonKey(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

func jsonFind(data map[string]any, name string) (any, bool) {
	key := jsonKey(name)
	for k, v := range data {
		if jsonKey(k) == key {
			return v, true
		}
	}
	return nil, false
}

//export cob_json_str
func cob_json_str(body *C.char, bodyLen C.long, fieldName *C.char, out *C.char, outSize C.long) C.int {
	jsonBytes := C.GoBytes(unsafe.Pointer(body), C.int(bodyLen))
	var data map[string]any
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return 1
	}
	name := C.GoString(fieldName)
	val, ok := jsonFind(data, name)
	if !ok {
		return 1
	}
	str := fmt.Sprintf("%v", val)
	if out != nil && outSize > 0 {
		size := int(outSize)
		buf := make([]byte, size)
		copy(buf, str)
		for i := len(str); i < size; i++ {
			buf[i] = ' '
		}
		C.memcpy(unsafe.Pointer(out), unsafe.Pointer(&buf[0]), C.size_t(size))
	}
	return 0
}

//export cob_json_int
func cob_json_int(body *C.char, bodyLen C.long, fieldName *C.char, out *C.int) C.int {
	jsonBytes := C.GoBytes(unsafe.Pointer(body), C.int(bodyLen))
	var data map[string]any
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return 1
	}
	name := C.GoString(fieldName)
	val, ok := jsonFind(data, name)
	if !ok {
		return 1
	}
	switch v := val.(type) {
	case float64:
		if out != nil {
			*out = C.int(int(v))
		}
		return 0
	case string:
		var iv int
		if _, err := fmt.Sscanf(v, "%d", &iv); err != nil {
			return 1
		}
		if out != nil {
			*out = C.int(iv)
		}
		return 0
	}
	return 1
}

//export cob_http_listen
func cob_http_listen(port C.int, statusCode *C.int) C.int {
	// Close previous server if any (handler is done since cob_http_respond was called)
	if listenSrv != nil {
		listenSrv.Close()
		listenSrv = nil
	}
	listenSrv = &http.Server{
		Addr: fmt.Sprintf(":%d", int(port)),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			r.Body.Close()
			req := &httpRequest{
				method:      r.Method,
				path:        r.URL.Path,
				body:        string(body),
				contentType: r.Header.Get("Content-Type"),
				headers:     make(map[string]string),
				respC:       make(chan *httpResponse, 1),
			}
			for k := range r.Header {
				req.headers[k] = r.Header.Get(k)
			}
			requestCh <- req

			resp := <-req.respC
			for k, v := range resp.headers {
				w.Header().Set(k, v)
			}
			if resp.contentType != "" {
				w.Header().Set("Content-Type", resp.contentType)
			}
			w.WriteHeader(resp.status)
			w.Write([]byte(resp.body))
			handlerDoneCh <- struct{}{}
		}),
	}

	go listenSrv.ListenAndServe()

	currentReq = <-requestCh

	if statusCode != nil {
		*statusCode = 0
	}

	return 0
}

//export cob_http_respond_set_header
func cob_http_respond_set_header(name *C.char, val *C.char, valSize C.long) C.int {
	if pendingHeaders == nil {
		pendingHeaders = make(map[string]string)
	}
	n := C.GoString(name)
	v := strings.TrimRight(C.GoString(val), " \t")
	if v == "" {
		delete(pendingHeaders, n)
	} else {
		pendingHeaders[n] = v
	}
	return 0
}

//export cob_http_respond
func cob_http_respond(status C.int, body *C.char, contentType *C.char) C.int {
	req := currentReq
	if req == nil {
		return 1
	}

	resp := &httpResponse{
		status:  int(status),
		headers: pendingHeaders,
	}
	pendingHeaders = nil
	if body != nil {
		resp.body = C.GoString(body)
	}
	if contentType != nil {
		resp.contentType = C.GoString(contentType)
	}

	req.respC <- resp

	// Wait for handler to finish writing the HTTP response
	<-handlerDoneCh

	if listenSrv != nil {
		listenSrv.Close()
		listenSrv = nil
	}

	return 0
}

//export cob_http_request_field
func cob_http_request_field(fieldName *C.char, out *C.char, outSize C.long) C.int {
	req := currentReq
	if req == nil {
		return 1
	}

	rawName := C.GoString(fieldName)
	name := jsonKey(rawName)
	var val string

	// Use suffix matching so REQ-METHOD, REQ-PATH, REQ-BODY, REQUEST-BODY,
	// RESP-CONTENT-TYPE etc. all match the built-in properties.
	switch {
	case strings.HasSuffix(name, "method"):
		val = req.method
	case strings.HasSuffix(name, "path"):
		val = req.path
	case strings.HasSuffix(name, "body"):
		val = req.body
	case strings.HasSuffix(name, "contenttype"):
		val = req.contentType
	default:
		// Fall through to request headers
		if req.headers != nil {
			for hk, hv := range req.headers {
				if jsonKey(hk) == name {
					val = hv
					break
				}
			}
		}
		if val == "" {
			return 1
		}
	}

	if out != nil && outSize > 0 {
		size := int(outSize)
		buf := make([]byte, size)
		copy(buf, val)
		for i := len(val); i < size; i++ {
			buf[i] = ' '
		}
		C.memcpy(unsafe.Pointer(out), unsafe.Pointer(&buf[0]), C.size_t(size))
	}
	return 0
}

//export cob_picx_eq
func cob_picx_eq(a *C.char, aSize C.long, b *C.char) C.int {
	aStr := C.GoBytes(unsafe.Pointer(a), C.int(aSize))
	bStr := C.GoString(b)
	aTrimmed := strings.TrimRight(string(aStr), " ")
	if aTrimmed == bStr {
		return 1
	}
	return 0
}

func main() {}

type stringReader struct {
	s string
	n int
}

func (r *stringReader) Read(p []byte) (int, error) {
	if r.n >= len(r.s) {
		return 0, io.EOF
	}
	n := copy(p, r.s[r.n:])
	r.n += n
	return n, nil
}

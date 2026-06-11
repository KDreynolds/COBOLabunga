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
	bTrimmed := strings.TrimRight(bStr, " ")
	if aTrimmed == bTrimmed {
		return 1
	}
	return 0
}

// --- STRING runtime ---

var sState struct {
	dest     unsafe.Pointer
	destSize int64
	pos      int64
	overflow bool
}

//export cob_string_init
func cob_string_init(dest unsafe.Pointer, destSize C.long, pointer *C.long) {
	sState.dest = dest
	sState.destSize = int64(destSize)
	sState.pos = 1
	sState.overflow = false
	if pointer != nil {
		sState.pos = int64(*pointer)
	}
}

//export cob_string_add_size
func cob_string_add_size(src unsafe.Pointer, srcSize C.long) {
	if sState.overflow {
		sState.pos += int64(srcSize)
		return
	}
	n := int64(srcSize)
	dp := sState.pos - 1
	for i := int64(0); i < n; i++ {
		if dp >= sState.destSize {
			sState.overflow = true
			dp++
			break
		}
		b := *(*byte)(unsafe.Pointer(uintptr(src) + uintptr(i)))
		*(*byte)(unsafe.Pointer(uintptr(sState.dest) + uintptr(dp))) = b
		dp++
	}
	sState.pos = dp + 1
}

//export cob_string_add_until_space
func cob_string_add_until_space(src unsafe.Pointer, srcSize C.long) {
	if sState.overflow {
		sState.pos += int64(srcSize)
		return
	}
	n := int64(srcSize)
	dp := sState.pos - 1
	var i int64
	for i = 0; i < n; i++ {
		b := *(*byte)(unsafe.Pointer(uintptr(src) + uintptr(i)))
		if b == ' ' {
			i++ // skip the space
			break
		}
		if dp >= sState.destSize {
			sState.overflow = true
			i++
			break
		}
		*(*byte)(unsafe.Pointer(uintptr(sState.dest) + uintptr(dp))) = b
		dp++
	}
	sState.pos = dp + 1
}

//export cob_string_add_until_delim
func cob_string_add_until_delim(src unsafe.Pointer, srcSize C.long, delim unsafe.Pointer, delimSize C.long) {
	if sState.overflow {
		sState.pos += int64(srcSize)
		return
	}
	delimBytes := C.GoBytes(delim, C.int(delimSize))
	delimStr := string(delimBytes)
	n := int64(srcSize)
	dp := sState.pos - 1
	var i int64
	for i = 0; i < n; i++ {
		b := *(*byte)(unsafe.Pointer(uintptr(src) + uintptr(i)))
		// Check if current position matches delimiter start
		if byte(delimStr[0]) == b {
			// check remaining delimiter bytes
			match := true
			var j int64
			for j = 1; j < int64(len(delimStr)); j++ {
				if i+j >= n {
					break
				}
				cb := *(*byte)(unsafe.Pointer(uintptr(src) + uintptr(i+j)))
				if cb != delimStr[j] {
					match = false
					break
				}
			}
			if match && j == int64(len(delimStr)) {
				i += int64(len(delimStr))
				break
			}
		}
		if dp >= sState.destSize {
			sState.overflow = true
			i++
			break
		}
		*(*byte)(unsafe.Pointer(uintptr(sState.dest) + uintptr(dp))) = b
		dp++
	}
	sState.pos = dp + 1
}

//export cob_string_finish
func cob_string_finish(pointer *C.long, overflow *C.int) {
	if pointer != nil {
		*pointer = C.long(sState.pos)
	}
	if sState.overflow {
		*overflow = 1
	}
	sState.dest = nil
}

// --- UNSTRING runtime ---

var uState struct {
	src        unsafe.Pointer
	srcSize    int64
	pos        int64
	fieldIndex int
	tally      int64
	overflow   bool
}

//export cob_unstring_init
func cob_unstring_init(src unsafe.Pointer, srcSize C.long, pointer *C.long, tally *C.long) {
	uState.src = src
	uState.srcSize = int64(srcSize)
	uState.pos = 1
	uState.fieldIndex = 0
	uState.tally = 0
	uState.overflow = false
	if pointer != nil {
		uState.pos = int64(*pointer)
	}
}

//export cob_unstring_extract
func cob_unstring_extract(dest unsafe.Pointer, destSize C.long,
	delimIn *C.char, delimInSize C.long,
	countIn *C.long,
	delim1 unsafe.Pointer, delim1Size C.long,
	delim2 unsafe.Pointer, delim2Size C.long) C.int {

	if uState.overflow {
		return 0
	}

	srcSize := uState.srcSize
	pos := uState.pos - 1
	if pos >= srcSize {
		uState.overflow = true
		return 0
	}

	// Find the next delimiter in source starting at pos
	nextDelim := srcSize // position of next delimiter in source (0-based, after pos)
	var foundDelim []byte

	if delim1 != nil {
		d1 := C.GoBytes(delim1, C.int(delim1Size))
		idx := indexAt(unsafe.Pointer(uintptr(uState.src)+uintptr(pos)), srcSize-pos, d1)
		if idx >= 0 && pos+idx < nextDelim {
			nextDelim = pos + idx
			foundDelim = d1
		}
	}
	if delim2 != nil {
		d2 := C.GoBytes(delim2, C.int(delim2Size))
		idx := indexAt(unsafe.Pointer(uintptr(uState.src)+uintptr(pos)), srcSize-pos, d2)
		if idx >= 0 && pos+idx < nextDelim {
			nextDelim = pos + idx
			foundDelim = d2
		}
	}

	// Copy from pos to nextDelim into destination
	cpLen := nextDelim - pos
	if dest != nil && destSize > 0 {
		maxCopy := int64(destSize)
		if cpLen < maxCopy {
			maxCopy = cpLen
		}
		for i := int64(0); i < maxCopy; i++ {
			b := *(*byte)(unsafe.Pointer(uintptr(uState.src) + uintptr(pos+i)))
			*(*byte)(unsafe.Pointer(uintptr(dest) + uintptr(i))) = b
		}
		// Pad remaining with spaces
		for i := cpLen; i < int64(destSize); i++ {
			*(*byte)(unsafe.Pointer(uintptr(dest) + uintptr(i))) = ' '
		}
	}

	// Store delimiter in
	if delimIn != nil && foundDelim != nil {
		maxCopy := int64(delimInSize)
		if int64(len(foundDelim)) < maxCopy {
			maxCopy = int64(len(foundDelim))
		}
		for i := int64(0); i < maxCopy; i++ {
			*(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(delimIn)) + uintptr(i))) = foundDelim[i]
		}
		for i := int64(len(foundDelim)); i < int64(delimInSize); i++ {
			*(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(delimIn)) + uintptr(i))) = ' '
		}
	}

	// Store count
	if countIn != nil {
		*countIn = C.long(cpLen)
	}

	// Advance position past the extracted data and delimiter
	uState.pos = pos + cpLen + 1
	if foundDelim != nil {
		uState.pos = pos + cpLen + int64(len(foundDelim)) + 1
	}

	uState.tally++
	uState.fieldIndex++

	return 0
}

//export cob_unstring_finish
func cob_unstring_finish(pointer *C.long, tally *C.long, overflow *C.int) {
	if pointer != nil {
		*pointer = C.long(uState.pos)
	}
	if tally != nil {
		*tally = C.long(uState.tally)
	}
	if uState.overflow {
		*overflow = 1
	}
	uState.src = nil
}

func indexAt(buf unsafe.Pointer, bufLen int64, needle []byte) int64 {
	if len(needle) == 0 {
		return -1
	}
	needleLen := int64(len(needle))
	for i := int64(0); i <= bufLen-needleLen; i++ {
		match := true
		for j := int64(0); j < needleLen; j++ {
			b := *(*byte)(unsafe.Pointer(uintptr(buf) + uintptr(i+j)))
			if b != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
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

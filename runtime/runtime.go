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

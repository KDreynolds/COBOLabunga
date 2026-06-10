package main

/*
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"io"
	"net/http"
	"os"
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

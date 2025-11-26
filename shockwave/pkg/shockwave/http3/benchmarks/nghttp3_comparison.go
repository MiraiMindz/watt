package benchmarks

// Comparative benchmark against nghttp3
// This file provides utilities to benchmark Shockwave against nghttp3

/*
#cgo pkg-config: libnghttp3
#include <nghttp3/nghttp3.h>
#include <stdlib.h>
#include <string.h>

// Callback stubs for nghttp3
static int acked_stream_data(nghttp3_conn *conn, int64_t stream_id,
                             uint64_t datalen, void *user_data,
                             void *stream_user_data) {
    return 0;
}

static int stream_close(nghttp3_conn *conn, int64_t stream_id,
                       uint64_t app_error_code, void *conn_user_data,
                       void *stream_user_data) {
    return 0;
}

static int recv_data(nghttp3_conn *conn, int64_t stream_id,
                    const uint8_t *data, size_t datalen,
                    void *user_data, void *stream_user_data) {
    return 0;
}

static int recv_header(nghttp3_conn *conn, int64_t stream_id,
                      int32_t token, nghttp3_rcbuf *name,
                      nghttp3_rcbuf *value, uint8_t flags,
                      void *user_data, void *stream_user_data) {
    return 0;
}

// Helper to create nghttp3 connection
static nghttp3_conn* create_nghttp3_conn() {
    nghttp3_conn *conn;
    nghttp3_callbacks callbacks = {
        acked_stream_data,
        stream_close,
        recv_data,
        NULL, // deferred_consume
        NULL, // begin_headers
        recv_header,
        NULL, // end_headers
        NULL, // begin_trailers
        NULL, // end_trailers
        NULL, // stop_sending
        NULL, // end_stream
        NULL, // reset_stream
        NULL, // shutdown
        NULL, // recv_settings
    };

    nghttp3_settings settings;
    nghttp3_settings_default(&settings);

    nghttp3_conn_client_new(&conn, &callbacks, &settings, NULL, NULL);
    return conn;
}

// Helper to encode headers with nghttp3
static int nghttp3_encode_headers_helper(nghttp3_conn *conn, int64_t stream_id,
                                        const char **headers, size_t nheaders) {
    nghttp3_nv *nva = malloc(sizeof(nghttp3_nv) * nheaders);
    if (!nva) return -1;

    for (size_t i = 0; i < nheaders; i += 2) {
        nva[i/2].name = (uint8_t*)headers[i];
        nva[i/2].namelen = strlen(headers[i]);
        nva[i/2].value = (uint8_t*)headers[i+1];
        nva[i/2].valuelen = strlen(headers[i+1]);
        nva[i/2].flags = NGHTTP3_NV_FLAG_NONE;
    }

    int rv = nghttp3_conn_submit_request(conn, stream_id, nva, nheaders/2,
                                         NULL, NULL);
    free(nva);
    return rv;
}

static void destroy_nghttp3_conn(nghttp3_conn *conn) {
    nghttp3_conn_del(conn);
}
*/
import "C"

import (
	"unsafe"
)

// NgHttp3Encoder wraps nghttp3 for benchmarking
type NgHttp3Encoder struct {
	conn *C.nghttp3_conn
}

// NewNgHttp3Encoder creates a new nghttp3 encoder for benchmarking
func NewNgHttp3Encoder() *NgHttp3Encoder {
	return &NgHttp3Encoder{
		conn: C.create_nghttp3_conn(),
	}
}

// EncodeHeaders encodes headers using nghttp3
func (e *NgHttp3Encoder) EncodeHeaders(headers map[string]string) error {
	// Convert Go headers to C array
	cHeaders := make([]*C.char, 0, len(headers)*2)
	for name, value := range headers {
		cHeaders = append(cHeaders, C.CString(name))
		cHeaders = append(cHeaders, C.CString(value))
	}
	defer func() {
		for _, ch := range cHeaders {
			C.free(unsafe.Pointer(ch))
		}
	}()

	// Call nghttp3
	rv := C.nghttp3_encode_headers_helper(e.conn, 0, &cHeaders[0], C.size_t(len(cHeaders)))
	if rv != 0 {
		return ErrEncodingFailed
	}

	return nil
}

// Close cleans up the nghttp3 connection
func (e *NgHttp3Encoder) Close() {
	if e.conn != nil {
		C.destroy_nghttp3_conn(e.conn)
		e.conn = nil
	}
}

var ErrEncodingFailed = &encodingError{}

type encodingError struct{}

func (e *encodingError) Error() string {
	return "nghttp3: encoding failed"
}

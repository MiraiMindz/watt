package http11

import (
	"testing"
)

// Map-based implementation for comparison
var statusLinesMap = map[int][]byte{
	// 1xx Informational
	100: []byte("HTTP/1.1 100 Continue\r\n"),
	101: []byte("HTTP/1.1 101 Switching Protocols\r\n"),

	// 2xx Success
	200: []byte("HTTP/1.1 200 OK\r\n"),
	201: []byte("HTTP/1.1 201 Created\r\n"),
	202: []byte("HTTP/1.1 202 Accepted\r\n"),
	203: []byte("HTTP/1.1 203 Non-Authoritative Information\r\n"),
	204: []byte("HTTP/1.1 204 No Content\r\n"),
	205: []byte("HTTP/1.1 205 Reset Content\r\n"),
	206: []byte("HTTP/1.1 206 Partial Content\r\n"),

	// 3xx Redirection
	300: []byte("HTTP/1.1 300 Multiple Choices\r\n"),
	301: []byte("HTTP/1.1 301 Moved Permanently\r\n"),
	302: []byte("HTTP/1.1 302 Found\r\n"),
	303: []byte("HTTP/1.1 303 See Other\r\n"),
	304: []byte("HTTP/1.1 304 Not Modified\r\n"),
	307: []byte("HTTP/1.1 307 Temporary Redirect\r\n"),
	308: []byte("HTTP/1.1 308 Permanent Redirect\r\n"),

	// 4xx Client Error
	400: []byte("HTTP/1.1 400 Bad Request\r\n"),
	401: []byte("HTTP/1.1 401 Unauthorized\r\n"),
	403: []byte("HTTP/1.1 403 Forbidden\r\n"),
	404: []byte("HTTP/1.1 404 Not Found\r\n"),
	405: []byte("HTTP/1.1 405 Method Not Allowed\r\n"),
	406: []byte("HTTP/1.1 406 Not Acceptable\r\n"),
	408: []byte("HTTP/1.1 408 Request Timeout\r\n"),
	409: []byte("HTTP/1.1 409 Conflict\r\n"),
	410: []byte("HTTP/1.1 410 Gone\r\n"),
	411: []byte("HTTP/1.1 411 Length Required\r\n"),
	412: []byte("HTTP/1.1 412 Precondition Failed\r\n"),
	413: []byte("HTTP/1.1 413 Payload Too Large\r\n"),
	414: []byte("HTTP/1.1 414 URI Too Long\r\n"),
	415: []byte("HTTP/1.1 415 Unsupported Media Type\r\n"),
	429: []byte("HTTP/1.1 429 Too Many Requests\r\n"),

	// 5xx Server Error
	500: []byte("HTTP/1.1 500 Internal Server Error\r\n"),
	501: []byte("HTTP/1.1 501 Not Implemented\r\n"),
	502: []byte("HTTP/1.1 502 Bad Gateway\r\n"),
	503: []byte("HTTP/1.1 503 Service Unavailable\r\n"),
	504: []byte("HTTP/1.1 504 Gateway Timeout\r\n"),
}

// getStatusLineMap is the map-based implementation
func getStatusLineMap(code int) []byte {
	if line, ok := statusLinesMap[code]; ok {
		return line
	}
	// Fallback for uncommon codes
	return buildStatusLine(code)
}

// Benchmark current switch-based implementation
func BenchmarkStatusLine_Switch_200(b *testing.B) {
	b.ReportAllocs()
	var result []byte
	for i := 0; i < b.N; i++ {
		result = getStatusLine(200)
	}
	_ = result
}

func BenchmarkStatusLine_Switch_404(b *testing.B) {
	b.ReportAllocs()
	var result []byte
	for i := 0; i < b.N; i++ {
		result = getStatusLine(404)
	}
	_ = result
}

func BenchmarkStatusLine_Switch_500(b *testing.B) {
	b.ReportAllocs()
	var result []byte
	for i := 0; i < b.N; i++ {
		result = getStatusLine(500)
	}
	_ = result
}

func BenchmarkStatusLine_Switch_UncommonCode(b *testing.B) {
	b.ReportAllocs()
	var result []byte
	for i := 0; i < b.N; i++ {
		result = getStatusLine(418) // I'm a teapot
	}
	_ = result
}

// Benchmark map-based implementation
func BenchmarkStatusLine_Map_200(b *testing.B) {
	b.ReportAllocs()
	var result []byte
	for i := 0; i < b.N; i++ {
		result = getStatusLineMap(200)
	}
	_ = result
}

func BenchmarkStatusLine_Map_404(b *testing.B) {
	b.ReportAllocs()
	var result []byte
	for i := 0; i < b.N; i++ {
		result = getStatusLineMap(404)
	}
	_ = result
}

func BenchmarkStatusLine_Map_500(b *testing.B) {
	b.ReportAllocs()
	var result []byte
	for i := 0; i < b.N; i++ {
		result = getStatusLineMap(500)
	}
	_ = result
}

func BenchmarkStatusLine_Map_UncommonCode(b *testing.B) {
	b.ReportAllocs()
	var result []byte
	for i := 0; i < b.N; i++ {
		result = getStatusLineMap(418) // I'm a teapot
	}
	_ = result
}

// Benchmark realistic distribution (80% 200, 10% 404, 5% 500, 5% others)
func BenchmarkStatusLine_Switch_Realistic(b *testing.B) {
	b.ReportAllocs()
	codes := []int{200, 200, 200, 200, 200, 200, 200, 200, 404, 500, 301, 204, 400, 503, 200, 200, 200, 404, 500, 200}
	var result []byte
	for i := 0; i < b.N; i++ {
		result = getStatusLine(codes[i%len(codes)])
	}
	_ = result
}

func BenchmarkStatusLine_Map_Realistic(b *testing.B) {
	b.ReportAllocs()
	codes := []int{200, 200, 200, 200, 200, 200, 200, 200, 404, 500, 301, 204, 400, 503, 200, 200, 200, 404, 500, 200}
	var result []byte
	for i := 0; i < b.N; i++ {
		result = getStatusLineMap(codes[i%len(codes)])
	}
	_ = result
}

// Benchmark worst case - hitting every code sequentially
func BenchmarkStatusLine_Switch_Sequential(b *testing.B) {
	b.ReportAllocs()
	codes := []int{100, 101, 200, 201, 202, 203, 204, 205, 206, 300, 301, 302, 303, 304, 307, 308, 400, 401, 403, 404, 405, 406, 408, 409, 410, 411, 412, 413, 414, 415, 429, 500, 501, 502, 503, 504}
	var result []byte
	for i := 0; i < b.N; i++ {
		result = getStatusLine(codes[i%len(codes)])
	}
	_ = result
}

func BenchmarkStatusLine_Map_Sequential(b *testing.B) {
	b.ReportAllocs()
	codes := []int{100, 101, 200, 201, 202, 203, 204, 205, 206, 300, 301, 302, 303, 304, 307, 308, 400, 401, 403, 404, 405, 406, 408, 409, 410, 411, 412, 413, 414, 415, 429, 500, 501, 502, 503, 504}
	var result []byte
	for i := 0; i < b.N; i++ {
		result = getStatusLineMap(codes[i%len(codes)])
	}
	_ = result
}

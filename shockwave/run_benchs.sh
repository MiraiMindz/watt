#!/bin/bash
set -e

echo "Starting Shockwave comprehensive benchmarks..."

# HTTP/1.1 benchmarks
go test -bench=. -benchmem -cpu=1,2,4,8 -count=5 -benchtime=5s -timeout=3h -cpuprofile=cpu_http11.out -memprofile=mem_http11.out -trace=trace_http11.out ./pkg/shockwave/http11 | tee shockwave_http11_benchmarks.result

# HTTP/2 benchmarks
go test -bench=. -benchmem -cpu=1,2,4,8 -count=5 -benchtime=5s -timeout=3h -cpuprofile=cpu_http2.out -memprofile=mem_http2.out -trace=trace_http2.out ./pkg/shockwave/http2 | tee shockwave_http2_benchmarks.result

# HTTP/3 benchmarks (main)
go test -bench=. -benchmem -cpu=1,2,4,8 -count=5 -benchtime=5s -timeout=3h -cpuprofile=cpu_http3.out -memprofile=mem_http3.out -trace=trace_http3.out ./pkg/shockwave/http3 | tee shockwave_http3_benchmarks.result

# HTTP/3 QPACK benchmarks
go test -bench=. -benchmem -cpu=1,2,4,8 -count=5 -benchtime=5s -timeout=3h -cpuprofile=cpu_qpack.out -memprofile=mem_qpack.out -trace=trace_qpack.out ./pkg/shockwave/http3/qpack | tee shockwave_qpack_benchmarks.result

# HTTP/3 QUIC benchmarks
go test -bench=. -benchmem -cpu=1,2,4,8 -count=5 -benchtime=5s -timeout=3h -cpuprofile=cpu_quic.out -memprofile=mem_quic.out -trace=trace_quic.out ./pkg/shockwave/http3/quic | tee shockwave_quic_benchmarks.result

# HTTP/3 comparison benchmarks
go test -bench=. -benchmem -cpu=1,2,4,8 -count=5 -benchtime=5s -timeout=3h -cpuprofile=cpu_http3_comp.out -memprofile=mem_http3_comp.out -trace=trace_http3_comp.out ./pkg/shockwave/http3/benchmarks | tee shockwave_http3_comparison_benchmarks.result

# Server benchmarks
go test -bench=. -benchmem -cpu=1,2,4,8 -count=5 -benchtime=5s -timeout=3h -cpuprofile=cpu_server.out -memprofile=mem_server.out -trace=trace_server.out ./pkg/shockwave/server | tee shockwave_server_benchmarks.result

# Client benchmarks
go test -bench=. -benchmem -cpu=1,2,4,8 -count=5 -benchtime=5s -timeout=3h -cpuprofile=cpu_client.out -memprofile=mem_client.out -trace=trace_client.out ./pkg/shockwave/client | tee shockwave_client_benchmarks.result

# Memory management benchmarks
go test -bench=. -benchmem -cpu=1,2,4,8 -count=5 -benchtime=5s -timeout=3h -cpuprofile=cpu_memory.out -memprofile=mem_memory.out -trace=trace_memory.out ./pkg/shockwave/memory | tee shockwave_memory_benchmarks.result

# Socket benchmarks
go test -bench=. -benchmem -cpu=1,2,4,8 -count=5 -benchtime=5s -timeout=3h -cpuprofile=cpu_socket.out -memprofile=mem_socket.out -trace=trace_socket.out ./pkg/shockwave/socket | tee shockwave_socket_benchmarks.result

# Buffer pool benchmarks
go test -bench=. -benchmem -cpu=1,2,4,8 -count=5 -benchtime=5s -timeout=3h -cpuprofile=cpu_bufpool.out -memprofile=mem_bufpool.out -trace=trace_bufpool.out ./pkg/shockwave | tee shockwave_bufpool_benchmarks.result

# Comprehensive benchmark (root level)
go test -bench=. -benchmem -cpu=1,2,4,8 -count=5 -benchtime=5s -timeout=3h -cpuprofile=cpu_comprehensive.out -memprofile=mem_comprehensive.out -trace=trace_comprehensive.out ./comprehensive_benchmark_test.go | tee shockwave_comprehensive_benchmarks.result

# All combined (no profiles)
go test -bench=. -benchmem -cpu=1,2,4,8 -count=5 -benchtime=5s -timeout=3h ./pkg/shockwave/... | tee shockwave_all_benchmarks.result

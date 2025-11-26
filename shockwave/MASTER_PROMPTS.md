# Master Prompts for Full Shockwave Development

This document contains the strategic prompts to fully develop Shockwave using the entire Claude Code ecosystem.

## 🎯 Phase 0: Foundation & Competitive Benchmarking

### Prompt 0.1: Create Competitor Benchmark Suite

```
I need to establish baseline benchmarks against our competitors before we start building Shockwave.

**Your Task:**
1. Create a comprehensive benchmark suite in `benchmarks/competitors/` that measures:
   - **net/http** (stdlib): Simple HTTP server performance
   - **valyala/fasthttp**: Their fastest configuration
   - **gorilla/websocket**: WebSocket throughput and latency

2. For each competitor, benchmark:
   - Simple GET request throughput (req/sec)
   - Request parsing latency (ns/op)
   - Memory allocations (B/op, allocs/op)
   - Keep-alive connection reuse
   - Concurrent request handling
   - Large request/response bodies

3. Create comparison test file: `benchmarks/competitors/comparison_test.go`

4. Run all benchmarks and save results to `results/competitors_baseline.txt`

5. Generate a report showing:
   - Performance gaps we need to close
   - Areas where we can exceed their performance
   - Specific targets for Shockwave

**Use:**
- The `benchmark-analysis` skill to design proper benchmarks
- Statistical significance (count=10, benchtime=3s)
- Save results for future comparison

**Deliverables:**
- Benchmark suite code
- Initial results
- Target performance metrics
- Report showing what we need to beat

This becomes our performance contract.
```

---

## 🎯 Phase 1: Core HTTP/1.1 Engine

### Prompt 1.1: Plan HTTP/1.1 Implementation

```
**Strategic Planning Prompt:**

Use the **Plan** agent to create a detailed implementation plan for the HTTP/1.1 engine.

The agent should:

1. **Analyze the architecture** from CLAUDE.md and implementation_analysis.md
2. **Design the zero-allocation parser** following these principles:
   - Inline header array (max 32 headers)
   - State machine-based incremental parsing
   - No string allocations (use []byte views)
   - Pre-compiled constants for common values
3. **Define the file structure** in `pkg/shockwave/http11/`:
   - parser.go - Request parsing
   - response.go - Response writing
   - connection.go - Connection management
   - constants.go - Pre-compiled values
   - pool.go - Object pooling
4. **Create the implementation sequence**:
   - What to build first
   - Dependencies between components
   - Validation at each step
5. **Define success criteria**:
   - 0 allocs/op for parsing
   - Faster than net/http
   - RFC 7230 compliant

**Output needed:**
- Detailed implementation plan
- File-by-file breakdown
- Test strategy
- Benchmark strategy

Once I approve the plan, we'll proceed to implementation.
```

### Prompt 1.2: Implement Zero-Allocation Parser

```
**Implementation Prompt:**

Based on the approved plan, implement the HTTP/1.1 zero-allocation parser.

**Requirements:**
1. Follow CLAUDE.md principles strictly
2. Use the `go-performance-optimization` skill for guidance
3. Target: 0 allocs/op for requests with ≤32 headers

**File: pkg/shockwave/http11/parser.go**

Implement:
- Request struct with inline arrays
- ParseRequest() function with state machine
- Zero-copy string views into buffer
- Method lookup using constants

**File: pkg/shockwave/http11/constants.go**

Implement:
- Pre-compiled HTTP versions
- Method constants and lookup
- Common header names

**After implementation:**
1. Run `/check-allocs` to verify zero allocations
2. Run `/profile-mem` to confirm no escapes
3. Run escape analysis: `go build -gcflags="-m -m" ./pkg/shockwave/http11`

**Validation:**
- Create benchmark in `pkg/shockwave/http11/parser_test.go`
- Compare against net/http parser
- Verify 0 allocs/op

Use TodoWrite to track implementation progress for each file.
```

### Prompt 1.3: Implement Response Writer

```
**Implementation Prompt:**

Implement the zero-allocation response writer.

**Requirements:**
1. Pooled buffer management
2. Pre-compiled status lines
3. Header writing without allocation
4. Chunked encoding support

**Files to create:**
- pkg/shockwave/http11/response.go
- pkg/shockwave/http11/pool.go
- pkg/shockwave/http11/response_test.go

**After implementation:**
1. `/check-allocs` - Verify status line writing is 0 allocs
2. `/bench` - Compare with net/http response writing
3. `/profile-cpu` - Ensure no hot spots

**Success criteria:**
- BenchmarkWriteResponse: 0 allocs/op
- Faster than net/http
- Proper pooling with >95% hit rate
```

### Prompt 1.4: Connection & Keep-Alive

```
**Implementation Prompt:**

Implement connection handling with keep-alive support.

**Critical requirement:**
Keep-alive connection reuse MUST be 0 allocs/op.

**Files:**
- pkg/shockwave/http11/connection.go
- pkg/shockwave/http11/conn_reader.go (thread-safe reader)

**Implement:**
1. Connection state tracking
2. Keep-alive timeout handling
3. Thread-safe connection reader (prevents "lost first byte" bug)
4. Graceful shutdown

**After implementation:**
1. `/check-allocs` - Verify BenchmarkKeepAlive shows 0 allocs/op
2. Create integration test for keep-alive
3. Test against curl with keep-alive

**Validation:**
Compare 10,000 sequential keep-alive requests vs net/http:
- Throughput
- Allocations
- Latency
```

### Prompt 1.5: HTTP/1.1 Server

```
**Implementation Prompt:**

Implement the main HTTP/1.1 server with multiple allocation modes.

**Files:**
- pkg/shockwave/server/server.go
- pkg/shockwave/server/server_shockwave.go (custom implementation)
- pkg/shockwave/memory/arena.go (arena allocation)
- pkg/shockwave/pool_greentea.go (Green Tea GC)

**Server must support:**
1. Standard pooling mode (default)
2. Arena allocation mode (experimental)
3. Green Tea GC mode
4. Configurable via build tags

**After implementation:**
1. Run benchmarks with all three modes
2. Compare arena vs standard vs green tea
3. `/bench` to compare with net/http
4. Generate performance report

**Success criteria:**
- All modes functional
- Arena mode shows <0.1% GC time
- Faster than net/http in all modes

Arenas and GreenTeaGC are enabled via BuildFlags
```

### Prompt 1.6: Phase 1 Validation

```
**Comprehensive Validation Prompt:**

Use the **performance-auditor** agent to conduct a full audit of the HTTP/1.1 implementation.

The agent should:
1. Review all HTTP/1.1 code for anti-patterns
2. Run escape analysis on all files
3. Execute comprehensive benchmarks
4. Compare against net/http baseline
5. Verify all critical paths are 0 allocs/op
6. Check for memory leaks
7. Validate RFC 7230 compliance

Then use the **protocol-validator** agent to:
1. Check RFC 7230-7235 compliance
2. Test edge cases
3. Validate security (request smuggling, etc.)
4. Run interoperability tests

**Deliverables:**
- Performance audit report
- Protocol compliance report
- Comparison vs net/http showing improvements
- Any issues to fix before Phase 2

Run `/compare-nethttp` to generate final comparison report.
```

---

## 🎯 Phase 2: Performance Optimizations

### Prompt 2.1: Socket-Level Optimizations

```
**Implementation Prompt:**

Implement Linux-specific socket optimizations.

**Files:**
- pkg/shockwave/socket/tuning_linux.go
- pkg/shockwave/socket/tuning_darwin.go
- pkg/shockwave/socket/tuning.go (cross-platform)

**Implement:**
1. TCP_QUICKACK - Immediate ACKs
2. TCP_DEFER_ACCEPT - Don't wake until data
3. TCP_FASTOPEN - TFO support
4. Zero-copy sendfile for large responses
5. TCP_NODELAY - Disable Nagle
6. SO_RCVBUF/SO_SNDBUF tuning

**Testing:**
1. Benchmark latency before/after TCP_QUICKACK
2. Measure throughput improvement from TFO
3. Test sendfile vs userspace copy for files
4. `/bench` to measure impact

**Expected:**
- 20-30% latency reduction (TCP_QUICKACK + TFO)
- 70% CPU reduction for file serving (sendfile)
```

### Prompt 2.2: Advanced Memory Management

```
**Implementation Prompt:**

Implement arena allocation and Green Tea GC optimizations.

**Arena Allocation (GOEXPERIMENT=arenas):**
- pkg/shockwave/memory/arena.go
- pkg/shockwave/pool_arena.go
- Bulk deallocation of request-lifetime objects
- Zero GC pressure

**Green Tea GC:**
- pkg/shockwave/pool_greentea.go
- Spatial locality optimization
- Temporal locality grouping
- Batch allocation

**Testing:**
1. Run benchmarks with each mode
2. Measure GC time with GODEBUG=gctrace=1
3. Compare:
   - Standard: ~2% GC time
   - Green Tea: ~1% GC time
   - Arena: <0.1% GC time

**Validation:**
Create report showing:
- GC overhead for each mode
- Throughput comparison
- When to use each mode

Use `/profile-mem` and `memory-profiling` skill for analysis.
```

### Prompt 2.3: Buffer Pool Optimization

```
**Implementation Prompt:**

Optimize buffer management with size-specific pools.

**File:** pkg/shockwave/buffer_pool.go

**Implement:**
1. Size-specific pools (2KB, 4KB, 8KB)
2. Pool hit rate tracking
3. Automatic buffer reuse
4. Reset on return

**After implementation:**
1. Add metrics to track pool efficiency
2. Run load test and measure hit rate
3. Target: >95% hit rate
4. `/profile-mem` to verify pool effectiveness

**Optimization:**
Use the `go-performance-optimization` skill to ensure:
- No allocations on pool hit
- Proper reset before Put
- Balanced Get/Put calls
```

---

## 🎯 Phase 3: HTTP/2 Implementation

### Prompt 3.1: Plan HTTP/2 Architecture

```
**Planning Prompt:**

Use the **Plan** agent to design the HTTP/2 implementation.

Requirements:
1. RFC 7540 compliance
2. Stream multiplexing
3. HPACK compression
4. Flow control
5. Server push support

**Agent should deliver:**
- File structure for pkg/shockwave/http2/
- Frame parsing strategy
- HPACK implementation approach
- Stream state management
- Flow control algorithm
- Integration with HTTP/1.1 server (ALPN)

**Key files:**
- frame.go - Frame parsing
- hpack.go - HPACK compression
- connection.go - HTTP/2 connection
- stream.go - Stream management
- flow_control.go - Flow control

Once approved, proceed to implementation.
```

### Prompt 3.2: Implement Frame Parser

```
**Implementation Prompt:**

Implement HTTP/2 frame parser following RFC 7540 Section 4.

**File:** pkg/shockwave/http2/frame.go

**Implement all frame types:**
- DATA, HEADERS, PRIORITY
- RST_STREAM, SETTINGS, PUSH_PROMISE
- PING, GOAWAY, WINDOW_UPDATE
- CONTINUATION

**Requirements:**
1. Zero-allocation for frame header parsing
2. Validation per RFC 7540
3. Frame size limits enforced
4. Stream ID validation

**Testing:**
1. `/test-protocol` for RFC compliance
2. Create test vectors from RFC 7540
3. Test malformed frames
4. Benchmark frame parsing speed

Use `http-protocol-testing` skill for RFC guidance.
```

### Prompt 3.3: Implement HPACK Compression

```
**Implementation Prompt:**

Implement HPACK header compression (RFC 7541).

**File:** pkg/shockwave/http2/hpack.go

**Implement:**
1. Static table (Appendix A)
2. Dynamic table management
3. Huffman encoding
4. Compression ratio tracking

**Testing:**
1. Test against RFC 7541 examples
2. Measure compression ratio
3. Target: 30-80% size reduction
4. Benchmark encode/decode speed

**Validation:**
1. `/test-protocol` for RFC 7541 compliance
2. Compare compression ratio with net/http
3. Ensure no security issues (CRIME/BREACH)
```

### Prompt 3.4: Stream Multiplexing & Flow Control

```
**Implementation Prompt:**

Implement stream multiplexing and flow control.

**Files:**
- pkg/shockwave/http2/stream.go
- pkg/shockwave/http2/flow_control.go
- pkg/shockwave/http2/connection.go

**Requirements:**
1. Concurrent stream handling
2. Stream priority
3. Per-stream flow control
4. Per-connection flow control
5. WINDOW_UPDATE handling

**Testing:**
1. Test with h2load for concurrent streams
2. Benchmark concurrent stream performance
3. Validate flow control prevents overflow
4. Test priority handling

**Success criteria:**
- Handle 100+ concurrent streams
- 50-70% better throughput vs HTTP/1.1
- Proper flow control (no deadlocks)
```

### Prompt 3.5: HTTP/2 Validation

```
**Comprehensive Validation:**

1. Use **protocol-validator** agent to check RFC 7540 compliance
2. Run h2load benchmarks:
   ```bash
   h2load -n 10000 -c 100 https://localhost:8443
   ```
3. Compare vs net/http HTTP/2
4. `/bench` for comprehensive metrics
5. Test interoperability with various clients

**Deliverables:**
- RFC 7540 compliance report
- Performance comparison vs net/http
- Interoperability test results
- Any issues found and fixed
```

<!--
☐ Optimize HPACK decoding with pre-allocation
☐ Implement sharded stream map
☐ Implement buffer pooling for stream I/O
☐ Add comprehensive tests for security features
☐ Run benchmarks and compare performance
☐ Update validation report
    ## Optional Future Enhancements

     While the implementation is **production-ready**, these optional enhancements could provide additional benefits:

     ### 1. Buffer Pooling with sync.Pool
     - **Effort**: 2-3 hours
     - **Impact**: Additional 50-70% reduction in allocations
     - **Priority**: Low (current allocations already minimal)

     ### 2. Concurrent Stream Creation Benchmark
     - **Effort**: 1 hour
     - **Impact**: Validate actual speedup from sharded map
     - **Priority**: Medium (validation of optimization)

     ### 3. h2load Real-World Benchmarking
     - **Effort**: 2-3 hours (after Phase 4 server implementation)
     - **Impact**: Real-world performance comparison vs net/http
     - **Priority**: Medium (requires HTTP/2 server implementation)

     ### 4. Arena Allocator for Large Header Sets
     - **Effort**: 4-6 hours (requires GOEXPERIMENT=arenas)
     - **Impact**: Faster GC for high-request-rate scenarios
     - **Priority**: Low (specialized use cases only)

     ---

-->

---

## 🎯 Phase 4: WebSocket Implementation

### Prompt 4.1: WebSocket Implementation

```
**Implementation Prompt:**

Implement RFC 6455 compliant WebSocket support.

**Files:**
- pkg/shockwave/websocket/upgrade.go
- pkg/shockwave/websocket/conn.go
- pkg/shockwave/websocket/protocol.go
- pkg/shockwave/websocket/frame.go

**Implement:**
1. Upgrade handshake with key validation
2. Frame parsing (text, binary, control)
3. Masking (client→server must mask)
4. Ping/Pong/Close control frames
5. Fragmentation support

**Requirements:**
1. RFC 6455 compliance
2. Zero-copy frame parsing where possible
3. Efficient masking algorithm

**Testing:**
1. Test with websocat
2. Benchmark vs gorilla/websocket
3. `/test-protocol` for RFC 6455 compliance
4. Test all frame types

**Success criteria:**
- 30%+ faster than gorilla/websocket
- 50% less memory usage
- 100% RFC 6455 compliant
```

---

## 🎯 Phase 5-8: Advanced Features

### Prompt 5.1: HTTP/3 & QUIC

```
Plan and implement HTTP/3 over QUIC (RFC 9114).

Use **Plan** agent to design architecture, then implement:
- QUIC transport
- 0-RTT support
- QPACK compression
- Unreliable datagrams

Test with QUIC-capable clients.
```

### Prompt 6.1: TLS & ACME

```
Implement TLS support with automatic Let's Encrypt certificates.

Files:
- pkg/shockwave/tls/acme.go
- pkg/shockwave/tls/config.go
- pkg/shockwave/tls/cert.go

Implement auto-renewal and staging support.
```

### Prompt 7.1: HTTP Client

```
Implement high-performance HTTP client with connection pooling.

Must support:
- HTTP/1.1, HTTP/2, HTTP/3
- Connection pooling with health checks
- Keep-alive
- Timeouts

Benchmark vs net/http client.
```

### Prompt 8.1: Production Readiness

```
**Final Validation:**

Use ALL agents in parallel:

1. **performance-auditor**: Full codebase audit
2. **protocol-validator**: Complete RFC compliance check
3. **benchmark-runner**: Comprehensive performance suite

Then:
1. Run fuzzing: `go test -fuzz=. -fuzztime=1h`
2. Security audit
3. Generate final benchmark report vs all competitors
4. Create examples and documentation

**Final deliverable:**
Production-ready library with proof of performance superiority.
```

---

## 🎯 Ultimate Meta-Prompt (Run Everything)

```
**FULL SHOCKWAVE DEVELOPMENT - AUTONOMOUS MODE**

I want you to fully implement the Shockwave HTTP library using the entire Claude Code ecosystem.

**Context:**
- Read CLAUDE.md for project philosophy
- Read implementation_analysis.md for planned features
- Read IMPLEMENTATION_ROADMAP.md for phases
- Read this file (MASTER_PROMPTS.md) for the plan

**Strategy:**
1. Execute Phase 0 first: Create competitor benchmarks
2. For each subsequent phase:
   - Use Plan agent to design architecture
   - Implement following CLAUDE.md principles
   - Skills will activate automatically for guidance
   - Run commands (/bench, /check-allocs, etc.) after each component
   - Use agents for validation at phase completion
   - Hooks will enforce quality automatically

**Phases to complete:**
- [ ] Phase 0: Competitor benchmarks and targets
- [ ] Phase 1: HTTP/1.1 engine (zero-allocation)
- [ ] Phase 2: Performance optimizations
- [ ] Phase 3: HTTP/2 implementation
- [ ] Phase 4: WebSocket support
- [ ] Phase 5: HTTP/3 & QUIC
- [ ] Phase 6: TLS & ACME
- [ ] Phase 7: HTTP client
- [ ] Phase 8: Production readiness

**Validation at each phase:**
1. Run performance-auditor agent
2. Run protocol-validator agent
3. Run benchmark-runner agent
4. Compare vs competitors
5. Fix all issues before next phase

**Success criteria:**
- Beat net/http by 70% throughput
- Beat fasthttp by 20% throughput
- Beat gorilla/websocket by 30% throughput
- Zero allocations in all hot paths
- 100% RFC compliance
- 80%+ test coverage
- Production-ready code

**Working mode:**
- Use TodoWrite to track progress within each phase
- Update IMPLEMENTATION_ROADMAP.md as phases complete
- Save all benchmark results to results/
- Generate reports after each phase

**Final deliverable:**
A complete, production-grade HTTP library with comprehensive benchmarks proving it's the fastest Go HTTP library available.

Begin with Phase 0: Create the competitor benchmark suite.
```

---

## 💡 How to Use These Prompts

### Sequential Approach (Recommended)
1. Start with Phase 0 prompt
2. Review and approve agent plans
3. Execute implementation prompts one by one
4. Run validation prompts after each component
5. Move to next phase only after validation passes

### Autonomous Approach (Advanced)
Use the "Ultimate Meta-Prompt" and let Claude work through all phases with agent coordination.

### Iterative Approach
Focus on one phase at a time, iterate until perfect, then move forward.

---

## 🎯 Expected Timeline

- **Phase 0**: 1 session (competitor benchmarks)
- **Phase 1**: 3-5 sessions (HTTP/1.1 core)
- **Phase 2**: 2-3 sessions (optimizations)
- **Phase 3**: 3-4 sessions (HTTP/2)
- **Phase 4**: 2 sessions (WebSocket)
- **Phase 5**: 3-4 sessions (HTTP/3)
- **Phase 6**: 1-2 sessions (TLS)
- **Phase 7**: 2-3 sessions (Client)
- **Phase 8**: 2-3 sessions (Production)

**Total**: ~20-30 focused sessions for complete implementation.

---

**Remember:** The ecosystem is designed to work together. Skills activate automatically, hooks enforce quality, agents provide deep analysis, and commands give quick validation. Trust the system!

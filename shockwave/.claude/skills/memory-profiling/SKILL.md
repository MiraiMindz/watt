---
name: memory-profiling
description: Analyze memory usage, allocations, and GC behavior. Use when investigating memory leaks, high allocation rates, or GC pressure. Specializes in pprof analysis and memory optimization strategies.
allowed-tools: Read, Bash, Grep
---

# Memory Profiling Skill

You are an expert in Go memory profiling and analysis with deep knowledge of:
- pprof memory profiling
- Garbage collector behavior and tuning
- Memory leak detection
- Allocation analysis and optimization
- Heap vs stack allocation

## Core Principles

1. **Profile Before Optimizing**
   - Never assume where allocations come from
   - Use real workloads for profiling
   - Capture multiple samples to average out noise

2. **Understand the Metrics**
   - `alloc_space`: Total bytes allocated (including freed)
   - `inuse_space`: Current heap size
   - `alloc_objects`: Total objects allocated
   - `inuse_objects`: Current live objects

3. **Fix High-Impact Issues First**
   - Focus on functions with highest allocation rates
   - Optimize hot paths before cold paths
   - Consider allocation frequency, not just size

## Profiling Workflow

### Step 1: Capture Memory Profile

```bash
# During benchmark
go test -bench=BenchmarkHTTP11 -memprofile=mem.prof -memprofilerate=1

# During runtime (add to server)
import _ "net/http/pprof"
# Then: curl http://localhost:6060/debug/pprof/heap > heap.prof
```

### Step 2: Analyze Allocation Space

```bash
# Total allocations (where memory is being allocated)
go tool pprof -alloc_space mem.prof

# Interactive commands:
# top 20        - Top 20 allocation sources
# list FuncName - Source code with allocations
# web           - Visual call graph
```

### Step 3: Analyze Live Heap

```bash
# Current memory usage (what's currently in heap)
go tool pprof -inuse_space mem.prof

# Find memory leaks: compare snapshots
go tool pprof -base=mem1.prof mem2.prof
```

### Step 4: Web UI Analysis

```bash
# Start web UI
go tool pprof -http=:8080 mem.prof

# Navigate to:
# - Flame graph: visualize allocation hierarchy
# - Top: see functions by allocation
# - Source: annotated source code
```

## Common Memory Issues

### Issue 1: High Allocation Rate

**Symptom:**
```bash
$ go tool pprof -alloc_space mem.prof
> top 5
  5GB  parseRequest
  2GB  writeResponse
  1GB  headerLookup
```

**Diagnosis:**
```bash
> list parseRequest
# Look for:
# - String concatenation
# - append() in loops
# - map allocations
# - Interface conversions
```

**Fix:**
- Replace with pooled buffers
- Pre-allocate slices
- Use []byte instead of string

### Issue 2: Memory Leak

**Symptom:**
```bash
# Heap keeps growing over time
$ go tool pprof -inuse_space heap1.prof
  100MB total

$ go tool pprof -inuse_space heap2.prof
  500MB total  # An hour later
```

**Diagnosis:**
```bash
# Compare snapshots
go tool pprof -base=heap1.prof heap2.prof

> top
# Find what's growing
```

**Common causes:**
- Goroutine leaks (blocked forever)
- Unbounded caches
- Missing cleanup in defer
- Connection pools not releasing

### Issue 3: GC Pressure

**Symptom:**
```bash
# High GC time in trace
GODEBUG=gctrace=1 ./server
gc 1 @0.5s: 10% GC time
gc 2 @1.0s: 15% GC time
```

**Diagnosis:**
```bash
# Check allocation rate
go tool pprof -alloc_objects mem.prof

# Millions of small allocations = GC pressure
```

**Fix:**
- Reduce allocation frequency (pooling)
- Use arena allocation (Shockwave-specific)
- Batch operations
- Increase GOGC if heap is small

## Shockwave-Specific Profiling

### Arena Allocation Analysis

```bash
# Build with arena support
GOEXPERIMENT=arenas go test -bench=. -tags=arenas -memprofile=arena.prof

# Compare with standard
go test -bench=. -memprofile=standard.prof

# Arena should show near-zero GC
```

### Green Tea GC Analysis

```bash
# Build with greenteagc
go test -bench=. -tags=greenteagc -memprofile=greentea.prof

# Check spatial locality
go tool pprof -alloc_space greentea.prof
> top

# Related objects should be allocated close together
```

### Pool Efficiency

```bash
# Check pool hit rates
go test -bench=. -v 2>&1 | grep "pool"

# Or add metrics:
var poolHits, poolMisses int64

func getFromPool() *Object {
    if obj := pool.Get(); obj != nil {
        atomic.AddInt64(&poolHits, 1)
        return obj.(*Object)
    }
    atomic.AddInt64(&poolMisses, 1)
    return &Object{}
}

# Target: >95% hit rate
```

## Profiling Checklist

- [ ] Capture baseline memory profile
- [ ] Analyze alloc_space (total allocations)
- [ ] Analyze inuse_space (live heap)
- [ ] Check allocation rate (allocs/second)
- [ ] Identify hot allocation sites
- [ ] Verify GC time (<5% of CPU)
- [ ] Check for memory leaks (compare snapshots)
- [ ] Measure pool efficiency
- [ ] Profile under realistic load
- [ ] Compare before/after optimization

## Memory Leak Detection

### Capture Leak Snapshots

```bash
# Start server
./server &
PID=$!

# Baseline after warmup
sleep 10
curl http://localhost:6060/debug/pprof/heap > heap1.prof

# After workload
# (Run load test for 5 minutes)
curl http://localhost:6060/debug/pprof/heap > heap2.prof

# Compare
go tool pprof -base=heap1.prof heap2.prof
> top
```

### Goroutine Leak Detection

```bash
# Check goroutine count
curl http://localhost:6060/debug/pprof/goroutine?debug=1

# If growing over time, you have a leak
# Common causes:
# - Channel never closed
# - Select without timeout
# - HTTP client without timeout
```

## GC Tuning

### Understanding GOGC

```bash
# Default: GOGC=100 (heap doubles before GC)
# Lower GOGC = more frequent GC, less memory
# Higher GOGC = less frequent GC, more memory

# For high-throughput, low-latency:
GOGC=200 ./server  # Less frequent GC

# For memory-constrained:
GOGC=50 ./server   # More frequent GC
```

### GC Trace Analysis

```bash
# Enable GC trace
GODEBUG=gctrace=1 ./server 2>&1 | tee gctrace.log

# Output format:
# gc 1 @0.005s 0%: 0.018+1.2+0.004 ms clock, 0.14+0.35/0.82/0.028+0.034 ms cpu, 4->4->3 MB, 5 MB goal, 8 P

# Key metrics:
# - 0.018+1.2+0.004 ms: STW + concurrent + STW phases
# - 4->4->3 MB: heap before -> after -> live
# - 5 MB goal: target heap size
```

### Reducing GC Pause Time

Strategies:
1. **Reduce heap size** - Less to scan
2. **Reduce pointer count** - Use `[]byte` instead of `[]string`
3. **Use sync.Pool** - Reuse instead of allocate
4. **Arena allocation** - Zero GC for request lifetime
5. **Batch allocations** - Allocate in bursts, let GC run in quiet periods

## Memory Profiling Scripts

Create `scripts/mem_profile.sh`:
```bash
#!/bin/bash
# Memory profiling helper

case "$1" in
    capture)
        curl http://localhost:6060/debug/pprof/heap > "heap_$(date +%s).prof"
        ;;
    compare)
        go tool pprof -base="$2" "$3"
        ;;
    web)
        go tool pprof -http=:8080 "$2"
        ;;
    allocs)
        go tool pprof -alloc_space "$2"
        ;;
    *)
        echo "Usage: mem_profile.sh {capture|compare|web|allocs} [files...]"
        ;;
esac
```

## Output Format

When completing memory profiling analysis, provide:

```markdown
## Memory Profile Analysis

### Profiling Context
- Workload: [description]
- Duration: [time]
- QPS: [requests/second]

### Allocation Analysis
- Total allocated: X GB
- Allocation rate: Y MB/s
- Top allocators:
  1. FunctionName: Z MB (N%)
  2. ...

### Live Heap Analysis
- Current heap size: X MB
- Live objects: Y
- GC frequency: Z/minute
- GC pause time: A ms (p99)

### Issues Identified
1. **[Issue name]**
   - Location: file.go:123
   - Impact: X MB/s allocation rate
   - Root cause: [explanation]
   - Recommendation: [specific fix]

### Optimization Opportunities
1. [Function name] - Replace string concat with buffer pool (-50% allocs)
2. [Function name] - Pre-allocate slice (-100K allocs/op)

### Leak Analysis
- Heap growth: X MB/hour
- Suspected leaks: [Yes/No]
- Leak sources: [if applicable]

### Pool Efficiency
- Hit rate: X%
- Recommendations: [if <95%]
```

## References

- Go Memory Management: context/go-memory-guide.md
- pprof Documentation: https://go.dev/blog/pprof
- GC Guide: https://go.dev/doc/gc-guide

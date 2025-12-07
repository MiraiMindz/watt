# Sequential Prompts for Building Capacitor DAL/DTO

This document provides a step-by-step sequence of prompts to guide Claude through building the complete Capacitor DAL/DTO library, leveraging the entire `.claude/` ecosystem.

## Prerequisites

Before starting, ensure:
- [ ] Go 1.21+ installed
- [ ] Docker installed (for integration tests)
- [ ] You've read `CLAUDE.md` to understand the project philosophy
- [ ] You've read `CLAUDE_ECOSYSTEM_GUIDE.md` to understand the ecosystem

## Phase 1: Foundation and Core Interfaces

### Prompt 1.1: Project Initialization
```
Review CLAUDE.md thoroughly and explain the key principles and architecture
of the Capacitor project. Then initialize the Go module and create any
missing foundational files.
```

**Expected Outcome**: Claude understands the project, verifies go.mod, creates any missing setup files.

---

### Prompt 1.2: Core Error Types
```
Create a comprehensive error system in pkg/capacitor/errors.go following
the error handling patterns in CLAUDE.md. Include:
- Custom error types for each layer (cache, database, validation)
- Error wrapping utilities
- Sentinel errors for common cases
- Examples in documentation
```

**Expected Outcome**: `pkg/capacitor/errors.go` with proper error types and godoc.

---

### Prompt 1.3: Enhance DAL Interface
```
Review pkg/capacitor/dal.go and enhance it with additional methods that
would be useful for a production DAL:
- Batch operations (GetMulti, SetMulti, DeleteMulti)
- Iteration support (Range, Keys, Values)
- Transaction support (BeginTx, Commit, Rollback)

Ensure all methods follow zero-allocation principles where possible.
```

**Expected Outcome**: Enhanced DAL interface with batch operations and proper documentation.

---

### Prompt 1.4: DTO Framework
```
Create a DTO (Data Transfer Object) framework in pkg/capacitor/dto.go that:
- Provides base interfaces for DTOs
- Supports validation
- Supports serialization/deserialization
- Uses generics for type safety
- Includes common DTO patterns

Follow the patterns in CLAUDE.md and ensure zero allocations where possible.
```

**Expected Outcome**: `pkg/capacitor/dto.go` with DTO framework and examples.

---

## Phase 2: Memory Cache Implementation

### Prompt 2.1: Generate Memory Cache
```
Use the dal-generator skill to create a high-performance in-memory cache
implementation in pkg/cache/memory/.

Requirements:
- Generic types: K comparable, V any
- Features: TTL support, size limits, LRU eviction
- Zero allocations in Get operations
- sync.Pool for entry objects
- Thread-safe with RWMutex
- Comprehensive metrics

Generate:
1. Implementation (cache.go)
2. Entry pooling (pool.go)
3. LRU eviction (lru.go)
4. Tests (cache_test.go)
5. Benchmarks (cache_bench_test.go)
6. Examples (example_test.go)
```

**Expected Outcome**: Complete memory cache implementation with tests and benchmarks.

---

### Prompt 2.2: Test Memory Cache
```
/test

After tests complete, review the coverage report and create additional
tests for any uncovered edge cases. Ensure coverage is above 85%.
```

**Expected Outcome**: Full test coverage, all tests passing.

---

### Prompt 2.3: Benchmark Memory Cache
```
/benchmark

After benchmarks complete, verify that the memory cache meets these targets:
- Get: < 100 ns/op, 0 allocs/op
- Set: < 200 ns/op, ≤1 alloc/op
- Delete: < 150 ns/op, 0 allocs/op

If targets are not met, analyze why and suggest optimizations.
```

**Expected Outcome**: Benchmark results meeting or exceeding targets.

---

### Prompt 2.4: Optimize Memory Cache (if needed)
```
If benchmarks didn't meet targets, launch the cache-optimizer agent:

Launch the cache-optimizer agent to analyze and optimize the memory cache
implementation to achieve zero-allocation performance in hot paths. Focus on:
- Get operations: target 0 allocs/op
- Lock contention reduction
- Cache-line alignment
```

**Expected Outcome**: Optimized cache meeting all performance targets.

---

### Prompt 2.5: Profile Memory Cache
```
/profile

Generate CPU and memory profiles for the memory cache. Analyze the results
and create a performance report documenting:
- Top CPU consumers
- Allocation sources (should be minimal)
- Any optimization opportunities
- Assembly analysis of hot paths
```

**Expected Outcome**: Profiles and performance analysis report.

---

## Phase 3: Multi-Layer DAL Implementation

### Prompt 3.1: Create Multi-Layer DAL
```
Implement the multi-layer DAL in pkg/capacitor/multilayer.go that:
- Implements the DAL[K, V] interface
- Uses the Config and Builder from config.go
- Supports multiple layers (L1, L2, L3, persistent)
- Implements cache promotion (pull values to faster layers on hit)
- Implements write-through to all layers
- Collects aggregate statistics
- Handles layer failures gracefully

Follow the architecture diagram in CLAUDE.md and use zero-allocation patterns.
```

**Expected Outcome**: Complete multi-layer DAL implementation.

---

### Prompt 3.2: Test Multi-Layer DAL
```
Create comprehensive tests for the multi-layer DAL in
pkg/capacitor/multilayer_test.go:
- Layer promotion tests
- Write-through tests
- Layer failure scenarios
- Statistics collection
- Concurrent access tests
- Configuration validation tests

Use table-driven tests and ensure >85% coverage.
```

**Expected Outcome**: Comprehensive test suite for multi-layer DAL.

---

### Prompt 3.3: Multi-Layer Benchmarks
```
Create benchmarks comparing:
1. Single-layer vs multi-layer overhead
2. L1 hit vs L2 hit vs persistent hit
3. Write-through vs write-to-L1-only
4. Different layer combinations

Document the performance characteristics of each scenario.
```

**Expected Outcome**: Comparative benchmarks with analysis.

---

## Phase 4: Database Backend - PostgreSQL

### Prompt 4.1: Generate PostgreSQL Backend
```
Use the dal-generator skill to create a PostgreSQL backend implementation
in pkg/database/sql/:

Requirements:
- Generic types: K comparable, V any
- Prepared statements for performance
- Connection pooling (user provides *sql.DB)
- JSON serialization for V type
- Transaction support
- Batch operations
- Context cancellation support
- Proper error handling with retries

Generate:
1. Backend implementation (postgres.go)
2. Serializer interface and JSON implementation (serializer.go)
3. Schema setup utilities (schema.go)
4. Tests with test database (postgres_test.go)
5. Benchmarks comparing with raw SQL (postgres_bench_test.go)
6. Docker compose for testing (docker-compose.test.yml)
```

**Expected Outcome**: Complete PostgreSQL backend with tests and benchmarks.

---

### Prompt 4.2: PostgreSQL Integration Tests
```
Create integration tests that:
- Use docker-compose to start a real PostgreSQL instance
- Test all CRUD operations
- Test transaction support
- Test concurrent access
- Test connection pool behavior
- Test error scenarios (connection loss, deadlocks)

Run with: make test-integration
```

**Expected Outcome**: Integration tests passing with real database.

---

### Prompt 4.3: PostgreSQL Benchmarks
```
/compare

Create comparative benchmarks showing:
1. Capacitor PostgreSQL backend vs raw database/sql queries
2. Different serialization formats (JSON, MessagePack, Protobuf)
3. Batch operations vs individual operations
4. Connection pool sizes

Target: Capacitor should be within 20% of raw SQL performance but with
better ergonomics.
```

**Expected Outcome**: Comparative benchmark results and analysis.

---

## Phase 5: Type System Support

### Prompt 5.1: PostgreSQL Types
```
Create PostgreSQL type support in pkg/types/postgres/:
- JSONB support with zero-copy where possible
- Array types ([]string, []int, etc.)
- UUID type
- INET/CIDR types
- Timestamp with timezone
- Interval type
- Custom types (enum, composite)

Each type should:
- Implement sql.Scanner and driver.Valuer
- Have zero-allocation paths where possible
- Include comprehensive tests
- Have usage examples
```

**Expected Outcome**: PostgreSQL type support package.

---

### Prompt 5.2: Geographical Types
```
Create geographical type support in pkg/types/geo/:
- Point (lat/lon)
- Polygon
- Circle
- Distance calculations
- Bounding box
- Integration with PostGIS

Use efficient representations and avoid allocations in hot paths.
```

**Expected Outcome**: Geographical type support package.

---

### Prompt 5.3: Time Types
```
Create enhanced time type support in pkg/types/time/:
- Range types (start/end time)
- Recurring events (cron-like)
- Time zone handling
- Duration with nanosecond precision
- Business day calculations

Optimize for common operations like "is in range" checks.
```

**Expected Outcome**: Time type support package.

---

### Prompt 5.4: JSON/BLOB Types
```
Create efficient JSON/BLOB handling in pkg/types/json/:
- Lazy JSON parsing (don't parse until needed)
- JSON path queries without full parse
- Compression support (gzip, zstd)
- Binary format support (CBOR, MessagePack)
- Zero-copy where possible

Include benchmarks comparing formats.
```

**Expected Outcome**: JSON/BLOB type support with benchmarks.

---

## Phase 6: Advanced Cache Implementations

### Prompt 6.1: Disk Cache
```
/gen-cache

Generate a disk-based cache implementation in pkg/cache/disk/:
- Uses OS filesystem for storage
- Supports TTL and size-based eviction
- Uses memory-mapped files for hot data
- Implements write-ahead log for durability
- Thread-safe
- Graceful degradation if disk is full

Target: < 100 μs/op for SSD storage
```

**Expected Outcome**: Disk cache implementation with tests and benchmarks.

---

### Prompt 6.2: Network Cache (Redis-compatible)
```
/gen-cache

Generate a network cache implementation in pkg/cache/network/ that:
- Wraps a Redis client (user provides connection)
- Implements connection pooling
- Supports pipelining for batch operations
- Handles network failures with retries
- Falls back to local cache on network issues
- Supports Redis cluster mode

Include benchmarks comparing with direct Redis usage.
```

**Expected Outcome**: Network cache implementation with Redis support.

---

### Prompt 6.3: Hybrid Cache
```
Create a hybrid cache in pkg/cache/hybrid/ that combines:
- Small L1 in-process memory cache (hot data)
- Medium L2 disk cache (warm data)
- Large L3 network cache (cold data)

Implements intelligent promotion/demotion based on access patterns.
Include a comprehensive example showing the benefits.
```

**Expected Outcome**: Hybrid cache with intelligent tiering.

---

## Phase 7: Specialized Database Backends

### Prompt 7.1: Vector Database Backend
```
/gen-backend

Generate a vector database backend in pkg/database/vector/:
- Support for embedding storage ([]float32, []float64)
- Similarity search (cosine, euclidean, dot product)
- Metadata filtering
- Batch insert for efficiency
- Support for multiple providers (Pinecone, Weaviate, Qdrant)

Include benchmarks for different vector sizes and search strategies.
```

**Expected Outcome**: Vector database backend with similarity search.

---

### Prompt 7.2: NoSQL Backend (MongoDB)
```
/gen-backend

Generate a MongoDB backend in pkg/database/nosql/:
- Document storage with flexible schema
- Index management
- Aggregation pipeline support
- Change streams for real-time updates
- Bulk operations

Compare performance with PostgreSQL for different use cases.
```

**Expected Outcome**: MongoDB backend implementation.

---

### Prompt 7.3: Key-Value Backend (BadgerDB)
```
Create an embedded key-value backend in pkg/database/kv/ using BadgerDB:
- Zero-dependency embedded database
- Transaction support
- Prefix scan support
- TTL support
- Efficient for read-heavy workloads

Benchmark against memory cache and PostgreSQL.
```

**Expected Outcome**: Embedded KV database backend.

---

## Phase 8: Validation Framework

### Prompt 8.1: Core Validation
```
Create a validation framework in pkg/validation/:
- Rule-based validation (required, min/max, pattern, custom)
- Field-level and struct-level validation
- Composable validators
- Context-aware validation
- Detailed error messages
- Zero allocations for successful validations

Follow builder pattern for ergonomics.
```

**Expected Outcome**: Validation framework with comprehensive rules.

---

### Prompt 8.2: Type-Specific Validators
```
Create type-specific validators in pkg/validation/validators/:
- Email, URL, UUID validators
- Credit card, phone number validators
- Geographical coordinate validators
- Custom business rule validators

All with thorough tests and examples.
```

**Expected Outcome**: Library of common validators.

---

### Prompt 8.3: Integration with DAL
```
Integrate validation into the DAL:
- Validate on Set operations (optional, configurable)
- Return detailed validation errors
- Support for async validation (external API checks)
- Benchmarks showing overhead

Update examples to show validation usage.
```

**Expected Outcome**: Validation integrated into DAL with minimal overhead.

---

## Phase 9: Comprehensive Examples

### Prompt 9.1: Basic Usage Example
```
Create a comprehensive example in examples/basic/:
- Simple in-memory cache usage
- CRUD operations
- Error handling
- Graceful shutdown
- Well-commented for beginners

Include a README explaining the example.
```

**Expected Outcome**: Basic usage example with documentation.

---

### Prompt 9.2: Multi-Layer Example
```
Create a multi-layer example in examples/multi-layer/:
- L1 memory cache (1 minute TTL)
- L2 Redis cache (1 hour TTL)
- Persistent PostgreSQL storage
- Demonstrates promotion and write-through
- Shows performance benefits with benchmarks

Include realistic User/Product data models.
```

**Expected Outcome**: Multi-layer caching example with performance comparison.

---

### Prompt 9.3: E-commerce Example
```
Create a realistic e-commerce example in examples/ecommerce/:
- Product catalog (PostgreSQL + Redis cache)
- Shopping cart (memory cache with Redis backup)
- User sessions (Redis)
- Order history (PostgreSQL)
- Inventory management with validation
- Transaction support for orders

Show complete CRUD operations and edge cases.
```

**Expected Outcome**: Production-like e-commerce application example.

---

### Prompt 9.4: Microservices Example
```
Create a microservices example in examples/microservices/:
- Multiple services sharing Redis cache
- Service-to-service communication
- Distributed tracing integration
- Metrics collection
- Health checks
- Docker Compose for local development

Demonstrate caching strategies for microservices.
```

**Expected Outcome**: Microservices architecture example.

---

### Prompt 9.5: Vector Search Example
```
Create a vector search example in examples/vector-search/:
- Embedding storage (product descriptions, images)
- Semantic search
- Recommendation engine
- Hybrid search (keyword + vector)

Show performance characteristics and use cases.
```

**Expected Outcome**: Vector search/recommendation example.

---

## Phase 10: Performance Optimization

### Prompt 10.1: Comprehensive Benchmarking
```
Create a comprehensive benchmark suite in benchmarks/comparative/:
- Capacitor vs GORM
- Capacitor vs sqlc
- Capacitor vs ent
- Capacitor vs raw SQL
- Capacitor vs hand-rolled caching

Run with: make bench-compare

Document methodology and results in benchmarks/METHODOLOGY.md
```

**Expected Outcome**: Comprehensive competitive benchmarks.

---

### Prompt 10.2: Optimization Pass
```
/profile

Profile all major components and optimize any that don't meet targets:
1. Memory cache (target: 0 allocs in Get)
2. Multi-layer DAL (target: <5% overhead)
3. PostgreSQL backend (target: <20% slower than raw SQL)
4. Serialization (compare formats, choose fastest)

Document all optimizations made.
```

**Expected Outcome**: All components meeting performance targets.

---

### Prompt 10.3: Memory Analysis
```
Run memory profiling on all components:

1. Check for memory leaks
2. Verify pool effectiveness (hit rates)
3. Identify unexpected allocations
4. Optimize hot paths

Create a report in docs/MEMORY_ANALYSIS.md
```

**Expected Outcome**: Memory analysis report with any fixes applied.

---

### Prompt 10.4: Concurrency Analysis
```
Test concurrent performance:
- Run benchmarks with -cpu=1,2,4,8,16
- Check for lock contention with mutex profiling
- Verify linear scaling where expected
- Document any bottlenecks

Create scaling analysis in docs/CONCURRENCY_ANALYSIS.md
```

**Expected Outcome**: Concurrency analysis and scaling documentation.

---

## Phase 11: Documentation

### Prompt 11.1: API Documentation
```
Generate comprehensive API documentation in docs/api/:
- All public interfaces
- Usage examples for each major component
- Best practices
- Common pitfalls
- Performance characteristics
- Migration guides

Use godoc format with rich examples.
```

**Expected Outcome**: Complete API documentation.

---

### Prompt 11.2: Architecture Guide
```
Create an architecture guide in docs/ARCHITECTURE.md:
- System overview with diagrams
- Component interaction
- Data flow diagrams
- Caching strategies
- Scaling considerations
- Design decisions and rationale

Use Mermaid diagrams where helpful.
```

**Expected Outcome**: Comprehensive architecture documentation.

---

### Prompt 11.3: Performance Guide
```
Create a performance guide in docs/PERFORMANCE.md:
- Benchmarking methodology
- Performance targets for each component
- Optimization techniques used
- Profiling guide
- Tuning recommendations
- Case studies

Include before/after examples of optimizations.
```

**Expected Outcome**: Performance guide with tuning recommendations.

---

### Prompt 11.4: Migration Guides
```
Create migration guides in docs/migrations/:
- From GORM to Capacitor
- From raw database/sql to Capacitor
- From custom caching to Capacitor
- Version upgrade guides

Include code examples and gotchas.
```

**Expected Outcome**: Migration guides for common scenarios.

---

### Prompt 11.5: Contributing Guide
```
Create a comprehensive CONTRIBUTING.md:
- Development setup
- Code standards (from CLAUDE.md)
- Testing requirements
- Benchmarking requirements
- PR process
- Code review checklist
- Performance regression policy

Make it welcoming to new contributors.
```

**Expected Outcome**: Contributing guide.

---

## Phase 12: Testing and Quality Assurance

### Prompt 12.1: Integration Test Suite
```
Create comprehensive integration tests in tests/integration/:
- Multi-layer DAL with real databases
- Failure scenarios (network issues, database crashes)
- Data consistency tests
- Performance regression tests
- Long-running stability tests

Use Docker Compose for test infrastructure.
```

**Expected Outcome**: Complete integration test suite.

---

### Prompt 12.2: Fuzzing Tests
```
Create fuzzing tests for:
- Serialization/deserialization
- Validation rules
- Concurrent operations
- Edge cases in cache eviction

Use Go's native fuzzing support (go test -fuzz).
```

**Expected Outcome**: Fuzzing tests finding edge cases.

---

### Prompt 12.3: Property-Based Tests
```
Create property-based tests using a testing library:
- Cache should never exceed size limit
- Eviction should remove oldest entries
- Serialization should be reversible
- Concurrent operations should be linearizable

Document interesting properties discovered.
```

**Expected Outcome**: Property-based test suite.

---

### Prompt 12.4: Stress Tests
```
Create stress tests in tests/stress/:
- High concurrency (1000+ goroutines)
- Large datasets (millions of entries)
- Memory pressure (limited heap)
- Network instability (packet loss, latency)

Run overnight to find rare issues.
```

**Expected Outcome**: Stress test suite with results.

---

### Prompt 12.5: Full Coverage Check
```
/test

Review coverage report and ensure:
- Overall coverage >85%
- All critical paths >95%
- All error paths tested
- All edge cases covered

Create additional tests for any gaps found.
```

**Expected Outcome**: >85% code coverage with all critical paths tested.

---

## Phase 13: Tools and Utilities

### Prompt 13.1: CLI Tool
```
Create a CLI tool in cmd/capacitor/:
- Inspect cache contents
- Monitor statistics
- Benchmark different configurations
- Migration utilities
- Schema management

Make it useful for operations and debugging.
```

**Expected Outcome**: CLI tool for operations.

---

### Prompt 13.2: Monitoring Integration
```
Create monitoring integrations in pkg/monitoring/:
- Prometheus metrics exporter
- OpenTelemetry tracing
- Structured logging
- Health check endpoints

Include example Grafana dashboards.
```

**Expected Outcome**: Monitoring integration with dashboards.

---

### Prompt 13.3: Code Generation Tools
```
Create code generation tools in tools/codegen/:
- Generate DAL implementations from schema
- Generate DTO types from database schema
- Generate validation rules from OpenAPI specs

Make it extensible for custom generators.
```

**Expected Outcome**: Code generation utilities.

---

## Phase 14: Finalization

### Prompt 14.1: Full Test Suite
```
Run the complete test suite:

make test          # Unit tests
make test-race     # Race detector
make test-integration  # Integration tests
make coverage      # Coverage report

Ensure all tests pass and coverage >85%.
```

**Expected Outcome**: All tests passing, coverage target met.

---

### Prompt 14.2: Full Benchmark Suite
```
Run comprehensive benchmarks and create baseline:

make bench         # All benchmarks
make bench-compare # Compare with competitors

Save results to benchmarks/baseline.txt for future comparisons.
```

**Expected Outcome**: Benchmark baseline established.

---

### Prompt 14.3: Documentation Review
```
Review all documentation for:
- Accuracy
- Completeness
- Clarity
- Up-to-date code examples
- Working links

Fix any issues found.
```

**Expected Outcome**: Polished, accurate documentation.

---

### Prompt 14.4: License and Legal
```
Add appropriate license files:
- LICENSE (MIT recommended)
- NOTICE (if using third-party code)
- Copyright headers in source files

Ensure all dependencies have compatible licenses.
```

**Expected Outcome**: License files and compliance.

---

### Prompt 14.5: Release Preparation
```
Prepare for v1.0.0 release:
- Update CHANGELOG.md
- Tag version in git
- Create GitHub release notes
- Update README badges
- Verify all examples work
- Final performance validation

Create a release checklist in docs/RELEASE_CHECKLIST.md
```

**Expected Outcome**: Ready for v1.0.0 release.

---

## Phase 15: Advanced Features (Post-v1.0)

### Prompt 15.1: Distributed Caching
```
Implement distributed caching support:
- Consistent hashing for cache distribution
- Replication for high availability
- Distributed eviction policies
- Network partition handling

Add to pkg/cache/distributed/
```

**Expected Outcome**: Distributed caching support.

---

### Prompt 15.2: Query Builder
```
Create a type-safe query builder:
- Fluent API for building queries
- Compile-time type checking
- Support for joins, aggregations
- Integration with existing backends

Add to pkg/query/
```

**Expected Outcome**: Query builder with type safety.

---

### Prompt 15.3: GraphQL Support
```
Add GraphQL integration:
- GraphQL schema generation from DTOs
- Resolver generation
- N+1 query prevention with DataLoader
- Integration with caching layers

Add to pkg/graphql/
```

**Expected Outcome**: GraphQL support package.

---

### Prompt 15.4: Real-time Updates
```
Implement real-time update support:
- Change data capture (CDC)
- WebSocket subscriptions
- Event sourcing support
- Pub/sub integration

Add to pkg/realtime/
```

**Expected Outcome**: Real-time update capabilities.

---

### Prompt 15.5: Advanced Security
```
Add security features:
- Field-level encryption
- Row-level security
- Audit logging
- Data masking/redaction
- GDPR compliance utilities

Add to pkg/security/
```

**Expected Outcome**: Security enhancement package.

---

## Verification Checklist

After completing all phases, verify:

- [ ] All tests passing (`make test`)
- [ ] No race conditions (`make test-race`)
- [ ] Coverage >85% (`make coverage`)
- [ ] All benchmarks meet targets (`make bench`)
- [ ] Documentation complete and accurate
- [ ] Examples all work
- [ ] License files in place
- [ ] CHANGELOG up to date
- [ ] README polished with badges
- [ ] CI/CD pipeline configured
- [ ] Docker images built and tested
- [ ] Performance regression tests in CI
- [ ] Security audit completed
- [ ] API stability guaranteed (no breaking changes)
- [ ] Migration guides tested

## Performance Target Summary

Final verification against targets:

| Component | Operation | Target | Actual | Status |
|-----------|-----------|--------|--------|--------|
| Memory Cache | Get | <100ns, 0 allocs | ___ | ☐ |
| Memory Cache | Set | <200ns, ≤1 alloc | ___ | ☐ |
| Disk Cache | Get | <100μs, ≤2 allocs | ___ | ☐ |
| Disk Cache | Set | <200μs, ≤2 allocs | ___ | ☐ |
| PostgreSQL | Get | <1ms, ≤5 allocs | ___ | ☐ |
| PostgreSQL | Set | <2ms, ≤5 allocs | ___ | ☐ |
| Multi-Layer | L1 Hit | <150ns, 0 allocs | ___ | ☐ |
| Multi-Layer | L2 Hit | <10μs, ≤1 alloc | ___ | ☐ |

## Notes

- Each prompt builds on previous work
- Use `/test` and `/benchmark` frequently to validate progress
- Use the cache-optimizer agent when performance targets aren't met
- Leverage the dal-generator skill for consistent implementations
- Let hooks catch quality issues early
- Document design decisions as you go
- Keep the Claude ecosystem updated with new patterns discovered

## Customization

Feel free to:
- Reorder phases based on priorities
- Skip features not needed for your use case
- Add custom backends for your specific databases
- Extend type system for your domain
- Customize validation rules for your business logic

The ecosystem is designed to be flexible and adapt to your needs while maintaining the core zero-allocation philosophy.

---

**Happy Building! 🚀**

Remember: Performance is a feature, not an afterthought.

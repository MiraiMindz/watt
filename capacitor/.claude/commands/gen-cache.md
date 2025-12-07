Generate a new cache layer implementation with full tests and benchmarks.

Prompt the user for:
1. Cache type (memory, disk, network)
2. Key type (string, int, custom)
3. Value type ([]byte, string, custom struct)
4. Features:
   - TTL support?
   - Size limits?
   - Eviction strategy? (LRU, LFU, TTL-based, size-based)
   - Compression?
   - Encryption?
5. Performance requirements:
   - Expected operations/second
   - Expected data sizes
   - Concurrency level

Then generate:
1. Main cache implementation file
2. Comprehensive test file with table-driven tests
3. Benchmark suite with:
   - Basic operations (Get, Set, Delete)
   - Varying sizes
   - Concurrency tests
   - Comparative benchmarks
4. Example usage file
5. README with:
   - API documentation
   - Performance characteristics
   - Usage examples
   - Benchmark results

Follow all Capacitor standards:
- Zero-allocation hot paths
- Use sync.Pool for frequent allocations
- Generic type safety
- Proper error handling
- Full godoc comments
- Integration with main DAL interface

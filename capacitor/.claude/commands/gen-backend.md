Generate a new database backend implementation with full tests and benchmarks.

Prompt the user for:
1. Database type:
   - SQL (PostgreSQL, MySQL, SQLite)
   - NoSQL (MongoDB, Redis, Cassandra)
   - Vector (Pinecone, Weaviate, Qdrant)
   - Graph (Neo4j, ArangoDB)
   - Time-series (InfluxDB, TimescaleDB)
2. Key type
3. Value type (may be complex struct)
4. Features:
   - Transaction support?
   - Batch operations?
   - Query capabilities?
   - Index requirements?
   - Serialization format? (JSON, Protobuf, MessagePack, GOB)

Then generate:
1. Backend implementation with:
   - Connection pooling
   - Prepared statements (for SQL)
   - Proper error handling
   - Context cancellation support
   - Metrics collection
2. Serializer/deserializer implementation
3. Schema migration scripts (if applicable)
4. Comprehensive tests:
   - Unit tests for each method
   - Integration tests with real database (using docker-compose)
   - Transaction tests (if supported)
   - Concurrent access tests
5. Benchmarks comparing:
   - Capacitor backend vs raw database access
   - Different serialization formats
   - Batch vs individual operations
6. Docker compose file for testing
7. Documentation:
   - Setup instructions
   - Configuration options
   - Performance characteristics
   - Example usage

Ensure:
- User provides initialized database connection
- No hidden initialization or magic
- Clean error wrapping
- Resource cleanup (Close methods)
- Connection pool configuration
- Retry logic with exponential backoff

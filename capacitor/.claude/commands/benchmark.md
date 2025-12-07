Run comprehensive benchmarks on the current package or specified path.

Steps:
1. Identify the target package (use current directory if not specified)
2. Run benchmarks with memory allocation tracking
3. Run with multiple iterations for statistical significance
4. Generate CPU and memory profiles if requested
5. Compare with baseline if available
6. Report results with clear formatting

Options to consider:
- -benchtime: Duration to run each benchmark (default: 5s)
- -count: Number of times to run each benchmark (default: 5)
- -cpuprofile: Generate CPU profile
- -memprofile: Generate memory profile
- -compare: Compare with baseline from benchmarks/baseline.txt

Output should include:
- Benchmark results (ns/op, B/op, allocs/op)
- Statistical analysis using benchstat if multiple runs
- Warnings for any allocations in hot paths
- Performance targets comparison
- Suggestions for improvements if targets not met

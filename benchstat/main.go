// Package main provides a comprehensive benchmarking orchestrator for BOLT and SHOCKWAVE
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// Default values
	defaultBenchTime  = "1s"
	defaultCount      = 5
	defaultCPUList    = "1,4"  // Reduced from "1,2,4,8" to make time budgets more achievable
	minBenchTime      = 100 * time.Millisecond
	maxBenchTime      = 30 * time.Second
	minCount          = 1
	maxCount          = 50

	// Safety margins
	discoveryOverhead = 1.1  // 10% overhead for discovery and parsing
	executionBuffer   = 0.95 // Use 95% of available time for actual benchmarking
)

// Config holds the benchmarking configuration
type Config struct {
	TotalTime        time.Duration
	BenchTime        time.Duration
	Count            int
	CPUList          string
	BoltPath         string
	ShockwavePath    string
	OutputDir        string
	Verbose          bool
	SkipOptimization bool
	ExcludePattern   string
}

// BenchmarkInfo stores metadata about a discovered benchmark
type BenchmarkInfo struct {
	Package   string
	Name      string
	Path      string
	Project   string // "bolt" or "shockwave"
	Framework string // Extracted framework/library name (e.g., "shockwave", "nethttp", "fasthttp", "bolt", "gin", "echo", "fiber")
}

// BenchmarkResult represents a single benchmark run result
type BenchmarkResult struct {
	Name        string
	Package     string
	Project     string
	Framework   string
	NsPerOp     float64
	BytesPerOp  int64
	AllocsPerOp int64
	Iterations  int
	CPUCount    int
}

// StatisticalSummary contains aggregated statistics for a benchmark
type StatisticalSummary struct {
	Name        string
	Package     string
	Project     string
	Framework   string
	MeanNs      float64
	StdDevNs    float64
	MinNs       float64
	MaxNs       float64
	MeanBytes   float64
	MeanAllocs  float64
	SampleCount int
}

// RankingReport contains comparative rankings
type RankingReport struct {
	Scenario       string
	Category       string // "http-server" or "web-framework"
	Rankings       []ProjectRanking
	BestPerformers map[string]string // metric -> framework name
}

// ProjectRanking represents a project's performance in a scenario
type ProjectRanking struct {
	Project      string
	Score        float64
	NsPerOp      float64
	BytesPerOp   float64
	AllocsPerOp  float64
	Rank         int
}

func main() {
	config := parseFlags()

	if config.Verbose {
		log.Printf("Starting benchmark orchestrator with config: %+v\n", config)
	}

	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Discover benchmarks
	log.Println("Discovering benchmarks...")
	benchmarks, err := discoverBenchmarks(config)
	if err != nil {
		log.Fatalf("Failed to discover benchmarks: %v", err)
	}

	log.Printf("Found %d benchmarks total (%d BOLT, %d SHOCKWAVE)\n",
		len(benchmarks),
		countByProject(benchmarks, "bolt"),
		countByProject(benchmarks, "shockwave"))

	// Filter out excluded benchmarks
	if config.ExcludePattern != "" {
		originalCount := len(benchmarks)
		benchmarks, excludedCount := filterBenchmarks(benchmarks, config.ExcludePattern)
		if excludedCount > 0 {
			log.Printf("Excluded %d benchmarks matching pattern '%s'\n", excludedCount, config.ExcludePattern)
			log.Printf("Remaining: %d benchmarks (%d BOLT, %d SHOCKWAVE)\n",
				len(benchmarks),
				countByProject(benchmarks, "bolt"),
				countByProject(benchmarks, "shockwave"))
		}
		_ = originalCount
	}

	// Calculate optimal flags if time budget is specified
	if config.TotalTime > 0 && !config.SkipOptimization {
		log.Printf("Optimizing benchmark parameters for %v total time...\n", config.TotalTime)
		optimizeFlags(config, len(benchmarks))
		log.Printf("Optimized: benchtime=%v, count=%d\n", config.BenchTime, config.Count)
	}

	// Run benchmarks with timeout
	log.Println("Running benchmarks...")
	startTime := time.Now()

	// Create a context with timeout if total time is specified
	var results []BenchmarkResult
	var benchErr error

	if config.TotalTime > 0 {
		// Add 10% buffer for cleanup
		timeout := time.Duration(float64(config.TotalTime) * 1.1)
		deadline := time.Now().Add(timeout)
		log.Printf("Hard deadline: %v (will abort if exceeded)\n", deadline.Format("15:04:05"))

		done := make(chan bool)
		go func() {
			results, benchErr = runBenchmarks(config, benchmarks, startTime, config.TotalTime)
			done <- true
		}()

		select {
		case <-done:
			// Completed normally
		case <-time.After(timeout):
			log.Printf("\n!!! TIMEOUT: Hard deadline exceeded after %v !!!\n", timeout)
			log.Printf("Benchmarks did not complete within time budget.\n")
			log.Printf("Collected %d results before timeout.\n", len(results))
			benchErr = fmt.Errorf("timeout exceeded")
		}
	} else {
		results, benchErr = runBenchmarks(config, benchmarks, startTime, 0)
	}

	if benchErr != nil && len(results) == 0 {
		log.Fatalf("Failed to run benchmarks: %v", benchErr)
	}
	elapsed := time.Since(startTime)

	log.Printf("Completed %d benchmark runs in %v\n", len(results), elapsed)

	// Analyze results
	log.Println("Analyzing results...")
	summaries := analyzeResults(results)

	// Generate rankings
	rankings := generateRankings(summaries)

	// Generate report
	reportPath := filepath.Join(config.OutputDir, fmt.Sprintf("benchmark_report_%s.txt", time.Now().Format("20060102_150405")))
	if err := generateReport(reportPath, config, summaries, rankings, elapsed); err != nil {
		log.Fatalf("Failed to generate report: %v", err)
	}

	log.Printf("Report generated: %s\n", reportPath)

	// Print summary to console
	printSummary(summaries, rankings)
}

func parseFlags() *Config {
	config := &Config{}

	var totalTimeStr string
	var benchTimeStr string

	flag.StringVar(&totalTimeStr, "total-time", "", "Total time budget (e.g., 30m, 1h)")
	flag.StringVar(&benchTimeStr, "benchtime", defaultBenchTime, "Time per benchmark (e.g., 1s, 100ms)")
	flag.IntVar(&config.Count, "count", defaultCount, "Number of times to run each benchmark")
	flag.StringVar(&config.CPUList, "cpu", defaultCPUList, "Comma-separated CPU counts to test")
	flag.StringVar(&config.BoltPath, "bolt", "bolt", "Path to BOLT project")
	flag.StringVar(&config.ShockwavePath, "shockwave", "shockwave", "Path to SHOCKWAVE project")
	flag.StringVar(&config.OutputDir, "output", "benchmark_results", "Output directory for results")
	flag.BoolVar(&config.Verbose, "v", false, "Verbose output (streams benchmark results)")
	flag.BoolVar(&config.SkipOptimization, "no-optimize", false, "Skip automatic flag optimization")
	flag.StringVar(&config.ExcludePattern, "exclude", "Panic|Recovery|KeepAlive|WebSocketEcho", "Exclude benchmarks matching pattern (regex)")

	flag.Parse()

	// Inform about default exclusions
	if config.ExcludePattern != "" {
		log.Printf("Excluding benchmarks matching: %s\n", config.ExcludePattern)
		log.Printf("(These benchmarks can hang or produce excessive output)\n")
	}

	// Warn about verbose mode with high count
	if config.Verbose && config.Count == 0 {
		log.Println("WARNING: Verbose mode with high count values can create very large logs.")
		log.Println("         Consider using -count with a lower value (e.g., -count 5)")
	}

	// Parse total time
	if totalTimeStr != "" {
		duration, err := time.ParseDuration(totalTimeStr)
		if err != nil {
			log.Fatalf("Invalid total-time: %v", err)
		}
		config.TotalTime = duration
	}

	// Parse benchtime
	duration, err := time.ParseDuration(benchTimeStr)
	if err != nil {
		log.Fatalf("Invalid benchtime: %v", err)
	}
	config.BenchTime = duration

	// Validate CPU list
	cpuCounts := parseCPUList(config.CPUList)
	if len(cpuCounts) == 0 {
		log.Fatalf("Invalid CPU list: %s", config.CPUList)
	}

	return config
}

func parseCPUList(cpuList string) []int {
	parts := strings.Split(cpuList, ",")
	var counts []int
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		count, err := strconv.Atoi(part)
		if err != nil || count < 1 {
			continue
		}
		counts = append(counts, count)
	}
	return counts
}

func discoverBenchmarks(config *Config) ([]BenchmarkInfo, error) {
	var benchmarks []BenchmarkInfo

	// Discover BOLT benchmarks
	boltBenches, err := discoverBenchmarksInProject(config.BoltPath, "bolt")
	if err != nil {
		return nil, fmt.Errorf("discovering BOLT benchmarks: %w", err)
	}
	benchmarks = append(benchmarks, boltBenches...)

	// Discover SHOCKWAVE benchmarks
	shockwaveBenches, err := discoverBenchmarksInProject(config.ShockwavePath, "shockwave")
	if err != nil {
		return nil, fmt.Errorf("discovering SHOCKWAVE benchmarks: %w", err)
	}
	benchmarks = append(benchmarks, shockwaveBenches...)

	return benchmarks, nil
}

func discoverBenchmarksInProject(projectPath, projectName string) ([]BenchmarkInfo, error) {
	var benchmarks []BenchmarkInfo

	// Find all *_test.go files
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor and hidden directories
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == ".git" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Process test files
		if strings.HasSuffix(path, "_test.go") {
			benches, err := extractBenchmarksFromFile(path, projectPath, projectName)
			if err != nil {
				log.Printf("Warning: failed to extract benchmarks from %s: %v", path, err)
				return nil
			}
			benchmarks = append(benchmarks, benches...)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return benchmarks, nil
}

func extractBenchmarksFromFile(filePath, projectPath, projectName string) ([]BenchmarkInfo, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Extract package name
	packageRe := regexp.MustCompile(`package\s+(\w+)`)
	packageMatch := packageRe.FindSubmatch(content)
	if packageMatch == nil {
		return nil, fmt.Errorf("no package declaration found")
	}
	packageName := string(packageMatch[1])

	// Find all benchmark functions
	benchmarkRe := regexp.MustCompile(`func\s+(Benchmark\w+)\s*\(`)
	matches := benchmarkRe.FindAllSubmatch(content, -1)

	var benchmarks []BenchmarkInfo
	dirPath := filepath.Dir(filePath)

	for _, match := range matches {
		benchName := string(match[1])
		benchmarks = append(benchmarks, BenchmarkInfo{
			Package:   packageName,
			Name:      benchName,
			Path:      dirPath,
			Project:   projectName,
			Framework: extractFramework(benchName, projectName),
		})
	}

	return benchmarks, nil
}

// filterBenchmarks removes benchmarks matching the exclude pattern
func filterBenchmarks(benchmarks []BenchmarkInfo, excludePattern string) ([]BenchmarkInfo, int) {
	if excludePattern == "" {
		return benchmarks, 0
	}

	excludeRe, err := regexp.Compile(excludePattern)
	if err != nil {
		log.Printf("Warning: invalid exclude pattern '%s': %v", excludePattern, err)
		return benchmarks, 0
	}

	var filtered []BenchmarkInfo
	excluded := 0

	for _, b := range benchmarks {
		if excludeRe.MatchString(b.Name) {
			excluded++
			continue
		}
		filtered = append(filtered, b)
	}

	return filtered, excluded
}

// extractFramework extracts the framework/library name from a benchmark name
// Examples:
//   - BenchmarkShockwave_RequestParsing -> "shockwave"
//   - BenchmarkComparisonSimpleGET/net/http -> "nethttp"
//   - BenchmarkComparisonSimpleGET/fasthttp -> "fasthttp"
//   - BenchmarkBolt_StaticRoute -> "bolt"
//   - BenchmarkGin_StaticRoute -> "gin"
//   - BenchmarkEcho_StaticRoute -> "echo"
//   - BenchmarkFiber_StaticRoute -> "fiber"
func extractFramework(benchmarkName, projectName string) string {
	name := strings.TrimPrefix(benchmarkName, "Benchmark")

	// Handle comparison benchmarks with "/" separator (e.g., "ComparisonSimpleGET/net/http")
	if strings.Contains(name, "/") {
		parts := strings.Split(name, "/")
		lastPart := parts[len(parts)-1]

		// Map net/http to nethttp for consistency
		if lastPart == "http" && len(parts) >= 2 && parts[len(parts)-2] == "net" {
			return "nethttp"
		}
		return strings.ToLower(lastPart)
	}

	// Handle standard benchmarks with "_" separator (e.g., "Bolt_StaticRoute", "Gin_StaticRoute")
	parts := strings.SplitN(name, "_", 2)
	if len(parts) >= 1 {
		framework := strings.ToLower(parts[0])

		// Normalize framework names
		switch framework {
		case "nethttp", "net":
			return "nethttp"
		case "shockwave":
			return "shockwave"
		case "fasthttp":
			return "fasthttp"
		case "bolt":
			return "bolt"
		case "gin":
			return "gin"
		case "echo":
			return "echo"
		case "fiber":
			return "fiber"
		case "comparison":
			// For benchmarks like "BenchmarkComparison..." without sub-parts
			// Assume it's part of the main project
			return projectName
		default:
			return framework
		}
	}

	return projectName
}

func countByProject(benchmarks []BenchmarkInfo, project string) int {
	count := 0
	for _, b := range benchmarks {
		if b.Project == project {
			count++
		}
	}
	return count
}

func optimizeFlags(config *Config, benchmarkCount int) {
	if config.TotalTime == 0 || benchmarkCount == 0 {
		return
	}

	// Calculate time per project (split evenly)
	timePerProject := config.TotalTime / 2

	// CRITICAL: Account for real-world overhead
	// - Compilation time per package: ~5-30 seconds
	// - benchtime is MINIMUM, actual time is often 2-5x longer
	// - Setup/teardown overhead
	// - System variance
	const (
		compilationOverhead = 0.15  // 15% for compilation
		runtimeMultiplier   = 5.0   // Benchmarks run 5x longer than benchtime (conservative)
		safetyMargin        = 0.50  // Use only 50% of available time to be very safe
	)

	// Available time after accounting for all overhead
	availableTime := float64(timePerProject) * safetyMargin * (1.0 - compilationOverhead)

	// Number of CPU configurations to test
	cpuCounts := parseCPUList(config.CPUList)
	numCPUConfigs := len(cpuCounts)

	// Average benchmarks per project
	benchmarksPerProject := benchmarkCount / 2
	if benchmarksPerProject == 0 {
		benchmarksPerProject = benchmarkCount
	}

	// Estimate number of packages (assume ~20 benchmarks per package)
	estimatedPackages := benchmarksPerProject / 20
	if estimatedPackages < 1 {
		estimatedPackages = 1
	}

	log.Printf("  Optimization estimates: %d benchmarks across ~%d packages\n",
		benchmarksPerProject, estimatedPackages)

	// Total benchmark executions = benchmarks * cpuCounts * count
	// Actual time = executions * benchtime * runtimeMultiplier
	// Solve for best combination

	// Calculate minimum possible time needed
	minPossibleTime := float64(benchmarksPerProject*numCPUConfigs) * float64(minBenchTime) * runtimeMultiplier
	minPossibleTimeTotal := minPossibleTime / (1.0 - compilationOverhead)

	if minPossibleTimeTotal > availableTime {
		log.Printf("  WARNING: Time budget is too small for %d benchmarks!\n", benchmarksPerProject)
		log.Printf("  Minimum time needed: ~%v (available: %v)\n",
			time.Duration(minPossibleTimeTotal), time.Duration(availableTime))
		log.Printf("  Using absolute minimum settings: count=1, benchtime=%v\n", minBenchTime)
		log.Printf("  Results may be incomplete or statistically unreliable.\n")
		log.Printf("  Consider: -total-time 30m or higher for reliable results\n")

		// Use absolute minimum
		config.BenchTime = minBenchTime
		config.Count = 1
		return
	}

	// Try different count values and calculate corresponding benchtime
	bestScore := 0.0
	bestBenchTime := config.BenchTime
	bestCount := config.Count

	for count := minCount; count <= minInt(10, maxCount); count++ { // Cap at 10 for time budget
		totalExecutions := float64(benchmarksPerProject * numCPUConfigs * count)

		// Calculate benchtime accounting for runtime multiplier
		benchTime := time.Duration(availableTime / (totalExecutions * runtimeMultiplier))

		// Validate benchtime is within acceptable range
		if benchTime < minBenchTime {
			continue // Skip this count, it's too low
		}
		if benchTime > maxBenchTime {
			benchTime = maxBenchTime
		}

		// Estimate actual time for this combination
		estimatedTime := float64(totalExecutions) * float64(benchTime) * runtimeMultiplier / (1.0 - compilationOverhead)
		if estimatedTime > availableTime*1.2 {
			break // This and higher counts will be too slow
		}

		// Score = count * sqrt(benchTime) - prefer higher count with reasonable benchtime
		score := float64(count) * math.Sqrt(float64(benchTime))

		if score > bestScore {
			bestScore = score
			bestBenchTime = benchTime
			bestCount = count
		}
	}

	// If we didn't find any valid combination, use minimum
	if bestScore == 0 {
		log.Printf("  No optimal combination found, using minimum: count=1, benchtime=%v\n", minBenchTime)
		config.BenchTime = minBenchTime
		config.Count = 1
	} else {
		config.BenchTime = bestBenchTime
		config.Count = bestCount

		// Estimate actual runtime
		estimatedRuntime := float64(benchmarksPerProject*numCPUConfigs*bestCount) *
			float64(bestBenchTime) * runtimeMultiplier / (1.0 - compilationOverhead)
		log.Printf("  Estimated actual runtime per project: ~%v (target: %v)\n",
			time.Duration(estimatedRuntime), timePerProject)
	}
}

func runBenchmarks(config *Config, benchmarks []BenchmarkInfo, startTime time.Time, totalTime time.Duration) ([]BenchmarkResult, error) {
	var allResults []BenchmarkResult

	// Calculate deadline if totalTime is set
	var deadline time.Time
	if totalTime > 0 {
		// Use same buffer as main (10%)
		deadline = startTime.Add(time.Duration(float64(totalTime) * 1.1))
	}

	// Group benchmarks by project and package
	boltBenches := filterByProject(benchmarks, "bolt")
	shockwaveBenches := filterByProject(benchmarks, "shockwave")

	// Run BOLT benchmarks
	log.Printf("\n========================================\n")
	log.Printf("Running BOLT benchmarks (%d total)\n", len(boltBenches))
	log.Printf("========================================\n")

	// Check deadline before starting BOLT
	if !deadline.IsZero() && time.Now().After(deadline) {
		log.Printf("!!! Time budget exceeded before BOLT benchmarks !!!\n")
		return allResults, fmt.Errorf("time budget exceeded")
	}

	boltResults, err := runProjectBenchmarks(config, config.BoltPath, boltBenches, "bolt", deadline)
	if err != nil {
		// If we hit timeout, return what we have
		if strings.Contains(err.Error(), "deadline exceeded") {
			log.Printf("BOLT benchmarks aborted due to deadline\n")
			return append(allResults, boltResults...), err
		}
		return nil, fmt.Errorf("running BOLT benchmarks: %w", err)
	}
	allResults = append(allResults, boltResults...)
	log.Printf("BOLT: Collected %d results\n", len(boltResults))

	// Check deadline before starting SHOCKWAVE
	if !deadline.IsZero() && time.Now().After(deadline) {
		log.Printf("!!! Time budget exceeded after BOLT, skipping SHOCKWAVE !!!\n")
		return allResults, fmt.Errorf("time budget exceeded")
	}

	// Run SHOCKWAVE benchmarks
	log.Printf("\n========================================\n")
	log.Printf("Running SHOCKWAVE benchmarks (%d total)\n", len(shockwaveBenches))
	log.Printf("========================================\n")
	shockwaveResults, err := runProjectBenchmarks(config, config.ShockwavePath, shockwaveBenches, "shockwave", deadline)
	if err != nil {
		// If we hit timeout, return what we have
		if strings.Contains(err.Error(), "deadline exceeded") {
			log.Printf("SHOCKWAVE benchmarks aborted due to deadline\n")
			return append(allResults, shockwaveResults...), err
		}
		return nil, fmt.Errorf("running SHOCKWAVE benchmarks: %w", err)
	}
	allResults = append(allResults, shockwaveResults...)
	log.Printf("SHOCKWAVE: Collected %d results\n", len(shockwaveResults))

	return allResults, nil
}

func filterByProject(benchmarks []BenchmarkInfo, project string) []BenchmarkInfo {
	var filtered []BenchmarkInfo
	for _, b := range benchmarks {
		if b.Project == project {
			filtered = append(filtered, b)
		}
	}
	return filtered
}

func runProjectBenchmarks(config *Config, projectPath string, benchmarks []BenchmarkInfo, projectName string, deadline time.Time) ([]BenchmarkResult, error) {
	// Group by package path
	packageGroups := make(map[string][]BenchmarkInfo)
	for _, b := range benchmarks {
		packageGroups[b.Path] = append(packageGroups[b.Path], b)
	}

	var allResults []BenchmarkResult
	pkgNum := 0
	totalPkgs := len(packageGroups)

	for pkgPath, pkgBenches := range packageGroups {
		// Check deadline before each package
		if !deadline.IsZero() && time.Now().After(deadline) {
			log.Printf("!!! Deadline exceeded at package %d/%d, aborting remaining packages !!!\n", pkgNum, totalPkgs)
			return allResults, fmt.Errorf("deadline exceeded")
		}

		pkgNum++
		relPath := strings.TrimPrefix(pkgPath, projectPath+"/")

		log.Printf("[%d/%d] Running %d benchmarks in %s...\n", pkgNum, totalPkgs, len(pkgBenches), relPath)

		// Build benchmark command
		// Calculate timeout based on deadline or use default
		packageTimeout := "5m" // Default timeout
		if !deadline.IsZero() {
			remaining := time.Until(deadline)
			if remaining > 0 {
				// Use remaining time, but cap at 10m to prevent single package from consuming all time
				pkgTimeout := minDuration(remaining, 10*time.Minute)
				packageTimeout = pkgTimeout.String()
			} else {
				// Already past deadline
				log.Printf("  Already past deadline, skipping remaining packages\n")
				return allResults, fmt.Errorf("deadline exceeded")
			}
		}

		args := []string{
			"test",
			"-bench=.",
			"-benchmem",
			fmt.Sprintf("-benchtime=%s", config.BenchTime),
			fmt.Sprintf("-count=%d", config.Count),
			fmt.Sprintf("-cpu=%s", config.CPUList),
			"-run=^$", // Don't run tests, only benchmarks
			fmt.Sprintf("-timeout=%s", packageTimeout), // Dynamic timeout based on deadline
		}

		// NOTE: We don't add -v even in verbose mode because it creates
		// astronomical output (shows every iteration). Instead, we stream
		// the summary output which is much more reasonable.

		cmd := exec.Command("go", args...)
		cmd.Dir = pkgPath

		var stdout, stderr bytes.Buffer

		// Always capture output for parsing
		// In verbose mode, also stream to console
		if config.Verbose {
			cmd.Stdout = io.MultiWriter(&stdout, os.Stdout)
			cmd.Stderr = io.MultiWriter(&stderr, os.Stderr)
		} else {
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
		}

		startTime := time.Now()
		err := cmd.Run()
		elapsed := time.Since(startTime)

		log.Printf("[%d/%d] Completed %s in %v\n", pkgNum, totalPkgs, relPath, elapsed)

		if err != nil {
			log.Printf("ERROR: Benchmark failed for %s: %v\n", relPath, err)
			if stderr.Len() > 0 {
				if !config.Verbose {
					// Only print stderr if not already printed in verbose mode
					log.Printf("Stderr:\n%s\n", stderr.String())
				}
			}
			if stdout.Len() > 0 && !config.Verbose {
				log.Printf("Stdout:\n%s\n", stdout.String())
			}
			continue
		}

		outputStr := stdout.String()

		results, err := parseBenchmarkOutput(outputStr, pkgBenches[0].Package, projectName)
		if err != nil {
			log.Printf("Warning: failed to parse benchmark output for %s: %v\n", relPath, err)
			if config.Verbose && outputStr != "" {
				log.Printf("Output was:\n%s\n", outputStr[:minInt(len(outputStr), 500)])
			}
			continue
		}

		log.Printf("  Parsed %d results\n", len(results))
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func parseBenchmarkOutput(output, packageName, projectName string) ([]BenchmarkResult, error) {
	var results []BenchmarkResult

	// Benchmark output format:
	// BenchmarkName-8    1000000    1234 ns/op    512 B/op    3 allocs/op
	re := regexp.MustCompile(`^(Benchmark\w+)-(\d+)\s+(\d+)\s+([\d.]+)\s+ns/op\s+(\d+)\s+B/op\s+(\d+)\s+allocs/op`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		match := re.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		name := match[1]
		cpuCount, _ := strconv.Atoi(match[2])
		iterations, _ := strconv.Atoi(match[3])
		nsPerOp, _ := strconv.ParseFloat(match[4], 64)
		bytesPerOp, _ := strconv.ParseInt(match[5], 10, 64)
		allocsPerOp, _ := strconv.ParseInt(match[6], 10, 64)

		results = append(results, BenchmarkResult{
			Name:        name,
			Package:     packageName,
			Project:     projectName,
			Framework:   extractFramework(name, projectName),
			NsPerOp:     nsPerOp,
			BytesPerOp:  bytesPerOp,
			AllocsPerOp: allocsPerOp,
			Iterations:  iterations,
			CPUCount:    cpuCount,
		})
	}

	return results, nil
}

func analyzeResults(results []BenchmarkResult) []StatisticalSummary {
	// Group by benchmark name, project, and framework
	groups := make(map[string][]BenchmarkResult)
	for _, r := range results {
		key := fmt.Sprintf("%s:%s:%s", r.Project, r.Framework, r.Name)
		groups[key] = append(groups[key], r)
	}

	var summaries []StatisticalSummary
	for key, group := range groups {
		parts := strings.SplitN(key, ":", 3)
		project := parts[0]
		framework := parts[1]
		name := parts[2]

		summary := StatisticalSummary{
			Name:        name,
			Package:     group[0].Package,
			Project:     project,
			Framework:   framework,
			SampleCount: len(group),
		}

		// Calculate statistics
		var nsValues []float64
		var bytesValues []float64
		var allocsValues []float64

		for _, r := range group {
			nsValues = append(nsValues, r.NsPerOp)
			bytesValues = append(bytesValues, float64(r.BytesPerOp))
			allocsValues = append(allocsValues, float64(r.AllocsPerOp))
		}

		summary.MeanNs = mean(nsValues)
		summary.StdDevNs = stdDev(nsValues, summary.MeanNs)
		summary.MinNs = min(nsValues)
		summary.MaxNs = max(nsValues)
		summary.MeanBytes = mean(bytesValues)
		summary.MeanAllocs = mean(allocsValues)

		summaries = append(summaries, summary)
	}

	return summaries
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stdDev(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	return math.Sqrt(variance / float64(len(values)-1))
}

func min(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	minVal := values[0]
	for _, v := range values[1:] {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}

func max(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	maxVal := values[0]
	for _, v := range values[1:] {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

func generateRankings(summaries []StatisticalSummary) []RankingReport {
	// Group by scenario and category
	// category is determined by the frameworks present
	type ScenarioKey struct {
		Scenario string
		Category string
	}

	scenarios := make(map[ScenarioKey][]StatisticalSummary)

	for _, s := range summaries {
		scenario := extractScenario(s.Name)
		category := categorizeFramework(s.Framework)

		key := ScenarioKey{
			Scenario: scenario,
			Category: category,
		}

		scenarios[key] = append(scenarios[key], s)
	}

	var reports []RankingReport

	for key, group := range scenarios {
		report := RankingReport{
			Scenario:       key.Scenario,
			Category:       key.Category,
			BestPerformers: make(map[string]string),
		}

		// Create rankings by framework (not project)
		frameworkPerf := make(map[string]*ProjectRanking)

		for _, s := range group {
			if _, exists := frameworkPerf[s.Framework]; !exists {
				frameworkPerf[s.Framework] = &ProjectRanking{
					Project: s.Framework, // Using "Project" field to store framework name
				}
			}

			pr := frameworkPerf[s.Framework]
			pr.NsPerOp += s.MeanNs
			pr.BytesPerOp += s.MeanBytes
			pr.AllocsPerOp += s.MeanAllocs
		}

		// Average the values
		for framework, pr := range frameworkPerf {
			count := float64(countFrameworkInGroup(group, framework))
			if count > 0 {
				pr.NsPerOp /= count
				pr.BytesPerOp /= count
				pr.AllocsPerOp /= count
			}

			// Calculate composite score (lower is better)
			// Normalize: ns/op is primary, bytes and allocs are secondary
			pr.Score = pr.NsPerOp + (pr.BytesPerOp * 0.1) + (pr.AllocsPerOp * 100)
		}

		// Sort by score
		var rankings []ProjectRanking
		for _, pr := range frameworkPerf {
			rankings = append(rankings, *pr)
		}

		sort.Slice(rankings, func(i, j int) bool {
			return rankings[i].Score < rankings[j].Score
		})

		// Assign ranks
		for i := range rankings {
			rankings[i].Rank = i + 1
		}

		report.Rankings = rankings

		// Determine best performers
		if len(rankings) > 0 {
			bestNs := rankings[0].Project
			bestBytes := rankings[0].Project
			bestAllocs := rankings[0].Project

			for _, r := range rankings {
				if r.NsPerOp < frameworkPerf[r.Project].NsPerOp {
					bestNs = r.Project
				}
				if r.BytesPerOp < frameworkPerf[r.Project].BytesPerOp {
					bestBytes = r.Project
				}
				if r.AllocsPerOp < frameworkPerf[r.Project].AllocsPerOp {
					bestAllocs = r.Project
				}
			}

			report.BestPerformers["ns/op"] = bestNs
			report.BestPerformers["B/op"] = bestBytes
			report.BestPerformers["allocs/op"] = bestAllocs
		}

		reports = append(reports, report)
	}

	// Sort reports by category and scenario name
	sort.Slice(reports, func(i, j int) bool {
		if reports[i].Category != reports[j].Category {
			return reports[i].Category < reports[j].Category
		}
		return reports[i].Scenario < reports[j].Scenario
	})

	return reports
}

// categorizeFramework determines if a framework is an HTTP server or web framework
func categorizeFramework(framework string) string {
	switch framework {
	case "shockwave", "nethttp", "fasthttp":
		return "http-server"
	case "bolt", "gin", "echo", "fiber":
		return "web-framework"
	default:
		// Try to infer from name
		if strings.Contains(framework, "http") {
			return "http-server"
		}
		return "web-framework"
	}
}

// countFrameworkInGroup counts how many summaries in the group belong to a framework
func countFrameworkInGroup(group []StatisticalSummary, framework string) int {
	count := 0
	for _, s := range group {
		if s.Framework == framework {
			count++
		}
	}
	return count
}

func extractScenario(benchmarkName string) string {
	// Remove project prefix and CPU suffix
	// BenchmarkBolt_StaticRoute -> StaticRoute
	// BenchmarkShockwave_RequestParsing -> RequestParsing

	// Remove "Benchmark" prefix
	name := strings.TrimPrefix(benchmarkName, "Benchmark")

	// Remove project prefix (Bolt_, Shockwave_, Gin_, etc.)
	parts := strings.SplitN(name, "_", 2)
	if len(parts) == 2 {
		return parts[1]
	}

	return name
}

func countInGroup(group []StatisticalSummary, project string) int {
	count := 0
	for _, s := range group {
		if s.Project == project {
			count++
		}
	}
	return count
}

func generateReport(path string, config *Config, summaries []StatisticalSummary, rankings []RankingReport, elapsed time.Duration) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "=" + strings.Repeat("=", 78) + "\n")
	fmt.Fprintf(f, " BOLT & SHOCKWAVE Comprehensive Benchmark Report\n")
	fmt.Fprintf(f, "=" + strings.Repeat("=", 78) + "\n\n")

	fmt.Fprintf(f, "Generated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(f, "System: %s/%s (CPUs: %d)\n", runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	fmt.Fprintf(f, "Total Execution Time: %v\n\n", elapsed)

	fmt.Fprintf(f, "Configuration:\n")
	fmt.Fprintf(f, "  Benchtime: %v\n", config.BenchTime)
	fmt.Fprintf(f, "  Count: %d\n", config.Count)
	fmt.Fprintf(f, "  CPU List: %s\n", config.CPUList)
	if config.TotalTime > 0 {
		fmt.Fprintf(f, "  Time Budget: %v (optimized)\n", config.TotalTime)
	}
	fmt.Fprintf(f, "\n")

	fmt.Fprintf(f, "-" + strings.Repeat("-", 78) + "\n")
	fmt.Fprintf(f, " Overall Rankings by Category and Scenario\n")
	fmt.Fprintf(f, "-" + strings.Repeat("-", 78) + "\n\n")

	// Group rankings by category for better presentation
	currentCategory := ""
	for _, report := range rankings {
		// Print category header if it changed
		if report.Category != currentCategory {
			currentCategory = report.Category
			categoryTitle := "HTTP SERVER BENCHMARKS"
			if report.Category == "web-framework" {
				categoryTitle = "WEB FRAMEWORK BENCHMARKS"
			}
			fmt.Fprintf(f, "\n" + strings.Repeat("=", 79) + "\n")
			fmt.Fprintf(f, " %s\n", categoryTitle)
			fmt.Fprintf(f, strings.Repeat("=", 79) + "\n\n")
		}

		fmt.Fprintf(f, "Scenario: %s\n", report.Scenario)
		fmt.Fprintf(f, strings.Repeat("-", 79) + "\n")
		fmt.Fprintf(f, "%-6s %-15s %12s %12s %12s %12s\n", "Rank", "Framework", "Score", "ns/op", "B/op", "allocs/op")
		fmt.Fprintf(f, strings.Repeat("-", 79) + "\n")

		for _, r := range report.Rankings {
			fmt.Fprintf(f, "%-6d %-15s %12.2f %12.2f %12.2f %12.2f\n",
				r.Rank, r.Project, r.Score, r.NsPerOp, r.BytesPerOp, r.AllocsPerOp)
		}

		fmt.Fprintf(f, "\nBest Performers:\n")
		fmt.Fprintf(f, "  CPU (ns/op):     %s\n", report.BestPerformers["ns/op"])
		fmt.Fprintf(f, "  Memory (B/op):   %s\n", report.BestPerformers["B/op"])
		fmt.Fprintf(f, "  Allocs:          %s\n", report.BestPerformers["allocs/op"])
		fmt.Fprintf(f, "\n\n")
	}

	fmt.Fprintf(f, "-" + strings.Repeat("-", 78) + "\n")
	fmt.Fprintf(f, " Detailed Statistics\n")
	fmt.Fprintf(f, "-" + strings.Repeat("-", 78) + "\n\n")

	// Group by project
	boltSummaries := filterSummariesByProject(summaries, "bolt")
	shockwaveSummaries := filterSummariesByProject(summaries, "shockwave")

	fmt.Fprintf(f, "BOLT Benchmarks (%d):\n", len(boltSummaries))
	fmt.Fprintf(f, strings.Repeat("-", 79) + "\n")
	printDetailedStats(f, boltSummaries)

	fmt.Fprintf(f, "\nSHOCKWAVE Benchmarks (%d):\n", len(shockwaveSummaries))
	fmt.Fprintf(f, strings.Repeat("-", 79) + "\n")
	printDetailedStats(f, shockwaveSummaries)

	fmt.Fprintf(f, "\n" + strings.Repeat("=", 79) + "\n")
	fmt.Fprintf(f, " Summary\n")
	fmt.Fprintf(f, strings.Repeat("=", 79) + "\n\n")

	printOverallSummary(f, boltSummaries, shockwaveSummaries)

	return nil
}

func filterSummariesByProject(summaries []StatisticalSummary, project string) []StatisticalSummary {
	var filtered []StatisticalSummary
	for _, s := range summaries {
		if s.Project == project {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func printDetailedStats(f *os.File, summaries []StatisticalSummary) {
	fmt.Fprintf(f, "%-40s %12s %12s %12s %12s\n", "Benchmark", "Mean (ns)", "StdDev", "B/op", "allocs/op")
	fmt.Fprintf(f, strings.Repeat("-", 79) + "\n")

	for _, s := range summaries {
		fmt.Fprintf(f, "%-40s %12.2f %12.2f %12.2f %12.2f\n",
			truncate(s.Name, 40), s.MeanNs, s.StdDevNs, s.MeanBytes, s.MeanAllocs)
	}
}

func printOverallSummary(f *os.File, boltSummaries, shockwaveSummaries []StatisticalSummary) {
	boltAvgNs := avgMetric(boltSummaries, func(s StatisticalSummary) float64 { return s.MeanNs })
	boltAvgBytes := avgMetric(boltSummaries, func(s StatisticalSummary) float64 { return s.MeanBytes })
	boltAvgAllocs := avgMetric(boltSummaries, func(s StatisticalSummary) float64 { return s.MeanAllocs })

	shockwaveAvgNs := avgMetric(shockwaveSummaries, func(s StatisticalSummary) float64 { return s.MeanNs })
	shockwaveAvgBytes := avgMetric(shockwaveSummaries, func(s StatisticalSummary) float64 { return s.MeanBytes })
	shockwaveAvgAllocs := avgMetric(shockwaveSummaries, func(s StatisticalSummary) float64 { return s.MeanAllocs })

	fmt.Fprintf(f, "Overall Averages:\n\n")
	fmt.Fprintf(f, "%-20s %15s %15s %15s\n", "Project", "Avg ns/op", "Avg B/op", "Avg allocs/op")
	fmt.Fprintf(f, strings.Repeat("-", 79) + "\n")
	fmt.Fprintf(f, "%-20s %15.2f %15.2f %15.2f\n", "BOLT", boltAvgNs, boltAvgBytes, boltAvgAllocs)
	fmt.Fprintf(f, "%-20s %15.2f %15.2f %15.2f\n", "SHOCKWAVE", shockwaveAvgNs, shockwaveAvgBytes, shockwaveAvgAllocs)

	fmt.Fprintf(f, "\nPerformance Comparison:\n")
	if boltAvgNs > 0 && shockwaveAvgNs > 0 {
		ratio := boltAvgNs / shockwaveAvgNs
		if ratio > 1 {
			fmt.Fprintf(f, "  SHOCKWAVE is %.2fx faster than BOLT on average\n", ratio)
		} else {
			fmt.Fprintf(f, "  BOLT is %.2fx faster than SHOCKWAVE on average\n", 1/ratio)
		}
	}

	if boltAvgBytes > 0 && shockwaveAvgBytes > 0 {
		ratio := boltAvgBytes / shockwaveAvgBytes
		if ratio > 1 {
			fmt.Fprintf(f, "  SHOCKWAVE uses %.2fx less memory than BOLT on average\n", ratio)
		} else {
			fmt.Fprintf(f, "  BOLT uses %.2fx less memory than SHOCKWAVE on average\n", 1/ratio)
		}
	}
}

func avgMetric(summaries []StatisticalSummary, extractor func(StatisticalSummary) float64) float64 {
	if len(summaries) == 0 {
		return 0
	}
	sum := 0.0
	for _, s := range summaries {
		sum += extractor(s)
	}
	return sum / float64(len(summaries))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func printSummary(summaries []StatisticalSummary, rankings []RankingReport) {
	fmt.Println("\n" + strings.Repeat("=", 79))
	fmt.Println(" Quick Summary")
	fmt.Println(strings.Repeat("=", 79))

	// Group by category
	currentCategory := ""
	for _, report := range rankings {
		if len(report.Rankings) == 0 {
			continue
		}

		// Print category header if it changed
		if report.Category != currentCategory {
			currentCategory = report.Category
			categoryTitle := "\nHTTP SERVERS:"
			if report.Category == "web-framework" {
				categoryTitle = "\nWEB FRAMEWORKS:"
			}
			fmt.Println(categoryTitle)
		}

		winner := report.Rankings[0]
		fmt.Printf("  %-28s: %s (score: %.2f)\n", report.Scenario, winner.Project, winner.Score)
	}

	fmt.Println()
}

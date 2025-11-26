// goxc is the GoX compiler CLI tool
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/user/gox/pkg/analyzer"
	"github.com/user/gox/pkg/lexer"
	"github.com/user/gox/pkg/optimizer"
	"github.com/user/gox/pkg/parser"
	"github.com/user/gox/pkg/transpiler/csr"
	"github.com/user/gox/pkg/transpiler/ssr"
)

const version = "0.1.0"

func main() {
	// Define subcommands
	buildCmd := flag.NewFlagSet("build", flag.ExitOnError)
	mode := buildCmd.String("mode", "ssr", "Compilation mode: ssr or csr")
	output := buildCmd.String("o", "dist", "Output directory")
	verbose := buildCmd.Bool("v", false, "Verbose output")
	production := buildCmd.Bool("production", false, "Production build with optimizations")
	watch := buildCmd.Bool("watch", false, "Watch for file changes")

	watchCmd := flag.NewFlagSet("watch", flag.ExitOnError)
	watchMode := watchCmd.String("mode", "ssr", "Compilation mode: ssr or csr")
	watchOutput := watchCmd.String("o", "dist", "Output directory")
	watchVerbose := watchCmd.Bool("v", false, "Verbose output")

	versionCmd := flag.NewFlagSet("version", flag.ExitOnError)

	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	projectName := initCmd.String("name", "my-gox-app", "Project name")

	// Parse command
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "build":
		buildCmd.Parse(os.Args[2:])
		if *watch {
			err := runWatch(*mode, *output, buildCmd.Args(), *verbose)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Watch failed: %v\n", err)
				os.Exit(1)
			}
		} else {
			err := runBuild(*mode, *output, buildCmd.Args(), *verbose, *production)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Build failed: %v\n", err)
				os.Exit(1)
			}
		}

	case "watch":
		watchCmd.Parse(os.Args[2:])
		err := runWatch(*watchMode, *watchOutput, watchCmd.Args(), *watchVerbose)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Watch failed: %v\n", err)
			os.Exit(1)
		}

	case "version":
		versionCmd.Parse(os.Args[2:])
		fmt.Printf("goxc version %s\n", version)

	case "init":
		initCmd.Parse(os.Args[2:])
		err := runInit(*projectName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Init failed: %v\n", err)
			os.Exit(1)
		}

	case "help":
		printUsage()

	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`GoX Compiler - A React-like frontend framework for Go

Usage:
  goxc <command> [options] [files...]

Commands:
  build    Compile .gox files to Go
  watch    Watch for changes and rebuild
  init     Initialize a new GoX project
  version  Show version information
  help     Show this help message

Build Options:
  -mode    Compilation mode: ssr (server-side) or csr (client-side) [default: ssr]
  -o       Output directory [default: dist]
  -v       Verbose output

Examples:
  goxc build -mode=ssr -o=dist src/*.gox
  goxc watch -mode=csr src/
  goxc init -name=my-app`)
}

func runBuild(mode, outputDir string, files []string, verbose, production bool) error {
	if verbose {
		fmt.Printf("Building in %s mode...\n", mode)
		if production {
			fmt.Printf("Production build: enabled\n")
		}
		fmt.Printf("Output directory: %s\n", outputDir)
	}

	// Create output directory
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Find .gox files if no files specified
	if len(files) == 0 {
		files, err = findGoXFiles(".")
		if err != nil {
			return fmt.Errorf("failed to find .gox files: %w", err)
		}
	}

	if len(files) == 0 {
		return fmt.Errorf("no .gox files found")
	}

	if verbose {
		fmt.Printf("Found %d .gox files\n", len(files))
	}

	// Process each file
	for _, file := range files {
		if verbose {
			fmt.Printf("Building %s...\n", file)
		}

		err := buildFile(file, mode, outputDir, verbose, production)
		if err != nil {
			return fmt.Errorf("failed to build %s: %w", file, err)
		}
	}

	fmt.Println("Build complete!")
	return nil
}

func buildFile(file, mode, outputDir string, verbose, production bool) error {
	// Read source file
	source, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Create lexer
	l := lexer.New(source, file)

	// Parse
	p := parser.New(l, file)
	parsedFile, err := p.ParseFile()
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	// Check for parser errors
	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		return fmt.Errorf("parsing failed with %d errors", len(p.Errors()))
	}

	// Analyze
	a := analyzer.New()
	result, err := a.Analyze(parsedFile)
	if err != nil {
		return fmt.Errorf("analysis error: %w", err)
	}

	// Optimize if production build
	if production {
		opt := optimizer.New(optimizer.ProductionOptions())
		for name, comp := range result.Components {
			result.Components[name] = opt.OptimizeComponent(comp)
		}
	}

	// Print warnings if verbose
	if verbose && len(result.Warnings) > 0 {
		for _, warning := range result.Warnings {
			fmt.Printf("Warning: %s\n", warning)
		}
	}

	// Transpile based on mode
	switch mode {
	case "ssr":
		err = transpileSSR(result, file, outputDir, verbose, production)
	case "csr":
		err = transpileCSR(result, file, outputDir, verbose, production)
	default:
		return fmt.Errorf("unknown mode: %s", mode)
	}

	if err != nil {
		return fmt.Errorf("transpilation error: %w", err)
	}

	return nil
}

func transpileSSR(result *analyzer.AnalysisResult, sourceFile, outputDir string, verbose, production bool) error {
	// Get package name from source file
	packageName := "main"
	if result.Components != nil && len(result.Components) > 0 {
		// Use the directory name as package name
		dir := filepath.Dir(sourceFile)
		if dir != "." && dir != "/" {
			packageName = filepath.Base(dir)
			// Replace hyphens with underscores for valid Go package names
			packageName = strings.ReplaceAll(packageName, "-", "_")
			// If the name starts with a digit, prepend "pkg"
			if len(packageName) > 0 && packageName[0] >= '0' && packageName[0] <= '9' {
				packageName = "pkg" + packageName
			}
		}
	}

	// Create transpiler
	t := ssr.New(packageName)

	// Convert components to array
	var components []*analyzer.ComponentIR
	for _, comp := range result.Components {
		components = append(components, comp)
	}

	// Generate Go code
	code, err := t.GenerateFile(components)
	if err != nil {
		return err
	}

	// Write output file
	outputFile := filepath.Join(outputDir, strings.TrimSuffix(filepath.Base(sourceFile), ".gox")+".go")
	err = ioutil.WriteFile(outputFile, code, 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	if verbose {
		fmt.Printf("  Generated %s\n", outputFile)
	}

	return nil
}

func transpileCSR(result *analyzer.AnalysisResult, sourceFile, outputDir string, verbose, production bool) error {
	// Get package name
	packageName := "main"

	// Create CSR transpiler
	t := csr.New(packageName)

	// Convert components to array
	var components []*analyzer.ComponentIR
	for _, comp := range result.Components {
		components = append(components, comp)
	}

	// Generate WASM-compatible Go code for each component
	for _, comp := range components {
		code, err := t.Transpile(comp)
		if err != nil {
			return fmt.Errorf("failed to transpile component %s: %w", comp.Name, err)
		}

		// Apply production optimizations
		if production {
			opt := optimizer.New(optimizer.ProductionOptions())
			code = opt.OptimizeGo(code)
		}

		// Write output file
		outputFile := filepath.Join(outputDir, strings.TrimSuffix(filepath.Base(sourceFile), ".gox")+"_wasm.go")
		err = ioutil.WriteFile(outputFile, code, 0644)
		if err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}

		if verbose {
			fmt.Printf("Generated WASM component: %s\n", outputFile)
		}
	}

	// Generate build script for WASM compilation
	buildScript := generateWASMBuildScript(outputDir)
	scriptFile := filepath.Join(outputDir, "build_wasm.sh")
	err := ioutil.WriteFile(scriptFile, []byte(buildScript), 0755)
	if err != nil {
		return fmt.Errorf("failed to write build script: %w", err)
	}

	if verbose {
		fmt.Printf("Generated WASM build script: %s\n", scriptFile)
		fmt.Println("To compile to WASM, run: sh " + scriptFile)
	}

	return nil
}

func generateWASMBuildScript(outputDir string) string {
	return fmt.Sprintf(`#!/bin/bash
# GoX WASM Build Script
# Generated by goxc

echo "Building WASM components..."

# Set WASM build environment
export GOOS=js
export GOARCH=wasm

# Find all _wasm.go files
for file in %s/*_wasm.go; do
    if [ -f "$file" ]; then
        base=$(basename "$file" _wasm.go)
        echo "Compiling $base to WASM..."
        go build -o "%s/${base}.wasm" "$file"

        if [ $? -eq 0 ]; then
            echo "✓ Generated ${base}.wasm"
        else
            echo "✗ Failed to compile ${base}"
            exit 1
        fi
    fi
done

# Copy WASM support file
if [ ! -f "%s/wasm_exec.js" ]; then
    echo "Copying wasm_exec.js..."
    cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" "%s/"
fi

echo ""
echo "WASM build complete!"
echo "Files generated in: %s"
echo ""
echo "To serve your WASM component:"
echo "1. Include wasm_exec.js in your HTML"
echo "2. Load and instantiate the .wasm file"
echo ""
echo "Example HTML:"
echo '<!DOCTYPE html>'
echo '<html>'
echo '<head>'
echo '    <script src="wasm_exec.js"></script>'
echo '    <script>'
echo '        const go = new Go();'
echo '        WebAssembly.instantiateStreaming(fetch("component.wasm"), go.importObject).then((result) => {'
echo '            go.run(result.instance);'
echo '        });'
echo '    </script>'
echo '</head>'
echo '<body>'
echo '    <div id="root"></div>'
echo '</body>'
echo '</html>'
`, outputDir, outputDir, outputDir, outputDir, outputDir)
}

func runWatch(mode, outputDir string, paths []string, verbose bool) error {
	fmt.Printf("Watching for changes in %s mode...\n", mode)
	fmt.Println("Press Ctrl+C to stop")

	// Watch implementation would go here
	// For now, just build once
	return runBuild(mode, outputDir, paths, verbose, false)
}

func runInit(projectName string) error {
	fmt.Printf("Initializing new GoX project: %s\n", projectName)

	// Create project directory
	err := os.MkdirAll(projectName, 0755)
	if err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Create go.mod
	goModContent := fmt.Sprintf(`module %s

go 1.21

require github.com/user/gox v0.1.0
`, projectName)

	err = ioutil.WriteFile(filepath.Join(projectName, "go.mod"), []byte(goModContent), 0644)
	if err != nil {
		return err
	}

	// Create src directory
	err = os.MkdirAll(filepath.Join(projectName, "src"), 0755)
	if err != nil {
		return err
	}

	// Create example component
	exampleComponent := `package main

import "gox"

component App() {
	title, setTitle := gox.UseState[string]("Hello, GoX!")
	count, setCount := gox.UseState[int](0)

	handleClick := func() {
		setCount(count + 1)
	}

	render {
		<div className="app">
			<h1>{title}</h1>
			<p>You clicked {count} times</p>
			<button onClick={handleClick}>
				Click me
			</button>
		</div>
	}

	style {
		.app {
			max-width: 600px;
			margin: 0 auto;
			padding: 20px;
			font-family: sans-serif;
		}

		h1 {
			color: #333;
		}

		button {
			background: #007bff;
			color: white;
			border: none;
			padding: 10px 20px;
			border-radius: 4px;
			cursor: pointer;
			font-size: 16px;
		}

		button:hover {
			background: #0056b3;
		}
	}
}
`

	err = ioutil.WriteFile(filepath.Join(projectName, "src", "App.gox"), []byte(exampleComponent), 0644)
	if err != nil {
		return err
	}

	// Create README
	readmeContent := fmt.Sprintf(`# %s

A GoX application.

## Getting Started

1. Install dependencies:
   ` + "```bash" + `
   go mod tidy
   ` + "```" + `

2. Build the application:
   ` + "```bash" + `
   goxc build -o dist src/*.gox
   ` + "```" + `

3. Run the application:
   ` + "```bash" + `
   go run dist/*.go
   ` + "```" + `

## Development

Watch for changes:
` + "```bash" + `
goxc watch src/
` + "```" + `

## Learn More

Visit [GoX Documentation](https://github.com/user/gox) for more information.
`, projectName)

	err = ioutil.WriteFile(filepath.Join(projectName, "README.md"), []byte(readmeContent), 0644)
	if err != nil {
		return err
	}

	fmt.Printf("\nProject created successfully!\n")
	fmt.Printf("Next steps:\n")
	fmt.Printf("  cd %s\n", projectName)
	fmt.Printf("  goxc build src/*.gox\n")

	return nil
}

func findGoXFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".gox") {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}
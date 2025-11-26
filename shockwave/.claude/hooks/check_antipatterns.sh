#!/bin/bash
# Check for common performance anti-patterns in Go code

# Get the tool input (file content or file path)
INPUT="$1"

# Define anti-patterns to check
declare -A ANTIPATTERNS=(
    ["string concatenation"]='[^"]\+[[:space:]]*"'
    ["fmt.Sprintf in loop"]='for.*fmt\.Sprintf'
    ["defer in tight loop"]='for.*\{.*defer'
    ["append without capacity"]='append\([^,]*\)'
    ["map for fixed set"]='map\[string\]'
)

# Track if any issues found
ISSUES_FOUND=0

# Check for anti-patterns
for pattern_name in "${!ANTIPATTERNS[@]}"; do
    pattern="${ANTIPATTERNS[$pattern_name]}"

    if echo "$INPUT" | grep -q "$pattern"; then
        echo "⚠️  Warning: Potential performance anti-pattern detected: $pattern_name" >&2
        ISSUES_FOUND=1
    fi
done

# Check for specific hot path files
if echo "$INPUT" | grep -qE "parser\.go|pool.*\.go"; then
    echo "ℹ️  Note: Editing hot path file. Please verify zero allocations with /check-allocs after changes." >&2
fi

# Exit 0 to allow (warning only), exit 1 to block
exit 0

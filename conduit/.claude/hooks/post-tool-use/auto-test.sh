#!/bin/bash
# Post-write hook to auto-run tests for modified packages

# Check if a Go test file was written/edited
if echo "$TOOL_NAME" | grep -qE '(Write|Edit)'; then
    if echo "$TOOL_INPUT" | grep -q '_test\.go'; then
        # Extract package directory
        FILE_PATH=$(echo "$TOOL_INPUT" | grep -oE '/[^"]+_test\.go' | head -1)
        PKG_DIR=$(dirname "$FILE_PATH")

        if [ -d "$PKG_DIR" ]; then
            echo "🧪 Running tests for package: $PKG_DIR"
            go test "$PKG_DIR"

            if [ $? -eq 0 ]; then
                echo "✅ Tests passed"
            else
                echo "❌ Tests failed - review output above"
            fi
        fi
    fi
fi

exit 0

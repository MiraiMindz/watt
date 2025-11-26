#!/bin/bash
# Post-write hook to auto-format Go code

# Check if a .go file was written/edited
if echo "$TOOL_NAME" | grep -qE '(Write|Edit)'; then
    if echo "$TOOL_INPUT" | grep -q '\.go'; then
        # Extract file path
        FILE_PATH=$(echo "$TOOL_INPUT" | grep -oE '/[^"]+\.go' | head -1)

        if [ -f "$FILE_PATH" ]; then
            echo "🔧 Auto-formatting Go file: $FILE_PATH"
            gofmt -w "$FILE_PATH"

            if [ $? -eq 0 ]; then
                echo "✅ Formatted successfully"
            else
                echo "⚠️  gofmt failed"
            fi
        fi
    fi
fi

exit 0

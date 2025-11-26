#!/usr/bin/env python3
"""
Fix HTTP/11 benchmark tests to use new NewConnection API.
Changes:
1. Move handler definition before the loop/NewConnection call
2. Pass handler as 3rd argument to NewConnection
3. Call Serve() with no arguments
"""

import re
import sys

def fix_benchmark_pattern(content):
    # Pattern 1: Handler defined inside loop - move it outside and fix calls
    pattern1 = r'(for i := 0; i < b\.N; i\+\+ \{\s+mockConn := newMockConn\(requestData\)\s+config := DefaultConnectionConfig\(\)\s+)conn := NewConnection\(mockConn, config\)(\s+)(handler := func\(req \*Request, rw \*ResponseWriter\) error \{[^}]+\})(\s+)conn\.Serve\(handler\)'

    def replace1(match):
        return f"conn := NewConnection(mockConn, config, handler){match.group(2)}{match.group(4)}conn.Serve()"

    # First, move handlers outside loops
    # Find pattern: for loop { ... NewConnection(...) ... handler := ... Serve(handler) }
    lines = content.split('\n')
    result = []
    i = 0

    while i < len(lines):
        line = lines[i]

        # Check if this is a 'for i := 0; i < b.N' loop
        if 'for i := 0; i < b.N; i++' in line:
            # Look ahead for the pattern
            j = i + 1
            handler_def = None
            newconn_line_idx = None
            serve_line_idx = None

            while j < len(lines) and j < i + 20:  # Look up to 20 lines ahead
                if 'handler := func(req *Request, rw *ResponseWriter)' in lines[j]:
                    # Found handler definition, extract it (may span multiple lines)
                    handler_start = j
                    brace_count = 0
                    handler_lines = []
                    for k in range(j, min(j + 15, len(lines))):
                        handler_lines.append(lines[k])
                        brace_count += lines[k].count('{') - lines[k].count('}')
                        if brace_count == 0 and '{' in lines[k]:
                            handler_def = '\n'.join(handler_lines)
                            serve_line_idx = k + 2  # Serve is usually 2 lines after handler end
                            break
                    break
                if 'NewConnection(mockConn, config)' in lines[j]:
                    newconn_line_idx = j
                j += 1

            # If we found the pattern, fix it
            if handler_def and newconn_line_idx:
                # Add handler before the for loop
                result.append('\t' + handler_def.strip())
                result.append('')
                result.append(line)  # Add the for loop line

                # Skip to after handler definition and fix the calls
                i += 1
                while i <= serve_line_idx:
                    if i == newconn_line_idx:
                        # Fix NewConnection call
                        result.append(lines[i].replace('NewConnection(mockConn, config)', 'NewConnection(mockConn, config, handler)'))
                    elif 'handler := func' in lines[i]:
                        # Skip handler definition lines (already moved)
                        pass
                    elif 'conn.Serve(handler)' in lines[i]:
                        # Fix Serve call
                        result.append(lines[i].replace('conn.Serve(handler)', 'conn.Serve()'))
                    else:
                        result.append(lines[i])
                    i += 1
                continue

        result.append(line)
        i += 1

    return '\n'.join(result)

def main():
    if len(sys.argv) != 2:
        print("Usage: fix_benchmarks.py <file>")
        sys.exit(1)

    filename = sys.argv[1]

    with open(filename, 'r') as f:
        content = f.read()

    fixed_content = fix_benchmark_pattern(content)

    with open(filename, 'w') as f:
        f.write(fixed_content)

    print(f"Fixed {filename}")

if __name__ == '__main__':
    main()

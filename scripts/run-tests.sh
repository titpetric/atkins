#!/bin/bash
# run-tests.sh - Run all documentation examples and capture output with termos shot
#
# Usage: ./scripts/run-tests.sh
#
# This script:
# 1. Discovers all .yml example files in docs/content/
# 2. Runs each with: atkins -f file.yml --log file.log --final
# 3. Captures output to file.txt
# 4. Takes a screenshot with termos shot
# 5. Verifies exit code is 0
# 6. Reports results

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DOCS_DIR="$ROOT_DIR/docs/content"

# Timeout for each test (seconds)
TIMEOUT=30

# Files to skip (not valid atkins pipelines)
SKIP_PATTERNS="taskfile-before.yml workflow-before.yml theme.yml"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0
SKIPPED=0
TOTAL=0

# Track failures for summary
FAILURES=""

should_skip() {
    local file="$1"
    local basename
    basename=$(basename "$file")

    for pattern in $SKIP_PATTERNS; do
        if [[ "$basename" == "$pattern" ]]; then
            return 0
        fi
    done
    return 1
}

run_test() {
    local yml_file="$1"
    local yml_dir
    local yml_name
    local rel_path
    local exit_code

    yml_dir=$(dirname "$yml_file")
    yml_name=$(basename "$yml_file" .yml)
    rel_path="${yml_file#$ROOT_DIR/}"

    TOTAL=$((TOTAL + 1))

    # Skip non-atkins files
    if should_skip "$yml_file"; then
        echo -e "${YELLOW}[SKIP]${NC} $rel_path"
        SKIPPED=$((SKIPPED + 1))
        return 0
    fi

    # Change to the yml directory so relative paths work
    cd "$yml_dir"

    # Run atkins with timeout and capture output
    if timeout "$TIMEOUT" atkins -f "$(basename "$yml_file")" -w "$yml_dir" --log "$yml_name.log" --final > "$yml_name.txt" 2>&1; then
        exit_code=0
    else
        exit_code=$?
    fi

    # Check exit code
    if [[ $exit_code -eq 0 ]]; then
        echo -e "${GREEN}[PASS]${NC} $rel_path"
        PASSED=$((PASSED + 1))

        # Take screenshot with termos shot if available
        if command -v termos &> /dev/null; then
            termos shot -c 80 --filename "$yml_name" -- atkins -f "$(basename "$yml_file")" --final >/dev/null 2>&1 || true
        fi
    elif [[ $exit_code -eq 124 ]]; then
        echo -e "${RED}[FAIL]${NC} $rel_path (timeout after ${TIMEOUT}s)"
        FAILED=$((FAILED + 1))
        FAILURES="$FAILURES\n  - $rel_path (timeout)"
    else
        echo -e "${RED}[FAIL]${NC} $rel_path (exit code: $exit_code)"
        FAILED=$((FAILED + 1))
        FAILURES="$FAILURES\n  - $rel_path"

        # Show output content on failure
        if [[ -f "$yml_name.txt" ]]; then
            echo "  Output:"
            head -10 "$yml_name.txt" | sed 's/^/    /'
        fi
    fi

    cd "$ROOT_DIR"
}

# Find all yml files in docs/content
echo "Running documentation example tests..."
echo "========================================"
echo ""

cd "$ROOT_DIR"

# Use find and process files
for yml_file in $(find "$DOCS_DIR" -type f -name "*.yml" ! -path "*/data/*" ! -path "*/.vale/*" | sort); do
    run_test "$yml_file"
done

echo ""
echo "========================================"
echo "Summary: $PASSED passed, $FAILED failed, $SKIPPED skipped (total: $TOTAL)"

# Print failures if any
if [[ -n "$FAILURES" ]]; then
    echo ""
    echo "Failed tests:"
    echo -e "$FAILURES"
fi

# Exit with appropriate code
if [[ $FAILED -gt 0 ]]; then
    exit 1
fi

exit 0

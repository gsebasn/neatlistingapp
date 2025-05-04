#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print section headers
print_section() {
    echo -e "\n${YELLOW}=== $1 ===${NC}"
}

# Function to check if a command exists
check_command() {
    if ! command -v $1 &> /dev/null; then
        echo -e "${RED}Error: $1 is not installed${NC}"
        exit 1
    fi
}

# Check if required commands are installed
check_command go
check_command staticcheck

# Print Go version
echo -e "${YELLOW}Go version:${NC}"
go version

# Run go mod tidy
print_section "Running go mod tidy"
go mod tidy
if [ $? -ne 0 ]; then
    echo -e "${RED}Error: go mod tidy failed${NC}"
    exit 1
fi

# Run go vet
print_section "Running go vet"
go vet listing/...
if [ $? -ne 0 ]; then
    echo -e "${RED}Error: go vet found issues${NC}"
    exit 1
fi

# Run gofmt
print_section "Running gofmt"
gofmt_files=$(gofmt -l -s .)
if [ ! -z "$gofmt_files" ]; then
    echo -e "${RED}Error: The following files need formatting:${NC}"
    echo "$gofmt_files"
    exit 1
fi

# Run staticcheck
print_section "Running staticcheck"
staticcheck listing/...
if [ $? -ne 0 ]; then
    echo -e "${RED}Error: staticcheck found issues${NC}"
    exit 1
fi

# Run tests
print_section "Running tests"
go test -v listing/...
if [ $? -ne 0 ]; then
    echo -e "${RED}Error: Tests failed${NC}"
    exit 1
fi

# If we get here, everything passed
echo -e "\n${GREEN}All checks passed successfully!${NC}" 
#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print section headers
print_section() {
    echo -e "\n${YELLOW}=== $1 ===${NC}"
}

# Function to print info messages
print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

# Function to print success messages
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Function to print error messages
print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Function to check if a command exists
check_command() {
    if ! command -v $1 &> /dev/null; then
        print_error "$1 is not installed"
        exit 1
    fi
}

# Check if required commands are installed
check_command go
check_command staticcheck

# Print Go version
print_section "Environment"
go version

# Run the linter script
print_section "Running Linter"
./scripts/lint.sh
if [ $? -ne 0 ]; then
    print_error "Linting failed. Aborting build."
    exit 1
fi

# Build the application
print_section "Building Application"
print_info "Building main application..."
go build -o main
if [ $? -ne 0 ]; then
    print_error "Build failed"
    exit 1
fi
print_success "Build successful"

# Check if environment file exists
if [ ! -f "env.dev" ]; then
    print_error "env.dev file not found"
    exit 1
fi

# Run the application
print_section "Starting Application"
print_info "Loading environment from env.dev..."
print_info "Starting server..."

# Run the application with the development environment
./main
if [ $? -ne 0 ]; then
    print_error "Application crashed"
    exit 1
fi 
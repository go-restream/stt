#!/bin/bash

# Local CI Testing Script
# This script provides a simple way to run CI-like tests locally

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if required tools are installed
check_requirements() {
    print_status "Checking requirements..."

    if ! command -v go &> /dev/null; then
        print_error "Go is not installed"
        exit 1
    fi

    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed"
        exit 1
    fi

    if ! command -v act &> /dev/null; then
        print_error "act is not installed. Install with: brew install act"
        exit 1
    fi

    print_status "All requirements are satisfied"
}

# Run Go tests
run_tests() {
    print_status "Running Go tests..."

    if go test -v -race -coverprofile=coverage.out ./...; then
        print_status "✅ Tests passed"

        # Show coverage summary
        if command -v go &> /dev/null; then
            go tool cover -func=coverage.out | tail -1
        fi
    else
        print_error "❌ Tests failed"
        exit 1
    fi
}

# Build project
build_project() {
    print_status "Building project..."

    # Clean previous builds
    make clean

    # Use Makefile to build (handles platform-specific dependencies)
    if make build; then
        print_status "✅ Build successful"
        ls -la build/
    else
        print_error "❌ Build failed"
        exit 1
    fi
}

# Run security scan
run_security_scan() {
    print_status "Running security scan..."

    # Install gosec if not present
    if ! command -v gosec &> /dev/null; then
        print_warning "gosec not found, installing..."
        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
    fi

    # Run gosec
    if gosec -no-fail -fmt sarif -out results.sarif ./...; then
        print_status "✅ Security scan completed"
    else
        print_warning "⚠️ Security scan found issues (continuing)"
    fi
}

# Build Docker image
build_docker() {
    print_status "Building Docker image..."

    if docker build -t streamasr:local .; then
        print_status "✅ Docker build successful"

        # Show image info
        docker images streamasr:local
    else
        print_error "❌ Docker build failed"
        exit 1
    fi
}

# Run act workflow
run_act_workflow() {
    local job_name=${1:-"test"}

    print_status "Running act workflow: $job_name"

    # Create a simple secrets file for act
    cat > .act-secrets <<EOF
GITHUB_TOKEN=mock-token-for-local-testing
CODECOV_TOKEN=
EOF

    # Run act with proper flags for Apple Silicon
    if act -j $job_name \
        --platform ubuntu-latest=nektos/act-environments-ubuntu:18.04 \
        --secret-file .act-secrets \
        --container-architecture linux/amd64 \
        --bind; then
        print_status "✅ Act workflow $job_name completed"
    else
        print_warning "⚠️ Act workflow $job_name had issues (this is normal for local testing)"
    fi

    # Clean up secrets file
    rm -f .act-secrets
}

# Show usage
show_usage() {
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  test          Run Go tests"
    echo "  build         Build project"
    echo "  security      Run security scan"
    echo "  docker        Build Docker image"
    echo "  act [job]     Run act workflow (default: test)"
    echo "  all           Run all checks (test, build, security, docker)"
    echo "  help          Show this help message"
}

# Main script logic
main() {
    case "${1:-help}" in
        "test")
            check_requirements
            run_tests
            ;;
        "build")
            check_requirements
            build_project
            ;;
        "security")
            check_requirements
            run_security_scan
            ;;
        "docker")
            check_requirements
            build_docker
            ;;
        "act")
            check_requirements
            run_act_workflow ${2:-"test"}
            ;;
        "all")
            check_requirements
            run_tests
            build_project
            run_security_scan
            build_docker
            print_status "🎉 All checks completed successfully!"
            ;;
        "help"|*)
            show_usage
            ;;
    esac
}

# Run main function with all arguments
main "$@"
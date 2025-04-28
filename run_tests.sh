#!/bin/bash

# Don't halt on errors for the entire script
set +e

# Check if we need sudo
if docker info > /dev/null 2>&1; then
    USE_SUDO=""
else
    echo "Docker requires elevated permissions. Using sudo..."
    USE_SUDO="sudo"
fi

# Ensure we're in the project root directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Create test requirements file if it doesn't exist
if [ ! -f test_requirements.txt ]; then
    echo "Creating test_requirements.txt..."
    echo "selenium==4.15.2" > test_requirements.txt
    echo "psycopg2-binary==2.9.9" >> test_requirements.txt
    echo "pytest==7.4.3" >> test_requirements.txt
    echo "requests==2.31.0" >> test_requirements.txt
fi

echo "Building and starting services..."

# First, make sure to clean up any existing containers
echo "Cleaning up any existing containers..."
$USE_SUDO docker-compose -f docker/docker-compose.test.yml down -v || true

# Build the test container
echo "Building test container..."
$USE_SUDO docker build -t minitwit-uitests -f docker/Dockerfile.test .

# Build the minitwit app
echo "Building minitwit application..."
$USE_SUDO docker build -t minitwit-app -f docker/Dockerfile .

# Start PostgreSQL first
echo "Starting PostgreSQL database..."
$USE_SUDO docker-compose -f docker/docker-compose.test.yml up -d postgres
echo "Waiting for PostgreSQL to be healthy..."
sleep 15

# Start minitwit
echo "Starting MiniTwit application..."
$USE_SUDO docker-compose -f docker/docker-compose.test.yml up -d minitwit

# Wait for minitwit to become ready
echo "Waiting for MiniTwit to initialize..."
sleep 15

# Check if MiniTwit container is running
echo "Checking if MiniTwit container is running:"
if ! $USE_SUDO docker ps | grep -q minitwit-app; then
    echo "ERROR: MiniTwit container is not running. Tests will not be executed."
    $USE_SUDO docker logs minitwit-app
    $USE_SUDO docker-compose -f docker/docker-compose.test.yml down -v
    exit 1
fi

# Run integration tests
echo "Running UI and API tests..."
$USE_SUDO docker-compose -f docker/docker-compose.test.yml up --abort-on-container-exit uitests

# Get the exit code of the test container
TEST_EXIT_CODE=$($USE_SUDO docker inspect -f '{{.State.ExitCode}}' minitwit-uitests || echo "1")

# Cleanup
echo "Cleaning up containers and volumes..."
$USE_SUDO docker-compose -f docker/docker-compose.test.yml down -v

# Now run the Go unit tests
echo "Running Go unit tests..."

# Initialize counters
TOTAL_TESTS=3
PASSED_TESTS=0
FAILED_TESTS=0
FAILED_TEST_NAMES=""

# Test handlers
echo "Running handlers_test.go..."
cd ./minitwit_test
go test -v handlers_test.go
if [ $? -eq 0 ]; then
    PASSED_TESTS=$((PASSED_TESTS+1))
else
    FAILED_TESTS=$((FAILED_TESTS+1))
    FAILED_TEST_NAMES="$FAILED_TEST_NAMES handlers_test"
fi

# Test models
echo "Running models_test.go..."
go test -v models_test.go
if [ $? -eq 0 ]; then
    PASSED_TESTS=$((PASSED_TESTS+1))
else
    FAILED_TESTS=$((FAILED_TESTS+1))
    FAILED_TEST_NAMES="$FAILED_TEST_NAMES models_test"
fi

# Test DB
echo "Running db_test.go..."
go test -v db_test.go
if [ $? -eq 0 ]; then
    PASSED_TESTS=$((PASSED_TESTS+1))
else
    FAILED_TESTS=$((FAILED_TESTS+1))
    FAILED_TEST_NAMES="$FAILED_TEST_NAMES db_test"
fi
cd ..

# Make sure we print the summary without trying to use /dev/tty
echo ""
echo "===== TEST SUMMARY ====="
echo "Integration tests: $([ "$TEST_EXIT_CODE" -eq 0 ] && echo "PASSED" || echo "FAILED")"
echo "Go unit tests: $PASSED_TESTS passed, $FAILED_TESTS failed out of $TOTAL_TESTS"

if [ "$FAILED_TESTS" -gt 0 ]; then
    echo "Failed tests:$FAILED_TEST_NAMES"
fi

# Calculate final exit code
FINAL_EXIT_CODE=0
if [ "$TEST_EXIT_CODE" -ne 0 ] || [ "$FAILED_TESTS" -gt 0 ]; then
    FINAL_EXIT_CODE=1
fi

echo "===== END TEST SUMMARY ====="
echo "Tests completed with final exit code: $FINAL_EXIT_CODE"

exit $FINAL_EXIT_CODE
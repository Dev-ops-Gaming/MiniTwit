#!/bin/bash

# Stop on first error
set -e

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
fi

echo "Building and starting services..."

# First, make sure to clean up any existing containers
echo "Cleaning up any existing containers..."
$USE_SUDO docker-compose -f docker/docker-compose.test.yml down -v || true

# Build only the test container since we already have minitwit
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

# Check that postgres is running and accessible
echo "Checking PostgreSQL container logs:"
$USE_SUDO docker logs minitwit-postgres

# Try accessing the database directly 
echo "Testing direct connection to PostgreSQL:"
$USE_SUDO docker exec minitwit-postgres psql -U myuser -d postgres -c "SELECT version();" || echo "Failed to connect to PostgreSQL"

# Start minitwit with full logs
echo "Starting MiniTwit application..."
$USE_SUDO docker-compose -f docker/docker-compose.test.yml up -d minitwit

# Wait a bit for minitwit to start
sleep 10

# Check container logs
echo "MiniTwit application logs:"
$USE_SUDO docker logs minitwit-app

# If MiniTwit containers exited, check its environment variables
echo "MiniTwit environment variables:"
$USE_SUDO docker inspect --format='{{range .Config.Env}}{{println .}}{{end}}' minitwit-app

# Check if MiniTwit container is running
echo "Checking if MiniTwit container is running:"
if $USE_SUDO docker ps | grep -q minitwit-app; then
    echo "MiniTwit container is running"
else
    echo "ERROR: MiniTwit container is not running"
    echo "Container state:"
    $USE_SUDO docker inspect minitwit-app | grep -A 20 "State"
    
    # Try to manually run minitwit to see the output
    echo "Trying to run minitwit manually to see direct output:"
    $USE_SUDO docker run --rm --network docker_minitwit-network \
        -e DB_HOST=postgres \
        -e DB_USER=myuser \
        -e DB_PASSWORD=mypassword \
        -e DB_DBNAME=postgres \
        -e DB_PORT=5432 \
        -e DB_SSLMODE=disable \
        -e DB_TIMEZONE="Europe/Copenhagen" \
        -e POSTGRES_CONNECTION="host=postgres user=myuser password=mypassword dbname=postgres port=5432 sslmode=disable TimeZone=Europe/Copenhagen" \
        minitwit-app
    
    echo "Manual minitwit run completed."
fi

# Don't run tests if the minitwit app isn't running
echo "Checking if MiniTwit is running before attempting tests..."
if ! $USE_SUDO docker ps | grep -q minitwit-app; then
    echo "ERROR: MiniTwit container is not running. Tests will not be executed."
    exit 1
fi

# Run tests
echo "Running tests..."
$USE_SUDO docker-compose -f docker/docker-compose.test.yml up --abort-on-container-exit uitests

# Get the exit code of the test container
TEST_EXIT_CODE=$($USE_SUDO docker inspect -f '{{.State.ExitCode}}' minitwit-uitests || echo "1")

# Cleanup
echo "Cleaning up containers and volumes..."
$USE_SUDO docker-compose -f docker/docker-compose.test.yml down -v

echo "Tests completed with exit code: $TEST_EXIT_CODE"
exit $TEST_EXIT_CODE
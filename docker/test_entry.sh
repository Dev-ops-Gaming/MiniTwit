#!/bin/bash
set -e

echo "Starting test entry script..."

# Ensure environment variables are set
GUI_HOST=${GUI_HOST:-minitwit}
GUI_PORT=${GUI_PORT:-8080}
MAX_RETRIES=${MAX_RETRIES:-30}
RETRY_DELAY=${RETRY_DELAY:-2}

# Display network info for debugging
echo "Network configuration:"
ip addr
echo "DNS servers:"
cat /etc/resolv.conf
echo "Hosts file:"
cat /etc/hosts

# Verify DNS resolution works
echo "Checking DNS resolution for $GUI_HOST..."
if ! getent hosts $GUI_HOST; then
  echo "DNS resolution for $GUI_HOST failed. Trying to find minitwit container..."
  
  # Try direct container IP lookup
  MINITWIT_IP=$(ping -c 1 minitwit-app 2>/dev/null | head -1 | grep -oE '([0-9]{1,3}\.){3}[0-9]{1,3}' | head -1)
  
  if [ -z "$MINITWIT_IP" ]; then
    echo "Trying to find host gateway IP..."
    HOST_GATEWAY=$(ip route show | grep default | awk '{print $3}')
    if curl -s -m 1 "http://$HOST_GATEWAY:$GUI_PORT" > /dev/null 2>&1; then
      MINITWIT_IP=$HOST_GATEWAY
      echo "Found minitwit app at host gateway IP: $MINITWIT_IP"
    fi
  fi
  
  if [ -n "$MINITWIT_IP" ]; then
    echo "Adding minitwit to hosts file with IP $MINITWIT_IP"
    echo "$MINITWIT_IP $GUI_HOST" >> /etc/hosts
  else
    echo "Warning: Could not determine IP for $GUI_HOST"
    echo "Will attempt to continue anyway..."
  fi
fi

# Try using direct IP as fallback
echo "Testing access to minitwit container..."
echo "Attempting with hostname: $GUI_HOST"
curl -v "http://$GUI_HOST:$GUI_PORT" || echo "Could not connect with hostname"

# Wait for MiniTwit service to be ready
echo "Waiting for MiniTwit service to be accessible at http://$GUI_HOST:$GUI_PORT..."
for i in $(seq 1 $MAX_RETRIES); do
  echo "Attempt $i/$MAX_RETRIES: Checking if MiniTwit is ready..."
  if curl -s -m 2 "http://$GUI_HOST:$GUI_PORT" > /dev/null 2>&1; then
    echo "MiniTwit is up and running!"
    break
  elif curl -s -m 2 "http://$GUI_HOST:$GUI_PORT/register" > /dev/null 2>&1; then
    echo "MiniTwit /register endpoint is accessible!"
    break
  fi
  
  echo "MiniTwit is not ready yet, waiting $RETRY_DELAY seconds..."
  sleep $RETRY_DELAY
  
  if [ $i -eq $MAX_RETRIES ]; then
    echo "Warning: Timed out waiting for MiniTwit! Continuing anyway..."
  fi
done

# Print environment for debugging
echo "Test environment:"
echo "GUI_HOST=$GUI_HOST"
echo "GUI_PORT=$GUI_PORT"
echo "DB_HOST=$DB_HOST"
echo "DB_PORT=$DB_PORT"

# Run the tests
echo "Running tests..."
exec python -m pytest test_itu_minitwit_ui.py -v

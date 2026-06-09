#!/bin/bash
# Restart script for trace-point server
# Run this to restart the server with the new binary

cd /home/rut/Project/trace-point-new

# Stop existing server (if running)
if pgrep -f "./server" > /dev/null; then
    echo "Stopping existing server..."
    pkill -f "./server" || sudo pkill -f "./server"
    sleep 2
fi

# Start new server
echo "Starting server..."
./server > /tmp/server.log 2>&1 &
sleep 3

# Verify it's running
if pgrep -f "./server" > /dev/null; then
    echo "Server started successfully!"
else
    echo "Failed to start server. Check /tmp/server.log"
fi
#!/bin/bash

echo "Starting Productivity Tracker test..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go and try again."
    exit 1
fi

# Check if SQLite is installed
if ! command -v sqlite3 &> /dev/null; then
    echo "Warning: SQLite3 command-line tool is not installed. You won't be able to directly query the database."
fi

# Build the application
echo "Building application..."
go build -o productivity-tracker main.go
if [ $? -ne 0 ]; then
    echo "Error: Failed to build the application."
    exit 1
fi

# Run the application in the background
echo "Starting server in the background..."
./productivity-tracker &
SERVER_PID=$!

# Wait for the server to start
echo "Waiting for server to start..."
sleep 2

# Check if the server is running
if ! ps -p $SERVER_PID > /dev/null; then
    echo "Error: Server failed to start."
    exit 1
fi

echo "Server started successfully with PID: $SERVER_PID"
echo "You can now test the application by opening http://localhost:3131/ambition in your browser."
echo "Press Ctrl+C to stop the server when you're done testing."

# Wait for user to press Ctrl+C
trap "kill $SERVER_PID; echo 'Server stopped.'; exit 0" INT
wait $SERVER_PID

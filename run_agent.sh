#!/bin/bash

# Set a flag to indicate whether the loop should continue
RUNNING=true

# Function to handle Ctrl+C
function ctrl_c() {
    echo "Exiting..."
    RUNNING=false
}

# Trap Ctrl+C and call the function to handle it
trap ctrl_c SIGINT

# Infinite loop
while $RUNNING; do
    # Your code here
    go run main.go ./secrets/
    sleep 10
done

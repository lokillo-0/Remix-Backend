#!/bin/bash

echo "Building Xenon..."
go build 

if [ $? -ne 0 ]; then
    echo "Build failed! Exiting."
    exit 1
fi

echo "Build successful! Starting Xenon with auto-restart..."

mkdir -p logs

restart_count=0
max_restarts_per_minute=5
last_restart_time=0

while true; do
    current_time=$(date +%s)
    
    if [ $((current_time - last_restart_time)) -gt 60 ]; then
        restart_count=0
    fi
    
    if [ $restart_count -ge $max_restarts_per_minute ]; then
        echo "Too many restarts ($restart_count) in the last minute. Waiting 60 seconds..."
        sleep 60
        restart_count=0
    fi
    
    echo "$(date): Starting Xenon (restart #$restart_count)..."
    
    ./xenon 2>&1 | tee "logs/xenon_$(date +%Y%m%d_%H%M%S).log"
    exit_code=$?
    
    if [ $exit_code -eq 0 ]; then
        echo "$(date): Xenon shut down gracefully. Exiting."
        break
    fi
    
    restart_count=$((restart_count + 1))
    last_restart_time=$current_time
    
    echo "$(date): Xenon crashed with exit code $exit_code. Restarting in 3 seconds..."
    echo "Crash #$restart_count logged to logs/xenon_$(date +%Y%m%d_%H%M%S).log"
    
    sleep 3
done
#!/bin/bash
# Stop any running server, rebuild, and start fresh

echo "Stopping existing server..."
pkill -f "main.go serve" 2>/dev/null
lsof -ti:8090 | xargs kill 2>/dev/null
sleep 1

echo "Generating templates..."
templ generate

echo "Compiling CSS..."
npx @tailwindcss/cli -i input.css -o static/css/output.css

echo "Starting server..."
go run main.go serve

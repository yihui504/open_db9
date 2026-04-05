#!/bin/bash
# Open-DB9 Quick Start Script for Linux/macOS

set -e

echo "🚀 Open-DB9 Quick Start"
echo "======================="
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first:"
    echo "   - macOS: https://docs.docker.com/desktop/install/mac-install/"
    echo "   - Linux: https://docs.docker.com/engine/install/"
    exit 1
fi

# Check if Docker is running
if ! docker info &> /dev/null; then
    echo "❌ Docker is not running. Please start Docker."
    exit 1
fi

echo "✅ Prerequisites check passed!"
echo ""

COMPOSE_FILE="deployments/docker/docker-compose.yml"
ENV_FILE="deployments/docker/.env.example"

echo "🐳 Starting services..."
docker-compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" up -d --build

echo ""
echo "✅ Services started!"
echo ""
echo "API Server: http://localhost:8080"
echo "Health Check: http://localhost:8080/health"
echo "FS9 Service: http://localhost:9090/health"
echo "RAG API: http://localhost:8001/health"
echo ""
echo "To run CLI commands:"
echo "  ./build/db9 --help"
echo ""
echo "To stop services:"
echo "  docker-compose -f $COMPOSE_FILE down"

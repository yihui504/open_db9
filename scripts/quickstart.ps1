$ErrorActionPreference = "Stop"

Write-Host "🚀 Open-DB9 Quick Start"
Write-Host "======================="
Write-Host ""

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-Host "❌ Docker 未安装，请先安装 Docker Desktop"
    exit 1
}

docker info *> $null
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Docker 未启动，请先启动 Docker Desktop"
    exit 1
}

$composeFile = "deployments/docker/docker-compose.yml"
$envFile = "deployments/docker/.env.example"

Write-Host "🐳 Starting services..."
docker-compose --env-file $envFile -f $composeFile up -d --build

Write-Host ""
Write-Host "✅ Services started!"
Write-Host "API Server: http://localhost:8080"
Write-Host "FS9 Service: http://localhost:9090/health"
Write-Host "RAG API: http://localhost:8001/health"
Write-Host ""
Write-Host "停止服务:"
Write-Host "  docker-compose -f $composeFile down"

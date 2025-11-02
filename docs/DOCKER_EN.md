# Docker Deployment Guide

**🌐 Language:** [English](DOCKER_EN.md) | [中文](DOCKER.md)

This document describes how to use Docker to build and run the streamASR project.

## Quick Start

### Using docker-compose (Recommended)

```bash
# Build and start services
make docker-deploy

# View logs
make docker-compose-logs

# Stop services
make docker-compose-down
```

### Using Native Docker Commands

```bash
# Build image
make docker-build

# Run container
make docker-run

# View logs
make docker-logs

# Stop container
make docker-stop
```

## Available Makefile Commands

### Basic Build and Run

- `make docker-build` - Build Docker image (production version)
- `make docker-build-dev` - Build development version image
- `make docker-run` - Run container
- `make docker-stop` - Stop and remove container

### Docker Compose Operations

- `make docker-compose-up` - Start services using docker-compose
- `make docker-compose-down` - Stop docker-compose services
- `make docker-compose-logs` - View service logs
- `make docker-compose-build` - Build using docker-compose

### Development and Debugging

- `make docker-dev` - Development environment (build dev version and run)
- `make docker-exec` - Enter running container
- `make docker-debug` - Start container in debug mode (interactive shell)
- `make docker-ps` - View container status

### Maintenance Operations

- `make docker-clean` - Clean all Docker-related resources
- `make docker-deploy` - Complete deployment (build and start services)

## Manual Docker Commands

### Build Images

```bash
# Production version
docker build -t streamasr:latest .

# Development version
docker build -t streamasr:dev .
```

### Run Container

```bash
# Run production version
docker run -d --name streamasr-container \
  -p 8088:8088 \
  -v $(pwd)/config/config.yaml:/app/config/config.yaml:ro \
  -v $(pwd)/vad/model:/app/vad/model:ro \
  -v $(pwd)/audio:/app/audio \
  -v $(pwd)/logs:/app/logs \
  streamasr:latest
```

### Using docker-compose

```bash
# Start services
docker-compose up -d

# View logs
docker-compose logs -f streamASR

# Stop services
docker-compose down

# Rebuild and start
docker-compose up -d --build
```

## Configuration Description

### Environment Variables

The following environment variables can be set in `docker-compose.yml`:

- `VERSION` - Application version
- `BUILD_TIME` - Build time
- `GIT_COMMIT` - Git commit hash
- `CONFIG_PATH` - Configuration file path

### Mounted Directories

- `./config/config.yaml` - Application configuration file (read-only)
- `./vad/model` - VAD model files (read-only)
- `./audio` - Audio file storage directory
- `./logs` - Log file directory
- `./static` - Static files directory (optional)

## Health Check

The container includes built-in health check functionality:

```bash
# Check container health status
docker ps

# View health check logs
docker inspect streamasr-container | grep Health -A 10
```

## Troubleshooting

### Common Issues

1. **Port Conflict**
   ```bash
   # Check port usage
   lsof -i :8088

   # Use different port
   docker run -p 9088:8088 streamasr:latest
   ```

2. **VAD Model File Missing**
   ```bash
   # Ensure VAD model file exists
   ls -la vad/model/

   # If model files are missing, download the appropriate model files
   ```

3. **Permission Issues**
   ```bash
   # Ensure audio and log directories have correct permissions
   chmod 755 audio logs
   ```

### Viewing Logs

```bash
# View container logs
docker logs streamasr-container

# View logs in real-time
docker logs -f streamasr-container

# View recent logs
docker logs --tail 100 streamasr-container
```

## Production Deployment Recommendations

1. **Use docker-compose** - Recommended for service orchestration in production environments
2. **Configuration Persistence** - Ensure audio files and log directories are properly mounted
3. **Resource Limits** - Set appropriate resource limits in production environments
4. **Log Management** - Configure log rotation and monitoring
5. **Health Check** - Enable health checks and configure appropriate monitoring

```yaml
# Example production configuration
services:
  streamASR:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 1G
        reservations:
          cpus: '1'
          memory: 512M
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```
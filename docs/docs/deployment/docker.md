---
sidebar_label: Docker Deployment
sidebar_position: 1
---

# Docker Deployment

Deploy Keyline using Docker and Docker Compose for development, testing, and production environments.

## Overview

Keyline provides official Docker images for easy deployment. This guide covers Docker installation, configuration, and best practices.

## Quick Start

### Pull the Image

```bash
docker pull ghcr.io/wasilak/keyline:latest
```

### Basic Docker Run

```bash
# Generate required secrets
export SESSION_SECRET=$(openssl rand -base64 32)
export CACHE_ENCRYPTION_KEY=$(openssl rand -base64 32)
export ES_ADMIN_PASSWORD=your-es-admin-password

# Run Keyline
docker run -d \
  --name keyline \
  -p 9000:9000 \
  -v $(pwd)/config.yaml:/etc/keyline/config.yaml \
  -e SESSION_SECRET="${SESSION_SECRET}" \
  -e CACHE_ENCRYPTION_KEY="${CACHE_ENCRYPTION_KEY}" \
  -e ES_ADMIN_PASSWORD="${ES_ADMIN_PASSWORD}" \
  ghcr.io/wasilak/keyline:latest \
  --config /etc/keyline/config.yaml
```

### Verify Installation

```bash
# Check container status
docker ps

# Check health endpoint
curl http://localhost:9000/healthz

# View logs
docker logs keyline
```

## Docker Compose

### Basic Setup (Development)

```yaml
version: '3.8'

services:
  keyline:
    image: ghcr.io/wasilak/keyline:latest
    container_name: keyline
    ports:
      - "9000:9000"
    volumes:
      - ./config.yaml:/etc/keyline/config.yaml:ro
    environment:
      - SESSION_SECRET=${SESSION_SECRET}
      - CACHE_ENCRYPTION_KEY=${CACHE_ENCRYPTION_KEY}
      - ES_ADMIN_PASSWORD=${ES_ADMIN_PASSWORD}
    command: ["--config", "/etc/keyline/config.yaml"]
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:9000/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    restart: unless-stopped
    networks:
      - keyline-network

networks:
  keyline-network:
    driver: bridge
```

### Production Setup (with Redis)

```yaml
version: '3.8'

services:
  keyline:
    image: ghcr.io/wasilak/keyline:latest
    container_name: keyline
    ports:
      - "9000:9000"
    volumes:
      - ./config.yaml:/etc/keyline/config.yaml:ro
    environment:
      - SESSION_SECRET=${SESSION_SECRET}
      - CACHE_ENCRYPTION_KEY=${CACHE_ENCRYPTION_KEY}
      - ES_ADMIN_PASSWORD=${ES_ADMIN_PASSWORD}
      - REDIS_URL=redis://redis:6379
    command: ["--config", "/etc/keyline/config.yaml"]
    depends_on:
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:9000/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    restart: unless-stopped
    networks:
      - keyline-network
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M

  redis:
    image: redis:7-alpine
    container_name: keyline-redis
    command: redis-server --appendonly yes
    volumes:
      - redis-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 3
    restart: unless-stopped
    networks:
      - keyline-network

volumes:
  redis-data:
    driver: local

networks:
  keyline-network:
    driver: bridge
```

### Full Stack (with Elasticsearch & Kibana)

```yaml
version: '3.8'

services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:9.3.1
    container_name: keyline-es
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=true
      - ELASTIC_PASSWORD=${ELASTIC_PASSWORD}
    volumes:
      - es-data:/usr/share/elasticsearch/data
    ports:
      - "9200:9200"
    networks:
      - keyline-network
    healthcheck:
      test: ["CMD-SHELL", "curl -s http://localhost:9200/_cluster/health | grep -q 'status'"]
      interval: 30s
      timeout: 10s
      retries: 5

  kibana:
    image: docker.elastic.co/kibana/kibana:9.3.1
    container_name: keyline-kibana
    environment:
      - ELASTICSEARCH_HOSTS=http://elasticsearch:9200
    ports:
      - "5601:5601"
    depends_on:
      elasticsearch:
        condition: service_healthy
    networks:
      - keyline-network

  keyline:
    image: ghcr.io/wasilak/keyline:latest
    container_name: keyline
    ports:
      - "9000:9000"
    volumes:
      - ./config.yaml:/etc/keyline/config.yaml:ro
    environment:
      - SESSION_SECRET=${SESSION_SECRET}
      - CACHE_ENCRYPTION_KEY=${CACHE_ENCRYPTION_KEY}
      - ES_ADMIN_PASSWORD=${ELASTIC_PASSWORD}
      - REDIS_URL=redis://redis:6379
    command: ["--config", "/etc/keyline/config.yaml"]
    depends_on:
      - elasticsearch
      - redis
    networks:
      - keyline-network

  redis:
    image: redis:7-alpine
    container_name: keyline-redis
    networks:
      - keyline-network

volumes:
  es-data:
  redis-data:

networks:
  keyline-network:
    driver: bridge
```

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `SESSION_SECRET` | Yes | Session cookie signing secret (min 32 bytes) |
| `CACHE_ENCRYPTION_KEY` | Yes* | Encryption key for credential cache (32 bytes) |
| `ES_ADMIN_PASSWORD` | Yes* | Elasticsearch admin password |
| `REDIS_URL` | No | Redis connection URL (required for production) |
| `OIDC_CLIENT_SECRET` | No | OIDC client secret (if using OIDC) |

*Required when using dynamic user management

### Volume Mounts

| Mount | Purpose |
|-------|---------|
| `./config.yaml:/etc/keyline/config.yaml:ro` | Configuration file (read-only) |
| `./ssl:/etc/keyline/ssl:ro` | SSL certificates (if using TLS) |

## Multi-Architecture Support

Keyline images support multiple architectures:

| Architecture | Platform | Status |
|--------------|----------|--------|
| **amd64** | Linux, Windows, macOS | ✅ Supported |
| **arm64** | Linux (ARM servers), macOS (M1/M2) | ✅ Supported |
| **arm/v7** | Raspberry Pi, ARM devices | ✅ Supported |

### Build for Specific Architecture

```bash
# AMD64
docker pull ghcr.io/wasilak/keyline:latest-linux-amd64

# ARM64
docker pull ghcr.io/wasilak/keyline:latest-linux-arm64

# ARM/v7
docker pull ghcr.io/wasilak/keyline:latest-linux-arm-v7
```

## Security Best Practices

### 1. Use Read-Only Config

```yaml
volumes:
  - ./config.yaml:/etc/keyline/config.yaml:ro  # Read-only
```

### 2. Don't Hardcode Secrets

```yaml
# ❌ BAD: Secrets in compose file
environment:
  - SESSION_SECRET=hardcoded-secret

# ✅ GOOD: Use environment variables
environment:
  - SESSION_SECRET=${SESSION_SECRET}
```

### 3. Use Secrets (Docker Swarm)

```yaml
version: '3.8'

services:
  keyline:
    image: ghcr.io/wasilak/keyline:latest
    secrets:
      - session_secret
      - encryption_key

secrets:
  session_secret:
    external: true
  encryption_key:
    external: true
```

### 4. Network Isolation

```yaml
networks:
  keyline-network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
```

### 5. Resource Limits

```yaml
deploy:
  resources:
    limits:
      cpus: '1.0'
      memory: 512M
    reservations:
      cpus: '0.5'
      memory: 256M
```

## Monitoring

### Health Check

```bash
# Check container health
docker inspect --format='{{.State.Health.Status}}' keyline

# Check health endpoint
curl http://localhost:9000/healthz
```

### Logs

```bash
# View logs
docker logs keyline

# Follow logs
docker logs -f keyline

# Last 100 lines
docker logs --tail 100 keyline
```

### Metrics

```bash
# Access Prometheus metrics
curl http://localhost:9000/_metrics
```

## Troubleshooting

### Container Won't Start

**Symptoms**: Container exits immediately

**Solution**:
```bash
# Check logs
docker logs keyline

# Test configuration
docker run --rm \
  -v $(pwd)/config.yaml:/etc/keyline/config.yaml:ro \
  ghcr.io/wasilak/keyline:latest \
  --validate-config --config /etc/keyline/config.yaml
```

### Can't Connect to Redis

**Symptoms**: Redis connection errors in logs

**Solution**:
```bash
# Check Redis is running
docker ps | grep redis

# Test connectivity
docker exec keyline wget --spider redis://redis:6379

# Check network
docker network inspect keyline-network
```

### Health Check Fails

**Symptoms**: Container marked as unhealthy

**Solution**:
```bash
# Check health endpoint manually
docker exec keyline wget -qO- http://localhost:9000/healthz

# Review health check configuration
docker inspect --format='{{.Config.Healthcheck}}' keyline
```

## Next Steps

- **[Kubernetes Deployment](./kubernetes.md)** - K8s deployment guide
- **[Binary Installation](./binary.md)** - Bare-metal installation
- **[Security Best Practices](./security-best-practices.md)** - Security guidelines

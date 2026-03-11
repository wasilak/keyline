# Docker Compose Files for Keyline Testing

## Files

### docker-compose.yml
- **Elasticsearch version**: 9.3.1
- **Authentication**: Disabled (`xpack.security.enabled=false`)
- **Use case**: Simple local testing without authentication
- **Nodes**: keyline-es01 (9200), keyline-es02 (9201), keyline-es03 (9202)

```bash
docker-compose up -d
curl http://localhost:9200/_cluster/health?pretty
```

### docker-compose-es8-auth.yml
- **Elasticsearch version**: 8.15.0
- **Authentication**: Enabled with basic auth (no SSL)
- **Use case**: Testing Keyline with authenticated Elasticsearch cluster
- **Nodes**: keyline-es8-01 (9200), keyline-es8-02 (9201), keyline-es8-03 (9202)
- **Username**: `elastic`
- **Password**: Set via `ELASTIC_PASSWORD` environment variable

```bash
export ELASTIC_PASSWORD=your-secure-password
docker-compose -f docker-compose-es8-auth.yml up -d
curl -u elastic:your-secure-password http://localhost:9200/_cluster/health?pretty
```

## Why Two Files?

**Elasticsearch 9.x Security Requirements**:
- ES 9.x requires transport SSL when security is enabled
- Setting up SSL certificates is complex for local testing
- For simple auth testing, ES 8.x allows security without transport SSL

**Recommendation**:
- Use `docker-compose.yml` (ES 9.3, no auth) for basic Keyline testing
- Use `docker-compose-es8-auth.yml` (ES 8.15, with auth) for testing Keyline's authentication proxy features

## Testing Keyline with Authentication

1. Start ES 8.x cluster with auth:
```bash
export ELASTIC_PASSWORD=changeme
docker-compose -f docker-compose-es8-auth.yml up -d
```

2. Configure Keyline (`config/es-auth-test.yaml`):
```yaml
elasticsearch:
  users:
    - username: elastic
      password: ${ES_PASSWORD}

upstream:
  url: http://localhost:9200
```

3. Start Keyline:
```bash
export ES_PASSWORD=changeme
./bin/keyline --config config/es-auth-test.yaml
```

4. Test:
```bash
curl -u testuser:password http://localhost:9000/_cluster/health?pretty
```

See [TESTING.md](TESTING.md) for complete testing instructions.

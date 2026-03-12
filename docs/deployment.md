# Keyline Deployment Guide

This guide covers deploying Keyline in various environments including Kubernetes, Docker, and integration with reverse proxies like Traefik and Nginx.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Docker Deployment](#docker-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Traefik Integration](#traefik-integration)
- [Nginx Integration](#nginx-integration)
- [Secret Management with Vault](#secret-management-with-vault)
- [Health Checks and Readiness Probes](#health-checks-and-readiness-probes)
- [Production Considerations](#production-considerations)

## Prerequisites

- Docker 20.10+ or Kubernetes 1.20+
- Redis 6.0+ (for production deployments)
- OIDC provider configured (if using OIDC authentication)
- Elasticsearch cluster with Security API enabled
- Elasticsearch admin user with `manage_security` privilege (if using dynamic user management)
- 32-byte encryption key for credential caching (if using dynamic user management)

## Docker Deployment

### Basic Docker Run

```bash
# Create a config file
cp config/config.example.yaml config.yaml
# Edit config.yaml with your settings

# Run with Docker
docker run -d \
  --name keyline \
  -p 9000:9000 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -e SESSION_SECRET=$(openssl rand -base64 32) \
  -e OIDC_CLIENT_SECRET=your-secret \
  -e REDIS_URL=redis://redis:6379 \
  keyline:latest \
  --config /app/config.yaml
```

### Docker Compose

```yaml
version: '3.8'

services:
  keyline:
    image: keyline:latest
    ports:
      - "9000:9000"
    environment:
      - SESSION_SECRET=${SESSION_SECRET}
      - OIDC_ISSUER_URL=${OIDC_ISSUER_URL}
      - OIDC_CLIENT_ID=${OIDC_CLIENT_ID}
      - OIDC_CLIENT_SECRET=${OIDC_CLIENT_SECRET}
      - REDIS_URL=redis://redis:6379
      - ES_ADMIN_PASSWORD=${ES_ADMIN_PASSWORD}
      - CACHE_ENCRYPTION_KEY=${CACHE_ENCRYPTION_KEY}
    volumes:
      - ./config.yaml:/app/config.yaml
    depends_on:
      - redis
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:9000/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 3

volumes:
  redis-data:
```

## Kubernetes Deployment

### ConfigMap for Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: keyline-config
  namespace: auth
data:
  config.yaml: |
    server:
      port: 9000
      mode: forward_auth
      read_timeout: 30s
      write_timeout: 30s
      max_concurrent: 1000
    
    oidc:
      enabled: true
      issuer_url: ${OIDC_ISSUER_URL}
      client_id: ${OIDC_CLIENT_ID}
      client_secret: ${OIDC_CLIENT_SECRET}
      redirect_url: https://auth.example.com/auth/callback
      scopes:
        - openid
        - email
        - profile
    
    local_users:
      enabled: true
      users:
        - username: monitoring
          password_bcrypt: ${MONITORING_PASSWORD_BCRYPT}
          groups:
            - monitoring
          email: monitoring@example.com
    
    # Dynamic user management configuration
    user_management:
      enabled: true
      password_length: 32
      credential_ttl: 1h
    
    # Role mappings for dynamic user management
    role_mappings:
      - claim: groups
        pattern: "admin"
        es_roles:
          - superuser
      - claim: groups
        pattern: "developers"
        es_roles:
          - developer
          - kibana_user
      - claim: groups
        pattern: "monitoring"
        es_roles:
          - monitoring_user
      - claim: email
        pattern: "*@admin.example.com"
        es_roles:
          - superuser
    
    default_es_roles:
      - viewer
      - kibana_user
    
    session:
      ttl: 24h
      cookie_name: keyline_session
      cookie_domain: .example.com
      cookie_path: /
      session_secret: ${SESSION_SECRET}
    
    cache:
      backend: redis
      redis_url: redis://keyline-redis:6379
      redis_db: 0
      credential_ttl: 1h
      encryption_key: ${CACHE_ENCRYPTION_KEY}
    
    elasticsearch:
      admin_user: admin
      admin_password: ${ES_ADMIN_PASSWORD}
      url: https://elasticsearch:9200
      timeout: 30s
      insecure_skip_verify: false
    
    observability:
      log_level: info
      log_format: json
      otel_enabled: true
      otel_endpoint: http://otel-collector:4318
      otel_service_name: keyline
      otel_service_version: v1.0.0
      otel_environment: production
      otel_trace_ratio: 0.1
      metrics_enabled: true
```

### Secret Management

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: keyline-secrets
  namespace: auth
type: Opaque
stringData:
  session-secret: "your-base64-encoded-secret"
  oidc-client-secret: "your-oidc-client-secret"
  es-admin-password: "your-es-admin-password"
  cache-encryption-key: "your-32-byte-base64-encoded-key"
  monitoring-password-bcrypt: "$2a$10$..."
```

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: keyline
  namespace: auth
  labels:
    app: keyline
spec:
  replicas: 3
  selector:
    matchLabels:
      app: keyline
  template:
    metadata:
      labels:
        app: keyline
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9000"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: keyline
        image: keyline:v1.0.0
        ports:
        - containerPort: 9000
          name: http
          protocol: TCP
        env:
        - name: SESSION_SECRET
          valueFrom:
            secretKeyRef:
              name: keyline-secrets
              key: session-secret
        - name: OIDC_ISSUER_URL
          value: "https://accounts.google.com"
        - name: OIDC_CLIENT_ID
          value: "your-client-id.apps.googleusercontent.com"
        - name: OIDC_CLIENT_SECRET
          valueFrom:
            secretKeyRef:
              name: keyline-secrets
              key: oidc-client-secret
        - name: ES_ADMIN_PASSWORD
          valueFrom:
            secretKeyRef:
              name: keyline-secrets
              key: es-admin-password
        - name: CACHE_ENCRYPTION_KEY
          valueFrom:
            secretKeyRef:
              name: keyline-secrets
              key: cache-encryption-key
        - name: MONITORING_PASSWORD_BCRYPT
          valueFrom:
            secretKeyRef:
              name: keyline-secrets
              key: monitoring-password-bcrypt
        volumeMounts:
        - name: config
          mountPath: /app/config.yaml
          subPath: config.yaml
        livenessProbe:
          httpGet:
            path: /healthz
            port: 9000
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /healthz
            port: 9000
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 2
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
      volumes:
      - name: config
        configMap:
          name: keyline-config
```

### Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: keyline
  namespace: auth
  labels:
    app: keyline
spec:
  type: ClusterIP
  ports:
  - port: 9000
    targetPort: 9000
    protocol: TCP
    name: http
  selector:
    app: keyline
```

### Redis Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: keyline-redis
  namespace: auth
spec:
  replicas: 1
  selector:
    matchLabels:
      app: keyline-redis
  template:
    metadata:
      labels:
        app: keyline-redis
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        ports:
        - containerPort: 6379
        volumeMounts:
        - name: redis-data
          mountPath: /data
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "1Gi"
            cpu: "500m"
      volumes:
      - name: redis-data
        persistentVolumeClaim:
          claimName: keyline-redis-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: keyline-redis
  namespace: auth
spec:
  type: ClusterIP
  ports:
  - port: 6379
    targetPort: 6379
  selector:
    app: keyline-redis
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: keyline-redis-pvc
  namespace: auth
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
```

## Traefik Integration

### ForwardAuth Middleware

```yaml
# Traefik dynamic configuration
http:
  middlewares:
    keyline-auth:
      forwardAuth:
        address: http://keyline:9000
        authResponseHeaders:
          - X-Es-Authorization
        trustForwardHeader: true

  routers:
    kibana:
      rule: "Host(`kibana.example.com`)"
      middlewares:
        - keyline-auth
      service: kibana
      tls:
        certResolver: letsencrypt

  services:
    kibana:
      loadBalancer:
        servers:
          - url: http://kibana:5601
```

### Kubernetes IngressRoute (Traefik CRD)

```yaml
apiVersion: traefik.containo.us/v1alpha1
kind: Middleware
metadata:
  name: keyline-auth
  namespace: auth
spec:
  forwardAuth:
    address: http://keyline.auth.svc.cluster.local:9000
    authResponseHeaders:
      - X-Es-Authorization
    trustForwardHeader: true
---
apiVersion: traefik.containo.us/v1alpha1
kind: IngressRoute
metadata:
  name: kibana
  namespace: elastic
spec:
  entryPoints:
    - websecure
  routes:
    - match: Host(`kibana.example.com`)
      kind: Rule
      middlewares:
        - name: keyline-auth
          namespace: auth
      services:
        - name: kibana
          port: 5601
  tls:
    certResolver: letsencrypt
```

## Nginx Integration

### auth_request Configuration

```nginx
# Nginx configuration
upstream keyline {
    server keyline:9000;
}

upstream kibana {
    server kibana:5601;
}

server {
    listen 443 ssl http2;
    server_name kibana.example.com;

    ssl_certificate /etc/nginx/certs/cert.pem;
    ssl_certificate_key /etc/nginx/certs/key.pem;

    # Authentication endpoint
    location = /auth {
        internal;
        proxy_pass http://keyline;
        proxy_pass_request_body off;
        proxy_set_header Content-Length "";
        proxy_set_header X-Original-URI $request_uri;
        proxy_set_header X-Original-Method $request_method;
        proxy_set_header X-Original-Host $host;
    }

    # Kibana proxy
    location / {
        auth_request /auth;
        auth_request_set $auth_header $upstream_http_x_es_authorization;
        proxy_set_header X-Es-Authorization $auth_header;
        
        proxy_pass http://kibana;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # OIDC callback (bypass auth)
    location /auth/callback {
        proxy_pass http://keyline;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Logout endpoint (bypass auth)
    location /auth/logout {
        proxy_pass http://keyline;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Kubernetes Ingress (Nginx Ingress Controller)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: kibana
  namespace: elastic
  annotations:
    nginx.ingress.kubernetes.io/auth-url: "http://keyline.auth.svc.cluster.local:9000"
    nginx.ingress.kubernetes.io/auth-response-headers: "X-Es-Authorization"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - kibana.example.com
    secretName: kibana-tls
  rules:
  - host: kibana.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: kibana
            port:
              number: 5601
```

## Secret Management with Vault

### Vault Integration

```bash
# Store secrets in Vault
vault kv put secret/keyline/prod \
  session_secret=$(openssl rand -base64 32) \
  oidc_client_secret=your-secret \
  es_admin_password=your-password \
  es_readonly_password=your-password

# Kubernetes with Vault Agent Injector
```

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: keyline
  namespace: auth
spec:
  template:
    metadata:
      annotations:
        vault.hashicorp.com/agent-inject: "true"
        vault.hashicorp.com/role: "keyline"
        vault.hashicorp.com/agent-inject-secret-secrets: "secret/data/keyline/prod"
        vault.hashicorp.com/agent-inject-template-secrets: |
          {{- with secret "secret/data/keyline/prod" -}}
          export SESSION_SECRET="{{ .Data.data.session_secret }}"
          export OIDC_CLIENT_SECRET="{{ .Data.data.oidc_client_secret }}"
          export ES_ADMIN_PASSWORD="{{ .Data.data.es_admin_password }}"
          export ES_READONLY_PASSWORD="{{ .Data.data.es_readonly_password }}"
          {{- end }}
    spec:
      containers:
      - name: keyline
        command: ["/bin/sh", "-c"]
        args:
          - source /vault/secrets/secrets && /app/keyline --config /app/config.yaml
```

## Health Checks and Readiness Probes

### Health Check Endpoint

The `/healthz` endpoint provides health status:

```bash
curl http://localhost:9000/healthz
```

Response:
```json
{
  "status": "healthy",
  "version": "v1.0.0",
  "oidc": {
    "status": "healthy",
    "issuer": "https://accounts.google.com"
  }
}
```

### Kubernetes Probes

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 9000
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /healthz
    port: 9000
  initialDelaySeconds: 10
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 2
```

## Dynamic User Management Deployment

### Overview

When dynamic user management is enabled, Keyline automatically creates and manages Elasticsearch users for all authenticated users. This requires additional configuration and considerations.

### Prerequisites

1. **Elasticsearch Security API**: Must be enabled on your ES cluster
2. **Admin Credentials**: ES user with `manage_security` privilege
3. **Encryption Key**: 32-byte key for encrypting cached credentials
4. **Redis**: Recommended for production (enables horizontal scaling)

### Generating Encryption Key

```bash
# Generate a 32-byte encryption key
openssl rand -base64 32
```

Store this key securely in your secrets management system (Vault, Kubernetes Secrets, etc.).

### Configuration Example

```yaml
# Enable dynamic user management
user_management:
  enabled: true
  password_length: 32
  credential_ttl: 1h

# Configure role mappings
role_mappings:
  - claim: groups
    pattern: "admin"
    es_roles:
      - superuser
  - claim: groups
    pattern: "developers"
    es_roles:
      - developer
      - kibana_user

# Default roles for users with no matching groups
default_es_roles:
  - viewer
  - kibana_user

# Elasticsearch admin credentials
elasticsearch:
  admin_user: admin
  admin_password: ${ES_ADMIN_PASSWORD}
  url: https://elasticsearch:9200
  timeout: 30s

# Cache configuration with encryption
cache:
  backend: redis
  redis_url: redis://redis:6379
  credential_ttl: 1h
  encryption_key: ${CACHE_ENCRYPTION_KEY}
```

### Security Considerations

1. **Encryption Key Management**:
   - Store encryption key in environment variables, not config files
   - Use the same key across all Keyline instances (for Redis)
   - Rotate keys periodically (invalidates cache)
   - Never commit keys to version control

2. **Admin Credentials**:
   - Use dedicated ES admin user with minimal privileges
   - Only grant `manage_security` privilege
   - Rotate credentials regularly
   - Never log admin credentials

3. **Password Security**:
   - Generated passwords are 32+ characters
   - Passwords are encrypted in cache using AES-256-GCM
   - Passwords are never logged
   - Cache TTL limits password lifetime

### Horizontal Scaling

When using Redis as the cache backend, multiple Keyline instances can share the credential cache:

```yaml
# All instances must use the same configuration
cache:
  backend: redis
  redis_url: redis://redis-cluster:6379
  encryption_key: ${CACHE_ENCRYPTION_KEY}  # Same key for all instances
```

Benefits:
- Consistent credentials across all instances
- Reduced ES API calls (shared cache)
- Better performance under load
- Seamless failover

### Monitoring

Key metrics to monitor:

- `keyline_user_upserts_total`: User creation/update rate
- `keyline_cred_cache_hits_total`: Cache hit rate (target: >95%)
- `keyline_cred_cache_misses_total`: Cache miss rate
- `keyline_es_api_calls_total`: ES API call rate and errors
- `keyline_role_mapping_matches_total`: Role mapping effectiveness

### Troubleshooting

See [User Management Troubleshooting Guide](troubleshooting-user-management.md) for common issues and solutions.

### Migration from Static Mapping

See [Migration Guide](migration-guide.md) for step-by-step instructions on migrating from static user mapping to dynamic user management.

## Production Considerations

### High Availability

- Deploy at least 3 replicas for redundancy
- Use Redis with persistence or Redis Sentinel/Cluster
- Configure pod anti-affinity to spread replicas across nodes
- Use horizontal pod autoscaling based on CPU/memory

### Security

- Always use HTTPS for OIDC redirect URLs
- Store secrets in Vault or Kubernetes Secrets
- Use network policies to restrict traffic
- Enable RBAC for Kubernetes resources
- Rotate session secrets regularly
- Use strong bcrypt cost for password hashing
- **Dynamic User Management**:
  - Rotate encryption keys periodically
  - Monitor ES API call patterns for anomalies
  - Audit ES user creation logs
  - Ensure admin credentials have minimal privileges
  - Use TLS for ES API connections

### Monitoring

- Enable Prometheus metrics endpoint
- Configure OpenTelemetry for distributed tracing
- Set up alerts for:
  - High error rates
  - Authentication failures
  - Redis connection failures
  - High response times
  - Pod restarts
  - **Dynamic User Management**:
    - Low cache hit rate (<95%)
    - High ES API error rate
    - Failed user upserts
    - Encryption/decryption failures

### Performance

- Tune `max_concurrent` based on load testing
- Configure appropriate resource limits
- Use Redis connection pooling
- Enable HTTP/2 for better performance
- Consider using a CDN for static assets

### Backup and Recovery

- Backup Redis data regularly
- Document OIDC provider configuration
- Store configuration in version control
- Test disaster recovery procedures
- Maintain runbooks for common issues

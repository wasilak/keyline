---
sidebar_label: Kubernetes Deployment
sidebar_position: 2
---

# Kubernetes Deployment

Deploy Keyline on Kubernetes with Helm charts, manifests, and best practices for production environments.

## Overview

Keyline can be deployed on Kubernetes using raw manifests, Kustomize, or Helm charts. This guide covers all three approaches.

## Prerequisites

- Kubernetes 1.20+
- kubectl configured
- Helm 3.0+ (for Helm deployment)
- Redis cluster (for production)
- Elasticsearch cluster with Security API enabled

## Quick Start with Helm

### Add Helm Repository

```bash
helm repo add keyline https://wasilak.github.io/keyline-helm-charts
helm repo update
```

### Install Keyline

```bash
helm install keyline ghcr.io/wasilak/keyline \
  --namespace auth \
  --create-namespace \
  --set session.secret=$(openssl rand -base64 32) \
  --set cache.encryptionKey=$(openssl rand -base64 32) \
  --set elasticsearch.adminPassword=${ES_ADMIN_PASSWORD}
```

### Verify Installation

```bash
# Check pods
kubectl get pods -n auth

# Check services
kubectl get svc -n auth

# Check health
kubectl port-forward svc/keyline 9000:9000 -n auth
curl http://localhost:9000/healthz
```

## Manual Deployment

### Namespace

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: auth
  labels:
    name: auth
```

### ConfigMap

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

    oidc:
      enabled: true
      issuer_url: ${OIDC_ISSUER_URL}
      client_id: ${OIDC_CLIENT_ID}
      client_secret: ${OIDC_CLIENT_SECRET}
      redirect_url: https://auth.example.com/auth/callback

    session:
      ttl: 24h
      cookie_name: keyline_session
      cookie_domain: .example.com

    cache:
      backend: redis
      redis_url: redis://redis.auth.svc.cluster.local:6379
      credential_ttl: 1h

    user_management:
      enabled: true
      password_length: 32
      credential_ttl: 1h

    role_mappings:
      - claim: groups
        pattern: "admin"
        es_roles:
          - superuser

    default_es_roles:
      - viewer

    elasticsearch:
      admin_user: keyline_admin
      admin_password: ${ES_ADMIN_PASSWORD}
      url: https://elasticsearch:9200
```

### Secrets

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: keyline-secrets
  namespace: auth
type: Opaque
stringData:
  SESSION_SECRET: <generate-with-openssl-rand-base64-32>
  CACHE_ENCRYPTION_KEY: <generate-with-openssl-rand-base64-32>
  ES_ADMIN_PASSWORD: <your-es-admin-password>
  OIDC_CLIENT_SECRET: <your-oidc-client-secret>
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
  replicas: 2
  selector:
    matchLabels:
      app: keyline
  template:
    metadata:
      labels:
        app: keyline
    spec:
      containers:
      - name: keyline
        image: ghcr.io/wasilak/keyline:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 9000
          name: http
        env:
        - name: SESSION_SECRET
          valueFrom:
            secretKeyRef:
              name: keyline-secrets
              key: SESSION_SECRET
        - name: CACHE_ENCRYPTION_KEY
          valueFrom:
            secretKeyRef:
              name: keyline-secrets
              key: CACHE_ENCRYPTION_KEY
        - name: ES_ADMIN_PASSWORD
          valueFrom:
            secretKeyRef:
              name: keyline-secrets
              key: ES_ADMIN_PASSWORD
        - name: OIDC_CLIENT_SECRET
          valueFrom:
            secretKeyRef:
              name: keyline-secrets
              key: OIDC_CLIENT_SECRET
        - name: REDIS_URL
          value: "redis://redis.auth.svc.cluster.local:6379"
        volumeMounts:
        - name: config
          mountPath: /etc/keyline
          readOnly: true
        livenessProbe:
          httpGet:
            path: /healthz
            port: 9000
          initialDelaySeconds: 30
          periodSeconds: 30
          timeoutSeconds: 10
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /healthz
            port: 9000
          initialDelaySeconds: 10
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
        securityContext:
          runAsNonRoot: true
          runAsUser: 1000
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop:
              - ALL
      volumes:
      - name: config
        configMap:
          name: keyline-config
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchLabels:
                  app: keyline
              topologyKey: kubernetes.io/hostname
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

### Ingress (Optional)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: keyline
  namespace: auth
  annotations:
    kubernetes.io/ingress.class: nginx
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  tls:
  - hosts:
    - auth.example.com
    secretName: keyline-tls
  rules:
  - host: auth.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: keyline
            port:
              number: 9000
```

## Kustomize Deployment

### kustomization.yaml

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: auth

resources:
  - namespace.yaml
  - configmap.yaml
  - secret.yaml
  - deployment.yaml
  - service.yaml
  - ingress.yaml

configMapGenerator:
  - name: keyline-config
    files:
      - config.yaml

secretGenerator:
  - name: keyline-secrets
    envs:
      - secrets.env

generatorOptions:
  disableNameSuffixHash: true
```

### secrets.env

```env
SESSION_SECRET=your-session-secret
CACHE_ENCRYPTION_KEY=your-encryption-key
ES_ADMIN_PASSWORD=your-es-admin-password
OIDC_CLIENT_SECRET=your-oidc-client-secret
```

### Deploy with Kustomize

```bash
kubectl apply -k overlays/production
```

## Horizontal Pod Autoscaler

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: keyline-hpa
  namespace: auth
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: keyline
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## Pod Disruption Budget

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: keyline-pdb
  namespace: auth
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: keyline
```

## Network Policy

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: keyline-network-policy
  namespace: auth
spec:
  podSelector:
    matchLabels:
      app: keyline
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 9000
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: auth
    ports:
    - protocol: TCP
      port: 6379  # Redis
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 9200  # Elasticsearch
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 443  # OIDC providers
```

## Monitoring

### Prometheus ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: keyline
  namespace: auth
  labels:
    app: keyline
spec:
  selector:
    matchLabels:
      app: keyline
  endpoints:
  - port: http
    path: /_metrics
    interval: 30s
```

### Logs with Loki

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: keyline-promtail-config
  namespace: auth
data:
  config.yaml: |
    scrape_configs:
      - job_name: keyline
        kubernetes_sd_configs:
          - role: pod
        relabel_configs:
          - source_labels: [__meta_kubernetes_pod_label_app]
            action: keep
            regex: keyline
```

## Troubleshooting

### Pod Won't Start

```bash
# Check pod status
kubectl get pods -n auth

# Describe pod
kubectl describe pod <pod-name> -n auth

# Check logs
kubectl logs <pod-name> -n auth
```

### Configuration Issues

```bash
# Validate ConfigMap
kubectl get configmap keyline-config -n auth -o yaml

# Validate Secret
kubectl get secret keyline-secrets -n auth -o yaml
```

### Health Check Fails

```bash
# Port forward
kubectl port-forward svc/keyline 9000:9000 -n auth

# Test health endpoint
curl http://localhost:9000/healthz
```

## Next Steps

- **[Docker Deployment](./docker.md)** - Docker deployment guide
- **[Binary Installation](./binary.md)** - Bare-metal installation
- **[Security Best Practices](./security-best-practices.md)** - Security guidelines

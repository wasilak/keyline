# Dynamic User Management Rollback Plan

This document provides detailed procedures for rolling back from dynamic user management to static user mapping if issues occur during or after migration.

## Table of Contents

- [When to Rollback](#when-to-rollback)
- [Rollback Scenarios](#rollback-scenarios)
- [Pre-Rollback Preparation](#pre-rollback-preparation)
- [Rollback Procedures](#rollback-procedures)
- [Post-Rollback Verification](#post-rollback-verification)
- [Root Cause Analysis](#root-cause-analysis)

## When to Rollback

Consider rolling back if you encounter:

### Critical Issues (Immediate Rollback)

- **Authentication failures**: Users cannot authenticate
- **Authorization failures**: Users cannot access Elasticsearch
- **Service unavailability**: Keyline instances crash or fail to start
- **Data loss**: Session data or credentials are lost
- **Security breach**: Credentials exposed or compromised
- **ES cluster impact**: High load or errors on Elasticsearch

### Non-Critical Issues (Evaluate Before Rollback)

- **Performance degradation**: Slower authentication (but functional)
- **Low cache hit rate**: <95% cache hits (but functional)
- **Intermittent errors**: Occasional failures (but mostly working)
- **Monitoring gaps**: Missing metrics or logs (but functional)

**Decision Matrix**:
- Critical issues → Immediate rollback
- Non-critical issues → Attempt fixes first, rollback if unfixable

## Rollback Scenarios

### Scenario 1: Rollback During Deployment

**Situation**: Issues discovered during initial deployment to production

**Impact**: Minimal (users not yet using new system)

**Procedure**: [Quick Rollback](#quick-rollback-procedure)

### Scenario 2: Rollback After Partial Rollout

**Situation**: Issues discovered after some users migrated

**Impact**: Moderate (some users on new system, some on old)

**Procedure**: [Staged Rollback](#staged-rollback-procedure)

### Scenario 3: Rollback After Full Rollout

**Situation**: Issues discovered after all users migrated

**Impact**: High (all users affected)

**Procedure**: [Full Rollback](#full-rollback-procedure)

### Scenario 4: Emergency Rollback

**Situation**: Critical security or availability issue

**Impact**: Critical (immediate action required)

**Procedure**: [Emergency Rollback](#emergency-rollback-procedure)

## Pre-Rollback Preparation

### 1. Assess the Situation

```bash
# Check Keyline logs
kubectl logs -n auth deployment/keyline --tail=100

# Check ES cluster health
curl -u admin:password https://elasticsearch:9200/_cluster/health

# Check Redis connectivity
redis-cli -h redis ping

# Check metrics
curl http://keyline:9000/metrics | grep keyline_user_upserts
```

### 2. Document the Issue

- [ ] Capture error messages from logs
- [ ] Capture metrics showing the problem
- [ ] Document affected users or services
- [ ] Note when the issue started
- [ ] Identify potential root cause

### 3. Notify Stakeholders

- [ ] Notify operations team
- [ ] Notify security team (if security issue)
- [ ] Notify affected users (if necessary)
- [ ] Create incident ticket
- [ ] Start incident timeline

### 4. Prepare Rollback Artifacts

- [ ] Locate previous configuration backup
- [ ] Locate previous Keyline version/image
- [ ] Verify backup integrity
- [ ] Prepare rollback commands
- [ ] Identify rollback window

## Rollback Procedures

### Quick Rollback Procedure

**Use when**: Issues discovered during initial deployment

**Time**: 5-10 minutes

**Steps**:

1. **Stop new deployment**:
   ```bash
   # Kubernetes
   kubectl rollout undo deployment/keyline -n auth
   
   # Docker Compose
   docker-compose down
   docker-compose -f docker-compose.old.yml up -d
   ```

2. **Verify rollback**:
   ```bash
   # Check pods are running
   kubectl get pods -n auth
   
   # Check logs
   kubectl logs -n auth deployment/keyline --tail=50
   
   # Test authentication
   curl -u testuser:password http://keyline:9000/healthz
   ```

3. **Confirm functionality**:
   - [ ] Keyline instances are running
   - [ ] Authentication works
   - [ ] Users can access Elasticsearch
   - [ ] No errors in logs

### Staged Rollback Procedure

**Use when**: Some users migrated, need gradual rollback

**Time**: 15-30 minutes

**Steps**:

1. **Identify affected instances**:
   ```bash
   # List all Keyline instances
   kubectl get pods -n auth -l app=keyline
   ```

2. **Rollback instances one by one**:
   ```bash
   # Scale down new version
   kubectl scale deployment/keyline-new -n auth --replicas=0
   
   # Scale up old version
   kubectl scale deployment/keyline-old -n auth --replicas=3
   ```

3. **Monitor during rollback**:
   ```bash
   # Watch pod status
   kubectl get pods -n auth -w
   
   # Monitor logs
   kubectl logs -n auth -l app=keyline -f
   ```

4. **Verify each instance**:
   - [ ] Instance starts successfully
   - [ ] Authentication works
   - [ ] No errors in logs
   - [ ] Metrics look normal

### Full Rollback Procedure

**Use when**: All users migrated, need complete rollback

**Time**: 30-60 minutes

**Steps**:

1. **Prepare rollback configuration**:
   ```bash
   # Restore old ConfigMap
   kubectl apply -f configmap-old.yaml
   
   # Restore old Secrets
   kubectl apply -f secrets-old.yaml
   ```

2. **Update deployment**:
   ```bash
   # Update image to previous version
   kubectl set image deployment/keyline \
     keyline=keyline:v1.0.0 -n auth
   
   # Or rollback to previous revision
   kubectl rollout undo deployment/keyline -n auth
   ```

3. **Wait for rollout**:
   ```bash
   # Watch rollout status
   kubectl rollout status deployment/keyline -n auth
   ```

4. **Verify all instances**:
   ```bash
   # Check all pods are ready
   kubectl get pods -n auth -l app=keyline
   
   # Check logs from all pods
   kubectl logs -n auth -l app=keyline --tail=20
   ```

5. **Test authentication flows**:
   - [ ] OIDC authentication works
   - [ ] Local user authentication works
   - [ ] Users can access Elasticsearch
   - [ ] Correct ES credentials are used
   - [ ] No errors in logs

### Emergency Rollback Procedure

**Use when**: Critical issue requiring immediate action

**Time**: 5 minutes

**Steps**:

1. **Immediate rollback** (no verification):
   ```bash
   # Kubernetes - rollback immediately
   kubectl rollout undo deployment/keyline -n auth
   
   # Docker - stop and start old version
   docker-compose down
   docker-compose -f docker-compose.old.yml up -d
   ```

2. **Verify basic functionality**:
   ```bash
   # Quick health check
   curl http://keyline:9000/healthz
   ```

3. **Monitor for stability**:
   ```bash
   # Watch logs for errors
   kubectl logs -n auth -l app=keyline -f
   ```

4. **Detailed verification later** (after emergency resolved)

## Configuration Rollback

### Restore Static User Mapping Configuration

**Old configuration structure**:

```yaml
# Restore OIDC mappings
oidc:
  enabled: true
  issuer_url: https://accounts.google.com
  client_id: your-client-id
  client_secret: ${OIDC_CLIENT_SECRET}
  redirect_url: https://auth.example.com/auth/callback
  scopes:
    - openid
    - email
    - profile
  mappings:
    - claim: email
      pattern: "*@admin.example.com"
      es_user: admin
    - claim: email
      pattern: "*@example.com"
      es_user: readonly
  default_es_user: readonly

# Restore local user mappings
local_users:
  enabled: true
  users:
    - username: testuser
      password_bcrypt: $2a$10$...
      es_user: readonly
    - username: admin
      password_bcrypt: $2a$10$...
      es_user: admin

# Restore static ES users
elasticsearch:
  users:
    - username: admin
      password: ${ES_ADMIN_PASSWORD}
    - username: readonly
      password: ${ES_READONLY_PASSWORD}

# Remove dynamic user management sections
# (Delete these sections)
# user_management:
# role_mappings:
# default_es_roles:
# elasticsearch.admin_user:
# elasticsearch.admin_password:
# cache.encryption_key:
```

### Environment Variables to Restore

```bash
# Remove dynamic user management variables
unset ES_ADMIN_PASSWORD  # (if used for admin API)
unset CACHE_ENCRYPTION_KEY

# Restore static user passwords
export ES_ADMIN_PASSWORD=your-admin-password
export ES_READONLY_PASSWORD=your-readonly-password
```

## Post-Rollback Verification

### 1. Functional Testing

- [ ] **OIDC Authentication**:
  ```bash
  # Test OIDC login flow
  curl -L http://keyline:9000/auth/login
  ```

- [ ] **Local User Authentication**:
  ```bash
  # Test basic auth
  curl -u testuser:password http://keyline:9000/healthz
  ```

- [ ] **Elasticsearch Access**:
  ```bash
  # Verify ES credentials work
  curl -H "Authorization: Basic $(echo -n 'readonly:password' | base64)" \
    https://elasticsearch:9200/_cluster/health
  ```

### 2. Performance Verification

- [ ] Check authentication latency
- [ ] Check error rates
- [ ] Check resource usage (CPU, memory)
- [ ] Verify no performance degradation

### 3. Monitoring Verification

- [ ] Metrics are being collected
- [ ] Logs are being generated
- [ ] Alerts are functional
- [ ] Dashboards show correct data

### 4. User Verification

- [ ] Sample users can authenticate
- [ ] Users can access Elasticsearch
- [ ] Users have correct permissions
- [ ] No user complaints

## Cleanup After Rollback

### 1. Remove Dynamic User Management Artifacts

```bash
# Remove dynamically created ES users (optional)
curl -X DELETE -u admin:password \
  https://elasticsearch:9200/_security/user/user@example.com

# Clear Redis cache
redis-cli FLUSHDB

# Remove unused ConfigMaps/Secrets
kubectl delete configmap keyline-config-new -n auth
kubectl delete secret keyline-secrets-new -n auth
```

### 2. Update Documentation

- [ ] Document rollback in incident report
- [ ] Update runbooks with lessons learned
- [ ] Note configuration changes
- [ ] Update deployment procedures

### 3. Communicate Status

- [ ] Notify stakeholders of rollback completion
- [ ] Provide incident summary
- [ ] Share timeline of events
- [ ] Outline next steps

## Root Cause Analysis

After rollback, conduct RCA to understand what went wrong:

### 1. Gather Evidence

- [ ] Collect all logs from incident period
- [ ] Collect metrics and graphs
- [ ] Collect configuration files
- [ ] Interview involved personnel
- [ ] Document timeline of events

### 2. Analyze Root Cause

Common root causes:

- **Configuration errors**: Invalid role mappings, wrong encryption key
- **Integration issues**: ES API connectivity, Redis connectivity
- **Performance issues**: High latency, low cache hit rate
- **Security issues**: Exposed credentials, weak encryption
- **Compatibility issues**: ES version incompatibility

### 3. Document Findings

- [ ] Write incident report
- [ ] Identify root cause
- [ ] List contributing factors
- [ ] Propose remediation steps
- [ ] Update procedures to prevent recurrence

### 4. Plan Remediation

- [ ] Fix identified issues
- [ ] Test fixes in dev/staging
- [ ] Update migration plan
- [ ] Schedule retry (if appropriate)

## Retry Migration

After fixing issues:

### 1. Validate Fixes

- [ ] Test fixes in development
- [ ] Test fixes in staging
- [ ] Verify all issues resolved
- [ ] Update configuration
- [ ] Update procedures

### 2. Plan Retry

- [ ] Schedule new migration window
- [ ] Notify stakeholders
- [ ] Prepare updated artifacts
- [ ] Review lessons learned
- [ ] Update rollback plan

### 3. Execute Retry

- [ ] Follow updated migration checklist
- [ ] Monitor closely
- [ ] Be prepared to rollback again
- [ ] Document any new issues

## Rollback Decision Tree

```
Issue Detected
    |
    ├─ Critical? (Auth failure, service down, security breach)
    |   └─ YES → Emergency Rollback (5 min)
    |
    └─ NO → Can it be fixed quickly? (<30 min)
        |
        ├─ YES → Attempt fix, monitor closely
        |   |
        |   └─ Fixed? → Continue monitoring
        |       |
        |       └─ NO → Staged Rollback (15-30 min)
        |
        └─ NO → Staged or Full Rollback (30-60 min)
```

## Contact Information

### Escalation Path

1. **On-call Engineer**: [contact info]
2. **Team Lead**: [contact info]
3. **Engineering Manager**: [contact info]
4. **Security Team**: [contact info] (for security issues)

### Support Resources

- **Documentation**: https://github.com/your-org/keyline/docs
- **Runbooks**: https://wiki.your-org.com/keyline
- **Incident Management**: https://incidents.your-org.com
- **Slack Channel**: #keyline-support

## Appendix

### A. Rollback Commands Reference

```bash
# Kubernetes rollback
kubectl rollout undo deployment/keyline -n auth
kubectl rollout history deployment/keyline -n auth
kubectl rollout status deployment/keyline -n auth

# Docker Compose rollback
docker-compose down
docker-compose -f docker-compose.old.yml up -d

# Configuration rollback
kubectl apply -f configmap-old.yaml
kubectl apply -f secrets-old.yaml

# Clear Redis cache
redis-cli FLUSHDB

# Check ES users
curl -u admin:password https://elasticsearch:9200/_security/user
```

### B. Verification Commands Reference

```bash
# Health check
curl http://keyline:9000/healthz

# Metrics check
curl http://keyline:9000/metrics | grep keyline_

# Log check
kubectl logs -n auth -l app=keyline --tail=100

# Pod status
kubectl get pods -n auth -l app=keyline

# ES cluster health
curl -u admin:password https://elasticsearch:9200/_cluster/health
```

### C. Backup Checklist

Before migration, ensure you have:

- [ ] Configuration backup (YAML files)
- [ ] Secrets backup (encrypted)
- [ ] Deployment manifests backup
- [ ] Current Keyline version/image tag
- [ ] ES user list backup
- [ ] Documentation of current state

## Summary

This rollback plan provides procedures for safely reverting from dynamic user management to static user mapping. Key points:

- **Assess before rolling back**: Not all issues require rollback
- **Choose appropriate procedure**: Match procedure to situation
- **Verify after rollback**: Ensure system is functional
- **Conduct RCA**: Understand what went wrong
- **Plan retry**: Fix issues and try again

Remember: Rollback is not failure - it's a safety mechanism to protect users and services.

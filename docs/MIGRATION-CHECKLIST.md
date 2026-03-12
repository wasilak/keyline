# Dynamic User Management Migration Checklist

This checklist guides you through migrating from static user mapping to dynamic user management in Keyline.

## Pre-Migration Phase

### 1. Review Current Configuration

- [ ] Document current `elasticsearch.users` configuration
- [ ] Document current OIDC `mappings` configuration
- [ ] Document current `default_es_user` configuration
- [ ] List all local users and their `es_user` mappings
- [ ] Identify all Elasticsearch roles currently in use

### 2. Plan Role Mappings

- [ ] Map OIDC groups to Elasticsearch roles
- [ ] Map local user groups to Elasticsearch roles
- [ ] Define default roles for users without group matches
- [ ] Verify all required ES roles exist in Elasticsearch
- [ ] Document role mapping strategy

### 3. Prepare Elasticsearch

- [ ] Verify Elasticsearch Security API is enabled
- [ ] Create admin user with `manage_security` privilege
- [ ] Test admin credentials with ES Security API
- [ ] Verify ES cluster is accessible from Keyline
- [ ] Review ES audit logging configuration

### 4. Generate Secrets

- [ ] Generate 32-byte encryption key: `openssl rand -base64 32`
- [ ] Store encryption key in secrets management system
- [ ] Verify admin password is stored securely
- [ ] Update environment variable configuration

### 5. Backup Current State

- [ ] Backup current Keyline configuration
- [ ] Backup Elasticsearch user list
- [ ] Backup Elasticsearch role mappings
- [ ] Document current authentication flows
- [ ] Create rollback plan

## Migration Phase

### 6. Update Configuration

- [ ] Add `user_management` section to config
  ```yaml
  user_management:
    enabled: true
    password_length: 32
    credential_ttl: 1h
  ```

- [ ] Add `role_mappings` section
  ```yaml
  role_mappings:
    - claim: groups
      pattern: "admin"
      es_roles:
        - superuser
    # Add all your mappings
  ```

- [ ] Add `default_es_roles` (if needed)
  ```yaml
  default_es_roles:
    - viewer
    - kibana_user
  ```

- [ ] Update `elasticsearch` section
  ```yaml
  elasticsearch:
    admin_user: admin
    admin_password: ${ES_ADMIN_PASSWORD}
    url: https://elasticsearch:9200
    timeout: 30s
  ```

- [ ] Update `cache` section
  ```yaml
  cache:
    backend: redis
    redis_url: redis://redis:6379
    credential_ttl: 1h
    encryption_key: ${CACHE_ENCRYPTION_KEY}
  ```

- [ ] Update `local_users` to use `groups` instead of `es_user`
  ```yaml
  local_users:
    users:
      - username: testuser
        password_bcrypt: $2a$10$...
        groups:
          - developers
        email: testuser@example.com
  ```

- [ ] Remove deprecated `elasticsearch.users` section
- [ ] Remove deprecated `oidc.mappings` section
- [ ] Remove deprecated `oidc.default_es_user` field
- [ ] Remove `es_user` field from all local users

### 7. Update Environment Variables

- [ ] Set `ES_ADMIN_PASSWORD` environment variable
- [ ] Set `CACHE_ENCRYPTION_KEY` environment variable
- [ ] Remove old `ES_*_PASSWORD` variables (if using static mapping)
- [ ] Verify all required environment variables are set

### 8. Validate Configuration

- [ ] Run configuration validation: `keyline --config config.yaml --validate`
- [ ] Check for validation errors
- [ ] Verify role mappings are valid
- [ ] Verify encryption key is 32 bytes
- [ ] Verify admin credentials are set

## Testing Phase

### 9. Test in Development/Staging

- [ ] Deploy updated configuration to dev/staging
- [ ] Verify Keyline starts successfully
- [ ] Check logs for user management initialization
- [ ] Test OIDC authentication flow
- [ ] Verify ES user is created automatically
- [ ] Verify correct roles are assigned
- [ ] Test local user authentication
- [ ] Verify credentials are cached
- [ ] Test cache expiration and refresh
- [ ] Verify ES audit logs show actual usernames

### 10. Performance Testing

- [ ] Test authentication latency (cache hit)
- [ ] Test authentication latency (cache miss)
- [ ] Verify cache hit rate >95%
- [ ] Test concurrent authentication requests
- [ ] Monitor ES API call rate
- [ ] Verify no performance degradation

### 11. Security Testing

- [ ] Verify passwords are encrypted in cache
- [ ] Verify passwords are never logged
- [ ] Verify admin credentials are never exposed
- [ ] Test encryption key rotation
- [ ] Verify TLS is used for ES API calls
- [ ] Review ES audit logs for anomalies

## Production Rollout Phase

### 12. Prepare Production Deployment

- [ ] Update production configuration
- [ ] Update production secrets
- [ ] Update deployment manifests (Kubernetes/Docker)
- [ ] Schedule maintenance window
- [ ] Notify users of upcoming changes
- [ ] Prepare rollback procedure

### 13. Deploy to Production

- [ ] Deploy updated Keyline version
- [ ] Verify all instances start successfully
- [ ] Monitor logs for errors
- [ ] Verify Redis connectivity
- [ ] Verify ES API connectivity
- [ ] Test authentication flows

### 14. Monitor Production

- [ ] Monitor authentication success rate
- [ ] Monitor cache hit rate (target: >95%)
- [ ] Monitor ES API call rate
- [ ] Monitor user upsert latency
- [ ] Check for errors in logs
- [ ] Verify metrics are being collected

### 15. Validate Production

- [ ] Test OIDC authentication
- [ ] Test local user authentication
- [ ] Verify ES users are created
- [ ] Verify correct roles are assigned
- [ ] Check ES audit logs
- [ ] Verify horizontal scaling (if using Redis)

## Post-Migration Phase

### 16. Cleanup

- [ ] Remove old static ES users (optional)
- [ ] Update documentation
- [ ] Update runbooks
- [ ] Archive old configuration
- [ ] Document lessons learned

### 17. Monitoring and Alerting

- [ ] Set up alerts for low cache hit rate
- [ ] Set up alerts for ES API errors
- [ ] Set up alerts for failed user upserts
- [ ] Set up alerts for encryption failures
- [ ] Configure Grafana dashboards

### 18. Training and Documentation

- [ ] Train operations team on new system
- [ ] Update troubleshooting guides
- [ ] Document common issues and solutions
- [ ] Update incident response procedures

## Rollback Procedure

If issues occur during migration:

### Immediate Rollback

- [ ] Stop Keyline instances
- [ ] Restore previous configuration
- [ ] Restore previous secrets
- [ ] Restart Keyline with old configuration
- [ ] Verify authentication works
- [ ] Notify users of rollback

### Post-Rollback

- [ ] Document what went wrong
- [ ] Analyze logs and metrics
- [ ] Identify root cause
- [ ] Plan remediation
- [ ] Schedule retry

## Success Criteria

Migration is successful when:

- [ ] All authentication methods work (OIDC, Basic Auth)
- [ ] ES users are created automatically
- [ ] Correct roles are assigned based on groups
- [ ] Cache hit rate >95%
- [ ] No increase in authentication latency
- [ ] ES audit logs show actual usernames
- [ ] No errors in logs
- [ ] Horizontal scaling works (if using Redis)
- [ ] All monitoring and alerts are functional

## Troubleshooting

If you encounter issues, see:

- [User Management Documentation](user-management.md)
- [Troubleshooting Guide](troubleshooting-user-management.md)
- [Migration Guide](migration-guide.md)

## Support

For additional help:

- Check GitHub issues
- Review documentation
- Contact support team
- Review ES audit logs
- Check Keyline logs with debug level

## Notes

- This is a **breaking change** - plan accordingly
- Test thoroughly in dev/staging before production
- Have a rollback plan ready
- Monitor closely after deployment
- Encryption key rotation invalidates cache
- All Keyline instances must use the same encryption key (for Redis)

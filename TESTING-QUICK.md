# Keyline Testing Quick Reference

## 5-Minute Test

```bash
# Start ES
docker-compose up -d
docker-compose logs -f setup  # Wait for "All done!"

# Run Keyline with go run (faster!)
ES_PASSWORD=changeme go run ./cmd/keyline --config config/test-config.yaml

# In another terminal, run tests
./test-dynamic-user-mgmt.sh
```

## Test Scenarios

| Scenario | Command | Auth | Port |
|----------|---------|------|------|
| Basic | `docker-compose up` | Local users | 9200 |
| Forward Auth | `docker-compose -f docker-compose-forwardauth.yml up` | Local + Traefik | 9200 |
| OIDC | `docker-compose -f docker-compose-oidc.yml up` | OIDC | 9201 |
| OIDC + FA | `docker-compose -f docker-compose-oidc-forwardauth.yml up` | OIDC + Traefik | 9202 |

## Manual Tests

```bash
# Test auth
curl -u testuser:password http://localhost:9000/_security/user

# Verify ES user
curl -k -u elastic:changeme https://localhost:9200/_security/user/testuser

# Check roles
curl -k -u elastic:changeme https://localhost:9200/_security/user/testuser | jq '.testuser.roles'
```

## Expected Results

✓ ES users created dynamically  
✓ Groups map to ES roles  
✓ Cache hits < 100ms  
✓ Audit logs show usernames  
✓ Default roles applied  

## Troubleshooting

```bash
# Check logs
docker logs keyline | grep -i "user management"

# Verify ES
curl -k -u elastic:changeme https://localhost:9200

# Check config
docker exec keyline cat /app/config.yaml
```

## Docs

- [Full Testing Guide](docs/TESTING-GUIDE.md)
- [Elastauth Evolution](docs/ELASTAUTH-TO-KEYLINE-EVOLUTION.md)
- [User Management](docs/user-management.md)
- [Configuration](docs/configuration.md)

---
sidebar_label: Logout
sidebar_position: 5
---

# Logout

Logout functionality terminates user sessions and optionally redirects to the OIDC provider's end session endpoint for single logout (SLO).

## Overview

Keyline provides a logout endpoint that:
1. Deletes the local session
2. Clears the session cookie
3. Optionally redirects to OIDC provider logout

## Logout Endpoint

### Endpoint Details

| Property | Value |
|----------|-------|
| **Path** | `/auth/logout` |
| **Methods** | GET, POST |
| **Authentication** | Not required |
| **Response** | 302 Redirect |

## Logout Flow

```mermaid
sequenceDiagram
    participant User
    participant Browser
    participant Keyline
    participant SessionStore
    participant OIDC
    
    User->>Browser: Click "Logout"
    Browser->>Keyline: GET /auth/logout
    
    Keyline->>Keyline: Extract session ID from cookie
    Keyline->>SessionStore: Delete session
    
    Note over Keyline,SessionStore: Session deleted
    
    Keyline->>Browser: Set-Cookie: keyline_session=;<br/>Max-Age=0;<br/>Path=/
    
    alt OIDC end_session_endpoint configured
        Keyline->>Browser: 302 to OIDC logout<br/>?id_token_hint=xxx<br/>&post_logout_redirect_uri=yyy
        Browser->>OIDC: Logout request
        OIDC->>OIDC: Terminate SSO session
        OIDC->>Browser: 302 to redirect_uri
    else No OIDC logout
        Keyline->>Browser: 302 to logout_redirect_url<br/>or 200 "Logged out"
    end
    
    Browser->>User: Logout complete
```

## Configuration

### Basic Logout Configuration

```yaml
session:
  ttl: 24h
  cookie_name: keyline_session
  cookie_domain: .example.com
  session_secret: ${SESSION_SECRET}
```

### OIDC Logout Configuration

For OIDC single logout, configure the provider's end session endpoint:

```yaml
oidc:
  enabled: true
  issuer_url: https://accounts.google.com
  # ... other OIDC config
  
# Optional: Post-logout redirect URL
logout_redirect_url: https://example.com/logged-out
```

## Logout Behavior

### With Active Session

1. Session is deleted from storage
2. Session cookie is cleared
3. User is redirected to:
   - OIDC provider logout (if configured)
   - `logout_redirect_url` (if set)
   - Default: 200 OK with "Logged out" message

### Without Active Session

1. No session to delete
2. Cookie is still cleared
3. User receives: 200 OK with "No active session" message

## OIDC Single Logout (SLO)

### Supported Providers

| Provider | End Session Endpoint | SLO Support |
|----------|---------------------|-------------|
| **Google** | `https://accounts.google.com/Logout` | ✅ Yes |
| **Azure AD** | `.../oauth2/v2.0/logout` | ✅ Yes |
| **Okta** | `.../oauth2/default/v1/logout` | ✅ Yes |
| **Auth0** | `.../v2/logout` | ✅ Yes |
| **Keycloak** | `.../protocol/openid-connect/logout` | ✅ Yes |

### ID Token Hint

Some providers require `id_token_hint` for logout:

```
GET https://provider.com/logout?
  id_token_hint={id_token}&
  post_logout_redirect_uri=https://example.com/logged-out
```

**Note**: Keyline stores the ID token in the session for this purpose.

## Examples

### Basic Logout

```yaml
# Minimal configuration
session:
  ttl: 24h
  cookie_name: keyline_session
  session_secret: ${SESSION_SECRET}
```

**Behavior**: Deletes session, returns 200 OK

### Logout with Redirect

```yaml
session:
  ttl: 24h
  cookie_name: keyline_session
  session_secret: ${SESSION_SECRET}

# Redirect after logout
logout_redirect_url: https://example.com/logged-out
```

**Behavior**: Deletes session, redirects to `logout_redirect_url`

### OIDC Single Logout

```yaml
oidc:
  enabled: true
  issuer_url: https://login.microsoftonline.com/{tenant-id}/v2.0
  client_id: ${CLIENT_ID}
  client_secret: ${CLIENT_SECRET}
  redirect_url: https://auth.example.com/auth/callback

session:
  ttl: 24h
  cookie_name: keyline_session
  session_secret: ${SESSION_SECRET}

logout_redirect_url: https://example.com
```

**Behavior**: 
1. Deletes Keyline session
2. Redirects to Azure AD logout
3. Azure AD redirects to `logout_redirect_url`

## Testing Logout

### Using curl

```bash
# Login first (get session cookie)
curl -c cookies.txt -L https://auth.example.com/

# Logout
curl -b cookies.txt -c cookies.txt -L https://auth.example.com/auth/logout

# Verify cookie is cleared
cat cookies.txt
# Should show expired cookie
```

### Using Browser

1. Open browser DevTools → Application → Cookies
2. Authenticate via OIDC
3. Note `keyline_session` cookie
4. Navigate to `/auth/logout`
5. Verify cookie is cleared
6. Verify redirect occurs

## Troubleshooting

### Session Not Cleared

**Symptoms**: User still authenticated after logout

**Causes**:
- Cookie domain mismatch
- Multiple cookies with same name
- Session storage issue

**Solution**:
1. Verify `cookie_domain` matches
2. Clear browser cookies manually
3. Check session storage

### OIDC Logout Fails

**Symptoms**: Provider returns error on logout

**Causes**:
- `id_token_hint` expired
- `post_logout_redirect_uri` not registered
- Provider doesn't support SLO

**Solution**:
1. Check provider logout documentation
2. Register redirect URI with provider
3. Use basic logout (no OIDC redirect)

### Redirect Loop

**Symptoms**: Logout → Login → Logout loop

**Causes**:
- Application redirects to protected page after logout
- Session not properly cleared

**Solution**:
1. Set `logout_redirect_url` to public page
2. Verify session deletion logic
3. Check application redirect logic

## Security Considerations

### Session Fixation

**Risk**: Attacker sets known session ID

**Mitigation**:
- Keyline generates new session ID on login
- Session ID is cryptographically random
- Session ID is never exposed in URLs

### CSRF on Logout

**Risk**: Attacker triggers logout for user

**Mitigation**:
- Logout requires GET or POST (both safe)
- No sensitive operations on logout
- User must re-authenticate

### Session Token Leakage

**Risk**: Session ID exposed in logs

**Mitigation**:
- Session ID is hashed in logs
- Use HTTPS for all traffic
- Set `HttpOnly` cookie attribute

## Best Practices

1. **Always use HTTPS**: Prevents session interception
2. **Set redirect URL**: Provide good UX after logout
3. **Clear all cookies**: Ensure complete logout
4. **Log logout events**: Audit trail for compliance
5. **Test SLO**: Verify OIDC logout works correctly

## Next Steps

- **[OIDC Authentication](./oidc-authentication.md)** - OIDC setup and configuration
- **[Session Management](./session-management.md)** - Session storage configuration
- **[Security Best Practices](../deployment/security-best-practices.md)** - Security guidelines

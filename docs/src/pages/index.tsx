import Layout from '@theme/Layout';

export default function Home() {
  return (
    <Layout
      title="Keyline - Authentication Proxy for Elasticsearch"
      description="Modern authentication proxy for Elasticsearch with OIDC and Basic Auth support"
    >
      <main style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '4rem 2rem',
        maxWidth: '1200px',
        margin: '0 auto',
      }}>
        <h1 style={{ fontSize: '3rem', marginBottom: '1rem' }}>
          Welcome to Keyline
        </h1>
        
        <p style={{ fontSize: '1.25rem', marginBottom: '2rem', textAlign: 'center', maxWidth: '800px' }}>
          <strong>Modern Authentication Proxy for Elasticsearch</strong>
        </p>

        <p style={{ fontSize: '1.1rem', marginBottom: '3rem', textAlign: 'center', maxWidth: '800px', lineHeight: '1.6' }}>
          Keyline is a unified authentication proxy service that provides dual authentication modes 
          (OIDC and Basic Auth) simultaneously, supports multiple deployment modes (forwardAuth, 
          auth_request, standalone proxy), and automatically injects Elasticsearch credentials 
          into authenticated requests.
        </p>

        <div style={{
          display: 'flex',
          gap: '1rem',
          flexWrap: 'wrap',
          justifyContent: 'center',
          marginBottom: '4rem',
        }}>
          <a
            href="/keyline/docs/getting-started/about"
            style={{
              padding: '1rem 2rem',
              backgroundColor: '#a6e3a1',
              color: '#1e1e2e',
              textDecoration: 'none',
              borderRadius: '8px',
              fontWeight: 'bold',
              fontSize: '1.1rem',
            }}
          >
            Get Started
          </a>
          <a
            href="/keyline/docs/configuration"
            style={{
              padding: '1rem 2rem',
              backgroundColor: '#313244',
              color: '#cdd6f4',
              textDecoration: 'none',
              borderRadius: '8px',
              fontWeight: 'bold',
              fontSize: '1.1rem',
            }}
          >
            Configuration
          </a>
          <a
            href="/keyline/docs/deployment/docker"
            style={{
              padding: '1rem 2rem',
              backgroundColor: '#313244',
              color: '#cdd6f4',
              textDecoration: 'none',
              borderRadius: '8px',
              fontWeight: 'bold',
              fontSize: '1.1rem',
            }}
          >
            Deployment
          </a>
        </div>

        <section style={{ maxWidth: '1000px', width: '100%' }}>
          <h2 style={{ fontSize: '2rem', marginBottom: '1.5rem' }}>Key Features</h2>
          
          <div style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))',
            gap: '1.5rem',
          }}>
            <FeatureCard
              title="Dual Authentication"
              description="Support both interactive (OIDC) and programmatic (Basic Auth) access simultaneously"
            />
            <FeatureCard
              title="Dynamic User Management"
              description="Automatically create and manage Elasticsearch users for all authenticated users"
            />
            <FeatureCard
              title="Multiple Deployment Modes"
              description="Works with Traefik (forwardAuth), Nginx (auth_request), or as standalone proxy"
            />
            <FeatureCard
              title="OIDC Support"
              description="Full OpenID Connect implementation with PKCE, auto-discovery, and token validation"
            />
            <FeatureCard
              title="Security First"
              description="Cryptographic randomness, secure cookies, HTTPS enforcement, bcrypt password hashing"
            />
            <FeatureCard
              title="Observability"
              description="Prometheus metrics, OpenTelemetry tracing, structured logging with context"
            />
          </div>
        </section>

        <section style={{ maxWidth: '1000px', width: '100%', marginTop: '4rem' }}>
          <h2 style={{ fontSize: '2rem', marginBottom: '1.5rem' }}>Installation</h2>
          
          <div style={{
            backgroundColor: '#1e1e2e',
            padding: '1.5rem',
            borderRadius: '8px',
            marginBottom: '1rem',
          }}>
            <h3 style={{ marginTop: 0 }}>Docker (Recommended)</h3>
            <pre style={{
              backgroundColor: '#11111b',
              padding: '1rem',
              borderRadius: '4px',
              overflow: 'auto',
            }}>
              <code>docker pull ghcr.io/wasilak/keyline:latest</code>
            </pre>
          </div>

          <div style={{
            backgroundColor: '#1e1e2e',
            padding: '1.5rem',
            borderRadius: '8px',
          }}>
            <h3 style={{ marginTop: 0 }}>Binary</h3>
            <pre style={{
              backgroundColor: '#11111b',
              padding: '1rem',
              borderRadius: '4px',
              overflow: 'auto',
            }}>
              <code>{`curl -LO https://github.com/wasilak/keyline/releases/latest/download/keyline-linux-amd64.tar.gz
tar -xzf keyline-linux-amd64.tar.gz
sudo mv keyline /usr/local/bin/`}</code>
            </pre>
          </div>
        </section>

        <section style={{ maxWidth: '1000px', width: '100%', marginTop: '4rem' }}>
          <h2 style={{ fontSize: '2rem', marginBottom: '1.5rem' }}>Community</h2>
          
          <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap' }}>
            <a
              href="https://github.com/wasilak/keyline"
              style={{
                padding: '0.75rem 1.5rem',
                backgroundColor: '#313244',
                color: '#cdd6f4',
                textDecoration: 'none',
                borderRadius: '6px',
              }}
            >
              GitHub
            </a>
            <a
              href="https://github.com/wasilak/keyline/issues"
              style={{
                padding: '0.75rem 1.5rem',
                backgroundColor: '#313244',
                color: '#cdd6f4',
                textDecoration: 'none',
                borderRadius: '6px',
              }}
            >
              Issues
            </a>
            <a
              href="https://github.com/wasilak/keyline/discussions"
              style={{
                padding: '0.75rem 1.5rem',
                backgroundColor: '#313244',
                color: '#cdd6f4',
                textDecoration: 'none',
                borderRadius: '6px',
              }}
            >
              Discussions
            </a>
          </div>
        </section>
      </main>
    </Layout>
  );
}

function FeatureCard({ title, description }) {
  return (
    <div style={{
      padding: '1.5rem',
      backgroundColor: '#313244',
      borderRadius: '8px',
      border: '1px solid #45475a',
    }}>
      <h3 style={{ marginTop: 0, marginBottom: '0.75rem' }}>{title}</h3>
      <p style={{ margin: 0, color: '#a6adc8', lineHeight: '1.5' }}>{description}</p>
    </div>
  );
}

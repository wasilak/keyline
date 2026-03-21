/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  docs: [
    {
      type: 'category',
      label: 'Getting Started',
      collapsed: false,
      items: [
        'getting-started/about',
        'getting-started/architecture',
        'getting-started/quick-start',
        'getting-started/configuration-basics',
        'getting-started/migration-from-elastauth',
      ],
    },
    {
      type: 'category',
      label: 'Authentication',
      collapsed: false,
      items: [
        'authentication/overview',
        'authentication/oidc-authentication',
        'authentication/local-users-basic-auth',
        'authentication/session-management',
        'authentication/logout',
      ],
    },
    {
      type: 'category',
      label: 'User Management',
      collapsed: false,
      items: [
        'user-management/dynamic-user-management',
        'user-management/role-mappings',
        'user-management/credential-caching',
      ],
    },
    {
      type: 'category',
      label: 'Deployment',
      collapsed: false,
      items: [
        'deployment-modes/forwardauth-traefik',
        'deployment-modes/auth-request-nginx',
        'deployment-modes/standalone-proxy',
        'deployment/docker',
        'deployment/kubernetes',
        'deployment/binary',
        'deployment/security-best-practices',
      ],
    },
    'configuration',
    'observability',
    'integrations',
    'troubleshooting',
    'faq',
  ],
};

module.exports = sidebars;

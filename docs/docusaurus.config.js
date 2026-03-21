// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'Keyline',
  tagline: 'Modern Authentication Proxy for Elasticsearch',
  favicon: 'img/favicon.svg',

  // Set the production url of your site here
  url: 'https://wasilak.github.io',
  // Set the /<baseUrl>/ pathname under which your site is served
  // For GitHub pages deployment, it is often '/<projectName>/'
  baseUrl: '/keyline/',

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: 'wasilak', // Usually your GitHub org/user name.
  projectName: 'keyline', // Usually your repo name.

  onBrokenLinks: 'warn',

  markdown: {
    hooks: {
      onBrokenMarkdownLinks: 'warn',
    },
  },

  // Even if you don't use internalization, you can use this field to set useful
  // metadata like html lang. For example, if your site is Chinese, you may want
  // to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  markdown: {
    mermaid: true,
    format: 'mdx',
  },

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          sidebarPath: require.resolve('./sidebars.js'),
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          editUrl:
            'https://github.com/wasilak/keyline/tree/main/docs/',
          // Versioning configuration
          lastVersion: 'current',
          versions: {
            current: {
              label: '1.0.x (Latest)',
              path: '/',
            },
          },
        },
        blog: false,
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      // Replace with your project's social card
      image: 'img/secan-social-card.jpg',
      navbar: {
        title: 'Keyline',
        logo: {
          alt: 'Keyline Logo',
          src: 'img/logo.svg',
        },
        items: [
          {
            type: 'docSidebar',
            sidebarId: 'docs',
            position: 'left',
            label: 'Documentation',
          },
          {
            type: 'docsVersionDropdown',
            position: 'right',
          },
          {
            href: 'https://github.com/wasilak/keyline',
            label: 'GitHub',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'Documentation',
            items: [
              {
                label: 'Getting Started',
                to: 'docs/01-getting-started/about',
              },
              {
                label: 'Configuration',
                to: 'docs/06-configuration/reference',
              },
              {
                label: 'Deployment',
                to: 'docs/05-deployment/docker',
              },
            ],
          },
          {
            title: 'Community',
            items: [
              {
                label: 'GitHub',
                href: 'https://github.com/wasilak/keyline',
              },
              {
                label: 'Issues',
                href: 'https://github.com/wasilak/keyline/issues',
              },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} Keyline. Built with Docusaurus.`,
      },
      colorMode: {
        defaultMode: 'dark',
        disableSwitch: false,
        respectPrefersColorScheme: true,
      },
      prism: {
        theme: require('prism-react-renderer').themes.github,
        darkTheme: require('prism-react-renderer').themes.dracula,
        additionalLanguages: ['go', 'yaml', 'json', 'bash', 'docker'],
      },
      mermaid: {
        theme: {
          light: 'default',
          dark: 'dark',
        },
        options: {
          themeVariables: {
            primaryColor: '#2e8555',
            primaryTextColor: '#fff',
            primaryBorderColor: '#29784c',
            lineColor: '#29784c',
            secondaryColor: '#33925d',
            tertiaryColor: '#3cad6e',
          },
        },
      },
    }),

  themes: ['@docusaurus/theme-mermaid'],

  // Custom webpack config to suppress warnings
  plugins: [
    function (context, options) {
      return {
        name: 'custom-webpack-config',
        configureWebpack(config, isServer) {
          return {
            ignoreWarnings: [
              {
                module: /vscode-languageserver-types/,
              },
            ],
          };
        },
      };
    },
  ],
};

module.exports = config;

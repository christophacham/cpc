// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

const lightCodeTheme = require('prism-react-renderer/themes/github');
const darkCodeTheme = require('prism-react-renderer/themes/dracula');

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'Cloud Price Compare (CPC)',
  tagline: 'A production-grade API service for aggregating and comparing cloud pricing data',
  favicon: 'img/favicon.ico',

  // Set the production url of your site here
  url: 'https://cpc-docs.localhost',
  // Set the /<baseUrl>/ pathname under which your site is served
  baseUrl: '/',

  // GitHub pages deployment config.
  organizationName: 'raulc0399',
  projectName: 'cpc',

  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',

  // Even if you don't use internalization, you can use this field to set useful
  // metadata like html lang. For example, if your site is Chinese, you may want
  // to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          routeBasePath: '/', // Serve docs at site root
          sidebarPath: require.resolve('./sidebars.js'),
          editUrl: 'https://github.com/raulc0399/cpc/tree/main/docs-site/',
        },
        blog: false, // Disable blog
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
      image: 'img/cpc-social-card.jpg',
      navbar: {
        title: 'CPC Docs',
        logo: {
          alt: 'CPC Logo',
          src: 'img/logo.svg',
        },
        items: [
          {
            type: 'docSidebar',
            sidebarId: 'tutorialSidebar',
            position: 'left',
            label: 'Documentation',
          },
          {
            href: 'http://localhost:8080',
            label: 'GraphQL API',
            position: 'right',
          },
          {
            href: 'https://github.com/raulc0399/cpc',
            label: 'GitHub',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'Docs',
            items: [
              {
                label: 'Getting Started',
                to: '/getting-started',
              },
              {
                label: 'API Reference',
                to: '/api-reference',
              },
            ],
          },
          {
            title: 'Tools',
            items: [
              {
                label: 'GraphQL Playground',
                href: 'http://localhost:8080',
              },
              {
                label: 'Database',
                href: 'http://localhost:5432',
              },
            ],
          },
          {
            title: 'More',
            items: [
              {
                label: 'GitHub',
                href: 'https://github.com/raulc0399/cpc',
              },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} CPC Documentation. Built with Docusaurus.`,
      },
      prism: {
        theme: lightCodeTheme,
        darkTheme: darkCodeTheme,
        additionalLanguages: ['bash', 'go', 'sql', 'graphql'],
      },
    }),
};

module.exports = config;
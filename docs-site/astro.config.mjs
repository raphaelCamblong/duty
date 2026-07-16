// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
	site: 'https://duty-cli.xyz',
	integrations: [
		starlight({
			title: 'duty',
      logo: { src: './src/assets/logo.png', alt: 'duty' },
      head: [
        // Social card image, site-wide. Starlight already emits og:title,
        // og:description, og:type, og:url and twitter:card per page.
        {
          tag: 'meta',
          attrs: { property: 'og:image', content: 'https://duty-cli.xyz/screens/board-dark.png' },
        },
        {
          tag: 'meta',
          attrs: { property: 'og:image:alt', content: 'The duty TUI showing a board of tasks and their statuses.' },
        },
        {
          tag: 'meta',
          attrs: { name: 'twitter:image', content: 'https://duty-cli.xyz/screens/board-dark.png' },
        },
      ],
			social: [
				{
					icon: 'github',
					label: 'GitHub',
					href: 'https://github.com/raphaelCamblong/duty',
				},
      ],
      customCss: [
        './src/fonts/font-face.css',
        './src/styles/custom.css',
      ],
			sidebar: [
				{
					label: 'Start',
					items: [
						{ label: 'Overview', link: '/' },
						{ label: 'Getting started', link: '/getting-started/' },
					],
				},
				{
					label: 'Guide',
					items: [
						{ label: 'Tasks', link: '/tasks/' },
						{ label: 'Tracks & boards', link: '/tracks/' },
						{ label: 'CLI', link: '/cli/' },
						{ label: 'Config', link: '/config/' },
						{ label: 'TUI', link: '/tui/' },
					],
				},
				{
					label: 'Reference',
					items: [
						{ label: 'Internals', link: '/internals/' },
						{ label: 'Convention', link: '/convention/' },
					],
				},
			],
		}),
	],
});

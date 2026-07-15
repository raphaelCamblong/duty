// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
	site: 'https://duty.raph-camblong.workers.dev',
	integrations: [
		starlight({
			title: 'duty',
      logo: { src: './src/assets/logo.png', alt: 'duty' },
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

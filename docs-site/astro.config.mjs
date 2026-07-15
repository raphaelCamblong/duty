// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
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
			sidebar: [
				{ label: 'Getting started', link: '/getting-started/' },
				{ label: 'The task convention', link: '/convention/' },
				{ label: 'The full spec', link: '/spec/' },
			],
		}),
	],
});

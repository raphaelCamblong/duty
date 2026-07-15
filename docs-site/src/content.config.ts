import { defineCollection } from 'astro:content';
import { glob } from 'astro/loaders';
import { docsLoader } from '@astrojs/starlight/loaders';
import { docsSchema } from '@astrojs/starlight/schema';

export const collections = {
	docs: defineCollection({ loader: docsLoader(), schema: docsSchema() }),
	// Sourced live from the repo — no copies, no symlinks.
	topics: defineCollection({
		loader: glob({
			pattern: ['tasks.md', 'tracks.md', 'cli.md', 'config.md', 'tui.md', 'internals.md'],
			base: '../docs',
		}),
	}),
	convention: defineCollection({
		loader: glob({ pattern: 'README.md', base: '../duty' }),
	}),
};

# docs-site

The duty docs site — Astro Starlight, served as Cloudflare Workers static
assets (`wrangler.jsonc`, no Worker code).

The spec and the task convention are sourced live from `../docs/spec.md` and
`../duty/README.md` at build time via glob loaders — no copies, no symlinks.
Only the landing page and the getting-started page live here, in
`src/content/docs/`.

```sh
npm install
npm run dev      # local dev server
npm run build    # static site → dist/
npm run preview  # serve dist/ the way Workers will (wrangler dev)
npm run deploy   # build + wrangler deploy (needs `wrangler login` once)
```

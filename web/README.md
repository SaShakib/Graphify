# graphify web

React + TypeScript + Vite frontend for graphify's code graph visualizer. See
the [repo root README](../README.md) for the full project overview and
[API_CONTRACT.md](../API_CONTRACT.md) for the REST contract this talks to.

## Development

```sh
npm install
npm run dev              # hits the real API at localhost:8420 via proxy
VITE_USE_MOCK=true npm run dev   # runs standalone against a small mock repo, no backend needed
```

## Build

```sh
npm run build             # outputs to dist/, picked up automatically by `graphify serve`
```

Do not commit a `.env.local` with `VITE_USE_MOCK=true` — Vite loads
`.env.local` for every mode, including production builds, so it would bake
mock data into `dist/`.

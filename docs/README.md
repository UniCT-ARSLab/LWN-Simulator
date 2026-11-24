# LWN Simulator documentation

This directory contains the multilingual documentation website for LWN
Simulator. It is built with Next.js, MDX and Nextra and is published with the
GitHub Pages deployment workflow.

The public documentation is available at:

<https://giuseppe-compagnone.github.io/LWN-Simulator/docs>

Available locales are English (`en`), Italian (`it`), Spanish (`es`), French
(`fr`) and German (`de`). English is the default locale.

## Development

From the repository root:

```bash
cd docs
yarn install
yarn dev
```

Open <http://localhost:3000> to browse the documentation. Locale routes are
available below `/en`, `/it`, `/es`, `/fr` and `/de`.

## Validation and build

```bash
yarn lint
yarn build
```

The production build exports a static site. The `deploy-docs` GitHub Actions
workflow publishes the documentation together with the UI component
Storybook.

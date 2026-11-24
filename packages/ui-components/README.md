# UI components

The UI components package contains the reusable visual building blocks shared
by the web frontend, desktop launcher and Storybook. It keeps common controls
and interaction patterns consistent across the application.

The deployed component catalog is available at
<https://giuseppe-compagnone.github.io/LWN-Simulator/storybook/>.

## Component groups

- `src/components/common/`: buttons, logos, progress indicators, spinners and
  toggles;
- `src/components/form/`: the form container and reusable fields;
- `src/components/layout/`: cards, page headers and entity details;
- `src/components/widget/`: popups, tables and other focused widgets;
- `src/hooks/`, `src/services/`, `src/types/` and `src/utils/`: shared support
  code;
- `src/styles/`: package-wide Sass entry points and design tokens.

Each component directory contains the component, its documented prop types,
styles, exports and—where relevant—Storybook stories. Stories cover the
normal, interactive, disabled and edge-case states that are important to the
component.

## Development

Start Storybook from the repository root with:

```bash
yarn workspace @lwn-simulator/ui-components storybook
```

Build the package and its static Storybook with:

```bash
yarn workspace @lwn-simulator/ui-components build
yarn workspace @lwn-simulator/ui-components build-storybook
```

Run the package checks with:

```bash
yarn workspace @lwn-simulator/ui-components lint:ts
yarn workspace @lwn-simulator/ui-components lint:css
```

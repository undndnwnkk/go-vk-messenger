# 0003. Frontend design tokens: import the handoff file, map it into Tailwind

Status: accepted (2026-11-18)

## Context

`docs/design/handoff/tokens.css` is the single source of truth for colors, type,
spacing, radii and shadows in both themes. The handoff asks to use it unchanged
and to keep the token names. The frontend is styled with Tailwind CSS.

## Decision

- `frontend/src/styles/index.css` imports `docs/design/handoff/tokens.css` directly
  instead of keeping a copy, so a design update is a single-file change.
- An `@theme inline` block maps every token into Tailwind (`bg-surface`,
  `text-text-2`, `rounded-md`, `h-touch`, `w-list`, `shadow-pop`, ...). Utilities
  emit `var(--token)`, so switching `pl-light` / `pl-dark` on `<html>` restyles
  everything without a rebuild.
- Tailwind's default palette, fonts, type scale, radii, shadows and breakpoints are
  reset with `--*: initial`. A hard-coded color such as `bg-blue-500` does not
  compile. `--spacing` is `4px`, so `p-1` equals `--sp-1`.
- Font families are exposed as `font-sans`, `font-heading` and `font-code`, because
  Tailwind's `--font-*` namespace would collide with `--font-ui` and `--font-display`.
- The only breakpoint is `desk` (760px), the handoff's mobile/desktop boundary.

## Consequences

- The frontend build reads one file outside `frontend/`. A future frontend Docker
  image needs the repository root as build context.
- New tokens in the handoff need one line in the `@theme inline` block.

# Accessibility review reference

Load this reference only when the PR content changes user-facing HTML, UI, or
interaction semantics. The goal is to catch interaction or semantics regressions
the change introduces.

## Focus areas

- Keyboard and focus: interactive elements reachable and operable by keyboard;
  focus order and visible focus preserved or intentionally changed.
- Semantics: meaningful labels, roles, and names for new interactive or live
  regions; changes that remove or override existing semantics.
- Contrast and targets: new colors or hit targets that drop below a clear minimum.
- Reduced motion and assistive tech: new animation or layout that ignores
  `prefers-reduced-motion` or breaks screen-reader announcements.

## Applying the rules

- Report only regressions the PR introduces or makes reachable, with the
  observable breakage (which user with which assistive need is blocked) and the
  causing diff location.
- Drop candidates that are preferences or that an existing mechanism or test
  already covers.

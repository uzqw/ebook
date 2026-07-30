# Relay Status: mobile-typography

- **State:** READY
- **Last completed leg:** 0
- **Next leg to run:** 1
- **Finish line:** See charter — headings visibly distinct, paragraphs
  segmented (no blob), no regression, all checks pass, focused commits.

## Next task (leg 1)

**Diagnose the blob rendering and land the first verifiable fix.**

Read plan item 1 in `plan.md`. Reproduce how extracted page text flows
through `reflowText()` into the reflow view on a narrow viewport, identify
why titles and paragraphs collapse into one blob, and fix the primary cause
with focused tests. If diagnosis shows two independent causes, fix only the
first in this leg and name the second as the next leg in this status.

## Minimal context for the next runner

- Reflow pipeline: `src/lib/reflow.ts` (`reflowText()` produces
  `{ kind: 'heading' | 'paragraph', text }` blocks; paragraph boundaries
  come from blank lines or lines starting with 2+ spaces / U+3000).
- Rendering: `src/views/books/ReaderView.vue` lines ~633–650 render
  `reflowBlocks` as `h2.reader-reflow-heading` / `p.reader-reflow-paragraph`.
- Styles: `src/styles/main.css` `.reader-reflow-page` (~285),
  `.reader-reflow-heading` (~294), `.reader-reflow-paragraph` (~307),
  narrow-viewport media query (~393).
- Existing tests: `src/lib/reflow.test.ts`.
- The completed `mobile-reflow` relay built this pipeline; its packet is at
  `.pi-web/relays/mobile-reflow/` — do not read it unless a specific lookup
  becomes necessary.
- npm commands run via `cnpm` under NVM Node 22 (per user requirement in the
  previous relay).

## Progress documentation and commit requirements

Before handoff or stop you must:

1. Update this `status.md` (state, leg counters, next task, context,
   blockers).
2. Append one concise entry to `log.md`.
3. Pass verification: focused reflow tests, then `cnpm run format:check`,
   `cnpm run lint`, `cnpm run typecheck`, `cnpm test`, `cnpm run build`.
4. Create one focused Conventional Commit per the charter's commit policy,
   including source, tests, and packet updates; exclude unrelated changes.
5. Then spawn the next leg once (see charter handover protocol), or stop if
   DONE/BLOCKED.

## Blockers and intervention state

- **Active blockers:** none known.
- Intervention conditions (stop, do not spawn): unfixable verification
  failure, out-of-scope changes required, ambiguous root cause or product
  trade-off, missing test data/environment, charter churn. See charter.

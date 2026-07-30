# Relay Status: mobile-typography

- **State:** READY
- **Last completed leg:** 1
- **Next leg to run:** 2
- **Finish line:** See charter — headings visibly distinct, paragraphs
  segmented (no blob), no regression, all checks pass, focused commits.

## Next task (leg 2)

**Fix heading detection: real book titles still render as body paragraphs.**

Leg 1 fixed paragraph segmentation (the primary blob cause) and confirmed
a second independent cause: `headingPattern` in `src/lib/reflow.ts` misses
real title formats, so titles render as ordinary paragraphs:

- ALL-CAPS / plain English titles and section headings, e.g.
  `The Power of Now`, `PAST PAIN: DISSOLVING THE PAIN-BODY`, `MY RECOVERY`
  (book `s8gcisy3gxy79l5` page 60, book `wcopgxn7rlu1e7k`).
- Numbered section headings, e.g. `3.1.2 不同场景下的不同架构案例`,
  `5. 不要有单点` (book `k7jfw2ne4auy9zk` page 15; these now get their own
  paragraph after leg 1, but still no heading style).

Read plan item 2 in `plan.md`. Improve heading detection with focused tests
in `src/lib/reflow.test.ts`, verified against the real extracted pages in
`.local/pb_data/data.db` (see query pattern below). Keep false positives
out: ordinary short body lines must not become headings.

## Minimal context for the next runner

- Reflow pipeline: `src/lib/reflow.ts` (`reflowText()` produces
  `{ kind: 'heading' | 'paragraph', text }` blocks).
- **Leg 1 landed:** paragraph-break inference in `reflowText()`. When a
  page has no real blank-line paragraph markers (PDF pages with no blank
  lines, or EPUB pages with synthetic `\n\n` after every visual line), a
  short final line (display width ≤ 0.7 of the page max, and ≤ 0.45 or
  ending in sentence-final punctuation; hyphen/comma-ending lines never
  close a paragraph) now ends the paragraph. Pages with genuine blank-line
  boundaries keep the old behavior exactly.
- Rendering: `src/views/books/ReaderView.vue` lines ~633–650 render
  `reflowBlocks` as `h2.reader-reflow-heading` / `p.reader-reflow-paragraph`.
- Styles: `src/styles/main.css` `.reader-reflow-page` (~285),
  `.reader-reflow-heading` (~294), `.reader-reflow-paragraph` (~307),
  narrow-viewport media query (~393) — all confirmed correct in leg 1.
- Real book text for reproduction lives in `.local/pb_data/data.db`
  (PocketBase SQLite). Query pattern:
  `sqlite3 .local/pb_data/data.db "select text from book_pages where book='<id>' and page_number=<n>;"`
  Blob-shape books: `5tdudhh3ei0bdc4` （断舍离， synthetic blanks),
  `mgbrm9gxutqeayb` (Book of Joy, synthetic blanks),
  `k7jfw2ne4auy9zk` (Java PDF, no blanks). Well-behaved books:
  `s8gcisy3gxy79l5`, `wcopgxn7rlu1e7k`, `71tc8gxub0n0h1j`, `8kad1osgk59j1lc`.
- Existing tests: `src/lib/reflow.test.ts`.
- npm commands run via `cnpm` under NVM Node 22 (per user requirement in
  the previous relay): `export NVM_DIR="$HOME/.nvm" && . "$NVM_DIR/nvm.sh"
  && nvm use 22`.

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

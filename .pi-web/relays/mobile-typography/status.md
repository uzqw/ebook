# Relay Status: mobile-typography

- **State:** READY
- **Last completed leg:** 2
- **Next leg to run:** 3
- **Finish line:** See charter — headings visibly distinct, paragraphs
  segmented (no blob), no regression, all checks pass, focused commits.

## Next task (leg 3)

**Final regression audit and finish** — plan item 3 in `plan.md`.

Both blob causes are now fixed (leg 1 paragraph segmentation, leg 2 heading
detection). Verify every finish-line item in `charter.md`: distinct headings
and segmented paragraphs on the mobile reflow view, no regression in
desktop/original-page mode, page navigation, reading progress, bookmarks,
and notes; confirm each leg produced its focused commit. If everything
holds, mark the relay `DONE`, append the final log entry, and stop without
spawning. Only make a final packet/docs commit if something is missing.

## Minimal context for the next runner

- Reflow pipeline: `src/lib/reflow.ts` (`reflowText()` produces
  `{ kind: 'heading' | 'paragraph', text }` blocks).
- **Leg 1 landed:** paragraph-break inference in `reflowText()`. Pages with
  no real blank-line markers now segment on short final lines (display width
  ≤ 0.7 of page max, and ≤ 0.45 or ending in sentence-final punctuation;
  hyphen/comma-ending lines never close a paragraph). Pages with genuine
  blank-line boundaries keep the old behavior exactly.
- **Leg 2 landed:** heading detection in `src/lib/reflow.ts`. Beyond the
  original `headingPattern` （第X章， Chapter N, …), `isHeading()` now also
  detects: numbered sections (`5. 不要有单点`, `3.1.2 …`, remainder must
  start CJK/uppercase, no sentence-final punctuation), ALL-CAPS English
  titles (`MY RECOVERY`; no lowercase, no CJK, ≥2 words or ≥4 letters), and
  — only for standalone flushed blocks, never mid-paragraph lines — plain
  title-case English titles (`The Power of Now`; ≤50 chars, no ending
  punctuation, >50% capitalized words). Verified against real pages:
  `s8gcisy3gxy79l5` p60, `wcopgxn7rlu1e7k` p10, `k7jfw2ne4auy9zk` p15.
- Rendering: `src/views/books/ReaderView.vue` lines ~633–650 render
  `reflowBlocks` as `h2.reader-reflow-heading` / `p.reader-reflow-paragraph`.
- Styles: `src/styles/main.css` `.reader-reflow-page` (~285),
  `.reader-reflow-heading` (~294), `.reader-reflow-paragraph` (~307),
  narrow-viewport media query (~393) — all confirmed correct in leg 1.
- Real book text for spot checks lives in `.local/pb_data/data.db`
  (PocketBase SQLite). Query pattern:
  `sqlite3 "file:.local/pb_data/data.db?mode=ro&immutable=1" "select text from book_pages where book='<id>' and page_number=<n>;"`
  Blob-shape books: `5tdudhh3ei0bdc4` （断舍离， synthetic blanks),
  `mgbrm9gxutqeayb` (Book of Joy, synthetic blanks),
  `k7jfw2ne4auy9zk` (Java PDF, no blanks). Well-behaved books:
  `s8gcisy3gxy79l5`, `wcopgxn7rlu1e7k`, `71tc8gxub0n0h1j`, `8kad1osgk59j1lc`.
- Existing tests: `src/lib/reflow.test.ts` (17 tests).
- npm commands run via `cnpm` under NVM Node 22: `export NVM_DIR="$HOME/.nvm"
  && . "$NVM_DIR/nvm.sh" && nvm use 22`.
- Prior commits: `b2fec30 fix(reflow): segment paragraphs via short final
  lines` (leg 1), leg 2 commit on HEAD.

## Progress documentation and commit requirements

Before handoff or stop you must:

1. Update this `status.md` (state, leg counters, next task, context,
   blockers).
2. Append one concise entry to `log.md`.
3. Pass verification: focused reflow tests, then `cnpm run format:check`,
   `cnpm run lint`, `cnpm run typecheck`, `cnpm test`, `cnpm run build`.
4. Create one focused Conventional Commit per the charter's commit policy,
   including source, tests, and packet updates; exclude unrelated changes.
   (Leg 3 may end with no code change — then commit only packet updates, or
   nothing if unchanged.)
5. Then spawn the next leg once (see charter handover protocol), or stop if
   DONE/BLOCKED.

## Blockers and intervention state

- **Active blockers:** none known.
- Intervention conditions (stop, do not spawn): unfixable verification
  failure, out-of-scope changes required, ambiguous root cause or product
  trade-off, missing test data/environment, charter churn. See charter.

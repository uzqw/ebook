# Relay Log: mobile-typography

## Setup — 2026-07-30

Created the relay packet from the approved choices: goal is visible heading
and paragraph structure in the mobile reflow view (no text blob); sizing is
one independently verifiable change per leg; task selection follows
`plan.md` dependency order when status names nothing; handover is automatic
with exactly one spawn at the end of each leg; reading discipline is charter
+ status + named plan sections only, never the full log; each leg ends in
one focused Conventional Commit after verification. Intervention conditions:
unfixable verification failure, scope creep, ambiguous root cause or product
trade-off, missing data/environment, charter churn. Seeded leg 1 to diagnose
the blob rendering and land the first verifiable fix.

## Leg 1 — diagnose blob, land paragraph segmentation fix

**Diagnosis.** Reproduced against real extracted pages in
`.local/pb_data/data.db`. Two extraction shapes produce the blob because
they carry no paragraph markers, so `reflowText()` merged the whole page
into one block:

- PDF pages with visual lines separated by single `\n` and zero blank
  lines (book `k7jfw2ne4auy9zk`).
- EPUB pages where MuPDF emits a synthetic `\n\n` after every visual line
  (books `5tdudhh3ei0bdc4`, `mgbrm9gxutqeayb`); the previous relay's
  `hasSyntheticLineSpacing` fix correctly rejoined wrapped sentences but
  as a side effect erased every paragraph boundary.

Rendering (`ReaderView.vue`) and styles (`main.css`) were confirmed
correct, and well-behaved EPUBs with real blank-line separators already
segmented fine.

**Fix (primary cause).** `reflowText()` now infers paragraph breaks only
when a page lacks real blank-line markers: a visual line ends a paragraph
when its display width (full-width chars 1, others 0.5) is ≤ 0.7 of the
page's widest line and it either is ≤ 0.45 or ends with sentence-final
punctuation; hyphen- or comma-ending lines never close a paragraph. Pages
with genuine blank-line boundaries keep the exact old behavior. Verified
on the real pages: 断舍离 p13 went from 1 blob to 8 true paragraphs, Java
PDF p15 from 1 to 10, Power of Now p60 unchanged. Also fixed shape
detection to trim the raw text first so a trailing newline does not
disable inference.

**Deferred second cause (leg 2).** `headingPattern` misses real titles:
ALL-CAPS/plain English headings (`MY RECOVERY`, `PAST PAIN: …`) and
numbered sections (`3.1.2 …`, `5. 不要有单点`) render as body paragraphs.

**Verification.** `reflow.test.ts` 13/13 (3 new focused tests);
`cnpm run format:check`, `lint`, `typecheck`, `cnpm test` (20/20),
`cnpm run build` all green under NVM Node 22.

**Handoff.** Spawning leg 2 (heading detection) per plan item 2.

## Leg 2 — heading detection for real book titles

- **Task (from status):** fix `headingPattern` missing real title formats, so
  titles render as headings instead of body paragraphs (plan item 2).
- **Did:** extended `isHeading()` in `src/lib/reflow.ts` with three detectors
  alongside the original pattern:
  - Numbered sections (`5. 不要有单点`, `3.1.2 不同场景下的不同架构案例`):
    digits + dots/、, remainder must open with CJK or uppercase Latin, no
    sentence-final punctuation, ≤40 chars. Checked per source line, so the
    heading is isolated even when leg-1 inference would not split there.
  - ALL-CAPS English titles (`MY RECOVERY`, `PAST PAIN: DISSOLVING THE
    PAIN-BODY`): no lowercase, no CJK, ≤60 chars, ≥2 words or ≥4 letters.
  - Plain title-case English titles (`The Power of Now`): ≤50 chars, no
    ending sentence punctuation, ≥2 capitalized words and >50% of words
    capitalized. Applied only to standalone flushed blocks
    (`allowTitleCase`), never per source line — a capitalized fragment
    mid-paragraph like "Fluid Latin text ..." must stay body text.
- **False-positive guards:** "3.5 million people attended the event"
  (lowercase remainder after number), "Watch out for any sign of
  unhappiness" (1 of 7 words capitalized), "OK" (single short word), and CJK
  lines containing Latin acronyms (excluded from ALL-CAPS) all stay
  paragraphs. Existing test expecting `5. 不要有单点` as paragraph updated
  to heading.
- **Verified against real extracted pages** (`.local/pb_data/data.db`, via a
  temporary scratch vitest file, deleted after the run): `s8gcisy3gxy79l5`
  p60 → `The Power of Now` + `PAST PAIN: DISSOLVING THE PAIN-BODY` headings,
  clean paragraphs; `k7jfw2ne4auy9zk` p15 → exactly `5. 不要有单点` and
  `3.1.2 不同场景下的不同架构案例` as headings; `wcopgxn7rlu1e7k` p10 →
  `MY RECOVERY` heading, no false positives in body text.
- **Checks:** 17/17 reflow tests, `cnpm run format:check`, `lint`,
  `typecheck`, `test` (24), `build` — all green under NVM Node 22.
- **Artifacts:** `src/lib/reflow.ts`, `src/lib/reflow.test.ts`, status.md,
  log.md. One focused commit.
- **Handoff:** spawn leg 3 (final regression audit and finish, plan item 3).

## Leg 3 — final regression audit and finish (plan item 3)

- **Audit:** verified every charter finish-line item.
  - Headings: spot-checked real pages through `reflowText()` (blob books
    `5tdudhh3ei0bdc4`, `mgbrm9gxutqeayb`, `k7jfw2ne4auy9zk` plus
    well-behaved `s8gcisy3gxy79l5` p60/p61, `wcopgxn7rlu1e7k` p10/p11) —
    headings emitted on title pages, rendered as `h2.reader-reflow-heading`.
  - Paragraphs: every audited page segments into multiple blocks; none
    renders as one undifferentiated blob.
  - Regression: legs 1–2 changed only `src/lib/reflow.ts` + its tests;
    `reflowBlocks` renders only under `reflowEnabled`, so original-page
    mode, navigation, progress, bookmarks, notes are untouched.
  - Commits: `b2fec30` (leg 1), `4d23b70` (leg 2) confirmed focused.
- **Checks:** 17/17 reflow tests, `cnpm run format:check`, `lint`,
  `typecheck`, `test` (24/24), `build` — all green under NVM Node 22.
- **Artifacts:** status.md, log.md only (no code change needed).
- **Stop:** relay marked DONE; no further legs spawned.

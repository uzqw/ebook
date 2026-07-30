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

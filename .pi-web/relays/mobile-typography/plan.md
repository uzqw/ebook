# Relay Plan: mobile-typography

Read only the item named by `status.md`.

## 1. Diagnose the blob and land the first fix

- Reproduce: trace how one page's extracted text (`currentPage.text`) passes
  through `reflowText()` and renders in the mobile reflow view. Determine
  why titles/paragraphs collapse into one blob (likely candidates: extracted
  text lacks blank lines and indentation so everything joins into one
  paragraph; heading pattern misses the book's title format; or reflow
  styles not applying on the narrow viewport).
- Fix the primary cause with focused tests in `src/lib/reflow.test.ts`
  (and/or component/style verification as applicable).
- If diagnosis reveals a second independent cause, do **not** fix it here —
  record it in `status.md` as the next leg.
- Run the full verification suite (see `status.md`), update the packet,
  commit, then hand off.

**State:** ready for leg 1.

## 2. Fix the remaining cause (if any)

- Implement the secondary fix identified by leg 1 (e.g. heading detection
  vs. paragraph segmentation, whichever was deferred).
- Focused tests, full verification, packet update, focused commit, handoff.

**State:** waiting on leg 1.

## 3. Final regression audit and finish

- Verify every finish-line item in `charter.md`: distinct headings,
  segmented paragraphs, no regression in desktop/original-page/navigation/
  progress/bookmarks/notes, full check suite green.
- Confirm each leg produced its focused commit; make a final packet/docs
  commit only if something is missing.
- Mark the relay `DONE` in `status.md`, append the log entry, and stop
  without spawning.

**State:** waiting on legs 1–2.

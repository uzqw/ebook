# Relay Charter: mobile-typography

## Relay identity and root

- **Name:** `mobile-typography`
- **Display identity:** 手机端标题与段落排版修复
- **Relay root:** `.pi-web/relays/mobile-typography/`
- Every runner performs exactly one leg of this relay.

## Goal and finish line

Fix the mobile reflow view so extracted book text renders as visually
structured titles and paragraphs instead of one undifferentiated blob of text.

The relay is complete only when all of these are true:

1. **Headings are visibly distinct.** On the mobile reflow view, chapter and
   section titles render as headings (size, weight, spacing), clearly
   separated from body text.
2. **Paragraphs are properly segmented.** Body text renders as separate
   paragraphs with visible spacing/indent; text is not one continuous blob.
3. **No regression.** Desktop behavior, original-page mode, page navigation,
   reading progress, bookmarks, and notes show no identified regression.
4. **Checks pass.** Focused reflow tests plus `format:check`, `lint`,
   `typecheck`, `test`, and `build` all pass. Per the completed
   `mobile-reflow` relay precedent, use the `cnpm` executable under NVM
   Node 22 for npm commands.
5. **Focused commits.** Every leg ends in one focused Conventional Commit
   (see Commit policy); unrelated worktree changes are excluded.

When every item is satisfied, mark the relay `DONE`, update the packet, and
stop without spawning another session.

## Sizing

One leg is **one independently verifiable change**: the implementation needed
for that single change, its focused tests, packet updates, verification, and
one focused commit. Do not start a second change in the same leg. A pure
diagnosis may be part of a leg, but the leg must still land one verifiable
change (fix or failing-test-then-fix) — not investigation alone.

## Task selection policy

1. Use the explicit next task in `status.md` when present.
2. If status does not name a task, read only the section of `plan.md`
   referenced by `status.md` and choose the first incomplete item whose
   dependencies are complete.
3. If that does not identify one unambiguous task, set the relay to
   `BLOCKED` and request human intervention. Do not invent work or read the
   full log to reconstruct intent.

## Handover protocol

Handoff is **automatic**: after completing a leg, the runner spawns exactly
one next session and stops. Before any handoff, the current runner must:

1. Finish exactly one leg and pass its verification.
2. Update `status.md`: state, last completed leg, next leg to run, next task
   (or task-selection pointer), minimal context the next runner needs,
   blockers.
3. Append one concise `log.md` entry: what was done, why, decisions,
   artifacts changed, verification results, handoff or stop.
4. Create the focused commit including source, tests, and packet updates.
5. Call `spawn_session` **once**, as the final action, with a prompt whose
   first line is a natural header containing the relay name and next leg
   number, e.g.:

```text
Relay "mobile-typography" leg <N> begins now.

You are the next runner in this Relay method chain.

Read:
- .pi-web/relays/mobile-typography/charter.md
- .pi-web/relays/mobile-typography/status.md

Do not read log.md end-to-end. Use it only for targeted lookup if status.md
or charter.md points you there.

Run one leg according to the charter. Before handing off, update status.md,
append log.md, make work durable, then either spawn the next leg once or
stop with a clear intervention note.
```

Never spawn more than once per leg, never spawn early, and never spawn when
the relay is `DONE` or `BLOCKED`.

## Intervention signal

Stop immediately, do **not** spawn, set `status.md` state to `BLOCKED` with
the reason, and append a clear `log.md` note when any of these fire:

1. **Verification fails and cannot be fixed within the leg** — focused tests,
   lint, typecheck, or build stay red.
2. **Scope creep** — the fix requires changing modules outside this relay's
   scope (backend, deployment, unrelated views).
3. **Ambiguous root cause or product trade-off** — the cause of the blob
   rendering cannot be determined, or the fix requires a design/product
   decision (e.g. typography style choices) that is the human's to make.
4. **Missing data or environment** — no test book, account, or runtime is
   available to reproduce or verify.
5. **Charter churn** — the goal or this charter itself appears to need
   changing. Do not quietly redefine the task.

A clean stop with a clear blocker is a success.

## Reading discipline

- Always read `charter.md` and `status.md` to orient.
- Read only the `plan.md` section that `status.md` names for the current leg.
- **Never read `log.md` end-to-end.** Use targeted lookup only when
  `status.md` or this charter points to a specific entry.
- Read task-relevant source and test files as needed to execute the leg;
  do not defensively rebuild the relay's full history or read unrelated
  artifact trees.
- If `status.md` is insufficient, repair the baton with targeted inspection
  and update `status.md`; if that requires broad archaeology, fire the
  intervention signal.

## Commit policy

- One focused Conventional Commit per leg, created only after verification
  passes, containing that leg's source, tests, and packet updates. Unrelated
  pre-existing worktree changes are never included.
- Format: `<type>(<scope>): <summary>` with allowed types `feat`, `fix`,
  `refactor`, `chore`, `docs`, `perf`.
- Scope is the module being changed: `reader`, `reflow`, `styles`, `relay`,
  or another precise module name.
- First line in imperative mood, under 50 characters.
- Body: one blank line, then bullet points (`-`) starting with strong
  imperative verbs (improve, modify, clean, remove); no past tense; every
  line under 72 characters; no markdown formatting or code blocks.

# Relay Charter: mobile-reflow

## Relay identity and root

- **Name:** `mobile-reflow`
- **Display identity:** 手机端排版自适应
- **Relay root:** `.pi-web/relays/mobile-reflow/`
- Every runner performs exactly one leg of this relay.

## Goal and finish line

Finish the ebook reader's mobile adaptive reflow feature without regressing the original fixed-page reader.

The relay is complete only when all of these are true:

1. On a viewport no wider than 768px, a user with no saved layout preference gets fluid text reflow by default.
2. Extracted book text wraps to the available width with readable CJK/Latin paragraph handling and responsive spacing/font sizing.
3. The reader can switch between reflow and original-page modes, and the explicit preference persists.
4. A page with no extractable text provides a clear path back to the original page.
5. Desktop/default original-page behavior, page navigation, reading progress, bookmarks, notes, and zoom controls have no identified regression from the change.
6. Focused reflow tests pass, along with `npm run format:check`, `npm run lint`, `npm run typecheck`, `npm test`, and `npm run build`.
7. Relevant source, tests, relay status/log updates, and verification results are committed in a focused commit; unrelated pre-existing worktree changes are not included.

When every item is satisfied, mark the relay `DONE`, update the packet, and stop without spawning another session.

## Sizing

One leg is **one independently verifiable change**. It includes only the implementation needed for that change, focused tests, required documentation, packet updates, verification, and a focused commit. Do not start a second change in the same leg.

## Task selection policy

1. Use the explicit next task in `status.md` when present.
2. If status does not name a task, read only the referenced section of `plan.md` and choose the first incomplete item whose dependencies are complete.
3. If that does not identify one unambiguous task, set the relay to `BLOCKED` and request human intervention. Do not invent work or read the full log to reconstruct intent.

## Handover protocol

Before any handoff, the current runner must:

1. Finish exactly one leg and its verification.
2. Update `status.md` with current state, last completed leg, next leg number/task, minimum context, and blockers.
3. Append one concise entry to `log.md` with work performed, decisions, artifacts, verification, and stop/handoff disposition.
4. Commit only the leg's relevant changes. Never stage unrelated pre-existing worktree changes.
5. If the finish line is reached or the relay is `BLOCKED`, stop and do **not** spawn.
6. Otherwise, as the final action, call `spawn_session` exactly once with a prompt in this form:

```text
Relay "mobile-reflow" leg <N> begins now.

You are the next runner in this Relay method chain.

Read:
- .pi-web/relays/mobile-reflow/charter.md
- .pi-web/relays/mobile-reflow/status.md

Do not read log.md end-to-end. Use it only for targeted lookup if status.md or charter.md points you there.

Run one leg according to the charter. Before handing off, update status.md, append log.md, make work durable, then either spawn the next leg once or stop with a clear intervention note.
```

Handoffs are automatic after a successful, committed leg when a clear next leg remains.

## Intervention signal

Set `State: BLOCKED` prominently in `status.md`, record the exact decision/input needed, append the blocker to `log.md`, and stop without spawning when any of these occurs:

- task, status, or acceptance criteria have material ambiguity;
- implementation reality requires goal, finish-line, architecture, or core charter changes;
- destructive operations, data migration, secrets, permissions, or security decisions are required;
- required verification repeatedly fails and cannot be resolved within the leg's sizing;
- a required external dependency, service, permission, or human input is unavailable;
- progress would require modifying, staging, or committing unrelated pre-existing worktree changes;
- any core charter agreement needs to change.

## Reading discipline

A runner normally reads only:

1. `.pi-web/relays/mobile-reflow/charter.md`;
2. `.pi-web/relays/mobile-reflow/status.md`;
3. the exact `plan.md` section, source files, tests, commands, or targeted log entry named by status.

Do not read `log.md` end-to-end by default. Do not defensively read the full repository, full history, backlog, or unrelated modules. If the baton is insufficient, repair it with targeted inspection; if that requires broad archaeology or judgment about intent, raise the intervention signal.

## Commit and worktree discipline

- Each completed leg produces one focused commit after verification.
- Include that leg's source/tests and its `status.md`/`log.md` updates.
- Inspect `git status` before staging and before committing.
- Stage explicit paths; never use broad staging that can capture unrelated work.
- The pre-existing `README.md` change is outside this relay unless status is explicitly changed by the human to include it.
- Do not amend, reset, discard, or rewrite pre-existing user work.

# Subly (Brightloop subscription management) — agent instructions

**Read `AGENT_HANDOFF.md` first.** It has the current step, the user's rules, the conventions, the status board and the next task.

Non-negotiable:
- Never run `git add` / `commit` / `push` / merge / branch switches; the user does Git by hand. Read-only Git is fine.
- Never print or commit `.env` (it contains a real SMTP app password). Only `*.example` files are tracked.
- After every completed step, update `AGENT_HANDOFF.md` (status board, step log, next step).
- Go isn't installed locally; run Go through Docker (see `AGENT_HANDOFF.md` section 3 / the Makefile).

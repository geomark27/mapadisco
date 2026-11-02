---
name: devlog
description: Mandatory project change log ("bitácora") — one entry per branch, one card per release, written in the language the project or user chooses. Use it ALWAYS when changing code, config, scripts or docs on a branch, before committing or opening a PR; when preparing a release; when orchestrating subagents that change code; when reviewing a PR; and when asked for a devlog, changelog, "bitácora", "registro de cambios", "qué cambió", or to set the log up in a project.
---

# devlog — change log per branch and per release

One **entry per branch** in `<log dir>/entries/` and one **card per release** in
`<log dir>/releases/`. Full guide, multi-agent scenarios and CI: [HOW-IT-WORKS.md](HOW-IT-WORKS.md).
Read it when in doubt, when several agents are involved, or before a release.

## Scripts

`S` = the `scripts/` folder next to this file. If the repo vendors the skill
(`.claude/skills/devlog/`), use that copy; otherwise `~/.claude/skills/devlog/scripts`.

| Script | Purpose |
|---|---|
| `$S/new-entry.sh "Title"` | Create the branch entry or return the existing one. Prints its path. |
| `$S/verify.sh [--base develop]` | Validate the entry. Exit 0 = valid. Read-only. |
| `$S/close-release.sh X.Y.Z --type … --previous vA.B.C --notes f.md` | Close a version (no commit). `--dry-run` to preview. |
| `$S/init.sh --lang es --dir docs/bitacora --vendor` | Set the log up in a project. |

Settings live in `.devlog.conf` at the repo root (`DEVLOG_DIR`, `DEVLOG_LANG`, …).

## Language

- This skill, its scripts and their messages are in English.
- **The log content is written in the project's language**: `DEVLOG_LANG` in `.devlog.conf`
  (`es`, `en`, …). Section headings come from that language's template; write titles and bodies
  in that language too. If the user asks for a different language, follow the user.
- Front matter keys and values (`branch`, `version: unreleased`, …) stay in English: they are
  machine data the scripts depend on.
- New project without a log: ask which language if it is not obvious from the user or repo.

## Precedence

Project rules win: its `CLAUDE.md`, a vendored copy of this skill, or shortcuts such as
`make bitacora` / `make release`. This skill is the default behaviour.

If the project has no log yet, do not create it in the middle of another task. Offer to set it
up (`init.sh`) and continue once the user agrees.

## Required flow (one person or one agent)

1. Never on `main`, `master` or `develop`: create the branch first.
2. `f=$($S/new-entry.sh "Short title")`. If it already existed, **update that one**; never add a second.
3. Fill it with the real change: the two required sections (*What changes* and *Why*, or their
   translation), then files/areas, how to test, notes (delete that one if empty).
4. Commit the entry on the same branch as the code.
5. `$S/verify.sh --base <target branch>` must pass before calling the work done or opening the PR.
6. Once the PR exists, set `pr: "#N"` in the front matter.

## Multiple agents

Principle: **one branch = one entry = one owner.**

- **Orchestrator (same branch as its subagents):** the only writer of the entry. Create it before
  dispatching anyone: `f=$(DEVLOG_AGENT="claude:orchestrator" $S/new-entry.sh "…")`.
  Put this line verbatim in every subagent prompt:
  > Do not edit the devlog folder. End your report with a `DEVLOG CONTRIBUTION` block with the lines
  > `agent:`, `what:`, `why:`, `files:`, `test:`, `follow-up:`.

  When the contributions arrive, write **one** coherent entry in the project language (do not paste
  the blocks as they are), add each agent to `agents:`, and run `verify.sh`.
- **Subagent on the same branch:** never touch the devlog folder. Return your `DEVLOG CONTRIBUTION`.
- **Agent on its own branch or worktree:** follow the normal flow on that branch with
  `DEVLOG_AGENT="claude:<role>"`.
- **Reviewer:** read-only. Run `verify.sh --base …` and check that the entry describes the actual
  diff. Ask for fixes; do not make them.
- **Another session on the same branch with no orchestrator:** `git pull` first, edit by section
  (append bullets, never rewrite the file), and merge both sides on conflict.

`new-entry.sh` is atomic: if several agents run it at once, a single file is created and all of
them get the same path.

## Releases

One actor per version. Inside the release process, after tests and build and **before** the
release commit: `$S/close-release.sh X.Y.Z --type … --previous … --notes …`, then add the log
folder to the release commit. Run `--dry-run` first; if no entries would ship, tell the user
before publishing.

## Rules that never bend

- `version:` stays `unreleased`; only the release changes it.
- Never hand-edit `releases/*.md` or the README index table (typos excepted).
- Released entries are not rewritten: if something changes, write a new entry.
- No secrets, tokens, private URLs, or real personal/customer data.
- Concrete and short: "the floor row left the round score negative", not "fixed a bug".

## Done checklist

- [ ] Exactly one entry for the branch, committed with the change.
- [ ] Required sections filled with concrete content, in the project language.
- [ ] `version: unreleased` untouched; agents listed if several took part.
- [ ] `verify.sh` passes.

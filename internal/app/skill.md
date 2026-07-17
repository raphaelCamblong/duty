---
name: duty
description: Drive the duty task system in this repo — markdown task files under a duty/ folder, ordered by BOARD.md indexes, kept in sync by the duty CLI. Use when asked what to work on next, to claim or pick up a task, start or update task status, work a board or track, report progress, or manage tasks and gates. Triggers on a duty/ folder, BOARD.md files, or T-NN task ids.
---

# duty — the agent loop

The repo has a `duty/` folder: a task system of one markdown file per task,
ordered by `BOARD.md` indexes and kept in sync by the `duty` binary. Use it —
never hand-edit the task files or boards.

## The loop

Four calls, repeated until the board is empty:

```
duty get next --claim           # take the next actionable task, marked in-progress
# ...do the work...
duty gates check <id> --all      # tick the task's gates once they pass
duty report <id> --status done   # append what changed and flip to done, one write
```

`get next --claim` hands each agent a distinct task under a lock, so a swarm
just works. Claim with `--as <your-name>` (or set `DUTY_AGENT`) so the board
shows who holds what. Read the whole task before starting:
`duty get task <id> --body`.

## Rules

- Never guess past a blocker. Missing input, a failed dependency, an unmade
  decision → `duty report <id> --status blocked` naming exactly what's missing,
  then stop. Don't invent scope to get unstuck.
- Reports accumulate: every `duty report` appends under `## Report`, never
  overwrites. Say what changed, paste the gate output, note deviations and why.
- Respect `blocked-by` — don't start a task whose dependencies aren't `done`.
- Tick gates honestly: only `check` a gate you actually verified.
- Finish the stated scope. If a task turns out to be two, do the stated one and
  name the split in your report; don't expand it yourself.

## Reading state

- Add `--agent` to any read for stable TSV — parse that, not the human table
  (`duty get tasks --agent`, `duty get task <id> --agent`).
- `--in <track>` targets a board by its path from the tree root (`.` = root),
  from anywhere in the tree.
- Prefer one-shot forms: `duty create task <title> --body` pipes a whole task
  in on stdin; `duty report <id> --status done` reports and flips in one write.
  Fewer calls, no half-written states.

## Flags live in --help

This skill is deliberately short and lists no flags. `duty --help` and
`duty <command> --help` are excellent and authoritative — consult them for
every command's flags, arguments, and exit codes. When unsure, ask the binary.

<!-- duty skill v2 · canonical: https://duty-cli.xyz/skill.md -->

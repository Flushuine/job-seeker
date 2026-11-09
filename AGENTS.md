# Project instructions

## Required project skills

Before starting a task in this repository, read and apply:

- [Ponytail](.agents/skills/ponytail/SKILL.md): use **full** by default for coding, fixes, reviews, and dependency decisions. Make the smallest complete change; reuse existing code before adding abstractions or dependencies.
- [Caveman](.agents/skills/caveman/SKILL.md): use concise, direct chat replies by default. Preserve technical facts, commands, errors, and necessary context. Keep persisted code, comments, documentation, and product text clear and grammatical.

These are project defaults. Follow the user's current instructions when they request another mode, more detail, or disable a skill. Skills guide engineering and communication; they must not alter application matching rules or user-facing content solely to fit a style. Only the two core skills are installed; do not claim missing upstream modes are available.

## Project guardrails

- Keep the application free and local-first. Use optional native Ollama, with one inference request at a time, bounded context, and deterministic fallback. Do not download models without the user's explicit authorization.
- Preserve evidence sources and unknown information. Never invent jobs, company facts, reviews, salary data, or employment arrangements.
- Keep secrets, the CV, and local database files out of tracked changes. Do not print `.env` values unless the user explicitly requests them.
- Respect user-managed development processes. Do not start another persistent API on port 8080 when it is occupied. Stop temporary verification servers when finished.
- Run checks appropriate to the change. Use `make test` for Go tests and `npm --prefix web run build` for frontend changes. Database integration tests require an isolated `TEST_DATABASE_URL`, never the application database. Documentation-only changes need link and content checks, not application tests.

## Approach

- Read existing files before writing. Don't re-read unless changed.
- Thorough in reasoning, concise in output.
- Skip files over 100KB unless required.
- No sycophantic openers or closing fluff.
- No emojis or em-dashes.
- Do not guess APIs, versions, flags, commit SHAs, or package names. Verify by reading code or docs before asserting.
- Always use caveman communication mode (invoke caveman:caveman skill at session start).

---

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

---

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and clarifying questions come before implementation rather than after mistakes.
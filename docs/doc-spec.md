---
title: Documentation Contract
description: The authoring rules every Virtuous doc follows — frontmatter, headings, code blocks, and callouts.
section: Specs
audience: both
status: stable
---

# Documentation Contract

This page is the contract for Virtuous documentation. Every doc in this tree
follows it, including this one. If a rule here is not being followed, the doc is
wrong — not the rule.

The contract is deliberately small. It only requires things that are enforceable
today on GitHub-rendered Markdown and useful to both humans and agents. It does
not mandate tooling (static site generator, build-time JSON index, versioned URL
trees) that the project has not built. Those ideas live in
[Not yet — deferred ideas](#not-yet--deferred-ideas) so the contract never claims
capabilities that do not exist.

## Why this exists

Virtuous treats code, schemas, and clients as a single runtime truth that cannot
drift. Documentation has historically been the exception — easy to let rot. A
short, mechanical contract keeps docs diff-friendly, predictably structured for
agents, and honest about what is stable versus experimental.

> [!NOTE]
> "Agent" below means an LLM coding agent reading these docs to write or migrate
> Virtuous code. Predictable structure and explicit stability signals are what
> let an agent consume docs deterministically.

## Frontmatter (required)

Every doc begins with YAML frontmatter. GitHub renders it as a table, so it is
visible to humans and parseable by agents.

Required fields:

| Field | Meaning |
| --- | --- |
| `title` | Human-readable page title. Matches the H1. |
| `description` | One sentence. What the page is for. |
| `section` | Top-level nav category (see [Sections](#sections)). |
| `audience` | `human`, `agent`, or `both`. |
| `status` | `stable`, `experimental`, or `internal`. |

Optional fields:

| Field | Meaning |
| --- | --- |
| `since` | Version the documented behavior was introduced (e.g. `0.0.55`). |
| `related` | List of related doc paths, relative to `docs/`. |

```yaml
---
title: RPC Router
description: How rpc.NewRouter wires handlers, prefixes, and guards.
section: RPC
audience: both
status: stable
related:
  - rpc/handlers.md
  - rpc/guards.md
---
```

> [!IMPORTANT]
> `status: internal` marks docs that ship in the tree but are not part of the
> public story (design notes, audits, trackers). Keep them under a clearly
> internal path and never link them from human onboarding pages.

## Headings

- Exactly one H1 (`#`) per page. It matches `title`.
- Do not skip heading levels (no H2 → H4).
- Use the canonical section names below when a page needs that kind of content.

## Sections

Use the subset of these sections that the page needs, in this order. Not every
page needs all of them — a reference page may be only `Overview` plus tables; a
concept page may add `Why this exists` and `Guarantees`. The point is that when a
section *is* present, it uses the canonical name and order, so agents can find it.

1. **Overview** — what this is, in two or three sentences.
2. **Why this exists** — the constraint or problem it solves. Optional for pure reference.
3. **How it works** — mechanics, signatures, flow.
4. **Example** — at least one runnable block. See [Code blocks](#code-blocks).
5. **Guarantees** — what callers can rely on. Use a `> [!IMPORTANT]` callout.
6. **Anti-patterns** — what not to do, and why.
7. **Notes for agents** — constraints, footguns, and verification steps specific
   to an agent generating code. Optional but encouraged on RPC/httpapi pages.

## Code blocks

- Every fenced block is language-tagged. No bare ``` fences.
- Go examples must be `gofmt`-clean (tabs, not spaces) and compile against the
  documented API. If a block is intentionally partial, prefix it with a sentence
  that says so ("Illustrative — omits imports").
- After a runnable block, say what success looks like: the URL to hit, the
  expected output, or the file that gets generated.

```go
router := rpc.NewRouter(rpc.WithPrefix("/rpc"))
router.HandleRPC(states.GetByCode)
router.ServeAllDocs()
// Docs now live at /rpc/docs; OpenAPI at /rpc/openapi.json.
```

## Callouts

Use GitHub alert syntax so callouts render natively on github.com. Do not write
ad-hoc `**Note:**` prose.

| Intent | Syntax |
| --- | --- |
| General note | `> [!NOTE]` |
| Recommended path / tip | `> [!TIP]` |
| Guarantee or required behavior | `> [!IMPORTANT]` |
| Footgun / easy to misuse | `> [!WARNING]` |
| Data loss / hard to reverse | `> [!CAUTION]` |

> [!WARNING]
> Map an anti-pattern to `> [!WARNING]`, not `> [!CAUTION]`. Reserve `CAUTION`
> for things that lose data or are hard to undo.

## Sections (canonical nav)

These map directly to folders under `docs/`. No hidden categories.

- Getting Started — `getting-started/`
- Concepts — `concepts/`
- Tutorials — `tutorials/`
- RPC — `rpc/`
- HTTP (httpapi) — `http-legacy/`
- Agents — `agents/`
- Reference — `reference/`
- Examples — `examples/`
- Specs — `specs/` (and this contract)
- Python Loader — `python-loader/`
- Internals — `internals/`

## Guarantees

> [!IMPORTANT]
> A doc that carries valid frontmatter, a single H1, language-tagged code blocks,
> and canonical section names where applicable is contract-compliant. Reviewers
> may reject docs that violate these four rules. Everything else in this page is
> guidance, not a gate.

## Anti-patterns

- Frontmatter that lies about `status` (marking experimental APIs `stable`).
- Untagged code fences, or Go blocks that have not been `gofmt`-ed.
- Inventing callout types outside the [table above](#callouts).
- Linking `internal` docs from onboarding or overview pages.
- Re-documenting the same cookbook in two places. Link to one canonical page.

## Not yet — deferred ideas

These were in the original plan and may return when the project is ready to build
and maintain them. They are explicitly **not** part of the current contract, so no
doc is judged against them:

- A static site generator with clean URLs and search indexing.
- A build-time machine-readable JSON index of all pages for agent discovery.
- Versioned doc trees (`/docs/v0.1`, `/docs/v0.2`) with per-page version
  constraints.

When one of these ships, promote it out of this section and into the body.

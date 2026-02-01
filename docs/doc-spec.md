Virtuous Documentation System Plan
Authoring, Rendering, and Agent-First Strategy

OVERVIEW

Virtuous documentation should be treated as a first-class system, not a collection of pages.

The goals are:

Fast human onboarding

Deterministic agent consumption

Zero ambiguity about contracts and guarantees

Tight coupling between code, specs, and docs

Long-term versionability without rewrites

Documentation should feel like:
A readable specification with executable examples.

AUTHORING FORMAT

Primary authoring format: Markdown
But not “loose” Markdown — disciplined, structured Markdown with strict conventions.

Markdown is chosen because:

Humans already know it

Agents can parse it reliably

Git-native and diff-friendly

Supported by all modern static site generators

Plain Markdown alone is insufficient. Every document must follow a contract.

FRONTMATTER (REQUIRED FOR ALL DOCS)

Each document begins with structured frontmatter metadata.

Required fields:

title: Human-readable page title

description: One-sentence summary

section: Top-level nav category

audience: human, agent, or both

stability: stable | experimental | internal

Optional but recommended:

since: version introduced

related: list of related doc paths

deprecated: true | false

Purpose of frontmatter:

Navigation

Search

Version filtering

Agent discovery

Stability signaling

Docs are data, not prose.

HEADING RULES (STRICT)

Exactly one top-level title per page

Headings must not skip levels

Heading hierarchy must be consistent across the site

Recommended semantic sections:

Overview

Why This Exists

How It Works

Example

Guarantees

Anti-Patterns

Notes for Agents

Agents rely on predictable section names.

CODE BLOCK RULES

Every code block must be language-tagged

No untyped code blocks allowed

Code examples must compile or be clearly labeled as illustrative

Code is not decoration. It is executable documentation.

CALLOUTS AND SEMANTIC BLOCKS

Avoid ad-hoc prose like “Note:” or “Warning:”.

Use semantic callouts that can be rendered consistently:

Warning

Info

Agent

Anti-Pattern

Guarantee

These should be parsed at render time, not improvised in content.

RENDERING STRATEGY

Documentation should be rendered using a static site generator (SSG).

Requirements:

Static output

Fast navigation

Versioning support

Search indexing

Clean URLs

CI-friendly builds

Recommended options:

Astro with Markdown and custom components (preferred)

Docusaurus (excellent if versioning is prioritized early)

Hugo (acceptable but less flexible for agent metadata)

The renderer must not obscure content structure.

DOCS AS DATA (AGENT-FIRST REQUIREMENT)

At build time, documentation should emit:

HTML for humans

A structured JSON index for agents

Each page should be representable as:

Path

Title

Audience

Stability

Section list

Version constraints

This allows:

Agent discovery of capabilities

Deterministic navigation

Filtering unstable or experimental content

Docs should be queryable, not just readable.

VERSIONING STRATEGY

Documentation must be versioned alongside the framework.

Recommended URL structure:

/docs/latest

/docs/v0.1

/docs/v0.2

Docs should not silently change meaning.

Each page should declare:

When it was introduced

Whether it is stable or experimental

Agents must be able to reason about compatibility.

SITE STRUCTURE (CANONICAL)

Top-level documentation sections:

Getting Started

Concepts

Tutorials

RPC

HTTP (Legacy)

Agents

Reference

Examples

Specs

Python Loader

Internals

This structure should map directly to folders on disk.

No hidden magic.

UX PRINCIPLES

For humans:

Minimal navigation depth

Fast search

Copyable code

Clear warnings and guarantees

Strong opinionation

For agents:

Predictable structure

Explicit guarantees

No prose ambiguity

Stable anchors

Machine-readable metadata

WHY THIS FITS VIRTUOUS

This documentation system:

Mirrors Virtuous’ typed, explicit design philosophy

Treats docs as runtime truth

Makes agent support intentional, not accidental

Avoids long-term documentation drift

Scales cleanly as the framework grows

Virtuous documentation should feel like:
A contract you can read and trust.

RECOMMENDED NEXT STEPS

Commit this document as the docs system contract

Lock the frontmatter schema

Choose the static site generator

Draft “Getting Started” using these rules

Build the agent docs index alongside HTML

Once the contract is locked, implementation is mechanical.

END OF DOCUMENT

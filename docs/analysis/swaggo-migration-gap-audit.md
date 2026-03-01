# Swaggo Migration Feedback Audit

Date: 2026-02-27

## Scope

This document captures the classification of user feedback into:

- Knowledge gaps: current behavior exists but is not clearly documented.
- Capability gaps: behavior is not currently supported (or is mismatched) and needs product/design work.

This file is intended as a working draft for follow-up prompts.

## Gap Classification (Q1-Q11)

| Q | Topic | Classification | Notes |
| --- | --- | --- | --- |
| 1 | Non-200 status preservation in `httpapi` migration | Resolved + Knowledge | Supported via `httpapi.HandlerMeta.Responses` / `httpapi.ResponseSpec`; docs need to point migration users to the explicit response metadata path. |
| 2 | Non-JSON routes (`image/png`, `text/html`, files) | Resolved + Knowledge | Typed handlers support `string`, `[]byte`, and custom text/byte media types via `httpapi.HandlerMeta.Responses`; runtime headers remain handler-owned. |
| 3 | Optional request body (`@Param body ... false`) | Resolved + Knowledge | Supported via `httpapi.Optional[Req]()`; docs/examples need to point migration users to this marker. |
| 4 | Mixed body+query with tag conflicts | Knowledge + Constraint | Supported when query/body use different fields. Same field cannot have both `query` and `json` tags by design. Needs explicit modeling guidance in docs. |
| 5 | Two security schemes (OR/AND + generated client semantics) | Capability + Knowledge | Runtime middleware composes all guards; OpenAPI encodes multiple security entries; generated clients currently use only first guard auth parameter. Needs behavior clarification and eventual feature work. |
| 6 | Path/query type fidelity (`int/bool` vs string) | Capability + Knowledge | OpenAPI and generated clients model path/query values as strings during migration path. Query string behavior is partially documented; path typing behavior is not explicit. |
| 7 | Handler factory methods (`func(...) http.HandlerFunc`) | Knowledge | Already supported through `WrapFunc` / `Wrap` because factory output is still a standard handler function. Needs explicit migration example. |
| 8 | Annotation vs router drift: source of truth | Knowledge | Runtime router registration is source of truth. Migration docs should explicitly define conflict policy. |
| 9 | Trailing slash normalization/preservation | Knowledge | Current docs do not define slash policy clearly for migration. Needs explicit guidance and examples. |
| 10 | Stale pinned version (`0.0.21` vs `VERSION`) | Knowledge | Prompt templates include stale literals and should rely on `VERSION` only. |
| 11 | RPC status model inconsistency (401 vs 200/422/500) | Knowledge | Must distinguish handler-return statuses (200/422/500) from guard-driven OpenAPI 401 response documentation. |

## Knowledge Gap Backlog (Documentation and Guidance Only)

- [x] Migration guide clarity
  Files updated: `docs/tutorials/migrate-swaggo.md`
  Notes: Added capability matrix, non-JSON lane, optional-body callout, mixed query/body guidance, source-of-truth policy, and trailing-slash guidance.
- [x] Security semantics clarification
  Files updated: `docs/tutorials/migrate-swaggo.md`, `docs/http-legacy/overview.md`, `docs/rpc/guards.md`
  Notes: Documented runtime middleware composition, OpenAPI security representation, and current generated-client auth limitation.
- [x] Status model unification
  Files updated: `README.md`, `docs/overview.md`, `docs/rpc/handlers.md`, `docs/internals/openapi.md`, `docs/agent_quickstart.md`
  Notes: Unified wording to handler-return statuses `200/422/500`, with guard-driven `401` clarified separately.
- [x] Version-template hygiene
  Files updated: `docs/overview.md`, `docs/tutorials/migrate-swaggo.md`, `docs/agents/overview.md`
  Notes: Removed stale pinned literals and standardized on "read `VERSION`".
- [x] Examples for advanced migrations
  Files updated: `docs/http-legacy/typed-handlers.md`, `docs/http-legacy/query-params.md`, `docs/tutorials/migrate-swaggo.md`, `docs/http-legacy/overview.md`
  Notes: Added factory handler, non-JSON side-by-side, and mixed query/body examples.
- [x] Documentation reference hygiene
  Files updated: `docs/specs/overview.md`, `docs/reference/public-api.md`, `AGENTS.md`
  Notes: Corrected stale spec-path references and aligned agent guidance references.

## Capability Gaps Ranked by Impact

### High / Big Impact

1. Dual/multi-security scheme behavior consistency across runtime, OpenAPI, and generated clients (Q5)

Reasoning: these directly affect large Swaggo migration cohorts and can block correctness at contract level.

### Medium Impact

2. Path/query typed fidelity beyond "string" in OpenAPI/client generation (Q6)

Reasoning: important for contract precision and strict consumers, but many teams can migrate with documented constraints.

### Low Impact

3. (No additional pure capability gaps beyond Q5/Q6 from this feedback set)

Reasoning: remaining items are primarily documentation/policy clarity problems rather than missing runtime/product capability.

## Working Notes

- This audit intentionally avoids proposing implementation details for capability gaps.
- Next iteration can convert "Knowledge Gap Backlog" into concrete doc patch tasks per file/section.

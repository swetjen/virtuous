---
title: RPC Overview
description: "The canonical RPC API style: inferred paths, POST transport, and typed handlers."
section: RPC
audience: both
status: stable
related:
  - rpc/handlers.md
  - rpc/router.md
  - rpc/guards.md
---

# RPC overview

## Overview

RPC is the canonical API style in Virtuous. Paths are derived from package and function names and all RPC calls use HTTP POST.

## Key properties

- Typed handlers reflected into OpenAPI and clients.
- Route paths inferred at registration time.
- Status codes limited to 200, 422, and 500.
- Guards define auth metadata and middleware.

## Start here

- `handlers.md` for signature rules.
- `router.md` for registration and path inference.
- `guards.md` for auth metadata.
- `docs-and-clients.md` for runtime outputs.

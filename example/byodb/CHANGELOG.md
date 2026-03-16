# Changelog

## 2026-03-15
- Add role-aware frontend routing with centralized route declarations in `frontend-web/src/App.tsx`.
- Add guest/user/admin layout + nav components and route guards for signed-in/admin-only pages.
- Add home/login/register/confirm/dashboard/admin user-management pages in the frontend template.
- Add backend RPC auth flows for register/confirm/login/me under `handlers/users`.
- Extend user schema and sqlc queries for password hash + confirmation state.
- Wire frontend auth context to backend auth routes with local fallback behavior.
- Harden auth with bcrypt password hashes and signed JWT session tokens.
- Replace static admin bearer middleware with JWT role-based session guard middleware.
- Refactor auth to a router-instanced `middleware.Auth` service injected through deps for guards + token/password ops.
- Re-check user/role from DB on each guarded request so role revocation is enforced immediately.
- Add `AUTH_TOKEN_TTL_SECONDS` config to control JWT lifetime without code changes.
- Set default JWT lifetime to 5 minutes (`300` seconds) for tighter default session hardening.
- Add `make start` and align dev runner with frontend-aware watch/build flow from `cf/user`.
- Add admin `UserDisable` flow so user detail pages can disable accounts.
- Enforce disabled-account checks in login and session guard auth validation.
- Update admin user creation to accept optional custom password and auto-generate a temporary password when omitted.
- Remove fallback explainer block text from admin users UI.
- Remove signed-in nav/routes for API Keys, Endpoints, Rules, and Traffic to keep a clean starter console surface.
- Remove Teams/Console links from admin nav and add a Dashboard button back to signed-in user view.
- Fix SPA deep-link refresh by emitting root-relative frontend assets and tightening static fallback behavior.
- Fix SPA fallback redirect behavior so refreshing deep links like `/login` does not bounce to `/`.
- Replace signed-in dashboard live-states panel with a simple “Protected User View” starter tile.
- Adjust signed-in/admin nav layout: dashboard links on the left, admin/dashboard actions on the right near sign-out, and remove the `Virtuous Admin` heading label.
- Pre-populate admin create-user password on the frontend and require it so admins can immediately share credentials.
- Remove static local-fallback credential hint text from sign-in UI.
- Seed fixture users on first boot (when no users exist) with random 12-character passwords and log credentials to console.
- Clear default sign-in form credentials so login uses emitted runtime fixture credentials.
- Move fixture seeding/password generation into `db` package helpers and keep router focused on dependency wiring + route mounting.
- Codify frontend route ownership + components/pages structure in `AGENTS.md`.

## 2026-02-01
- Add `make agent-run` to capture run output in `ERRORS`.
- Log router readiness with a clear slog message.
- Load defaults from `.env` and watch it for reloads.

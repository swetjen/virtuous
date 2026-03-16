import { Link } from "react-router-dom";

export function HomePage() {
  return (
    <div className="stack-lg">
      <section className="hero">
        <p className="kicker">Virtuous Byodb</p>
        <h1>Build typed APIs, then ship product UI on top.</h1>
        <p className="hero-copy">
          This starter demonstrates user/admin roles, a declarative route map, and generated API clients.
          It is intentionally easy to rebrand for production teams.
        </p>
        <div className="button-row">
          <Link className="btn" to="/login">
            Sign In
          </Link>
          <Link className="btn secondary" to="/register">
            Create Account
          </Link>
          <a className="btn ghost" href="/rpc/docs/">
            API Docs
          </a>
        </div>
      </section>

      <section className="grid-3">
        <article className="panel">
          <h3>Role-aware navigation</h3>
          <p>Guest, user, and admin layouts with route-level guards and explicit ownership in App.tsx.</p>
        </article>
        <article className="panel">
          <h3>Admin user management</h3>
          <p>Admins can list users, inspect details, and create users from a single control surface.</p>
        </article>
        <article className="panel">
          <h3>Generated client first</h3>
          <p>State and admin pages use the generated Virtuous client so contracts stay in sync.</p>
        </article>
      </section>
    </div>
  );
}

import { useState, type FormEvent } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";

import { useUser } from "../context/UserContext";

type LocationState = {
  from?: string;
};

export function LoginPage() {
  const { signIn, signOutReason } = useUser();
  const navigate = useNavigate();
  const location = useLocation();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError("");

    const result = await signIn({ email, password });
    setSubmitting(false);

    if (!result.ok) {
      setError(result.error);
      return;
    }

    const state = location.state as LocationState;
    if (state?.from) {
      navigate(state.from);
      return;
    }

    if (result.user.role === "admin") {
      navigate("/admin/user");
      return;
    }

    navigate("/console/getting-started");
  }

  return (
    <section className="auth-card">
      <h2>Sign In</h2>
      <p>Use one of the seeded demo users or create a new account.</p>

      {signOutReason && <div className="alert">{signOutReason}</div>}
      {error && <div className="alert">{error}</div>}

      <form onSubmit={onSubmit} className="stack-sm">
        <label className="field-label">
          Email
          <input value={email} onChange={(event) => setEmail(event.target.value)} required />
        </label>
        <label className="field-label">
          Password
          <input
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            required
          />
        </label>
        <button className="btn" type="submit" disabled={submitting}>
          {submitting ? "Signing in..." : "Sign In"}
        </button>
      </form>

      <p className="inline-links">
        Need an account? <Link to="/register">Sign up</Link>
      </p>
    </section>
  );
}

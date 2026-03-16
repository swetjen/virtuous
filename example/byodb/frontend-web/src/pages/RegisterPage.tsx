import { useState, type FormEvent } from "react";
import { Link } from "react-router-dom";

import { useUser } from "../context/UserContext";

export function RegisterPage() {
  const { signUp } = useUser();
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [confirmationCode, setConfirmationCode] = useState("");

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setConfirmationCode("");

    const result = await signUp({
      email,
      name,
      password,
    });

    if (!result.ok) {
      setError(result.error);
      return;
    }

    setConfirmationCode(result.code);
    setEmail("");
    setName("");
    setPassword("");
  }

  return (
    <section className="auth-card">
      <h2>Create Account</h2>
      <p>This template uses a simple confirmation flow so teams can swap in real auth later.</p>

      {error && <div className="alert">{error}</div>}
      {confirmationCode && (
        <div className="success-block">
          <strong>Confirmation ready:</strong> {confirmationCode}
          <p>
            Continue to <Link to={`/confirm/${confirmationCode}`}>confirm account</Link>.
          </p>
        </div>
      )}

      <form onSubmit={onSubmit} className="stack-sm">
        <label className="field-label">
          Name
          <input value={name} onChange={(event) => setName(event.target.value)} required />
        </label>
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
            minLength={6}
            required
          />
        </label>
        <button className="btn" type="submit">
          Register
        </button>
      </form>

      <p className="inline-links">
        Already registered? <Link to="/login">Sign in</Link>
      </p>
    </section>
  );
}

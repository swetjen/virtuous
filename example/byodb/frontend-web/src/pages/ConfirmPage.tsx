import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

import { useUser } from "../context/UserContext";

export function ConfirmPage() {
  const { code } = useParams();
  const { confirmCode } = useUser();
  const navigate = useNavigate();
  const [error, setError] = useState("");
  const [confirmed, setConfirmed] = useState(false);

  async function onConfirm() {
    if (!code) {
      setError("Missing confirmation code.");
      return;
    }
    const result = await confirmCode(code);
    if (!result.ok) {
      setError(result.error);
      return;
    }
      setConfirmed(true);
      setTimeout(() => navigate("/login"), 900);
  }

  return (
    <section className="auth-card">
      <h2>Confirm Account</h2>
      <p>Code: {code ?? "unknown"}</p>

      {error && <div className="alert">{error}</div>}
      {confirmed && <div className="success-block">Confirmed. Redirecting to sign-in...</div>}

      <button className="btn" type="button" onClick={onConfirm}>
        Confirm and continue
      </button>

      <p className="inline-links">
        Wrong code? <Link to="/register">Create a new registration</Link>
      </p>
    </section>
  );
}

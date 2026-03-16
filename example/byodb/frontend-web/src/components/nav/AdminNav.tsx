import { NavLink } from "react-router-dom";

import { useUser } from "../../context/UserContext";

export function AdminNav() {
  const { signOut, user } = useUser();

  return (
    <header className="top-nav top-nav-admin">
      <nav className="nav-links">
        <NavLink to="/admin/user">Users</NavLink>
      </nav>
      <div className="nav-actions">
        <NavLink className="btn secondary" to="/console/getting-started">
          Dashboard
        </NavLink>
        <span className="pill pill-admin">{user?.email ?? "admin"}</span>
        <button className="btn ghost" onClick={signOut} type="button">
          Sign Out
        </button>
      </div>
    </header>
  );
}

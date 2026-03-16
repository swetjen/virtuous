import { NavLink } from "react-router-dom";

import { useUser } from "../../context/UserContext";

export function UserNav() {
  const { signOut, user, isSuperUser } = useUser();

  return (
    <header className="top-nav">
      <nav className="nav-links">
        <NavLink to="/console/getting-started">Dashboard</NavLink>
      </nav>
      <div className="nav-actions">
        {isSuperUser() && (
          <NavLink className="btn secondary" to="/admin/user">
            Admin
          </NavLink>
        )}
        <span className="pill">{user?.name ?? "Unknown"}</span>
        <button className="btn ghost" onClick={signOut} type="button">
          Sign Out
        </button>
      </div>
    </header>
  );
}

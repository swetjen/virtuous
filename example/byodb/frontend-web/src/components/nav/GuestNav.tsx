import { Link, NavLink } from "react-router-dom";

export function GuestNav() {
  return (
    <header className="top-nav">
      <Link className="brand" to="/">
        Virtuous
      </Link>
      <nav className="nav-links">
        <NavLink to="/">Home</NavLink>
        <NavLink to="/login">Sign In</NavLink>
        <NavLink to="/register">Sign Up</NavLink>
      </nav>
    </header>
  );
}

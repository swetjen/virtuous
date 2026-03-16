import type { ReactNode } from "react";
import { Link } from "react-router-dom";

export function BasicGuestNav({ children }: { children: ReactNode }) {
  return (
    <div className="app-shell app-shell-compact">
      <header className="top-nav top-nav-compact">
        <Link className="brand" to="/">
          Virtuous
        </Link>
        <Link className="text-link" to="/">
          Back to home
        </Link>
      </header>
      <main className="page-content">{children}</main>
    </div>
  );
}

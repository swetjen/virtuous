import type { ReactNode } from "react";

import { GuestNav } from "../nav/GuestNav";

export function GuestLayout({ children }: { children: ReactNode }) {
  return (
    <div className="app-shell">
      <GuestNav />
      <main className="page-content">{children}</main>
    </div>
  );
}

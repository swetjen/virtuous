import type { ReactNode } from "react";

import { AdminNav } from "../nav/AdminNav";

export function AdminLayout({ children }: { children: ReactNode }) {
  return (
    <div className="app-shell">
      <AdminNav />
      <main className="page-content">{children}</main>
    </div>
  );
}

import type { ReactNode } from "react";

import { UserNav } from "../nav/UserNav";

export function SignedInLayout({ children }: { children: ReactNode }) {
  return (
    <div className="app-shell">
      <UserNav />
      <main className="page-content">{children}</main>
    </div>
  );
}

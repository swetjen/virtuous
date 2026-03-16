import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

import {
  clearSession,
  confirmPendingUser,
  createUserLocally,
  getSession,
  registerPendingUser,
  setSession,
  signInWithCreds,
} from "../lib/authStore";
import { api } from "../lib/api";
import type { LoginCredsRequest, RegisterRequest, SessionUser, UserRole } from "../types/auth";

type UserRPC = {
  id: number;
  email: string;
  name: string;
  role: string;
  is_superuser?: boolean;
};

type LoginRPCResponse = {
  user?: UserRPC;
  token?: string;
  error?: string;
};

type RegisterRPCResponse = {
  confirmation_code?: string;
  error?: string;
};

type ConfirmRPCResponse = {
  user?: UserRPC;
  error?: string;
};

type MeRPCResponse = {
  user?: UserRPC;
  error?: string;
};

type UserContextValue = {
  user: SessionUser | null;
  checkedSignIn: boolean;
  signOutReason: string;
  signIn: (req: LoginCredsRequest) => Promise<{ ok: true; user: SessionUser } | { ok: false; error: string }>;
  signUp: (req: RegisterRequest) => Promise<{ ok: true; code: string } | { ok: false; error: string }>;
  confirmCode: (code: string) => Promise<{ ok: true } | { ok: false; error: string }>;
  signOut: () => void;
  signOutSessionExpired: () => void;
  refreshUser: () => void;
  isSignedIn: () => boolean;
  isSuperUser: () => boolean;
  createLocalUser: (req: { email: string; name: string; role: UserRole; password?: string }) => {
    id: number;
    email: string;
    name: string;
    role: UserRole;
    temporaryPassword?: string;
  };
};

const UserContext = createContext<UserContextValue | null>(null);

function toSessionUser(user: UserRPC, token: string): SessionUser {
  const role = user.is_superuser || user.role === "admin" ? "admin" : "user";
  return {
    id: user.id,
    email: user.email,
    name: user.name,
    role,
    sessionToken: token,
    authToken: token,
  };
}

export function UserProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<SessionUser | null>(null);
  const [checkedSignIn, setCheckedSignIn] = useState(false);
  const [signOutReason, setSignOutReason] = useState("");

  const refreshUser = useCallback(() => {
    void (async () => {
      const fromStore = getSession();
      if (!fromStore) {
        setUser(null);
        setCheckedSignIn(true);
        return;
      }

      if (!fromStore.sessionToken || fromStore.sessionToken.startsWith("local-")) {
        setUser(fromStore);
        setCheckedSignIn(true);
        return;
      }

      try {
        const response = (await api.users.UserMe({ auth: fromStore.sessionToken })) as MeRPCResponse;
        if (!response.user || response.error) {
          throw new Error(response.error ?? "session invalid");
        }
        const fresh = toSessionUser(response.user, fromStore.sessionToken);
        setSession(fresh);
        setUser(fresh);
      } catch {
        clearSession();
        setUser(null);
      } finally {
        setCheckedSignIn(true);
      }
    })();
  }, []);

  useEffect(() => {
    refreshUser();
  }, [refreshUser]);

  const signIn = useCallback(async (req: LoginCredsRequest) => {
    try {
      const response = (await api.users.UserLogin(req)) as LoginRPCResponse;
      if (!response.user || !response.token || response.error) {
        throw new Error(response.error ?? "invalid credentials");
      }
      const session = toSessionUser(response.user, response.token);
      setSession(session);
      setSignOutReason("");
      setUser(session);
      return { ok: true as const, user: session };
    } catch {
      const fallback = signInWithCreds(req.email, req.password);
      if (!fallback) {
        return { ok: false as const, error: "Invalid email or password." };
      }
      setSignOutReason("");
      setUser(fallback);
      return { ok: true as const, user: fallback };
    }
  }, []);

  const signUp = useCallback(async (req: RegisterRequest) => {
    try {
      const response = (await api.users.UserRegister(req)) as RegisterRPCResponse;
      if (response.error) {
        return { ok: false as const, error: response.error };
      }
      return { ok: true as const, code: response.confirmation_code ?? "CHECK_EMAIL" };
    } catch {
      const fallback = registerPendingUser(req);
      if (!fallback.ok) {
        return fallback;
      }
      return { ok: true as const, code: fallback.code };
    }
  }, []);

  const confirmCode = useCallback(async (code: string) => {
    try {
      const response = (await api.users.UserConfirm({ code })) as ConfirmRPCResponse;
      if (response.error) {
        return { ok: false as const, error: response.error };
      }
      return { ok: true as const };
    } catch {
      const fallback = confirmPendingUser(code);
      if (!fallback.ok) {
        return fallback;
      }
      setUser(fallback.user);
      setSignOutReason("");
      return { ok: true as const };
    }
  }, []);

  const signOut = useCallback(() => {
    clearSession();
    setUser(null);
  }, []);

  const signOutSessionExpired = useCallback(() => {
    setSignOutReason("Session expired");
    signOut();
  }, [signOut]);

  const isSignedIn = useCallback(() => {
    return !!user && user.id > 0;
  }, [user]);

  const isSuperUser = useCallback(() => {
    return !!user && user.role === "admin";
  }, [user]);

  const createLocalUser = useCallback((req: { email: string; name: string; role: UserRole; password?: string }) => {
    return createUserLocally(req);
  }, []);

  const value = useMemo<UserContextValue>(
    () => ({
      user,
      checkedSignIn,
      signOutReason,
      signIn,
      signUp,
      confirmCode,
      signOut,
      signOutSessionExpired,
      refreshUser,
      isSignedIn,
      isSuperUser,
      createLocalUser,
    }),
    [
      user,
      checkedSignIn,
      signOutReason,
      signIn,
      signUp,
      confirmCode,
      signOut,
      signOutSessionExpired,
      refreshUser,
      isSignedIn,
      isSuperUser,
      createLocalUser,
    ],
  );

  return <UserContext.Provider value={value}>{children}</UserContext.Provider>;
}

export function useUser(): UserContextValue {
  const context = useContext(UserContext);
  if (!context) {
    throw new Error("useUser must be used within UserProvider");
  }
  return context;
}

import type { SessionUser, UserRole } from "../types/auth";

type StoredUser = {
  id: number;
  email: string;
  name: string;
  role: UserRole;
  disabled: boolean;
  password: string;
  confirmedAt: string;
};

type PendingRegistration = {
  code: string;
  email: string;
  name: string;
  password: string;
  role: UserRole;
  createdAt: string;
};

const USERS_KEY = "byodb.users.v1";
const PENDING_KEY = "byodb.pending.v1";
const SESSION_KEY = "byodb.session.v1";

const seededUsers: StoredUser[] = [
  {
    id: 1,
    email: "admin@virtuous.dev",
    name: "Virtuous Admin",
    role: "admin",
    disabled: false,
    password: "admin123",
    confirmedAt: new Date().toISOString(),
  },
  {
    id: 2,
    email: "user@virtuous.dev",
    name: "Virtuous User",
    role: "user",
    disabled: false,
    password: "user123",
    confirmedAt: new Date().toISOString(),
  },
];

function parseJSON<T>(value: string | null, fallback: T): T {
  if (!value) {
    return fallback;
  }
  try {
    return JSON.parse(value) as T;
  } catch {
    return fallback;
  }
}

function nextID(users: StoredUser[]): number {
  return users.reduce((max, current) => Math.max(max, current.id), 0) + 1;
}

function normalizeEmail(value: string): string {
  return value.trim().toLowerCase();
}

function toSessionUser(user: StoredUser): SessionUser {
  return {
    id: user.id,
    email: user.email,
    name: user.name,
    role: user.role,
    sessionToken: "",
    authToken: "",
  };
}

function writeUsers(users: StoredUser[]): void {
  localStorage.setItem(USERS_KEY, JSON.stringify(users));
}

export function readUsers(): StoredUser[] {
  const users = parseJSON<StoredUser[]>(localStorage.getItem(USERS_KEY), []).map((user) => ({
    ...user,
    disabled: !!user.disabled,
  }));
  if (users.length > 0) {
    writeUsers(users);
    return users;
  }
  writeUsers(seededUsers);
  return [...seededUsers];
}

function readPending(): PendingRegistration[] {
  return parseJSON<PendingRegistration[]>(localStorage.getItem(PENDING_KEY), []);
}

function writePending(items: PendingRegistration[]): void {
  localStorage.setItem(PENDING_KEY, JSON.stringify(items));
}

function randomLocalPassword(length = 14): string {
  const safeLength = length < 10 ? 10 : length;
  const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%^&*";
  const out: string[] = [];
  if (typeof window !== "undefined" && window.crypto?.getRandomValues) {
    const bytes = new Uint8Array(safeLength);
    window.crypto.getRandomValues(bytes);
    bytes.forEach((value) => {
      out.push(alphabet.charAt(value % alphabet.length));
    });
    return out.join("");
  }
  for (let index = 0; index < safeLength; index += 1) {
    out.push(alphabet.charAt(Math.floor(Math.random() * alphabet.length)));
  }
  return out.join("");
}

export function listUsersForUI(): Array<Omit<StoredUser, "password" | "confirmedAt">> {
  return readUsers().map((user) => ({
    id: user.id,
    email: user.email,
    name: user.name,
    role: user.role,
    disabled: user.disabled,
  }));
}

export function findUserByID(id: number): Omit<StoredUser, "password" | "confirmedAt"> | null {
  const user = readUsers().find((item) => item.id === id);
  if (!user) {
    return null;
  }
  return {
    id: user.id,
    email: user.email,
    name: user.name,
    role: user.role,
    disabled: user.disabled,
  };
}

export function signInWithCreds(email: string, password: string): SessionUser | null {
  const normalizedEmail = normalizeEmail(email);
  const user = readUsers().find(
    (item) => normalizeEmail(item.email) === normalizedEmail && item.password === password,
  );
  if (!user) {
    return null;
  }
  if (user.disabled) {
    return null;
  }
  const token = `local-${user.id}-${Date.now()}`;
  const session = {
    ...toSessionUser(user),
    sessionToken: token,
    authToken: token,
  };
  setSession(session);
  return session;
}

export function setSession(session: SessionUser): void {
  localStorage.setItem(SESSION_KEY, JSON.stringify(session));
}

export function getSession(): SessionUser | null {
  const session = parseJSON<SessionUser | null>(localStorage.getItem(SESSION_KEY), null);
  if (!session) {
    return null;
  }
  if (typeof session.sessionToken !== "string") {
    return { ...session, sessionToken: "" };
  }
  return session;
}

export function clearSession(): void {
  localStorage.removeItem(SESSION_KEY);
}

export function registerPendingUser(input: {
  email: string;
  name: string;
  password: string;
  role?: UserRole;
}): { ok: true; code: string } | { ok: false; error: string } {
  const email = normalizeEmail(input.email);
  const users = readUsers();
  const pending = readPending();

  if (users.some((item) => normalizeEmail(item.email) === email)) {
    return { ok: false, error: "Email is already registered." };
  }
  if (pending.some((item) => normalizeEmail(item.email) === email)) {
    return { ok: false, error: "A confirmation is already pending for this email." };
  }

  const code = Math.random().toString(36).slice(2, 8).toUpperCase();
  pending.push({
    code,
    email,
    name: input.name.trim(),
    password: input.password,
    role: input.role ?? "user",
    createdAt: new Date().toISOString(),
  });
  writePending(pending);
  return { ok: true, code };
}

export function confirmPendingUser(code: string): { ok: true; user: SessionUser } | { ok: false; error: string } {
  const cleanCode = code.trim().toUpperCase();
  const pending = readPending();
  const match = pending.find((item) => item.code === cleanCode);
  if (!match) {
    return { ok: false, error: "Confirmation code not found." };
  }

  const users = readUsers();
  const created: StoredUser = {
    id: nextID(users),
    email: match.email,
    name: match.name,
    role: match.role,
    disabled: false,
    password: match.password,
    confirmedAt: new Date().toISOString(),
  };

  users.push(created);
  writeUsers(users);
  writePending(pending.filter((item) => item.code !== cleanCode));

  const token = `local-${created.id}-${Date.now()}`;
  const session = {
    ...toSessionUser(created),
    sessionToken: token,
    authToken: token,
  };
  setSession(session);
  return { ok: true, user: session };
}

export function createUserLocally(input: {
  email: string;
  name: string;
  role: UserRole;
  password?: string;
}): { id: number; email: string; name: string; role: UserRole; temporaryPassword?: string } {
  const users = readUsers();
  const suppliedPassword = input.password?.trim() ?? "";
  const temporaryPassword = suppliedPassword.length > 0 ? undefined : randomLocalPassword();
  const created: StoredUser = {
    id: nextID(users),
    email: normalizeEmail(input.email),
    name: input.name.trim(),
    role: input.role,
    disabled: false,
    password: suppliedPassword || temporaryPassword || randomLocalPassword(),
    confirmedAt: new Date().toISOString(),
  };
  users.push(created);
  writeUsers(users);
  return {
    id: created.id,
    email: created.email,
    name: created.name,
    role: created.role,
    temporaryPassword,
  };
}

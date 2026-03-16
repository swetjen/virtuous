export type UserRole = "admin" | "user";

export interface SessionUser {
  id: number;
  email: string;
  name: string;
  role: UserRole;
  sessionToken: string;
  authToken: string;
}

export interface LoginCredsRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  name: string;
  password: string;
}

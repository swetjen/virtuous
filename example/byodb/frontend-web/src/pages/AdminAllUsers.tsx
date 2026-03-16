import { useEffect, useState, type FormEvent } from "react";
import { Link } from "react-router-dom";

import { useUser } from "../context/UserContext";
import { listUsersForUI } from "../lib/authStore";
import { api } from "../lib/api";
import type { UserRole } from "../types/auth";

type AdminUser = {
  id: number;
  email: string;
  name: string;
  role: string;
  disabled: boolean;
};

type AdminUsersResponse = {
  data?: AdminUser[];
  error?: string;
};

type AdminUserResponse = {
  user?: AdminUser;
  temporary_password?: string;
  error?: string;
};

function generatePassword(length = 14): string {
  const safeLength = length < 10 ? 10 : length;
  const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%^&*";
  const chars: string[] = [];
  if (typeof window !== "undefined" && window.crypto?.getRandomValues) {
    const bytes = new Uint8Array(safeLength);
    window.crypto.getRandomValues(bytes);
    bytes.forEach((value) => {
      chars.push(alphabet.charAt(value % alphabet.length));
    });
    return chars.join("");
  }
  for (let index = 0; index < safeLength; index += 1) {
    chars.push(alphabet.charAt(Math.floor(Math.random() * alphabet.length)));
  }
  return chars.join("");
}

export function AdminAllUsers() {
  const { user, createLocalUser } = useUser();
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [error, setError] = useState("");
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [role, setRole] = useState<UserRole>("user");
  const [password, setPassword] = useState(() => generatePassword());
  const [tempPassword, setTempPassword] = useState("");

  useEffect(() => {
    void loadUsers();
  }, []);

  async function loadUsers() {
    setError("");
    try {
      const response = (await api.admin.UsersGetMany({ auth: user?.authToken })) as AdminUsersResponse;
      if (response.error) {
        throw new Error(response.error);
      }
      setUsers(response.data ?? []);
    } catch {
      const localUsers = listUsersForUI().map((item) => ({ ...item, role: item.role }));
      setUsers(localUsers);
    }
  }

  async function onCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setTempPassword("");
    const passwordToShare = password.trim();
    if (!passwordToShare) {
      setError("Password is required.");
      return;
    }

    try {
      const response = (await api.admin.UserCreate(
        {
          email: email.trim(),
          name: name.trim(),
          role,
          password: passwordToShare,
        },
        { auth: user?.authToken },
      )) as AdminUserResponse;

      if (response.error) {
        throw new Error(response.error);
      }

      await loadUsers();
      setEmail("");
      setName("");
      setRole("user");
      setPassword(generatePassword());
      setTempPassword(passwordToShare);
    } catch {
      const created = createLocalUser({ email, name, role, password: passwordToShare });
      await loadUsers();
      setEmail("");
      setName("");
      setRole("user");
      setPassword(generatePassword());
      setTempPassword(created.temporaryPassword ?? passwordToShare);
    }
  }

  return (
    <div className="stack-lg">
      <section className="panel">
        <div className="panel-header">
          <h2>Users</h2>
          <button className="btn secondary" type="button" onClick={() => void loadUsers()}>
            Refresh
          </button>
        </div>

        {error && <div className="alert">{error}</div>}

        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Email</th>
                <th>Name</th>
                <th>Role</th>
                <th>Status</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {users.length === 0 && (
                <tr>
                  <td colSpan={6}>No users found.</td>
                </tr>
              )}
              {users.map((item) => (
                <tr key={item.id}>
                  <td>{item.id}</td>
                  <td>{item.email}</td>
                  <td>{item.name}</td>
                  <td>{item.role}</td>
                  <td>{item.disabled ? "disabled" : "active"}</td>
                  <td>
                    <Link className="text-link" to={`/admin/user/${item.id}`}>
                      Detail
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section className="panel">
        <h3>Create User</h3>
        <p className="small-note">New users are confirmed automatically. A password is pre-generated below; share it with the user after creation.</p>
        {tempPassword && (
          <div className="success-block">
            <strong>Password to share with user</strong>
            <div>
              <code>{tempPassword}</code>
            </div>
          </div>
        )}
        <form className="stack-sm" onSubmit={onCreate}>
          <label className="field-label">
            Email
            <input value={email} onChange={(event) => setEmail(event.target.value)} required />
          </label>
          <label className="field-label">
            Name
            <input value={name} onChange={(event) => setName(event.target.value)} required />
          </label>
          <label className="field-label">
            Role
            <select value={role} onChange={(event) => setRole(event.target.value as UserRole)}>
              <option value="user">user</option>
              <option value="admin">admin</option>
            </select>
          </label>
          <label className="field-label">
            Password
            <input
              type="text"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              required
            />
          </label>
          <button className="btn" type="submit">
            Create
          </button>
        </form>
      </section>
    </div>
  );
}

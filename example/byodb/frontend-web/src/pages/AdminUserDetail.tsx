import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { useUser } from "../context/UserContext";
import { findUserByID } from "../lib/authStore";
import { api } from "../lib/api";

type AdminUser = {
  id: number;
  email: string;
  name: string;
  role: string;
  disabled: boolean;
};

type AdminUserResponse = {
  user?: AdminUser;
  error?: string;
};

export function AdminUserDetail() {
  const { id } = useParams();
  const { user } = useUser();
  const [record, setRecord] = useState<AdminUser | null>(null);
  const [error, setError] = useState("");
  const [actionError, setActionError] = useState("");
  const [disabling, setDisabling] = useState(false);

  useEffect(() => {
    void loadUser();
  }, [id]);

  async function loadUser() {
    setError("");
    setActionError("");
    if (!id) {
      setError("Missing user id.");
      return;
    }

    try {
      const response = (await api.admin.UserByID({ id }, { auth: user?.authToken })) as AdminUserResponse;
      if (response.error || !response.user) {
        throw new Error(response.error ?? "Not found");
      }
      setRecord(response.user);
    } catch {
      const local = findUserByID(Number(id));
      if (!local) {
        setError("User not found.");
        setRecord(null);
        return;
      }
      setRecord({ ...local, disabled: !!local.disabled });
    }
  }

  async function onDisableUser() {
    setActionError("");
    if (!id || !record) {
      setActionError("Missing user id.");
      return;
    }
    if (record.disabled) {
      return;
    }

    setDisabling(true);
    try {
      const response = (await api.admin.UserDisable({ id }, { auth: user?.authToken })) as AdminUserResponse;
      if (response.error || !response.user) {
        throw new Error(response.error ?? "Failed to disable user");
      }
      setRecord(response.user);
    } catch {
      setActionError("Unable to disable user from this environment.");
    } finally {
      setDisabling(false);
    }
  }

  return (
    <section className="panel">
      <div className="panel-header">
        <h2>User Detail</h2>
        <Link className="text-link" to="/admin/user">
          Back to users
        </Link>
      </div>

      {error && <div className="alert">{error}</div>}
      {!error && actionError && <div className="alert">{actionError}</div>}
      {!error && !record && <div className="loading-state">Loading user...</div>}

      {record && (
        <dl className="detail-grid">
          <div>
            <dt>ID</dt>
            <dd>{record.id}</dd>
          </div>
          <div>
            <dt>Email</dt>
            <dd>{record.email}</dd>
          </div>
          <div>
            <dt>Name</dt>
            <dd>{record.name}</dd>
          </div>
          <div>
            <dt>Role</dt>
            <dd>{record.role}</dd>
          </div>
          <div>
            <dt>Status</dt>
            <dd>{record.disabled ? "disabled" : "active"}</dd>
          </div>
        </dl>
      )}

      {record && (
        <div>
          <button className="btn ghost" type="button" onClick={() => void onDisableUser()} disabled={disabling || record.disabled}>
            {record.disabled ? "Account disabled" : disabling ? "Disabling..." : "Disable account"}
          </button>
        </div>
      )}
    </section>
  );
}

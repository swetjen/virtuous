import { useEffect, useMemo, useState } from "react";
import "./index.css";
import { createClient } from "../api/client.gen.js";

type State = {
  id: number;
  code: string;
  name: string;
};

type StatesResponse = {
  data?: State[];
  error?: string;
};

type StateResponse = {
  state?: State;
  error?: string;
};

type Lang = "js" | "ts" | "py";

type ClientSnippet = {
  status: "idle" | "loading" | "ready" | "error";
  content: string;
};

const CLIENT_URLS: Record<Lang, string> = {
  js: "/rpc/client.gen.js",
  ts: "/rpc/client.gen.ts",
  py: "/rpc/client.gen.py",
};

export function App() {
  const api = useMemo(() => createClient(window.location.origin), []);
  const [states, setStates] = useState<State[]>([]);
  const [statesStatus, setStatesStatus] = useState<"idle" | "loading" | "error">("idle");
  const [statesError, setStatesError] = useState<string>("");
  const [lookupCode, setLookupCode] = useState("");
  const [lookupResult, setLookupResult] = useState<State | null>(null);
  const [lookupError, setLookupError] = useState("");
  const [createCode, setCreateCode] = useState("");
  const [createName, setCreateName] = useState("");
  const [createStatus, setCreateStatus] = useState<"idle" | "saving" | "done" | "error">("idle");
  const [createError, setCreateError] = useState("");
  const [openPanel, setOpenPanel] = useState<Lang | null>(null);
  const [snippets, setSnippets] = useState<Record<Lang, ClientSnippet>>({
    js: { status: "idle", content: "" },
    ts: { status: "idle", content: "" },
    py: { status: "idle", content: "" },
  });

  useEffect(() => {
    void refreshStates();
  }, []);

  async function refreshStates() {
    setStatesStatus("loading");
    setStatesError("");
    try {
      const data = (await api.states.StatesGetMany()) as StatesResponse;
      if (data?.error) {
        throw new Error(data.error);
      }
      setStates(data?.data ?? []);
      setStatesStatus("idle");
    } catch (err) {
      setStatesError(err instanceof Error ? err.message : "Failed to load states");
      setStatesStatus("error");
    }
  }

  async function lookupState() {
    setLookupError("");
    setLookupResult(null);
    if (!lookupCode.trim()) {
      setLookupError("Enter a state code to search.");
      return;
    }
    try {
      const data = (await api.states.StateByCode({ code: lookupCode.trim() })) as StateResponse;
      if (data?.error) {
        throw new Error(data.error);
      }
      if (data.state) {
        setLookupResult(data.state);
      }
    } catch (err) {
      setLookupError(err instanceof Error ? err.message : "State not found");
    }
  }

  async function createState() {
    setCreateStatus("saving");
    setCreateError("");
    try {
      const data = (await api.states.StateCreate({
        code: createCode.trim(),
        name: createName.trim(),
      })) as StateResponse;
      if (data?.error) {
        throw new Error(data.error);
      }
      setCreateStatus("done");
      setCreateCode("");
      setCreateName("");
      await refreshStates();
      setTimeout(() => setCreateStatus("idle"), 1200);
    } catch (err) {
      setCreateStatus("error");
      setCreateError(err instanceof Error ? err.message : "Failed to create state");
    }
  }

  async function togglePanel(lang: Lang) {
    const next = openPanel === lang ? null : lang;
    setOpenPanel(next);
    if (!next) return;
    const current = snippets[next];
    if (current.status !== "idle") return;
    setSnippets((prev) => ({
      ...prev,
      [next]: { status: "loading", content: "" },
    }));
    try {
      const res = await fetch(CLIENT_URLS[next]);
      if (!res.ok) {
        throw new Error(`${res.status} ${res.statusText}`);
      }
      const text = await res.text();
      setSnippets((prev) => ({
        ...prev,
        [next]: { status: "ready", content: text },
      }));
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unable to load client";
      setSnippets((prev) => ({
        ...prev,
        [next]: { status: "error", content: message },
      }));
    }
  }

  function copySnippet() {
    const text = `curl -H "Authorization: Bearer dev-admin-token" \\\n  -H "Content-Type: application/json" \\\n  -d '{}' \\\n  http://localhost:8000/rpc/admin/users-get-many`;
    void navigator.clipboard.writeText(text);
  }

  return (
    <div className="page">
      <div className="wrap">
        <div className="logo-mark">Virtuous</div>
        <div className="badge">Agent First API Framework</div>
        <h1>
          Built for <span>reliable APIs</span>
        </h1>
        <p>
          Virtuous introduces typing to routes, handlers and guards, producing OpenAPI and
          native clients without codegen or config drift.
        </p>

        <div className="links">
          <a className="btn primary" href="/rpc/docs/">
            View Docs
          </a>
          <a className="btn" href="/rpc/openapi.json">
            OpenAPI JSON
          </a>
          <a className="btn" href="https://github.com/swetjen/virtuous">
            GitHub
          </a>
        </div>

        <div className="card">
          <button className="copy-btn" onClick={copySnippet}>
            Copy
          </button>
          <pre>
            <code>{`curl -H "Authorization: Bearer dev-admin-token" \\
  -H "Content-Type: application/json" \\
  -d '{}' \\
  http://localhost:8000/rpc/admin/users-get-many`}</code>
          </pre>
        </div>

        <section className="states">
          <div className="states-header">
            <div>
              <h2>States API</h2>
              <p>Live RPC-backed data from the sqlc store.</p>
            </div>
            <button className="btn" onClick={refreshStates}>
              Refresh
            </button>
          </div>

          {statesStatus === "error" && <div className="alert">{statesError}</div>}

          <div className="states-grid">
            <div className="states-list">
              <div className="states-list-header">
                <span>Code</span>
                <span>Name</span>
              </div>
              {statesStatus === "loading" ? (
                <div className="states-loading">Loading states…</div>
              ) : (
                states.map((state) => (
                  <div className="state-row" key={state.id}>
                    <span className="pill">{state.code.toUpperCase()}</span>
                    <span>{state.name}</span>
                  </div>
                ))
              )}
            </div>

            <div className="states-actions">
              <div className="panel">
                <h3>Lookup</h3>
                <p>Search by two-letter code.</p>
                <div className="field">
                  <input
                    value={lookupCode}
                    onChange={(event) => setLookupCode(event.target.value)}
                    placeholder="mn"
                  />
                  <button className="btn primary" onClick={lookupState}>
                    Find
                  </button>
                </div>
                {lookupError && <div className="alert">{lookupError}</div>}
                {lookupResult && (
                  <div className="result">
                    <span className="pill">{lookupResult.code.toUpperCase()}</span>
                    <div>
                      <strong>{lookupResult.name}</strong>
                      <small>ID {lookupResult.id}</small>
                    </div>
                  </div>
                )}
              </div>

              <div className="panel">
                <h3>Create</h3>
                <p>Insert a new state record.</p>
                <div className="field column">
                  <input
                    value={createCode}
                    onChange={(event) => setCreateCode(event.target.value)}
                    placeholder="Code"
                  />
                  <input
                    value={createName}
                    onChange={(event) => setCreateName(event.target.value)}
                    placeholder="Name"
                  />
                  <button className="btn primary" onClick={createState}>
                    {createStatus === "saving" ? "Saving…" : "Create"}
                  </button>
                </div>
                {createError && <div className="alert">{createError}</div>}
                {createStatus === "done" && <div className="success">Created successfully.</div>}
              </div>
            </div>
          </div>
        </section>

        <p className="explainer">
          These clients are generated directly from the same typed routes and guards. Updating a
          route or type signature is immediately reflected below.
        </p>

        <div className="accordion">
          {([
            { lang: "js", label: "JavaScript Client", note: ["Lightweight", "JSDoc types", "Fetch-based", "Diff-friendly"] },
            { lang: "ts", label: "TypeScript Client", note: ["Typed", "Fetch-based", "ESM ready"] },
            { lang: "py", label: "Python Client", note: ["Dataclasses", "urllib", "Explicit errors"] },
          ] as const).map(({ lang, label, note }) => {
            const snippet = snippets[lang];
            return (
              <div key={lang}>
                <button
                  className={`accordion-toggle ${openPanel === lang ? "active" : ""}`}
                  onClick={() => void togglePanel(lang)}
                >
                  {label}
                </button>
                <div className={`accordion-panel ${openPanel === lang ? "open" : ""}`}>
                  <div className="client-notes">
                    {note.map((item) => (
                      <span key={item}>{item}</span>
                    ))}
                  </div>
                  <pre className={`code-block ${lang}`}>
                    <code>{snippet.status === "loading" ? "Loading…" : snippet.content || ""}</code>
                  </pre>
                  {snippet.status === "error" && <div className="alert">{snippet.content}</div>}
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

export default App;

"use client";

import { useState, useEffect, useRef, useCallback } from "react";

/* ── Initial form values (mirrors mockup data-initial attrs) ── */
const INITIAL: Record<string, string> = {
  "profile-name": "Ethan Brooks",
  "profile-email": "ethan@example.com",
  "g-client-id": "••••••••••••••••••••••4a2b",
  "g-client-secret": "••••••••",
  "gh-client-id": "••••••••••••••••8d3f",
  "gh-client-secret": "••••••••",
  "g-domains": "",
  "gh-orgs": "",
  "domain-list": "",
  "inst-name": "etalbaas",
  /* checkboxes stored as "checked" | "" */
  "oauth-google-toggle": "checked",
  "oauth-github-toggle": "checked",
  "domain-toggle": "",
};

export default function SettingsPage() {
  /* ── Form state ── */
  const [values, setValues] = useState<Record<string, string>>({ ...INITIAL });
  const [saved, setSaved] = useState<Record<string, string>>({ ...INITIAL });

  /* ── UI state ── */
  const [activeNav, setActiveNav] = useState("sec-profile");
  const [oauthOpen, setOauthOpen] = useState<Record<string, boolean>>({
    google: true,
    github: false,
  });
  const [toastMsg, setToastMsg] = useState("");
  const [toastVisible, setToastVisible] = useState(false);
  const toastTimer = useRef<ReturnType<typeof setTimeout>>();

  /* ── Dirty check ── */
  const isDirty = Object.keys(saved).some((k) => values[k] !== saved[k]);

  /* ── Handlers ── */
  const set = (key: string, val: string) =>
    setValues((prev) => ({ ...prev, [key]: val }));

  const showToast = useCallback((msg: string) => {
    setToastMsg(msg);
    setToastVisible(true);
    clearTimeout(toastTimer.current);
    toastTimer.current = setTimeout(() => setToastVisible(false), 1800);
  }, []);

  const resetForm = () => setValues({ ...saved });
  const saveForm = () => {
    setSaved({ ...values });
    showToast("Changes saved");
  };

  const copyText = (text: string) => {
    navigator.clipboard?.writeText(text).catch(() => {});
    showToast("Copied to clipboard");
  };

  /* ── Scroll-spy ── */
  useEffect(() => {
    const sections = document.querySelectorAll<HTMLElement>(".section-block");
    if (!sections.length) return;
    const io = new IntersectionObserver(
      (entries) => {
        entries.forEach((e) => {
          if (e.isIntersecting) setActiveNav(e.target.id);
        });
      },
      { rootMargin: "-30% 0px -55% 0px", threshold: 0 }
    );
    sections.forEach((s) => io.observe(s));
    return () => io.disconnect();
  }, []);

  const scrollTo = (id: string) => {
    document.getElementById(id)?.scrollIntoView({ behavior: "smooth", block: "start" });
  };

  /* ── Render ── */
  return (
    <>
      <div className="st-content-inner">
        <div className="st-page-head">
          <h1>Settings</h1>
          <p>Manage your account, authentication, and instance configuration.</p>
        </div>

        <div className="settings-split">
          {/* Section nav */}
          <nav className="settings-nav" aria-label="Settings sections">
            {[
              { id: "sec-profile", label: "Profile" },
              { id: "sec-auth", label: "Authentication" },
              { id: "sec-instance", label: "Instance" },
            ].map((s) => (
              <a
                key={s.id}
                href={`#${s.id}`}
                className={activeNav === s.id ? "active" : ""}
                onClick={(e) => { e.preventDefault(); scrollTo(s.id); }}
              >
                {s.label}
              </a>
            ))}
            <a href="#sec-billing" className="disabled" onClick={(e) => e.preventDefault()}>
              Billing
              <span className="phase-pill">Phase 2</span>
            </a>
          </nav>

          <div className="settings-main">
            {/* ===== Profile section ===== */}
            <section className="section-block" id="sec-profile">
              <h2 className="section-title">Profile</h2>

              {/* Account info card */}
              <div className="st-card">
                <div className="st-card-head">
                  <h3>Account information</h3>
                  <p className="sub">Your personal details for this instance.</p>
                </div>
                <div className="st-card-body">
                  <div className="profile-top">
                    <div className="profile-avatar">EB</div>
                    <div className="links">
                      <button>Change avatar</button>
                      <span className="since">Member since May 1, 2026</span>
                    </div>
                  </div>
                  <div className="st-field row2">
                    <div className="st-field">
                      <label htmlFor="profile-name">Name</label>
                      <input
                        type="text"
                        id="profile-name"
                        value={values["profile-name"]}
                        onChange={(e) => set("profile-name", e.target.value)}
                      />
                    </div>
                    <div className="st-field">
                      <label htmlFor="profile-email">Email</label>
                      <div className="copyable">
                        <input
                          type="text"
                          id="profile-email"
                          className="mono"
                          value={values["profile-email"]}
                          readOnly
                        />
                      </div>
                      <span className="helper">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="11" width="18" height="11" rx="2"></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path></svg>
                        Managed by your OAuth provider
                      </span>
                    </div>
                  </div>
                  <div className="st-field">
                    <label>Role</label>
                    <div>
                      <span className="role-badge owner">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 11, height: 11 }}><path d="M12 2 L15 8 L22 9 L17 14 L18 21 L12 18 L6 21 L7 14 L2 9 L9 8 Z"></path></svg>
                        Owner
                      </span>
                    </div>
                  </div>
                </div>
                <div className="st-card-foot">
                  <button className="btn btn-ghost" onClick={resetForm}>Reset</button>
                  <button className="btn btn-primary" disabled={!isDirty} onClick={saveForm}>Save Changes</button>
                </div>
              </div>

              {/* Connected accounts */}
              <div className="st-card">
                <div className="st-card-head">
                  <h3>Connected accounts</h3>
                  <p className="sub">OAuth providers linked to your account.</p>
                </div>
                <div className="st-card-body">
                  <div className="acct-row">
                    <span className="icon" style={{ background: "#fff" }}>
                      <svg viewBox="0 0 24 24" width="16" height="16"><path fill="#4285F4" d="M21.32 11.76c0-.64-.07-1.12-.16-1.56H12v3.92h5.45c-.2 1.18-1.06 2.84-3.18 3.84l3.04 2.48c1.78-1.64 2.97-4.08 2.97-7.12z"></path><path fill="#34A853" d="M6.48 14.4c-.2-.6-.32-1.24-.32-1.92s.12-1.32.3-1.92L3.24 8.1A9.27 9.27 0 0 0 2.68 12c0 1.5.36 2.92.98 4.18l2.82-1.78z"></path><path fill="#FBBC05" d="M12 6.12c1.87 0 3.12.8 3.84 1.48l2.6-2.5C16.84 3.6 14.66 2.6 12 2.6 8.36 2.6 5.22 4.62 3.88 7.34l2.84 2.22C7.5 7.46 9.56 6.12 12 6.12z"></path><path fill="#EA4335" d="M12 21.24c2.6 0 4.78-.86 6.38-2.34l-3.04-2.48c-.84.58-1.96.98-3.34.98-2.56 0-4.74-1.68-5.52-4l-2.82 2.18C5.06 18.86 8.24 21.24 12 21.24z"></path></svg>
                    </span>
                    <div className="info">
                      <div className="name">Google</div>
                      <div className="handle">ethan@gmail.com</div>
                    </div>
                    <span className="badge-ok"><span className="dt"></span>Connected</span>
                    <button className="btn btn-ghost small" onClick={() => showToast("Disconnect google (demo)")}>Disconnect</button>
                  </div>
                  <div className="acct-row">
                    <span className="icon" style={{ background: "rgba(255,255,255,0.04)" }}>
                      <svg viewBox="0 0 24 24" width="16" height="16" fill="#e6e9ef"><path d="M12 .5C5.65.5.5 5.65.5 12c0 5.08 3.29 9.39 7.86 10.91.57.1.78-.25.78-.55v-2.02c-3.2.7-3.87-1.37-3.87-1.37-.53-1.34-1.29-1.7-1.29-1.7-1.05-.72.08-.7.08-.7 1.16.08 1.77 1.19 1.77 1.19 1.03 1.77 2.71 1.26 3.37.96.1-.75.4-1.26.73-1.55-2.55-.29-5.24-1.28-5.24-5.7 0-1.26.45-2.29 1.19-3.1-.12-.29-.52-1.47.11-3.06 0 0 .97-.31 3.18 1.18a11 11 0 0 1 5.78 0c2.21-1.49 3.18-1.18 3.18-1.18.63 1.59.23 2.77.11 3.06.74.81 1.19 1.84 1.19 3.1 0 4.44-2.7 5.41-5.27 5.69.41.35.78 1.05.78 2.12v3.14c0 .3.21.66.79.55C20.21 21.39 23.5 17.07 23.5 12 23.5 5.65 18.35.5 12 .5z"></path></svg>
                    </span>
                    <div className="info">
                      <div className="name">GitHub</div>
                      <div className="handle">@ethanbrooks</div>
                    </div>
                    <span className="badge-ok"><span className="dt"></span>Connected</span>
                    <button className="btn btn-ghost small" onClick={() => showToast("Disconnect github (demo)")}>Disconnect</button>
                  </div>
                </div>
              </div>

              {/* Sessions */}
              <div className="st-card">
                <div className="st-card-head">
                  <h3>Active sessions</h3>
                  <p className="sub">Devices currently signed in to your account.</p>
                </div>
                <div className="st-card-body">
                  <div className="session-row">
                    <span className="badge-ok"><span className="dt"></span>Active now</span>
                    <div className="info">
                      <div className="name">Chrome on Windows</div>
                      <div className="meta">Tokyo, JP · <span className="mono">203.0.113.42</span> · this device</div>
                    </div>
                  </div>
                  <div className="session-row">
                    <span className="badge-info" style={{ opacity: 0.7 }}>2 others</span>
                    <div className="info">
                      <div className="name">2 other active sessions</div>
                      <div className="meta">Safari on macOS · Last seen 3 hours ago · iPhone · Last seen 2 days ago</div>
                    </div>
                    <button className="btn btn-danger-ghost small" onClick={() => showToast("Revoked 2 other sessions")}>Revoke all others</button>
                  </div>
                </div>
              </div>
            </section>

            {/* ===== Authentication section ===== */}
            <section className="section-block" id="sec-auth">
              <h2 className="section-title">Authentication</h2>

              {/* OAuth providers */}
              <div className="st-card">
                <div className="st-card-head">
                  <h3>OAuth providers</h3>
                  <p className="sub">Configure which providers appear on the login page.</p>
                </div>
                <div className="st-card-body" style={{ gap: 0, padding: "0 20px" }}>
                  {/* Google */}
                  <div className={`oauth-row${oauthOpen.google ? " open" : ""}`} id="oauth-google">
                    <div className="head-row" onClick={(e) => { if ((e.target as HTMLElement).closest(".stg-switch")) return; setOauthOpen((p) => ({ ...p, google: !p.google })); }}>
                      <span className="icon" style={{ background: "#fff" }}>
                        <svg viewBox="0 0 24 24" width="16" height="16"><path fill="#4285F4" d="M21.32 11.76c0-.64-.07-1.12-.16-1.56H12v3.92h5.45c-.2 1.18-1.06 2.84-3.18 3.84l3.04 2.48c1.78-1.64 2.97-4.08 2.97-7.12z"></path><path fill="#34A853" d="M6.48 14.4c-.2-.6-.32-1.24-.32-1.92s.12-1.32.3-1.92L3.24 8.1A9.27 9.27 0 0 0 2.68 12c0 1.5.36 2.92.98 4.18l2.82-1.78z"></path><path fill="#FBBC05" d="M12 6.12c1.87 0 3.12.8 3.84 1.48l2.6-2.5C16.84 3.6 14.66 2.6 12 2.6 8.36 2.6 5.22 4.62 3.88 7.34l2.84 2.22C7.5 7.46 9.56 6.12 12 6.12z"></path><path fill="#EA4335" d="M12 21.24c2.6 0 4.78-.86 6.38-2.34l-3.04-2.48c-.84.58-1.96.98-3.34.98-2.56 0-4.74-1.68-5.52-4l-2.82 2.18C5.06 18.86 8.24 21.24 12 21.24z"></path></svg>
                      </span>
                      <div className="info">
                        <div className="name">Google <span className="badge-ok"><span className="dt"></span>Configured</span></div>
                        <div className="sub">OAuth 2.0 / OpenID Connect</div>
                      </div>
                      <input
                        type="checkbox"
                        className="stg-switch"
                        id="oauth-google-toggle"
                        checked={values["oauth-google-toggle"] === "checked"}
                        onChange={(e) => { e.stopPropagation(); set("oauth-google-toggle", e.target.checked ? "checked" : ""); }}
                        onClick={(e) => e.stopPropagation()}
                      />
                      <svg className="chev" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><polyline points="6 9 12 15 18 9"></polyline></svg>
                    </div>
                    <div className="config">
                      <div className="st-field row2">
                        <div className="st-field">
                          <label htmlFor="g-client-id">Client ID</label>
                          <input type="text" id="g-client-id" className="mono" value={values["g-client-id"]} onChange={(e) => set("g-client-id", e.target.value)} />
                        </div>
                        <div className="st-field">
                          <label htmlFor="g-client-secret">Client Secret</label>
                          <input type="password" id="g-client-secret" className="mono" value={values["g-client-secret"]} onChange={(e) => set("g-client-secret", e.target.value)} />
                        </div>
                      </div>
                      <div className="st-field" style={{ marginTop: 14 }}>
                        <label>Redirect URI</label>
                        <div className="copyable">
                          <input type="text" className="mono" value="https://etalbaas.local/auth/callback/google" readOnly />
                          <button className="copy" onClick={() => copyText("https://etalbaas.local/auth/callback/google")} aria-label="Copy">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                          </button>
                        </div>
                        <span className="helper">Paste this into your Google OAuth client&apos;s authorized redirect URIs.</span>
                      </div>
                      <div className="st-field" style={{ marginTop: 14 }}>
                        <label htmlFor="g-domains">Allowed email domains</label>
                        <input type="text" id="g-domains" placeholder="Leave empty to allow all (e.g. example.com, company.org)" value={values["g-domains"]} onChange={(e) => set("g-domains", e.target.value)} />
                        <span className="helper">Comma-separated. Restricts sign-in to these domains.</span>
                      </div>
                    </div>
                  </div>

                  {/* GitHub */}
                  <div className={`oauth-row${oauthOpen.github ? " open" : ""}`} id="oauth-github">
                    <div className="head-row" onClick={(e) => { if ((e.target as HTMLElement).closest(".stg-switch")) return; setOauthOpen((p) => ({ ...p, github: !p.github })); }}>
                      <span className="icon" style={{ background: "rgba(255,255,255,0.04)" }}>
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="#e6e9ef"><path d="M12 .5C5.65.5.5 5.65.5 12c0 5.08 3.29 9.39 7.86 10.91.57.1.78-.25.78-.55v-2.02c-3.2.7-3.87-1.37-3.87-1.37-.53-1.34-1.29-1.7-1.29-1.7-1.05-.72.08-.7.08-.7 1.16.08 1.77 1.19 1.77 1.19 1.03 1.77 2.71 1.26 3.37.96.1-.75.4-1.26.73-1.55-2.55-.29-5.24-1.28-5.24-5.7 0-1.26.45-2.29 1.19-3.1-.12-.29-.52-1.47.11-3.06 0 0 .97-.31 3.18 1.18a11 11 0 0 1 5.78 0c2.21-1.49 3.18-1.18 3.18-1.18.63 1.59.23 2.77.11 3.06.74.81 1.19 1.84 1.19 3.1 0 4.44-2.7 5.41-5.27 5.69.41.35.78 1.05.78 2.12v3.14c0 .3.21.66.79.55C20.21 21.39 23.5 17.07 23.5 12 23.5 5.65 18.35.5 12 .5z"></path></svg>
                      </span>
                      <div className="info">
                        <div className="name">GitHub <span className="badge-ok"><span className="dt"></span>Configured</span></div>
                        <div className="sub">OAuth Apps</div>
                      </div>
                      <input
                        type="checkbox"
                        className="stg-switch"
                        id="oauth-github-toggle"
                        checked={values["oauth-github-toggle"] === "checked"}
                        onChange={(e) => { e.stopPropagation(); set("oauth-github-toggle", e.target.checked ? "checked" : ""); }}
                        onClick={(e) => e.stopPropagation()}
                      />
                      <svg className="chev" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><polyline points="6 9 12 15 18 9"></polyline></svg>
                    </div>
                    <div className="config">
                      <div className="st-field row2">
                        <div className="st-field">
                          <label htmlFor="gh-client-id">Client ID</label>
                          <input type="text" id="gh-client-id" className="mono" value={values["gh-client-id"]} onChange={(e) => set("gh-client-id", e.target.value)} />
                        </div>
                        <div className="st-field">
                          <label htmlFor="gh-client-secret">Client Secret</label>
                          <input type="password" id="gh-client-secret" className="mono" value={values["gh-client-secret"]} onChange={(e) => set("gh-client-secret", e.target.value)} />
                        </div>
                      </div>
                      <div className="st-field" style={{ marginTop: 14 }}>
                        <label>Redirect URI</label>
                        <div className="copyable">
                          <input type="text" className="mono" value="https://etalbaas.local/auth/callback/github" readOnly />
                          <button className="copy" onClick={() => copyText("https://etalbaas.local/auth/callback/github")} aria-label="Copy">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                          </button>
                        </div>
                      </div>
                      <div className="st-field" style={{ marginTop: 14 }}>
                        <label htmlFor="gh-orgs">Allowed GitHub orgs</label>
                        <input type="text" id="gh-orgs" placeholder="Leave empty to allow all (e.g. acme-co, etalbaas)" value={values["gh-orgs"]} onChange={(e) => set("gh-orgs", e.target.value)} />
                      </div>
                    </div>
                  </div>

                  {/* GitLab (disabled) */}
                  <div className="oauth-row disabled">
                    <div className="head-row">
                      <span className="icon" style={{ background: "rgba(255,255,255,0.04)" }}>
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="#fc6d26"><path d="M22.65 14.39 12 22.13 1.35 14.39a.84.84 0 0 1-.3-.94l1.22-3.78 2.44-7.51A.42.42 0 0 1 5.5 2.1l2.44 7.51h8.11l2.44-7.51a.42.42 0 0 1 .79 0l2.44 7.51 1.22 3.78a.84.84 0 0 1-.29.94z"></path></svg>
                      </span>
                      <div className="info">
                        <div className="name">GitLab <span className="phase-pill">Coming soon</span></div>
                        <div className="sub">OAuth 2.0</div>
                      </div>
                      <input type="checkbox" className="stg-switch" disabled />
                    </div>
                  </div>

                  {/* SAML (disabled) */}
                  <div className="oauth-row disabled">
                    <div className="head-row">
                      <span className="icon" style={{ background: "rgba(255,255,255,0.04)", color: "var(--fg-dim)" }}>
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path><circle cx="12" cy="11" r="3"></circle></svg>
                      </span>
                      <div className="info">
                        <div className="name">SAML / SSO <span className="phase-pill">Coming soon</span></div>
                        <div className="sub">Enterprise single sign-on</div>
                      </div>
                      <input type="checkbox" className="stg-switch" disabled />
                    </div>
                  </div>
                </div>
                <div className="st-card-foot">
                  <button className="btn btn-ghost" onClick={resetForm}>Reset</button>
                  <button className="btn btn-primary" disabled={!isDirty} onClick={saveForm}>Save Changes</button>
                </div>
              </div>

              {/* Allowed domains */}
              <div className="st-card">
                <div className="st-card-head">
                  <h3>Allowed email domains</h3>
                  <p className="sub">Restrict sign-ups to specific email domains.</p>
                </div>
                <div className="st-card-body">
                  <label style={{ display: "flex", alignItems: "center", gap: 10, cursor: "pointer" }}>
                    <input
                      type="checkbox"
                      className="stg-switch"
                      checked={values["domain-toggle"] === "checked"}
                      onChange={(e) => set("domain-toggle", e.target.checked ? "checked" : "")}
                    />
                    <span style={{ fontSize: 13 }}>Enable domain restriction</span>
                  </label>
                  {values["domain-toggle"] === "checked" && (
                    <div className="st-field">
                      <label htmlFor="domain-list">Allowed domains <span className="optional">— one per line</span></label>
                      <textarea
                        id="domain-list"
                        className="mono"
                        placeholder={"example.com\ncompany.org"}
                        value={values["domain-list"]}
                        onChange={(e) => set("domain-list", e.target.value)}
                      />
                      <span className="helper" style={{ color: "var(--warn)" }}>
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"></path><line x1="12" y1="9" x2="12" y2="13"></line><line x1="12" y1="17" x2="12.01" y2="17"></line></svg>
                        Users with emails outside these domains will be denied access.
                      </span>
                    </div>
                  )}
                </div>
              </div>
            </section>

            {/* ===== Instance section ===== */}
            <section className="section-block" id="sec-instance">
              <h2 className="section-title">Instance</h2>

              {/* Instance info */}
              <div className="st-card">
                <div className="st-card-head">
                  <h3>Instance information</h3>
                  <p className="sub">Identity and version of this EtalBaaS deployment.</p>
                </div>
                <div className="st-card-body">
                  <div className="st-field row2">
                    <div className="st-field">
                      <label htmlFor="inst-name">Instance name</label>
                      <input type="text" id="inst-name" value={values["inst-name"]} onChange={(e) => set("inst-name", e.target.value)} />
                      <span className="helper">Shown in the status pill across all pages.</span>
                    </div>
                    <div className="st-field">
                      <label>Instance URL</label>
                      <div className="copyable">
                        <input type="text" className="mono" value="https://etalbaas.local" readOnly />
                        <button className="copy" onClick={() => copyText("https://etalbaas.local")} aria-label="Copy">
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                        </button>
                      </div>
                      <span className="helper">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="11" width="18" height="11" rx="2"></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path></svg>
                        Configured via Helm values. Change requires redeployment.
                      </span>
                    </div>
                  </div>
                  <div className="inst-row">
                    <span className="label">Version</span>
                    <span className="value">
                      v0.1.0-sha-abc1234
                      <span className="badge-ok"><span className="dt"></span>Up to date</span>
                    </span>
                  </div>
                </div>
              </div>

              {/* TLS & domain */}
              <div className="st-card">
                <div className="st-card-head">
                  <h3>TLS &amp; domain</h3>
                  <p className="sub">Certificate and DNS configuration for this instance.</p>
                </div>
                <div className="st-card-body" style={{ padding: "0 20px" }}>
                  <div className="inst-row">
                    <span className="label">Primary domain</span>
                    <span className="value">etalbaas.local</span>
                  </div>
                  <div className="inst-row">
                    <span className="label">TLS certificate</span>
                    <span className="value">
                      Valid until Dec 15, 2026
                      <span className="badge-ok"><span className="dt"></span>Healthy</span>
                      <span style={{ color: "var(--fg-mute)", fontFamily: "'Inter',sans-serif", fontSize: 12, marginLeft: 4 }}>Let&apos;s Encrypt</span>
                    </span>
                  </div>
                  <div className="inst-row">
                    <span className="label">Wildcard</span>
                    <span className="value">*.etalbaas.local <span className="badge-ok"><span className="dt"></span>Active</span></span>
                  </div>
                </div>
                <div className="st-card-foot" style={{ justifyContent: "flex-start" }}>
                  <span style={{ color: "var(--fg-mute)", fontSize: 12, display: "inline-flex", alignItems: "center", gap: 6 }}>
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 11, height: 11 }}><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>
                    Domain &amp; TLS are managed by cert-manager.
                  </span>
                </div>
              </div>

              {/* Observability */}
              <div className="st-card">
                <div className="st-card-head">
                  <h3>Observability</h3>
                  <p className="sub">Metrics, tracing, and log destinations.</p>
                </div>
                <div className="st-card-body" style={{ padding: "0 20px" }}>
                  <div className="inst-row">
                    <span className="label">Metrics endpoint</span>
                    <span className="value">
                      http://localhost:9090/metrics
                      <button
                        className="copy"
                        onClick={() => copyText("http://localhost:9090/metrics")}
                        aria-label="Copy"
                        style={{ background: "transparent", border: "1px solid var(--line-2)", color: "var(--fg-mute)", cursor: "pointer", width: 24, height: 24, display: "grid", placeItems: "center", borderRadius: 5 }}
                      >
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 10, height: 10 }}><rect x="9" y="9" width="13" height="13" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                      </button>
                    </span>
                  </div>
                  <div className="inst-row">
                    <span className="label">Tracing</span>
                    <span className="value">
                      OpenTelemetry → Jaeger
                      <span className="badge-ok"><span className="dt"></span>Connected</span>
                    </span>
                  </div>
                  <div className="inst-row">
                    <span className="label">OTLP endpoint</span>
                    <span className="value">http://jaeger:4317</span>
                  </div>
                  <div className="inst-row">
                    <span className="label">Logs</span>
                    <span className="value" style={{ color: "var(--fg-dim)" }}>Forwarded to stdout (container logs)</span>
                  </div>
                </div>
              </div>

              {/* Backup */}
              <div className="st-card">
                <div className="st-card-head">
                  <h3>Backup configuration</h3>
                  <p className="sub">Periodic backups of project metadata and database snapshots.</p>
                </div>
                <div className="st-card-body" style={{ padding: "0 20px" }}>
                  <div className="inst-row">
                    <span className="label">Provider</span>
                    <span className="value">
                      <span className="badge-info">Cloudflare R2</span>
                      <span style={{ color: "var(--fg-mute)", fontFamily: "'Inter',sans-serif", fontSize: 12 }}>backups-etalbaas</span>
                    </span>
                  </div>
                  <div className="inst-row">
                    <span className="label">Schedule</span>
                    <span className="value">Daily at 03:00 UTC</span>
                  </div>
                  <div className="inst-row">
                    <span className="label">Last backup</span>
                    <span className="value">
                      12 hours ago
                      <span style={{ color: "var(--fg-dim)", fontFamily: "'Inter',sans-serif" }}>·</span>
                      2.3 GB
                      <span className="badge-ok"><span className="dt"></span>Healthy</span>
                    </span>
                  </div>
                  <div className="inst-row">
                    <span className="label">Retention</span>
                    <span className="value" style={{ color: "var(--fg-dim)" }}>30 days</span>
                  </div>
                </div>
                <div className="st-card-foot">
                  <span style={{ color: "var(--fg-mute)", fontSize: 12, marginRight: "auto" }}>Managed via Helm values.</span>
                  <button className="btn btn-ghost" onClick={() => showToast("Test backup queued…")}>
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 13, height: 13 }}><polyline points="3 12 3 6 9 6"></polyline><path d="M3 6 13 16a8 8 0 1 1-1 11"></path></svg>
                    Test Backup
                  </button>
                </div>
              </div>
            </section>

            {/* ===== Billing (disabled) ===== */}
            <section className="section-block" id="sec-billing">
              <h2 className="section-title">Billing</h2>
              <div className="st-card">
                <div className="billing-disabled">
                  <div className="icon-box">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="11" width="18" height="11" rx="2"></rect><path d="M7 11V7a5 5 0 0 1 10 0v4"></path></svg>
                  </div>
                  <h3>Billing &amp; usage</h3>
                  <p>Usage tracking and billing will be available in Phase 2.</p>
                  <p style={{ color: "var(--fg-mute)", marginTop: 8, fontSize: 12 }}>Self-hosted instances have no usage limits in Phase 1.</p>
                </div>
              </div>
            </section>
          </div>
        </div>

        {/* Unsaved bar */}
        <div className={`st-unsaved-bar${isDirty ? " show" : ""}`}>
          <span className="left"><span className="dot-warn"></span>You have unsaved changes</span>
          <div className="actions" style={{ display: "flex", gap: 6 }}>
            <button className="btn btn-ghost" onClick={resetForm}>Discard</button>
            <button className="btn btn-primary" onClick={saveForm}>Save Changes</button>
          </div>
        </div>
      </div>

      {/* Toast */}
      <div className={`toast${toastVisible ? " show" : ""}`}>
        <span>{toastMsg}</span>
      </div>
    </>
  );
}

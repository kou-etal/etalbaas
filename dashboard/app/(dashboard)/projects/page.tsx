"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useProjects, useCreateProject } from "@/features/projects/hooks";

function genId() {
  return (
    Math.random().toString(36).substring(2, 6) +
    Math.random().toString(36).substring(2, 6)
  );
}

function timeAgo(dateStr: string): string {
  if (!dateStr) return "";
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diff = now - then;
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return "just now";
  if (minutes < 60)
    return `${minutes} minute${minutes > 1 ? "s" : ""} ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} hour${hours > 1 ? "s" : ""} ago`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days} day${days > 1 ? "s" : ""} ago`;
  const weeks = Math.floor(days / 7);
  if (weeks < 5) return `${weeks} week${weeks > 1 ? "s" : ""} ago`;
  const months = Math.floor(days / 30);
  if (months < 12)
    return `${months} month${months > 1 ? "s" : ""} ago`;
  const years = Math.floor(days / 365);
  return `${years} year${years > 1 ? "s" : ""} ago`;
}

type FilterKey = "all" | "ready" | "provisioning" | "issues";

function matchesFilter(status: string, filter: FilterKey): boolean {
  if (filter === "all") return true;
  if (filter === "ready") return status === "ready";
  if (filter === "provisioning")
    return (
      status === "provisioning" ||
      status === "building" ||
      status === "pending"
    );
  if (filter === "issues") return status === "failed" || status === "paused";
  return true;
}

const FILTER_LABELS: Record<FilterKey, string> = {
  all: "All",
  ready: "Ready",
  provisioning: "Active",
  issues: "Issues",
};

export default function ProjectsPage() {
  const router = useRouter();
  const { data: projects, isLoading } = useProjects();
  const createProject = useCreateProject();

  /* ===== Search + filter ===== */
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<FilterKey>("all");
  const searchRef = useRef<HTMLInputElement>(null);

  /* ===== Modal state ===== */
  const [modalOpen, setModalOpen] = useState(false);
  const [modalView, setModalView] = useState<"form" | "success">("form");
  const [cpName, setCpName] = useState("");
  const [cpDesc, setCpDesc] = useState("");
  const [cpIdPreview, setCpIdPreview] = useState("________");
  const [pgEnabled, setPgEnabled] = useState(true);
  const [rdEnabled, setRdEnabled] = useState(false);
  const [apiEnabled, setApiEnabled] = useState(true);
  const [extVector, setExtVector] = useState(false);
  const [extCrypto, setExtCrypto] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const cpNameRef = useRef<HTMLInputElement>(null);
  const cpIdSeedRef = useRef("");

  /* ===== Toast ===== */
  const [toastText, setToastText] = useState("Creating project\u2026");
  const [toastShow, setToastShow] = useState(false);
  const toastTimerRef = useRef<ReturnType<typeof setTimeout>>();

  /* ===== body[data-state] for CSS toggling ===== */
  const pageState = isLoading
    ? "loading"
    : !projects || projects.length === 0
      ? "empty"
      : "loaded";

  useEffect(() => {
    document.body.dataset.state = pageState;
    return () => {
      delete document.body.dataset.state;
    };
  }, [pageState]);

  /* ===== Derived data ===== */
  const filtered = projects?.filter((p) => {
    const q = search.toLowerCase();
    const matchSearch =
      p.displayName.toLowerCase().includes(q) ||
      (p.description || "").toLowerCase().includes(q);
    const matchStatus = matchesFilter(p.status || "pending", statusFilter);
    return matchSearch && matchStatus;
  });

  const counts: Record<FilterKey, number> = {
    all: projects?.length || 0,
    ready:
      projects?.filter((p) => (p.status || "pending") === "ready").length || 0,
    provisioning:
      projects?.filter((p) =>
        ["provisioning", "building", "pending"].includes(p.status || "pending"),
      ).length || 0,
    issues:
      projects?.filter((p) =>
        ["failed", "paused"].includes(p.status || "pending"),
      ).length || 0,
  };

  /* ===== Name validation ===== */
  const nameVal: "none" | "valid" | "invalid" =
    cpName.trim() === ""
      ? "none"
      : cpName.trim().length > 100
        ? "invalid"
        : "valid";

  const nameInputClass =
    nameVal === "valid" ? "valid" : nameVal === "invalid" ? "invalid" : "";

  /* Update ID preview when name changes */
  useEffect(() => {
    const v = cpName.trim();
    if (!v) {
      setCpIdPreview("________");
      return;
    }
    if (cpIdSeedRef.current !== v) {
      cpIdSeedRef.current = v;
      setCpIdPreview(genId());
    }
  }, [cpName]);

  /* ===== Toast helper ===== */
  const showToast = useCallback((msg: string) => {
    setToastText(msg);
    setToastShow(true);
    if (toastTimerRef.current) clearTimeout(toastTimerRef.current);
    toastTimerRef.current = setTimeout(() => setToastShow(false), 1800);
  }, []);

  /* ===== Modal open / close ===== */
  const openModal = useCallback(() => {
    setCpName("");
    setCpDesc("");
    setCpIdPreview("________");
    setPgEnabled(true);
    setRdEnabled(false);
    setApiEnabled(true);
    setExtVector(false);
    setExtCrypto(false);
    setIsSubmitting(false);
    setModalView("form");
    setModalOpen(true);
    setTimeout(() => cpNameRef.current?.focus(), 60);
  }, []);

  const closeModal = useCallback(() => {
    setModalOpen(false);
  }, []);

  /* ===== PG toggle — PostgREST depends on PG ===== */
  const handlePgToggle = (checked: boolean) => {
    setPgEnabled(checked);
    if (!checked) {
      setApiEnabled(false);
    }
  };

  /* ===== Submit ===== */
  const handleSubmit = useCallback(async () => {
    if (nameVal !== "valid" || isSubmitting) return;
    setIsSubmitting(true);
    try {
      const extensions: string[] = [];
      if (extVector) extensions.push("pgvector");
      if (extCrypto) extensions.push("pgcrypto");
      await createProject.mutateAsync({
        displayName: cpName.trim(),
        description: cpDesc,
        postgresEnabled: pgEnabled,
        postgresExtensions: extensions,
        redisEnabled: rdEnabled,
        postgrestEnabled: apiEnabled,
      });
      setModalView("success");
      setTimeout(() => {
        setModalOpen(false);
        showToast(
          'Project "' + cpName.trim() + '" created \u2014 provisioning\u2026',
        );
      }, 1400);
    } catch (err) {
      console.error("CreateProject failed:", err);
      setIsSubmitting(false);
    }
  }, [
    nameVal,
    isSubmitting,
    extVector,
    extCrypto,
    cpName,
    cpDesc,
    pgEnabled,
    rdEnabled,
    apiEnabled,
    createProject,
    showToast,
  ]);

  /* ===== Keep a ref so keyboard handler always has latest ===== */
  const submitRef = useRef(handleSubmit);
  submitRef.current = handleSubmit;

  /* ===== Keyboard shortcuts ===== */
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      /* Modal-scoped shortcuts */
      if (modalOpen) {
        if (e.key === "Escape") {
          e.preventDefault();
          closeModal();
        }
        if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
          e.preventDefault();
          submitRef.current();
        }
        return;
      }

      /* Global shortcuts (not when focused on inputs) */
      const tag = (e.target as HTMLElement).tagName;
      if (tag === "INPUT" || tag === "TEXTAREA") return;
      if (e.metaKey || e.ctrlKey || e.altKey) return;
      const key = e.key.toLowerCase();
      if (key === "n") {
        e.preventDefault();
        openModal();
      }
      if (e.key === "/") {
        e.preventDefault();
        searchRef.current?.focus();
      }
    };

    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [modalOpen, openModal, closeModal]);

  /* ============================================================= */
  /*  RENDER                                                        */
  /* ============================================================= */
  return (
    <>
          <header className="topbar">
            <div className="crumbs">
              <strong>Projects</strong>
              <span className="sep">/</span>
              <span className="mono" style={{ fontSize: "12px", color: "var(--fg-mute)" }}>workspace-default</span>
            </div>
            <div className="right">
              <span className="pill"><span className="dot" /><span className="mono">etalbaas.local</span></span>
              <button className="btn btn-primary" id="new-project" onClick={openModal}>
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round"><path d="M12 5v14M5 12h14" /></svg>
                New Project
                <span className="kbd">N</span>
              </button>
            </div>
          </header>

          <main className="content">
            <div className="content-inner">
              <div className="page-head">
                <div>
                  <h1>Projects</h1>
                  <p>Deploy databases, functions, and APIs in minutes.</p>
                </div>
              </div>

              {/* Filters */}
              <div className="filters">
                <label className="search">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="11" cy="11" r="7" /><path d="m20 20-3.5-3.5" /></svg>
                  <input
                    ref={searchRef}
                    type="text"
                    placeholder="Search projects\u2026"
                    aria-label="Search projects"
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                  />
                  <span className="kbd-hint">/</span>
                </label>
                {(["all", "ready", "provisioning", "issues"] as FilterKey[]).map(
                  (key) => (
                    <button
                      key={key}
                      className={`chip${statusFilter === key ? " active" : ""}`}
                      onClick={() => setStatusFilter(key)}
                    >
                      {FILTER_LABELS[key]} <span className="count">{counts[key]}</span>
                    </button>
                  ),
                )}
              </div>

              {/* ===== Loaded grid ===== */}
              <div id="loaded-block">
                <div className="card-grid">
                  {filtered?.map((project) => (
                    <a
                      key={project.id}
                      className="card"
                      href={`/projects/${project.id}`}
                      onClick={(e) => {
                        e.preventDefault();
                        router.push(`/projects/${project.id}`);
                      }}
                    >
                      <div className="card-head">
                        <div className="card-title">
                          <div className="name">{project.displayName}</div>
                          <div className="id mono">{project.id}</div>
                        </div>
                        <span className={`badge ${(project.status || "pending").toLowerCase()}`}>
                          <span className="bd" />
                          {(project.status || "pending").charAt(0).toUpperCase() +
                            (project.status || "pending").slice(1)}
                        </span>
                      </div>
                      <p className="card-desc">{project.description}</p>
                      <div className="card-foot">
                        <div className="services">
                          <span className={`svc pg${!project.postgresEnabled ? " dimmed" : ""}`}><span className="svc-dot" />PG</span>
                          <span className={`svc rd${!project.redisEnabled ? " dimmed" : ""}`}><span className="svc-dot" />RD</span>
                          <span className={`svc api${!project.postgrestEnabled ? " dimmed" : ""}`}><span className="svc-dot" />API</span>
                        </div>
                        <span className="when">
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 2" /></svg>
                          {timeAgo(project.updatedAt || project.createdAt)}
                        </span>
                      </div>
                    </a>
                  ))}
                </div>

                <div className="pagination">
                  <button className="btn btn-ghost">Load more</button>
                </div>
              </div>

              {/* ===== Loading skeleton ===== */}
              <div id="loading-block">
                <div className="card-grid">
                  {[0, 1, 2, 3, 4, 5].map((i) => (
                    <div key={i} className="card skeleton">
                      <div className="card-head">
                        <div className="card-title" style={{ flex: 1 }}>
                          <div className="skel skel-line skel-1" />
                          <div className="skel skel-line skel-2" />
                        </div>
                        <div className="skel skel-pill" />
                      </div>
                      <div className="skel skel-line skel-3" />
                      <div className="skel skel-line skel-4" />
                      <div className="card-foot">
                        <div className="services">
                          <div className="skel skel-svc" />
                          <div className="skel skel-svc" />
                          <div className="skel skel-svc" />
                        </div>
                        <div className="skel skel-line" style={{ width: "60px" }} />
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* ===== Empty state ===== */}
              <div id="empty-block">
                <div className="empty">
                  <div className="art" aria-hidden="true">
                    <svg viewBox="0 0 96 96" fill="none">
                      <path d="M48 12 L84 24 L48 36 L12 24 Z" fill="var(--accent)" opacity="0.95" />
                      <path d="M12 42 L48 54 L84 42" stroke="var(--accent)" strokeWidth="3" strokeLinejoin="round" opacity="0.55" fill="none" />
                      <path d="M12 60 L48 72 L84 60" stroke="var(--accent)" strokeWidth="3" strokeLinejoin="round" opacity="0.3" fill="none" />
                      <path d="M12 78 L48 90 L84 78" stroke="var(--accent)" strokeWidth="3" strokeLinejoin="round" opacity="0.15" fill="none" />
                    </svg>
                  </div>
                  <h2>No projects yet</h2>
                  <p>Create your first project to start deploying databases, functions, and APIs.</p>
                  <button className="btn btn-primary" onClick={openModal}>
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round"><path d="M12 5v14M5 12h14" /></svg>
                    New Project
                    <span className="kbd">N</span>
                  </button>
                </div>
              </div>
            </div>
          </main>

      {/* Toast */}
      <div className={`toast${toastShow ? " show" : ""}`}>
        <span className="spinner" />
        <span>{toastText}</span>
      </div>

      {/* ====== Create Project modal ====== */}
      <div
        className={`modal-scrim${modalOpen ? " open" : ""}`}
        role="dialog"
        aria-modal="true"
        aria-labelledby="cp-title"
        onClick={(e) => {
          if (e.target === e.currentTarget) closeModal();
        }}
      >
        <div className="modal">
          {/* Form view */}
          <div
            id="cp-form"
            style={{
              display: modalView === "form" ? "flex" : "none",
              flexDirection: "column",
              minHeight: 0,
              flex: 1,
            }}
          >
            <div className="modal-head">
              <div style={{ flex: 1, minWidth: 0 }}>
                <h3 id="cp-title">Create a new project</h3>
                <p className="sub">Configure your project&apos;s name and services. You can change these later.</p>
              </div>
              <button className="close" onClick={closeModal} aria-label="Close">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: "15px", height: "15px" }}><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
              </button>
            </div>

            <div className="modal-body">
              <div className="cp-field">
                <label htmlFor="cp-name">Project name</label>
                <div className="input-wrap">
                  <input
                    ref={cpNameRef}
                    type="text"
                    id="cp-name"
                    placeholder="e.g. my-saas-app, analytics-prod"
                    maxLength={120}
                    autoComplete="off"
                    className={nameInputClass}
                    value={cpName}
                    onChange={(e) => setCpName(e.target.value)}
                  />
                  <span className="input-icon">
                    <svg className="check" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12" /></svg>
                    <svg className="x" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
                  </span>
                </div>
                <span className={`hint${nameVal === "invalid" ? " err" : ""}`}>
                  {nameVal === "invalid" && (
                    <>
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
                      {" "}Maximum 100 characters
                    </>
                  )}
                </span>
                <div className="endpoint-preview">
                  <span className="ep-label">Your project will be available at</span>
                  <span className="ep-url">https://<span className="id-slot">{cpIdPreview}</span>.etalbaas.io</span>
                  <span className="ep-note">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10" /><line x1="12" y1="16" x2="12" y2="12" /><line x1="12" y1="8" x2="12.01" y2="8" /></svg>
                    An 8-character ID will be assigned automatically.
                  </span>
                </div>
              </div>

              <div className="cp-field">
                <label htmlFor="cp-desc">Description <span className="optional">&mdash; optional</span></label>
                <div className="ta-wrap">
                  <textarea
                    id="cp-desc"
                    placeholder="What is this project for?"
                    maxLength={500}
                    rows={2}
                    value={cpDesc}
                    onChange={(e) => setCpDesc(e.target.value)}
                  />
                  <span className="ta-counter">{cpDesc.length} / 500</span>
                </div>
              </div>

              <div className="svc-section">
                <div className="head">
                  <h4>Enable services</h4>
                  <span className="sub">Choose which infrastructure to provision.</span>
                </div>

                {/* PostgreSQL */}
                <div className={`svc-card${pgEnabled ? " checked" : ""}`}>
                  <label className="head-row">
                    <span className="icon pg">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><ellipse cx="12" cy="5" rx="9" ry="3" /><path d="M3 5v14c0 1.66 4.03 3 9 3s9-1.34 9-3V5" /><path d="M3 12c0 1.66 4.03 3 9 3s9-1.34 9-3" /></svg>
                    </span>
                    <span className="info">
                      <span className="ttl">PostgreSQL <span className="ver">v16</span></span>
                      <p className="sub">Managed database with automatic backups.</p>
                    </span>
                    <input
                      type="checkbox"
                      className="cp-switch"
                      checked={pgEnabled}
                      onChange={(e) => handlePgToggle(e.target.checked)}
                    />
                  </label>
                  <div className="svc-extensions">
                    <div className="exp-title">Extensions <span style={{ color: "var(--fg-dim)", textTransform: "none", letterSpacing: 0, fontWeight: 400 }}>(optional)</span></div>
                    <label className="ext-row">
                      <input
                        type="checkbox"
                        checked={extVector}
                        onChange={(e) => setExtVector(e.target.checked)}
                      />
                      <span>
                        <span className="nm">pgvector</span>
                        <span className="desc">Vector similarity search for AI / ML workloads</span>
                      </span>
                    </label>
                    <label className="ext-row">
                      <input
                        type="checkbox"
                        checked={extCrypto}
                        onChange={(e) => setExtCrypto(e.target.checked)}
                      />
                      <span>
                        <span className="nm">pgcrypto</span>
                        <span className="desc">Cryptographic functions</span>
                      </span>
                    </label>
                  </div>
                </div>

                {/* Redis */}
                <div className={`svc-card${rdEnabled ? " checked" : ""}`}>
                  <label className="head-row">
                    <span className="icon rd">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" /></svg>
                    </span>
                    <span className="info">
                      <span className="ttl">Redis <span className="ver">v7.2</span></span>
                      <p className="sub">In-memory cache for high-speed data access.</p>
                    </span>
                    <input
                      type="checkbox"
                      className="cp-switch"
                      checked={rdEnabled}
                      onChange={(e) => setRdEnabled(e.target.checked)}
                    />
                  </label>
                </div>

                {/* PostgREST */}
                <div className={`svc-card${apiEnabled ? " checked" : ""}${!pgEnabled ? " disabled" : ""}`}>
                  <label className="head-row">
                    <span className="icon api">
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M22 12h-4l-3 9L9 3l-3 9H2" /></svg>
                    </span>
                    <span className="info">
                      <span className="ttl">PostgREST <span className="ver">v12</span></span>
                      <p className="sub">Auto-generated REST API from your database schema.</p>
                      <span className="dep-note" style={{ display: pgEnabled ? "none" : undefined }}>
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10" /><line x1="12" y1="16" x2="12" y2="12" /><line x1="12" y1="8" x2="12.01" y2="8" /></svg>
                        Requires PostgreSQL
                      </span>
                    </span>
                    <input
                      type="checkbox"
                      className="cp-switch"
                      checked={apiEnabled}
                      disabled={!pgEnabled}
                      onChange={(e) => setApiEnabled(e.target.checked)}
                    />
                  </label>
                </div>
              </div>
            </div>

            <div className="modal-foot">
              <span className="left">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="9" /><polyline points="12 7 12 12 15 14" /></svg>
                Provisioning takes ~2 minutes
              </span>
              <div className="right">
                <button className="btn btn-ghost" onClick={closeModal}>Cancel</button>
                <button
                  className="btn btn-primary btn-loading"
                  id="cp-submit"
                  disabled={nameVal !== "valid" || isSubmitting}
                  onClick={handleSubmit}
                >
                  {isSubmitting ? (
                    <>
                      <span className="spinner" />
                      <span>Creating&hellip;</span>
                    </>
                  ) : (
                    <span className="cp-label">Create Project</span>
                  )}
                </button>
              </div>
            </div>
          </div>

          {/* Success view */}
          <div id="cp-success" style={{ display: modalView === "success" ? undefined : "none" }}>
            <div className="cp-success">
              <div className="check-circle">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12" /></svg>
              </div>
              <h3>Project created!</h3>
              <p>Redirecting to your project&hellip;</p>
            </div>
          </div>
        </div>
      </div>
    </>
  );
}

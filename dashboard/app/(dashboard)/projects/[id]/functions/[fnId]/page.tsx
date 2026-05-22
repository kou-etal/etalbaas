"use client";
import { useState, useRef, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { useFunction, useDeleteFunction, useUpdateFunction, useInvocations, type FunctionItem } from "@/features/functions/hooks";
import { useProject } from "@/features/projects/hooks";
import { formatDate, formatRelative, formatBytes } from "@/lib/utils/format";

/* ===== Inline SVG icons (per point.txt — no lucide-react) ===== */
const IconCopy = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
);
const IconCheck = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
);

/* ===== CopyBtn helper ===== */
function CopyBtn({ text, bare }: { text: string; bare?: boolean }) {
  const [copied, setCopied] = useState(false);
  const copy = async (e: React.MouseEvent) => {
    e.stopPropagation();
    try { await navigator.clipboard.writeText(text); } catch { /* noop */ }
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };
  return (
    <button className={bare ? undefined : "copy-mini"} onClick={copy} aria-label="Copy">
      {copied ? <IconCheck /> : <IconCopy />}
    </button>
  );
}

/* ===== Helpers ===== */
function kindLabel(kind: string): string {
  switch (kind) {
    case "heavy_job": return "Heavy Job";
    case "light_function": return "Light Function";
    case "gpu_inference": return "GPU Inference";
    default: return kind.replace(/_/g, " ").replace(/\b\w/g, c => c.toUpperCase());
  }
}

function kindClass(kind: string): string {
  switch (kind) {
    case "heavy_job": return "heavy-job";
    case "light_function": return "light-fn";
    case "gpu_inference": return "gpu";
    default: return "";
  }
}

function statusLabel(status: string): string {
  switch (status) {
    case "ready": return "Ready";
    case "building": return "Building";
    case "failed": return "Failed";
    case "pending": return "Pending";
    default: return status.replace(/_/g, " ").replace(/\b\w/g, c => c.toUpperCase());
  }
}

function statusClass(status: string): string {
  switch (status) {
    case "ready": return "ready";
    case "building": return "building";
    case "failed": return "error";
    case "pending": return "pending";
    default: return "";
  }
}

function fmtDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

function triggerIcon(triggerType: string) {
  switch (triggerType) {
    case "object_storage":
      return <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>;
    case "database_change":
      return <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v14c0 1.66 4.03 3 9 3s9-1.34 9-3V5"/></svg>;
    default:
      return <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M13 2 L4 14 L12 14 L11 22 L20 10 L12 10 L13 2"/></svg>;
  }
}

function triggerLabel(triggerType: string): string {
  switch (triggerType) {
    case "object_storage": return "ObjectStorage";
    case "database_change": return "Database";
    case "http": return "HTTP";
    default: return triggerType;
  }
}

/* ===== Tab types ===== */
type TabKey = "overview" | "triggers" | "invocations" | "logs" | "settings";

/* ===== Invocation type (matches clients.ts) ===== */
interface Invocation {
  id: string;
  functionId: string;
  projectId: string;
  triggerType: string;
  mode: string;
  status: string;
  errorMessage: string;
  retryCount: number;
  durationMs: number;
  coldStartMs: number;
  gpuDurationMs: number;
  memoryPeakBytes: number;
  cpuMillis: number;
  gpuProvider: string;
  gpuType: string;
  traceId: string;
  startedAt: string;
  completedAt: string;
  createdAt: string;
}

export default function FunctionDetailPage() {
  const params = useParams();
  const projectId = params.id as string;
  const fnId = params.fnId as string;
  const router = useRouter();

  /* API hooks */
  const { data: fn, isLoading } = useFunction(projectId, fnId);
  const { data: project } = useProject(projectId);
  const { data: invocations } = useInvocations(projectId, fnId) as { data: Invocation[] | undefined };
  const deleteFn = useDeleteFunction();
  const updateFn = useUpdateFunction();

  /* State */
  const [activeTab, setActiveTab] = useState<TabKey>("overview");
  const [invExpanded, setInvExpanded] = useState<Set<string>>(new Set());
  const [logLive, setLogLive] = useState(true);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deleteInput, setDeleteInput] = useState("");

  /* Settings state */
  const [fnDispName, setFnDispName] = useState("");
  const [fnDesc, setFnDesc] = useState("");
  const [fnCpu, setFnCpu] = useState("500m");
  const [fnMem, setFnMem] = useState("512Mi");
  const [fnTimeout, setFnTimeout] = useState(300);
  const [fnReplicas, setFnReplicas] = useState(3);
  const [fnConcurrency, setFnConcurrency] = useState(10);
  const [scaleToZero, setScaleToZero] = useState(true);

  /* Initialize settings from API data */
  useEffect(() => {
    if (!fn) return;
    setFnDispName(fn.displayName || "");
    setFnDesc("");
    setFnTimeout(fn.timeoutSec || 300);
  }, [fn]);

  /* Log body ref for scroll-to-bottom */
  const logBodyRef = useRef<HTMLDivElement>(null);
  const [showScrollBtn, setShowScrollBtn] = useState(false);

  useEffect(() => {
    const el = logBodyRef.current;
    if (!el) return;
    const onScroll = () => {
      const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
      setShowScrollBtn(!atBottom);
    };
    el.addEventListener("scroll", onScroll);
    // Start at bottom
    setTimeout(() => { el.scrollTop = el.scrollHeight; }, 50);
    return () => el.removeEventListener("scroll", onScroll);
  }, [activeTab]);

  const scrollToBottom = () => {
    logBodyRef.current?.scrollTo({ top: logBodyRef.current.scrollHeight, behavior: "smooth" });
  };

  const toggleInv = (id: string) => {
    setInvExpanded(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  };

  const openDeleteModal = () => {
    setDeleteInput("");
    setDeleteModalOpen(true);
  };

  /* Handlers */
  const handleDelete = async () => {
    await deleteFn.mutateAsync({ projectId, functionId: fnId });
    router.push(`/projects/${projectId}`);
  };

  const handleSaveGeneral = async () => {
    await updateFn.mutateAsync({
      projectId,
      functionId: fnId,
      displayName: fnDispName,
    });
  };

  const handleSaveRuntime = async () => {
    await updateFn.mutateAsync({
      projectId,
      functionId: fnId,
      timeoutSec: fnTimeout,
    });
  };

  const handleSaveEnvVars = async () => {
    await updateFn.mutateAsync({
      projectId,
      functionId: fnId,
      envVars: fn?.envVars || [],
    });
  };

  const triggerCount = fn?.triggers?.length || 0;

  const tabs: { key: TabKey; label: string; count?: number }[] = [
    { key: "overview", label: "Overview" },
    { key: "triggers", label: "Triggers", count: triggerCount > 0 ? triggerCount : undefined },
    { key: "invocations", label: "Invocations" },
    { key: "logs", label: "Logs" },
    { key: "settings", label: "Settings" },
  ];

  /* Loading state */
  if (isLoading || !fn) {
    return (
      <>
        <header className="topbar">
          <div className="crumbs">
            <Link href="/projects">Projects</Link>
            <span className="sep">/</span>
            <span className="skel" style={{width:80,height:14}} />
            <span className="sep">/</span>
            <span>Functions</span>
            <span className="sep">/</span>
            <span className="skel" style={{width:120,height:14}} />
          </div>
        </header>
        <main className="content">
          <div className="content-inner">
            <div className="fn-header">
              <div className="left">
                <span className="skel" style={{width:200,height:24}} />
                <span className="skel" style={{width:140,height:16,marginTop:4}} />
              </div>
            </div>
          </div>
        </main>
      </>
    );
  }

  return (
    <>
      {/* Topbar */}
      <header className="topbar">
            <div className="crumbs">
              <Link href="/projects">Projects</Link>
              <span className="sep">/</span>
              <Link href={`/projects/${projectId}`}>{project?.displayName || projectId}</Link>
              <span className="sep">/</span>
              <Link href={`/projects/${projectId}`}>Functions</Link>
              <span className="sep">/</span>
              <span className="here">{fn.name}</span>
            </div>
            <div className="right">
              <button className="btn btn-ghost">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{width:13,height:13}}><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
                Rebuild
              </button>
              <button className="btn btn-danger-ghost" onClick={openDeleteModal}>
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{width:13,height:13}}><polyline points="3 6 5 6 21 6"/><path d="M19 6l-2 14a2 2 0 0 1-2 2H9a2 2 0 0 1-2-2L5 6"/></svg>
                Delete
              </button>
              <span className="pill"><span className="dot" /><span className="mono">etalbaas.local</span></span>
            </div>
          </header>

          <main className="content">
            <div className="content-inner">

              {/* ===== Function header ===== */}
              <div className="fn-header">
                <div className="left">
                  <h1>{fn.name}</h1>
                  {fn.displayName && <span className="disp">{fn.displayName}</span>}
                  <div className="badges">
                    <span className={`kind-badge ${kindClass(fn.kind)}`}><span className="dt" style={{width:5,height:5,borderRadius:"50%",background:"currentColor",display:"inline-block"}} />{kindLabel(fn.kind)}</span>
                    <span className="mode-pill">{fn.mode}</span>
                    <span className={`status-badge ${statusClass(fn.status)}`}><span className="dt" />{statusLabel(fn.status)}</span>
                  </div>
                </div>
                <div className="right">
                  <button className="btn btn-primary">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{width:13,height:13}}><polygon points="5 3 19 12 5 21 5 3"/></svg>
                    Invoke
                  </button>
                </div>
              </div>

              {/* ===== Tab bar ===== */}
              <div className="fd-tabs" role="tablist">
                {tabs.map(t => (
                  <button
                    key={t.key}
                    className={`tab${activeTab === t.key ? " active" : ""}`}
                    role="tab"
                    onClick={() => setActiveTab(t.key)}
                  >
                    {t.label}
                    {t.count != null && (
                      <span style={{color:"var(--fg-mute)",marginLeft:4,fontSize:11}}>{t.count}</span>
                    )}
                  </button>
                ))}
              </div>

              {/* ========== Overview pane ========== */}
              {activeTab === "overview" && (
                <div className="fd-pane">
                  {/* Build information */}
                  <div className="fd-card">
                    <div className="fd-card-head"><h3>Build information</h3><p className="sub">Container image and build pipeline status.</p></div>
                    <div className="fd-card-body" style={{padding:"8px 18px"}}>
                      <div className="meta-row">
                        <span className="lbl">Image</span>
                        <span className="val" style={{minWidth:0,maxWidth:"60%"}}>
                          {fn.buildImageRef ? (
                            <>
                              <span style={{overflow:"hidden",textOverflow:"ellipsis",whiteSpace:"nowrap",display:"inline-block",maxWidth:"100%"}}>{fn.buildImageRef}</span>
                              <CopyBtn text={fn.buildImageRef} />
                            </>
                          ) : <span style={{color:"var(--fg-mute)"}}>—</span>}
                        </span>
                      </div>
                      <div className="meta-row">
                        <span className="lbl">Digest</span>
                        <span className="val">
                          {fn.buildImageDigest ? (
                            <>{fn.buildImageDigest.slice(0, 20)}{"\u2026"} <CopyBtn text={fn.buildImageDigest} /></>
                          ) : <span style={{color:"var(--fg-mute)"}}>—</span>}
                        </span>
                      </div>
                      <div className="meta-row">
                        <span className="lbl">Built at</span>
                        <span className="val">
                          {fn.lastBuiltAt ? (
                            <>{formatDate(fn.lastBuiltAt)} <span className="rel">({formatRelative(fn.lastBuiltAt)})</span></>
                          ) : <span style={{color:"var(--fg-mute)"}}>—</span>}
                        </span>
                      </div>
                      <div className="meta-row">
                        <span className="lbl">Build duration</span>
                        <span className="val">{fn.buildDurationSec ? `${fn.buildDurationSec}s` : "—"}</span>
                      </div>
                      <div className="meta-row">
                        <span className="lbl">Pipeline</span>
                        <span className="val" style={{fontFamily:"'Inter',sans-serif"}}>
                          <div className="step-timeline">
                            <span className="step"><span className="dt"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12"/></svg></span>Source uploaded</span>
                            <span className="step"><span className="sep" /></span>
                            <span className="step"><span className="dt"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12"/></svg></span>Build started</span>
                            <span className="step"><span className="sep" /></span>
                            <span className="step"><span className="dt"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12"/></svg></span>Image pushed</span>
                            <span className="step"><span className="sep" /></span>
                            <span className="step"><span className="dt"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12"/></svg></span>Deployed</span>
                          </div>
                        </span>
                      </div>
                    </div>
                  </div>

                  {/* Resource configuration */}
                  <div className="fd-card">
                    <div className="fd-card-head"><h3>Resource configuration</h3><p className="sub">Compute limits per replica.</p></div>
                    <div className="fd-card-body">
                      <div className="res-grid">
                        <div className="res-tile"><div className="lbl">CPU</div><div className="val">500m</div><div className="desc">0.5 vCPU</div></div>
                        <div className="res-tile"><div className="lbl">Memory</div><div className="val">512Mi</div><div className="desc">512 MB</div></div>
                        <div className="res-tile"><div className="lbl">Timeout</div><div className="val">{fn.timeoutSec}s</div><div className="desc">{fn.timeoutSec >= 60 ? `${Math.floor(fn.timeoutSec / 60)} min` : `${fn.timeoutSec}s`}</div></div>
                        <div className="res-tile"><div className="lbl">Replicas</div><div className="val">{`0 \u2192 3`}</div><div className="desc">Scale to zero · max 3</div></div>
                        <div className="res-tile"><div className="lbl">GPU</div><div className="val" style={{color: fn.gpuConfig ? "var(--fg)" : "var(--fg-mute)"}}>{fn.gpuConfig ? fn.gpuConfig.product : "None"}</div><div className="desc">{fn.gpuConfig ? `${fn.gpuConfig.provider} · ${fn.gpuConfig.type}` : "CPU-only workload"}</div></div>
                        <div className="res-tile"><div className="lbl">Concurrency</div><div className="val">10</div><div className="desc">Requests per replica</div></div>
                      </div>
                    </div>
                  </div>

                  {/* Environment variables */}
                  <div className="fd-card">
                    <div className="fd-card-head" style={{display:"flex",justifyContent:"space-between",alignItems:"center",flexWrap:"wrap",gap:8}}>
                      <div><h3>Environment variables</h3><p className="sub">Secrets injected into this function at runtime.</p></div>
                      <a href="#" onClick={e => { e.preventDefault(); setActiveTab("settings"); }} style={{color:"var(--fg-dim)",fontSize:12,textDecoration:"none",borderBottom:"1px dotted var(--line-2)"}}>Edit in Settings tab →</a>
                    </div>
                    <div className="fd-card-body" style={{padding:"0 18px"}}>
                      {fn.envVars && fn.envVars.length > 0 ? fn.envVars.map(ev => (
                        <div className="env-row" key={ev.name}>
                          <span className="from">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
                            {ev.name}
                          </span>
                          <span className="arrow">{"\u2192"}</span>
                          <span style={{color:"var(--fg-dim)"}}>
                            {ev.secretName ? (<>from secret <span style={{color:"var(--fg)"}}>{ev.secretName}</span></>) : (ev.value || "—")}
                          </span>
                        </div>
                      )) : (
                        <div style={{padding:"18px 0",color:"var(--fg-mute)",fontSize:13}}>No environment variables configured.</div>
                      )}
                    </div>
                  </div>
                </div>
              )}

              {/* ========== Triggers pane ========== */}
              {activeTab === "triggers" && (
                <div className="fd-pane">
                  {fn.triggers && fn.triggers.length > 0 ? fn.triggers.map((t, i) => {
                    if (t.objectStorage) {
                      return (
                        <div className="trigger-card storage" key={i}>
                          <span className="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg></span>
                          <div className="body">
                            <div className="top">
                              <div className="top-left">
                                <span className="type-pill storage">Object Storage</span>
                                {t.objectStorage.events.map(ev => <span className="evt-pill" key={ev}>{ev}</span>)}
                              </div>
                              <div className="actions">
                                <button className="btn btn-ghost small">Edit</button>
                                <button className="btn btn-danger-ghost small">Remove</button>
                              </div>
                            </div>
                            <div className="trigger-config">
                              <span className="k">Bucket</span><span className="v">{t.objectStorage.bucket}</span>
                              <span className="k">Prefix filter</span><span className="v">{t.objectStorage.prefix || "—"}</span>
                              <span className="k">Events</span><span className="v">{t.objectStorage.events.join(", ")}</span>
                            </div>
                          </div>
                        </div>
                      );
                    }
                    if (t.databaseChange) {
                      return (
                        <div className="trigger-card db" key={i}>
                          <span className="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v14c0 1.66 4.03 3 9 3s9-1.34 9-3V5"/><path d="M3 12c0 1.66 4.03 3 9 3s9-1.34 9-3"/></svg></span>
                          <div className="body">
                            <div className="top">
                              <div className="top-left">
                                <span className="type-pill db">Database Change</span>
                                {t.databaseChange.events.map(ev => <span className="evt-pill" key={ev}>{ev}</span>)}
                              </div>
                              <div className="actions">
                                <button className="btn btn-ghost small">Edit</button>
                                <button className="btn btn-danger-ghost small">Remove</button>
                              </div>
                            </div>
                            <div className="trigger-config">
                              <span className="k">Table</span><span className="v">{t.databaseChange.table}</span>
                              {t.databaseChange.includeColumns && t.databaseChange.includeColumns.length > 0 && (
                                <><span className="k">Columns</span><span className="v">{t.databaseChange.includeColumns.join(", ")}</span></>
                              )}
                              {t.databaseChange.filter && (
                                <><span className="k">Filter</span><span><span className="filter-code">{t.databaseChange.filter}</span></span></>
                              )}
                            </div>
                          </div>
                        </div>
                      );
                    }
                    return null;
                  }) : (
                    <div style={{padding:"40px 0",textAlign:"center",color:"var(--fg-mute)",fontSize:13}}>
                      <p>No triggers configured for this function.</p>
                    </div>
                  )}

                  {/* Add trigger */}
                  <button className="trigger-card add">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" style={{width:14,height:14}}><path d="M12 5v14M5 12h14"/></svg>
                    Add Trigger
                  </button>
                </div>
              )}

              {/* ========== Invocations pane ========== */}
              {activeTab === "invocations" && (
                <div className="fd-pane">
                  <div className="inv-table">
                    {/* Header row */}
                    <div className="inv-row header">
                      <span>Status</span>
                      <span>Trigger</span>
                      <span>Duration</span>
                      <span>Started</span>
                      <span>Trace</span>
                    </div>

                    {invocations && invocations.length > 0 ? invocations.map((inv: Invocation) => (
                      <div key={inv.id}>
                        <div className={`inv-row body-row${invExpanded.has(inv.id) ? " expanded" : ""}`} onClick={e => { if ((e.target as HTMLElement).closest(".copy-mini")) return; toggleInv(inv.id); }}>
                          <span><span className={`badge-s ${inv.status}`}><span className="dt" />{inv.status}</span></span>
                          <span className="trigger-cell">{triggerIcon(inv.triggerType)}{triggerLabel(inv.triggerType)}</span>
                          <span className={`duration${inv.durationMs > 3000 ? " warn" : ""}`}>{fmtDuration(inv.durationMs)}</span>
                          <span className="when">{formatRelative(inv.startedAt)}</span>
                          <span className="trace">{inv.traceId?.slice(0, 8)} <CopyBtn text={inv.traceId || ""} /></span>
                        </div>
                        <div className="inv-expand"><div className="inv-expand-inner">
                          {inv.errorMessage && (
                            <div>
                              <div className="blk-label">Error</div>
                              <pre className="json-block" style={{color:"var(--err)"}}>{inv.errorMessage}</pre>
                            </div>
                          )}
                          <div style={{display:"flex",gap:24,flexWrap:"wrap"}}>
                            {inv.memoryPeakBytes > 0 && (
                              <div style={{flex:1,minWidth:180}}>
                                <div className="blk-label">Memory used</div>
                                <span style={{fontFamily:"'JetBrains Mono',monospace",fontSize:"12.5px",color:"var(--fg)"}}>{formatBytes(inv.memoryPeakBytes)}</span>
                              </div>
                            )}
                            <div>
                              <div className="blk-label">Cold start</div>
                              <span style={{fontSize:"12.5px",color:"var(--fg-dim)"}}>{inv.coldStartMs > 0 ? `${inv.coldStartMs}ms` : "No (warm replica)"}</span>
                            </div>
                          </div>
                        </div></div>
                      </div>
                    )) : (
                      <div style={{padding:"40px 0",textAlign:"center",color:"var(--fg-mute)",fontSize:13}}>
                        <p>No invocations yet.</p>
                      </div>
                    )}
                  </div>
                </div>
              )}

              {/* ========== Logs pane (mock — API not implemented) ========== */}
              {activeTab === "logs" && (
                <div className="fd-pane">
                  <div className="log-panel">
                    <div className="log-head">
                      <h3>Logs</h3>
                      <label className={`live-toggle${logLive ? " on" : ""}`}>
                        <input
                          type="checkbox"
                          className="switch"
                          checked={logLive}
                          onChange={e => setLogLive(e.target.checked)}
                        />
                        <span className="dt-live" />
                        Live
                      </label>
                      <div className="right">
                        <div className="select-wrap">
                          <select>
                            <option>All</option>
                            <option>Info</option>
                            <option>Warn</option>
                            <option>Error</option>
                          </select>
                        </div>
                        <button className="btn btn-ghost small">Clear</button>
                        <button className="btn btn-ghost small">
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{width:11,height:11}}><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                          Download
                        </button>
                      </div>
                    </div>
                    <div className="log-body" ref={logBodyRef}>
                      <div className="log-line"><span className="ts">2026-05-15 14:32:01.234</span><span className="lvl I">I</span><span className="msg">Function cold start <span className="mute">(3.2s)</span></span></div>
                      <div className="log-line"><span className="ts">2026-05-15 14:32:01.890</span><span className="lvl I">I</span><span className="msg">Processing file: <span className="acc">uploads/photo-001.jpg</span></span></div>
                      <div className="log-line"><span className="ts">2026-05-15 14:32:02.102</span><span className="lvl I">I</span><span className="msg">{`Resizing to 1920\u00d71080\u2026`}</span></div>
                      <div className="log-line warn"><span className="ts">2026-05-15 14:32:03.445</span><span className="lvl W">W</span><span className="msg">Image exceeds 10MB, using streaming decode</span></div>
                      <div className="log-line"><span className="ts">2026-05-15 14:32:04.201</span><span className="lvl I">I</span><span className="msg">Upload complete: <span className="ok">processed/photo-001.webp (2.4MB)</span></span></div>
                      <div className="log-line"><span className="ts">2026-05-15 14:32:04.205</span><span className="lvl I">I</span><span className="msg">Invocation complete <span className="mute">(duration: 2.97s)</span></span></div>

                      <div className="log-line"><span className="ts">2026-05-15 14:38:11.512</span><span className="lvl I">I</span><span className="msg">Processing file: <span className="acc">uploads/scan-2026-05.pdf</span></span></div>
                      <div className="log-line"><span className="ts">2026-05-15 14:38:11.690</span><span className="lvl I">I</span><span className="msg">Detected MIME: <span className="mute">application/pdf</span> {"\u2014"} extracting first page</span></div>
                      <div className="log-line err"><span className="ts">2026-05-15 14:38:13.024</span><span className="lvl E">E</span><span className="msg">Failed to decode PDF: <span style={{color:"var(--err)"}}>PdfReadError: Trailer not found</span></span></div>
                      <div className="log-line err"><span className="ts">2026-05-15 14:38:13.025</span><span className="lvl E">E</span><span className="msg" style={{color:"var(--err)"}}>{"  at PdfDocument.open (pdf-lib.js:88:11)"}</span></div>
                      <div className="log-line"><span className="ts">2026-05-15 14:38:13.030</span><span className="lvl I">I</span><span className="msg">{`Falling back to image-based extraction\u2026`}</span></div>

                      <div className="log-line"><span className="ts">2026-05-15 15:02:54.001</span><span className="lvl I">I</span><span className="msg">Scaling up to 2 replicas (queue depth: 18)</span></div>
                      <div className="log-line"><span className="ts">2026-05-15 15:02:55.224</span><span className="lvl I">I</span><span className="msg">Replica <span className="mono">fn-pu-7c4d</span> ready (cold-start 411ms)</span></div>
                      <div className="log-line"><span className="ts">2026-05-15 15:03:01.812</span><span className="lvl I">I</span><span className="msg">Processed batch of <span className="mono">12</span> files in <span className="ok">8.4s</span></span></div>
                      <div className="log-line warn"><span className="ts">2026-05-15 15:18:43.660</span><span className="lvl W">W</span><span className="msg">Backpressure: dropping new events <span className="mute">(queue full)</span></span></div>
                      <div className="log-line"><span className="ts">2026-05-15 15:18:48.001</span><span className="lvl I">I</span><span className="msg">{`Queue drained \u2014 resuming normal processing`}</span></div>
                      <div className="log-line"><span className="ts">2026-05-15 15:20:00.000</span><span className="lvl I">I</span><span className="msg">Health check OK · <span className="mute">CPU: 312m / 500m · Mem: 287Mi / 512Mi</span></span></div>
                      <div className="log-line"><span className="ts">2026-05-15 16:05:11.443</span><span className="lvl I">I</span><span className="msg">Scaling down to 1 replica <span className="mute">(idle 5m)</span></span></div>
                      <button className={`scroll-bottom-btn${showScrollBtn ? " visible" : ""}`} onClick={scrollToBottom}>
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="6 9 12 15 18 9"/></svg>
                        Scroll to latest
                      </button>
                    </div>
                  </div>
                </div>
              )}

              {/* ========== Settings pane ========== */}
              {activeTab === "settings" && (
                <div className="fd-pane">
                  <div className="stg-grid">

                    {/* General */}
                    <div className="fd-card">
                      <div className="fd-card-head"><h3>General</h3><p className="sub">Display information for this function.</p></div>
                      <div className="fd-card-body" style={{display:"flex",flexDirection:"column",gap:14}}>
                        <div className="fd-field">
                          <label htmlFor="fn-disp-name">Display name</label>
                          <input type="text" id="fn-disp-name" value={fnDispName} onChange={e => setFnDispName(e.target.value)} />
                        </div>
                        <div className="fd-field">
                          <label htmlFor="fn-desc">Description</label>
                          <textarea id="fn-desc" value={fnDesc} onChange={e => setFnDesc(e.target.value)} />
                        </div>
                      </div>
                      <div className="fd-card-foot">
                        <button className="btn btn-primary" onClick={handleSaveGeneral} disabled={updateFn.isPending}>
                          {updateFn.isPending ? "Saving\u2026" : "Save"}
                        </button>
                      </div>
                    </div>

                    {/* Runtime configuration */}
                    <div className="fd-card">
                      <div className="fd-card-head"><h3>Runtime configuration</h3><p className="sub">Compute and scaling settings.</p></div>
                      <div className="fd-card-body" style={{display:"flex",flexDirection:"column",gap:14}}>
                        <div className="row2" style={{display:"grid",gridTemplateColumns:"1fr 1fr",gap:14}}>
                          <div className="fd-field">
                            <label htmlFor="fn-cpu">CPU</label>
                            <select id="fn-cpu" value={fnCpu} onChange={e => setFnCpu(e.target.value)}>
                              <option>250m</option>
                              <option>500m</option>
                              <option>1000m</option>
                              <option>2000m</option>
                            </select>
                          </div>
                          <div className="fd-field">
                            <label htmlFor="fn-mem">Memory</label>
                            <select id="fn-mem" value={fnMem} onChange={e => setFnMem(e.target.value)}>
                              <option>128Mi</option>
                              <option>256Mi</option>
                              <option>512Mi</option>
                              <option>1Gi</option>
                              <option>2Gi</option>
                            </select>
                          </div>
                        </div>
                        <div className="row2" style={{display:"grid",gridTemplateColumns:"1fr 1fr",gap:14}}>
                          <div className="fd-field">
                            <label htmlFor="fn-timeout">Timeout (seconds)</label>
                            <input type="number" id="fn-timeout" value={fnTimeout} onChange={e => setFnTimeout(Number(e.target.value))} min={1} max={900} />
                          </div>
                          <div className="fd-field">
                            <label htmlFor="fn-replicas">Max replicas</label>
                            <input type="number" id="fn-replicas" value={fnReplicas} onChange={e => setFnReplicas(Number(e.target.value))} min={1} max={100} />
                          </div>
                        </div>
                        <div className="fd-field">
                          <label htmlFor="fn-concurrency">Concurrency (requests per replica)</label>
                          <input type="number" id="fn-concurrency" value={fnConcurrency} onChange={e => setFnConcurrency(Number(e.target.value))} min={1} max={1000} />
                        </div>
                        <div className="toggle-row">
                          <label>Scale to zero</label>
                          <span className="sub">Scale down to 0 replicas when idle</span>
                          <input type="checkbox" className="stg-switch" checked={scaleToZero} onChange={e => setScaleToZero(e.target.checked)} />
                        </div>
                        <span className="fd-field">
                          <span className="helper warn">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/></svg>
                            Changes require a redeployment to take effect.
                          </span>
                        </span>
                      </div>
                      <div className="fd-card-foot">
                        <button className="btn btn-primary" onClick={handleSaveRuntime} disabled={updateFn.isPending}>
                          {updateFn.isPending ? "Saving\u2026" : "Save"}
                        </button>
                      </div>
                    </div>

                    {/* Environment variables */}
                    <div className="fd-card">
                      <div className="fd-card-head" style={{display:"flex",justifyContent:"space-between",alignItems:"center",flexWrap:"wrap",gap:8}}>
                        <div><h3>Environment variables</h3><p className="sub">Map project secrets into this function{"'"}s environment.</p></div>
                      </div>
                      <div className="fd-card-body" style={{display:"flex",flexDirection:"column",gap:12}}>
                        <div className="env-mapping">
                          {fn.envVars && fn.envVars.length > 0 ? fn.envVars.map((ev, i) => (
                            <div className="env-map-row" key={i}>
                              <div className="fd-field">
                                <input type="text" className="mono" defaultValue={ev.secretName || ev.name} readOnly />
                              </div>
                              <span className="arrow">{"\u2192"}</span>
                              <div className="fd-field">
                                <input type="text" className="mono" defaultValue={ev.name} readOnly />
                              </div>
                              <button className="remove" aria-label="Remove">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                              </button>
                            </div>
                          )) : (
                            <div style={{padding:"12px 0",color:"var(--fg-mute)",fontSize:13}}>No environment variables configured.</div>
                          )}
                        </div>
                        <button className="btn btn-ghost" style={{alignSelf:"flex-start"}}>
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{width:12,height:12}}><path d="M12 5v14M5 12h14"/></svg>
                          Add Variable
                        </button>
                      </div>
                      <div className="fd-card-foot">
                        <button className="btn btn-primary" onClick={handleSaveEnvVars} disabled={updateFn.isPending}>
                          {updateFn.isPending ? "Saving\u2026" : "Save"}
                        </button>
                      </div>
                    </div>

                    {/* Danger zone */}
                    <div className="danger-card">
                      <div className="fd-card-head"><h3>Danger zone</h3></div>
                      <div className="fd-card-body">
                        <div className="danger-row">
                          <div className="info">
                            <div className="ttl">Delete function</div>
                            <p className="sub">Permanently delete this function and all its invocation history, logs, and triggers. This action cannot be undone.</p>
                          </div>
                          <button className="btn btn-danger" onClick={openDeleteModal}>
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{width:12,height:12}}><polyline points="3 6 5 6 21 6"/><path d="M19 6l-2 14a2 2 0 0 1-2 2H9a2 2 0 0 1-2-2L5 6"/></svg>
                            Delete Function
                          </button>
                        </div>
                      </div>
                    </div>

                  </div>
                </div>
              )}

            </div>
          </main>

      {/* ===== Delete function modal ===== */}
      <div className={`modal-scrim${deleteModalOpen ? " open" : ""}`} onClick={e => { if (e.target === e.currentTarget) setDeleteModalOpen(false); }}>
        <div className="modal">
          <div className="modal-head">
            <h3 className="err">Delete function</h3>
            <button className="close" onClick={() => setDeleteModalOpen(false)} aria-label="Close">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{width:14,height:14}}><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
            </button>
          </div>
          <div className="modal-body">
            <div className="modal-err">
              <span className="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg></span>
              <p>This permanently deletes <strong className="mono">{fn.name}</strong> along with all triggers, logs, and invocation history. Any events that reference it will fail.</p>
            </div>
            <div className="field">
              <label htmlFor="delete-fn-input">Type <span className="mono" style={{color:"var(--err)",fontFamily:"'JetBrains Mono',monospace"}}>{fn.name}</span> to confirm</label>
              <input
                type="text"
                id="delete-fn-input"
                className="mono"
                placeholder="Type the function name"
                autoComplete="off"
                value={deleteInput}
                onChange={e => setDeleteInput(e.target.value)}
              />
            </div>
          </div>
          <div className="modal-foot-inner">
            <button className="btn btn-ghost" onClick={() => setDeleteModalOpen(false)}>Cancel</button>
            <button className="btn btn-danger" disabled={deleteInput !== fn.name || deleteFn.isPending} onClick={handleDelete}>
              {deleteFn.isPending ? "Deleting\u2026" : "Delete Function"}
            </button>
          </div>
        </div>
      </div>
    </>
  );
}

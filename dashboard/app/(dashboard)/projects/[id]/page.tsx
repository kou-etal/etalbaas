"use client";
import { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { useProject } from "@/features/projects/hooks";
import { useFunctions } from "@/features/functions/hooks";
import { functionClient } from "@/lib/api/clients";
import { useBuckets } from "@/features/storage/hooks";
import { useApiKeys } from "@/features/api-keys/hooks";
import { formatDate, formatRelative } from "@/lib/utils/format";

/* ===== SVG icon helpers (inline per point.txt) ===== */
const IconCopy = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
);
const IconCheck = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
);
const IconChevRight = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="9 18 15 12 9 6"/></svg>
);
const IconBolt = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
);
const IconFolder = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
);
const IconKey = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>
);
const IconEye = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
);
const IconPlay = () => (
  <svg viewBox="0 0 24 24" fill="currentColor"><polygon points="6 4 20 12 6 20"/></svg>
);
const IconPause = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><rect x="6" y="5" width="4" height="14" rx="1"/><rect x="14" y="5" width="4" height="14" rx="1"/></svg>
);
const IconKebab = () => (
  <svg viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="5" r="1.6"/><circle cx="12" cy="12" r="1.6"/><circle cx="12" cy="19" r="1.6"/></svg>
);
const IconTrash = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-2 14a2 2 0 0 1-2 2H9a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6"/><path d="M14 11v6"/></svg>
);
const IconRefresh = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
);
const IconDownload = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
);
const IconDuplicate = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6"/></svg>
);
const IconSearch = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></svg>
);
const IconRocket = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M4.5 16.5c-1.5 1.26-2 5-2 5s3.74-.5 5-2c.71-.84.7-2.13-.09-2.91a2.18 2.18 0 0 0-2.91-.09z"/><path d="M12 15l-3-3a22 22 0 0 1 2-3.95A12.88 12.88 0 0 1 22 2c0 2.72-.78 7.5-6 11a22.35 22.35 0 0 1-4 2z"/><path d="M9 12H4s.55-3.03 2-4c1.62-1.08 5 0 5 0"/><path d="M12 15v5s3.03-.55 4-2c1.08-1.62 0-5 0-5"/></svg>
);
const IconLock = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
);
const IconShield = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
);
const IconKeyAlt = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="8" cy="15" r="4"/><path d="M10.85 12.15L19 4"/><path d="M18 5l3 3"/><path d="M15 8l3 3"/></svg>
);
const IconPlus = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><path d="M12 5v14M5 12h14"/></svg>
);
const IconLogs = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>
);

type TabName = "overview" | "functions" | "events" | "storage" | "secrets" | "apikeys" | "settings";

function CopyBtn({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    try { await navigator.clipboard.writeText(text); } catch { /* noop */ }
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };
  return (
    <button className="field-btn" onClick={copy} aria-label="Copy">
      {copied ? <IconCheck /> : <IconCopy />}
    </button>
  );
}

/* ===== Hardcoded mockup data ===== */
const HC_SECRETS = [
  { name: "OPENAI_API_KEY", desc: "OpenAI API key for ML inference functions", rotated: "3 days ago", rotatedNever: false, created: "2 weeks ago" },
  { name: "DATABASE_URL", desc: "PostgreSQL connection string for direct access", rotated: "Never rotated", rotatedNever: true, created: "1 month ago" },
  { name: "STRIPE_SECRET_KEY", desc: "Stripe payment processing secret key", rotated: "12 hours ago", rotatedNever: false, created: "3 weeks ago" },
  { name: "SMTP_PASSWORD", desc: "SMTP credentials for transactional emails", rotated: "Never rotated", rotatedNever: true, created: "2 weeks ago" },
  { name: "REDIS_URL", desc: "Redis connection string for caching layer", rotated: "Never rotated", rotatedNever: true, created: "1 month ago" },
  { name: "WEBHOOK_SIGNING_SECRET", desc: "Signature verification for incoming webhooks", rotated: "1 week ago", rotatedNever: false, created: "1 month ago" },
];

const HC_APIKEYS = [
  { name: "Frontend App", prefix: "etbs_a1b2", role: "anon", created: "1 month ago", expires: "Never", expCls: "never", status: "active", rowCls: "" },
  { name: "Mobile Client", prefix: "etbs_c3d4", role: "anon", created: "2 weeks ago", expires: "in 85 days", expCls: "", status: "active", rowCls: "" },
  { name: "Backend Worker", prefix: "etbs_e5f6", role: "service_role", created: "3 weeks ago", expires: "in 67 days", expCls: "", status: "active", rowCls: "" },
  { name: "CI Pipeline", prefix: "etbs_g7h8", role: "service_role", created: "2 months ago", expires: "Expires in 3d", expCls: "soon", status: "active", rowCls: "expiring" },
  { name: "Old Frontend", prefix: "etbs_i9j0", role: "anon", created: "3 months ago", expires: "Expired 5d ago", expCls: "expired", status: "revoked", rowCls: "revoked" },
  { name: "Test Runner", prefix: "etbs_k1l2", role: "service_role", created: "1 month ago", expires: "in 60 days", expCls: "", status: "revoked", rowCls: "revoked" },
];

const HC_BUCKETS_MOCK = [
  { name: "assets", access: "public", meta: "142 files \u00b7 3.2 GB" },
  { name: "user-avatars", access: "public", meta: "1,204 files \u00b7 890 MB" },
  { name: "backups", access: "private", meta: "28 files \u00b7 14.7 GB" },
];

type EvtRetry = { cls: string; num: string; what: string; when: string; muted?: boolean };
type EvtRow = {
  trace: string; status: "delivered" | "retrying" | "failed"; sourceIcon: "db" | "stor";
  verb: string; tbl: string; fn: string; attempts: string; attCls: string;
  error: string; trShort: string; when: string;
  resultStyle: React.CSSProperties; resultMsg: React.ReactNode | null;
  errMsg?: string; retries: EvtRetry[]; payload: string | null; payloadTokens: boolean;
};

const HC_EVENTS: EvtRow[] = [
  { trace:"a1b2c3d4f5e6",status:"delivered",sourceIcon:"db",verb:"INSERT",tbl:"users",fn:"send-welcome-email",attempts:"1/1",attCls:"",error:"",trShort:"a1b2c3d4",when:"5 min ago",resultStyle:{background:"rgba(134,239,172,0.05)",borderColor:"rgba(134,239,172,0.18)",color:"var(--ok)"},resultMsg:<>Delivered in 142ms &mdash; function returned <span style={{color:"var(--fg)"}}>200 OK</span></>,retries:[{cls:"ok",num:"1",what:"Delivered",when:"5 min ago"}],payload:'{\n  "type": "db.insert",\n  "table": "users",\n  "record": {\n    "id": "u_8f2a91",\n    "email": "alex@acme.io",\n    "created_at": "2026-05-16T08:35:14Z"\n  }\n}',payloadTokens:true },
  { trace:"e5f6a7b8c9d0",status:"delivered",sourceIcon:"stor",verb:"ObjectCreated",tbl:"assets",fn:"process-upload",attempts:"1/1",attCls:"",error:"",trShort:"e5f6a7b8",when:"12 min ago",resultStyle:{background:"rgba(134,239,172,0.05)",borderColor:"rgba(134,239,172,0.18)",color:"var(--ok)"},resultMsg:<>Delivered in 318ms &mdash; function returned <span style={{color:"var(--fg)"}}>200 OK</span></>,retries:[{cls:"ok",num:"1",what:"Delivered",when:"12 min ago"}],payload:'{\n  "type": "storage.created",\n  "bucket": "assets",\n  "object": {\n    "key": "uploads/2026-05-16/avatar.png",\n    "size": 214503,\n    "content_type": "image/png"\n  }\n}',payloadTokens:true },
  { trace:"9c8d7e6f5a4b",status:"retrying",sourceIcon:"db",verb:"UPDATE",tbl:"orders",fn:"sync-inventory",attempts:"3/5",attCls:"warn",error:"connection timeout to downstream",trShort:"9c8d7e6f",when:"28 min ago",resultStyle:{},resultMsg:null,errMsg:"Error: connect ETIMEDOUT inventory-api.internal:8443\n  at TCPConnectWrap.afterConnect [as oncomplete] (net.js:1146:16)\n  context: { attempt: 3, backoff_ms: 8000 }",retries:[{cls:"err",num:"1",what:"Timeout",when:"28 min ago"},{cls:"err",num:"2",what:"Timeout",when:"20 min ago"},{cls:"warn",num:"3",what:"In flight\u2026",when:"12 min ago"},{cls:"",num:"4",what:"Scheduled",when:"in 4 min",muted:true},{cls:"",num:"5",what:"Scheduled",when:"in 12 min",muted:true}],payload:'{\n  "type": "db.update",\n  "table": "orders",\n  "old": { "status": "pending" },\n  "new": { "status": "paid" }\n}',payloadTokens:true },
  { trace:"f1e2d3c4b5a6",status:"failed",sourceIcon:"db",verb:"DELETE",tbl:"sessions",fn:"cleanup-tokens",attempts:"5/5",attCls:"err",error:"function pod OOMKilled after 512Mi",trShort:"f1e2d3c4",when:"1 hour ago",resultStyle:{},resultMsg:null,errMsg:"FunctionRuntimeError: pod OOMKilled after 512Mi\n  signal: SIGKILL (137)\n  memory_peak: 524288 KiB\n  context: { rss: 537 MB, heap_used: 491 MB, attempt: 5/5 }\n  suggestion: increase memory limit or paginate the cleanup batch",retries:[{cls:"err",num:"1",what:"OOMKilled",when:"1h 4m ago"},{cls:"err",num:"2",what:"OOMKilled",when:"59 min ago"},{cls:"err",num:"3",what:"OOMKilled",when:"52 min ago"},{cls:"err",num:"4",what:"OOMKilled",when:"38 min ago"},{cls:"err",num:"5",what:"OOMKilled \u2014 gave up",when:"1 hour ago"}],payload:'{\n  "type": "db.delete",\n  "table": "sessions",\n  "affected_rows": 14392,\n  "deleted_at": "2026-05-16T07:40:00Z"\n}',payloadTokens:true },
  { trace:"7a8b9c0d1e2f",status:"delivered",sourceIcon:"stor",verb:"ObjectCreated",tbl:"exports",fn:"generate-report",attempts:"1/1",attCls:"",error:"",trShort:"7a8b9c0d",when:"2 hours ago",resultStyle:{background:"rgba(134,239,172,0.05)",borderColor:"rgba(134,239,172,0.18)",color:"var(--ok)"},resultMsg:<>Delivered in 4.2s &mdash; produced 1 report PDF (2.1 MB)</>,retries:[{cls:"ok",num:"1",what:"Delivered",when:"2 hours ago"}],payload:null,payloadTokens:false },
  { trace:"b4c5d6e7f8a9",status:"delivered",sourceIcon:"db",verb:"INSERT",tbl:"invoices",fn:"send-receipt",attempts:"2/5",attCls:"warn",error:"",trShort:"b4c5d6e7",when:"3 hours ago",resultStyle:{background:"rgba(134,239,172,0.05)",borderColor:"rgba(134,239,172,0.18)",color:"var(--ok)"},resultMsg:<>Delivered on attempt 2 after transient SMTP rate-limit</>,retries:[{cls:"err",num:"1",what:"SMTP 421 rate limit",when:"3h 2m ago"},{cls:"ok",num:"2",what:"Delivered",when:"3 hours ago"}],payload:null,payloadTokens:false },
  { trace:"2d3e4f5a6b7c",status:"delivered",sourceIcon:"stor",verb:"ObjectRemoved",tbl:"tmp",fn:"cleanup-orphans",attempts:"1/1",attCls:"",error:"",trShort:"2d3e4f5a",when:"5 hours ago",resultStyle:{background:"rgba(134,239,172,0.05)",borderColor:"rgba(134,239,172,0.18)",color:"var(--ok)"},resultMsg:<>Delivered in 89ms &mdash; 1 object cleaned</>,retries:[{cls:"ok",num:"1",what:"Delivered",when:"5 hours ago"}],payload:null,payloadTokens:false },
  { trace:"6f7a8b9c0d1e",status:"failed",sourceIcon:"db",verb:"INSERT",tbl:"payments",fn:"process-payment",attempts:"5/5",attCls:"err",error:"Stripe API: card_declined (4xx)",trShort:"6f7a8b9c",when:"6 hours ago",resultStyle:{},resultMsg:null,errMsg:"StripeCardError: Your card was declined.\n  code: card_declined\n  decline_code: insufficient_funds\n  http_status: 402\n  charge_id: ch_3Pq7xT2eZvKYlo2C0a4Z1xN3\n  context: { attempt: 5/5, idempotency_key: pay_8f2a91 }\n  notes: non-retriable \u2014 moved to dead-letter queue",retries:[{cls:"err",num:"1",what:"card_declined",when:"6h 4m ago"},{cls:"err",num:"2",what:"card_declined",when:"5h 58m ago"},{cls:"err",num:"3",what:"card_declined",when:"5h 50m ago"},{cls:"err",num:"4",what:"card_declined",when:"5h 28m ago"},{cls:"err",num:"5",what:"card_declined \u2014 moved to DLQ",when:"6 hours ago"}],payload:'{\n  "type": "db.insert",\n  "table": "payments",\n  "record": {\n    "id": "pay_8f2a91",\n    "amount_cents": 12900,\n    "currency": "usd",\n    "card_last4": "4242",\n    "status": null\n  }\n}',payloadTokens:true },
];

type FileItem = { file: string; folder: boolean; sub?: string; size: string; sizeEmpty: boolean; type: string; mod: string; iconCls?: string; hasUrl?: boolean };
const HC_FILES_LIST: FileItem[] = [
  { file:"images/",folder:true,sub:"48 files",size:"\u2014",sizeEmpty:true,type:"folder",mod:"2 days ago" },
  { file:"documents/",folder:true,sub:"12 files",size:"\u2014",sizeEmpty:true,type:"folder",mod:"1 week ago" },
  { file:"hero-banner.webp",folder:false,size:"2.4 MB",sizeEmpty:false,type:"image/webp",mod:"3 hours ago",iconCls:"thumb",hasUrl:true },
  { file:"logo-dark.svg",folder:false,size:"12 KB",sizeEmpty:false,type:"image/svg+xml",mod:"5 days ago",iconCls:"thumb green",hasUrl:true },
  { file:"terms-of-service.pdf",folder:false,size:"340 KB",sizeEmpty:false,type:"application/pdf",mod:"2 weeks ago",iconCls:"doc",hasUrl:true },
  { file:"demo-recording.mp4",folder:false,size:"48.2 MB",sizeEmpty:false,type:"video/mp4",mod:"1 month ago",iconCls:"video",hasUrl:true },
  { file:"data-export.csv.gz",folder:false,size:"8.7 MB",sizeEmpty:false,type:"application/gzip",mod:"3 days ago",iconCls:"archive",hasUrl:true },
];

type GridItem = { file: string; folder: boolean; previewCls: string; sizeBadge: string; docIcon?: boolean; videoIcon?: boolean; archiveIcon?: boolean };
const HC_FILES_GRID: GridItem[] = [
  { file:"images/",folder:true,previewCls:"",sizeBadge:"" },
  { file:"documents/",folder:true,previewCls:"",sizeBadge:"" },
  { file:"hero-banner.webp",folder:false,previewCls:"violet",sizeBadge:"2.4 MB" },
  { file:"logo-dark.svg",folder:false,previewCls:"green",sizeBadge:"12 KB" },
  { file:"terms-of-service.pdf",folder:false,previewCls:"",sizeBadge:"340 KB",docIcon:true },
  { file:"demo-recording.mp4",folder:false,previewCls:"warm",sizeBadge:"48.2 MB",videoIcon:true },
  { file:"data-export.csv.gz",folder:false,previewCls:"",sizeBadge:"8.7 MB",archiveIcon:true },
];

export default function ProjectDetailPage() {
  const params = useParams();
  const router = useRouter();
  const projectId = params.id as string;
  const { data: project, isLoading } = useProject(projectId);
  const { data: functions } = useFunctions(projectId);
  const { data: buckets } = useBuckets(projectId);
  const { data: apiKeys } = useApiKeys(projectId);

  const [activeTab, setActiveTab] = useState<TabName>("overview");
  const [menuOpen, setMenuOpen] = useState(false);
  const [revealAnon, setRevealAnon] = useState(false);
  const [revealService, setRevealService] = useState(false);

  /* Modal open states */
  const [pauseModalOpen, setPauseModalOpen] = useState(false);
  const [deleteProjectModalOpen, setDeleteProjectModalOpen] = useState(false);
  const [addSecretModalOpen, setAddSecretModalOpen] = useState(false);
  const [rotateSecretModalOpen, setRotateSecretModalOpen] = useState(false);
  const [deleteSecretModalOpen, setDeleteSecretModalOpen] = useState(false);
  const [createKeyModalOpen, setCreateKeyModalOpen] = useState(false);
  const [revokeKeyModalOpen, setRevokeKeyModalOpen] = useState(false);
  const [deployFnModalOpen, setDeployFnModalOpen] = useState(false);

  /* Delete project */
  const [deleteProjectInput, setDeleteProjectInput] = useState("");
  const [deleteProjectAck, setDeleteProjectAck] = useState(false);

  /* Create key */
  const [createKeyStep, setCreateKeyStep] = useState<1 | 2>(1);
  const [createKeyName, setCreateKeyName] = useState("");
  const [createKeyRole, setCreateKeyRole] = useState<"anon" | "service_role">("anon");
  const [createKeyExp, setCreateKeyExp] = useState("90");

  /* Revoke key */
  const [revokeKeyName, setRevokeKeyName] = useState("");
  const [revokeKeyPrefix, setRevokeKeyPrefix] = useState("");
  const [revokeKeyRole, setRevokeKeyRole] = useState("");
  const [revokeConfirmInput, setRevokeConfirmInput] = useState("");

  /* Secrets */
  const [targetSecretName, setTargetSecretName] = useState("");
  const [addSecretName, setAddSecretName] = useState("");
  const [addSecretValue, setAddSecretValue] = useState("");
  const [addSecretDesc, setAddSecretDesc] = useState("");
  const [rotateSecretValue, setRotateSecretValue] = useState("");
  const [deleteSecretConfirmInput, setDeleteSecretConfirmInput] = useState("");
  const [secValueRevealed, setSecValueRevealed] = useState(false);
  const [rotateValueRevealed, setRotateValueRevealed] = useState(false);

  /* Events */
  const [evtFilter, setEvtFilter] = useState("all");
  const [evtExpanded, setEvtExpanded] = useState<Set<string>>(new Set());
  const [liveToggle, setLiveToggle] = useState(true);

  /* Storage */
  const [activeBucket, setActiveBucket] = useState(0);
  const [storView, setStorView] = useState<"list" | "grid">("list");
  const [uploadOverlayOpen, setUploadOverlayOpen] = useState(false);
  const [fileDetailOpen, setFileDetailOpen] = useState(false);

  /* Deploy Function */
  const [fnName, setFnName] = useState("");
  const [fnDisplayName, setFnDisplayName] = useState("");
  const [fnKind, setFnKind] = useState("light-deployment");
  const [fnMode, setFnMode] = useState("sync");
  const [sourceTab, setSourceTab] = useState("inline");
  const [inlineFilename, setInlineFilename] = useState("main.py");
  const [inlineCode, setInlineCode] = useState("def handler(event, context):\n    body = event.get('body', {})\n    print(f\"processing: {body}\")\n    return {'ok': True}");
  const [gitUrl, setGitUrl] = useState("");
  const [gitBranch, setGitBranch] = useState("main");
  const [gitSubpath, setGitSubpath] = useState("/");
  const [zipPath, setZipPath] = useState("");
  const [runtimeTab, setRuntimeTab] = useState("preset");
  const [runtimePreset, setRuntimePreset] = useState("python-3.12");
  const [runtimeDeps, setRuntimeDeps] = useState("");
  const [customDockerfile, setCustomDockerfile] = useState("");
  const [fnTimeout, setFnTimeout] = useState(30);
  const [gpuEnabled, setGpuEnabled] = useState(false);
  const [gpuType, setGpuType] = useState("A100");
  const [gpuProvider, setGpuProvider] = useState("runpod");
  const [gpuProduct, setGpuProduct] = useState("serverless");
  const [dfmTriggers, setDfmTriggers] = useState<Array<{ id: number; type: string; table: string; dbEvents: string[]; filter: string; bucket: string; storageEvents: string[]; prefix: string }>>([]);
  const [dfmTriggerSeq, setDfmTriggerSeq] = useState(0);
  const [dfmEnvVars, setDfmEnvVars] = useState<Array<{ id: number; name: string; source: string; value: string; secretName: string }>>([]);
  const [dfmEnvSeq, setDfmEnvSeq] = useState(0);
  const [fnDeploying, setFnDeploying] = useState(false);

  /* Deploy Function — computed */
  const dfmGpuProviderNames: Record<string, string> = { runpod: "RunPod", lambda: "Lambda Labs", sakura: "Sakura", self: "Self-managed" };
  const dfmTimeoutHint = (() => {
    if (fnKind === "light-deployment") return { text: "Max: 30s for Light Deployment", cls: "hint", max: 30 };
    if (gpuEnabled) return { text: "No timeout limit with GPU", cls: "hint ok", max: 86400 };
    if (fnKind === "heavy-job") return { text: "Max: 3600s for Heavy Job", cls: "hint", max: 3600 };
    return { text: "Max: 600s for Heavy Deployment", cls: "hint", max: 600 };
  })();
  const dfmGpuStatusText = gpuEnabled ? `On — ${gpuType} / ${dfmGpuProviderNames[gpuProvider] || gpuProvider}` : "Off";
  const dfmNameValid = /^[a-z][a-z0-9-]{2,39}$/.test(fnName);

  /* Deploy Function — handlers */
  const addDfmTrigger = () => {
    const id = dfmTriggerSeq + 1;
    setDfmTriggerSeq(id);
    setDfmTriggers(prev => [...prev, { id, type: "db", table: "", dbEvents: [], filter: "", bucket: "", storageEvents: [], prefix: "" }]);
  };
  const removeDfmTrigger = (id: number) => setDfmTriggers(prev => prev.filter(t => t.id !== id));
  const updateDfmTrigger = (id: number, u: Record<string, unknown>) => setDfmTriggers(prev => prev.map(t => t.id === id ? { ...t, ...u } as typeof t : t));
  const toggleDfmTriggerEvent = (id: number, field: "dbEvents" | "storageEvents", evt: string) => {
    setDfmTriggers(prev => prev.map(t => {
      if (t.id !== id) return t;
      const arr = t[field];
      return { ...t, [field]: arr.includes(evt) ? arr.filter((e: string) => e !== evt) : [...arr, evt] };
    }));
  };
  const addDfmEnvVar = () => {
    const id = dfmEnvSeq + 1;
    setDfmEnvSeq(id);
    setDfmEnvVars(prev => [...prev, { id, name: "", source: "value", value: "", secretName: "OPENAI_API_KEY" }]);
  };
  const removeDfmEnvVar = (id: number) => setDfmEnvVars(prev => prev.filter(ev => ev.id !== id));
  const updateDfmEnvVar = (id: number, u: Record<string, unknown>) => setDfmEnvVars(prev => prev.map(ev => ev.id === id ? { ...ev, ...u } as typeof ev : ev));
  const handleDfmKindChange = (kind: string) => {
    setFnKind(kind);
    if (kind === "light-deployment" && fnTimeout > 30) setFnTimeout(30);
  };
  const handleDfmDeploy = async () => {
    setFnDeploying(true);
    try {
      const req: Record<string, unknown> = {
        projectId, name: fnName, displayName: fnDisplayName || fnName,
        kind: fnKind, mode: fnMode, timeoutSec: fnTimeout,
      };
      if (sourceTab === "inline") req.inlineSource = { code: inlineCode, filename: inlineFilename };
      else if (sourceTab === "git") req.gitSource = { repoUrl: gitUrl, branch: gitBranch, subpath: gitSubpath };
      if (runtimeTab === "preset") {
        const deps = runtimeDeps.trim() ? runtimeDeps.trim().split("\n").map(s => s.trim()).filter(Boolean) : undefined;
        req.presetRuntime = { preset: runtimePreset, ...(deps ? { requirements: deps } : {}) };
      } else req.customRuntime = { dockerfile: customDockerfile };
      if (gpuEnabled) req.gpuConfig = { type: gpuType, provider: gpuProvider, product: gpuProduct };
      if (dfmTriggers.length > 0) req.triggers = dfmTriggers.map(t => t.type === "db"
        ? { databaseChange: { table: t.table, events: t.dbEvents, ...(t.filter ? { filter: t.filter } : {}) } }
        : { objectStorage: { bucket: t.bucket, events: t.storageEvents, prefix: t.prefix } });
      if (dfmEnvVars.length > 0) req.envVars = dfmEnvVars.map(ev => ev.source === "secret"
        ? { name: ev.name, secretName: ev.secretName } : { name: ev.name, value: ev.value });
      await functionClient.createFunction(req as unknown as Parameters<typeof functionClient.createFunction>[0]);
      setFnName(""); setFnDisplayName(""); setFnKind("light-deployment"); setFnMode("sync");
      setSourceTab("inline"); setInlineFilename("main.py");
      setInlineCode("def handler(event, context):\n    body = event.get('body', {})\n    print(f\"processing: {body}\")\n    return {'ok': True}");
      setGitUrl(""); setGitBranch("main"); setGitSubpath("/"); setZipPath("");
      setRuntimeTab("preset"); setRuntimePreset("python-3.12"); setRuntimeDeps(""); setCustomDockerfile("");
      setFnTimeout(30); setGpuEnabled(false); setGpuType("A100"); setGpuProvider("runpod"); setGpuProduct("serverless");
      setDfmTriggers([]); setDfmTriggerSeq(0); setDfmEnvVars([]); setDfmEnvSeq(0);
      setDeployFnModalOpen(false);
    } catch { /* best-effort */ }
    setFnDeploying(false);
  };

  /* Secrets banner */
  const [secBannerDismissed, setSecBannerDismissed] = useState(false);

  /* Settings */
  const [settingsSection, setSettingsSection] = useState<"general" | "services" | "danger">("general");
  const [stgName, setStgName] = useState("");
  const [stgDesc, setStgDesc] = useState("");
  const [svcPg, setSvcPg] = useState(true);
  const [svcRd, setSvcRd] = useState(true);
  const [svcApi, setSvcApi] = useState(true);

  // Derived data
  const displayName = project?.displayName || projectId;
  const status = project?.status || "pending";
  const description = project?.description || "";
  const apiEndpoint = `https://${projectId}.etalbaas.io`;
  const anonKey = apiKeys?.find((k) => k.role === "anon") ?? { keyPrefix: "etbs_a1b2_", role: "anon" as const };
  const serviceKey = apiKeys?.find((k) => k.role === "service_role") ?? { keyPrefix: "etbs_s8k7_", role: "service_role" as const };
  const fnCount = functions?.length ?? 0;
  const fnReady = functions?.filter((f) => f.status === "ready").length ?? 0;
  const fnBuilding = functions?.filter((f) => f.status === "building").length ?? 0;
  const fnOther = fnCount - fnReady - fnBuilding;
  const bucketCount = buckets?.length ?? 0;
  const bucketPublic = buckets?.filter((b) => b.accessLevel === "public").length ?? 0;
  const bucketPrivate = bucketCount - bucketPublic;
  const akCount = apiKeys?.length ?? 0;

  // Close kebab on outside click
  useEffect(() => {
    if (!menuOpen) return;
    const handler = () => setMenuOpen(false);
    document.addEventListener("click", handler);
    return () => document.removeEventListener("click", handler);
  }, [menuOpen]);

  const toggleEvtExpanded = (trace: string) => {
    setEvtExpanded(prev => {
      const next = new Set(prev);
      if (next.has(trace)) next.delete(trace); else next.add(trace);
      return next;
    });
  };

  const tabs: { key: TabName; label: string; count?: number }[] = [
    { key: "overview", label: "Overview" },
    { key: "functions", label: "Functions", count: fnCount || undefined },
    { key: "events", label: "Events" },
    { key: "storage", label: "Storage", count: bucketCount || undefined },
    { key: "secrets", label: "Secrets" },
    { key: "apikeys", label: "API Keys", count: akCount || undefined },
    { key: "settings", label: "Settings" },
  ];

  if (isLoading) {
    return (
      <div style={{ padding: "24px 28px 80px" }}>
        <div className="pd-content-inner">
          <div className="panel" style={{ padding: "22px 24px" }}>
            <div className="skel-block" style={{ width: 120, height: 22, borderRadius: 999, marginBottom: 12 }} />
            <div className="skel-block" style={{ width: 220, height: 24, marginBottom: 8 }} />
            <div className="skel-block" style={{ width: 140, height: 12, marginBottom: 18 }} />
            <div className="skel-block" style={{ width: "90%", maxWidth: 520, height: 12, marginBottom: 6 }} />
            <div className="skel-block" style={{ width: "70%", maxWidth: 380, height: 12 }} />
          </div>
          <div className="pd-stats">
            {[0,1,2].map(i => (
              <div className="pd-stat" key={i}>
                <div className="skel-block" style={{ width: 90, height: 13, marginBottom: 14 }} />
                <div className="skel-block" style={{ width: 60, height: 28, marginBottom: 6 }} />
                <div className="skel-block" style={{ width: 120, height: 12 }} />
              </div>
            ))}
          </div>
          <div className="grid-2">
            <div className="panel" style={{ padding: "14px 18px" }}>
              <div className="skel-block" style={{ width: 140, height: 14, marginBottom: 18 }} />
              {[0,1,2,3].map(i => (
                <div className="skel-block" key={i} style={{ width: `${100 - i * 5}%`, height: 14, marginBottom: 14 }} />
              ))}
            </div>
            <div className="panel" style={{ padding: "14px 18px" }}>
              <div className="skel-block" style={{ width: 160, height: 14, marginBottom: 18 }} />
              {[0,1,2].map(i => (
                <div className="skel-block" key={i} style={{ width: "100%", height: 30, marginBottom: 12 }} />
              ))}
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <>
      {/* Top bar */}
      <header className="topbar">
        <nav className="crumbs">
          <Link href="/projects">Projects</Link>
          <span className="sep">/</span>
          <strong>{displayName}</strong>
        </nav>
        <div className="right">
          <span className="pill"><span className="dot" /><span className="mono">etalbaas.local</span></span>
          <button className="btn btn-ghost" onClick={() => setPauseModalOpen(true)}>
            <IconPause />
            <span className="label">Pause</span>
          </button>
          <div className="menu-wrap">
            <button
              className="btn btn-ghost btn-icon"
              aria-label="More actions"
              aria-haspopup="menu"
              aria-expanded={menuOpen}
              onClick={(e) => { e.stopPropagation(); setMenuOpen(!menuOpen); }}
            >
              <IconKebab />
            </button>
            <div className={`menu ${menuOpen ? "open" : ""}`} role="menu">
              <div className="menu-item"><IconDuplicate /> Duplicate project</div>
              <div className="menu-item"><IconDownload /> Export config</div>
              <div className="menu-divider" />
              <div className="menu-item danger" onClick={() => { setMenuOpen(false); setDeleteProjectModalOpen(true); }}><IconTrash /> Delete project</div>
            </div>
          </div>
        </div>
      </header>

      {/* Tabs */}
      <nav className="tabs" role="tablist">
        {tabs.map(t => (
          <button
            key={t.key}
            className={`tab ${activeTab === t.key ? "active" : ""}`}
            role="tab"
            aria-selected={activeTab === t.key}
            onClick={() => setActiveTab(t.key)}
          >
            {t.label}
            {t.count !== undefined && <span className="badge-count">{t.count}</span>}
          </button>
        ))}
      </nav>

      <div style={{ padding: "24px 28px 80px" }}>
        <div className="pd-content-inner">

          {/* ====== Overview tab ====== */}
          {activeTab === "overview" && (
            <>
              <section className={`status-card state-${status}`}>
                <div className="status-head">
                  <div className="left">
                    <div className="title-row">
                      <h1>{displayName}</h1>
                      <span className={`badge-lg ${status}`}>
                        <span className="bd" /><span className="text">{status.charAt(0).toUpperCase() + status.slice(1)}</span>
                      </span>
                    </div>
                    <div className="id-row">
                      <span className="mono">id:</span>
                      <span className="id">{projectId}</span>
                      <CopyBtn text={projectId} />
                    </div>
                    {description && <p className="status-desc">{description}</p>}
                  </div>
                </div>
                <div className="meta-grid">
                  <div><span className="lbl">Created</span><span className="val">{project?.createdAt ? formatDate(project.createdAt) : "\u2014"}</span></div>
                  <div><span className="lbl">Last updated</span><span className="val">{project?.updatedAt ? formatRelative(project.updatedAt) : "\u2014"}</span></div>
                  <div><span className="lbl">Region</span><span className="val mono" style={{ fontSize: 12 }}>self-hosted</span></div>
                  <div><span className="lbl">Plan</span><span className="val">Self-hosted &middot; Unlimited</span></div>
                  <div style={{ gridColumn: "1 / -1" }}>
                    <span className="lbl" style={{ marginBottom: 8 }}>Enabled services</span>
                    <div className="services-row">
                      {project?.postgresEnabled && (
                        <span className="svc-pill"><IconCheck /> PostgreSQL 16
                          {project.postgresExtensions && project.postgresExtensions.length > 0 && (
                            <span className="ext-pills">{project.postgresExtensions.map(ext => (<span key={ext} className="ext-pill">{ext}</span>))}</span>
                          )}
                        </span>
                      )}
                      {project?.redisEnabled && (<span className="svc-pill"><IconCheck /> Redis 7.2</span>)}
                      {project?.postgrestEnabled && (<span className="svc-pill"><IconCheck /> PostgREST 12</span>)}
                      {!project?.postgresEnabled && !project?.redisEnabled && !project?.postgrestEnabled && (
                        <span style={{ fontSize: 12, color: "var(--fg-mute)" }}>No services enabled</span>
                      )}
                    </div>
                  </div>
                </div>
              </section>

              <section className="pd-stats">
                <a className="pd-stat" onClick={() => setActiveTab("functions")} style={{ cursor: "pointer" }}>
                  <div className="head">
                    <span className="lbl"><svg className="icon-glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>Functions</span>
                    <svg className="arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="9 18 15 12 9 6"/></svg>
                  </div>
                  <div className="num">{fnCount}</div>
                  <div className="sub">{fnCount > 0 ? [fnReady > 0 && `${fnReady} ready`, fnBuilding > 0 && `${fnBuilding} building`, fnOther > 0 && `${fnOther} other`].filter(Boolean).join(", ") : "No functions deployed"}</div>
                </a>
                <a className="pd-stat" onClick={() => setActiveTab("storage")} style={{ cursor: "pointer" }}>
                  <div className="head">
                    <span className="lbl"><svg className="icon-glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>Storage buckets</span>
                    <svg className="arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="9 18 15 12 9 6"/></svg>
                  </div>
                  <div className="num">{bucketCount}</div>
                  <div className="sub">{bucketCount > 0 ? `${bucketPublic} public, ${bucketPrivate} private` : "No buckets"}</div>
                </a>
                <a className="pd-stat" onClick={() => setActiveTab("apikeys")} style={{ cursor: "pointer" }}>
                  <div className="head">
                    <span className="lbl"><svg className="icon-glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>API keys</span>
                    <svg className="arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="9 18 15 12 9 6"/></svg>
                  </div>
                  <div className="num">{akCount}</div>
                  <div className="sub">{akCount > 0 ? `${akCount} active` : "No API keys"}</div>
                </a>
              </section>

              <section className="grid-2">
                <div className="panel">
                  <div className="panel-head">
                    <h3>Recent activity</h3>
                    <div className="right"><a onClick={() => setActiveTab("events")} style={{ cursor: "pointer" }}>View all events<IconChevRight /></a></div>
                  </div>
                  <ul className="activity">
                    <li><div className="act-icon fn"><IconPlay /></div><div className="act-text"><div className="desc">Function <span className="mono">process-upload</span> triggered</div></div><span className="badge-s delivered">Delivered</span><span className="act-when">5 min ago</span></li>
                    <li><div className="act-icon key"><IconKeyAlt /></div><div className="act-text"><div className="desc">API key <span className="mono">etbs_a1b2&hellip;</span> created</div></div><span className="badge-s created">Created</span><span className="act-when">42 min ago</span></li>
                    <li><div className="act-icon fn"><IconPlay /></div><div className="act-text"><div className="desc">Function <span className="mono">send-receipt</span> failed</div></div><span className="badge-s failed">Failed</span><span className="act-when">1 hour ago</span></li>
                    <li><div className="act-icon bkt"><IconFolder /></div><div className="act-text"><div className="desc">Bucket <span className="mono">user-avatars</span> created</div></div><span className="badge-s created">Created</span><span className="act-when">3 hours ago</span></li>
                  </ul>
                </div>
                <div className="panel connect-card">
                  <div className="panel-head"><h3>Connect to your project</h3></div>
                  <div className="row"><span className="row-lbl">API endpoint</span><div className="connect-field"><span className="value">{apiEndpoint}</span><span className="field-btns"><CopyBtn text={apiEndpoint} /></span></div></div>
                  {anonKey && (
                    <div className="row"><span className="row-lbl">Anon key <span className="role-tag anon">public</span></span><div className="connect-field"><span className={`value ${revealAnon ? "" : "masked"}`}>{revealAnon ? `${anonKey.keyPrefix}XXXXXXXXXXXXXXXXXX` : `${anonKey.keyPrefix}\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022`}</span><span className="field-btns"><button className="field-btn" onClick={() => setRevealAnon(!revealAnon)} aria-label="Reveal key"><IconEye /></button><CopyBtn text={anonKey.keyPrefix} /></span></div></div>
                  )}
                  {serviceKey && (
                    <div className="row"><span className="row-lbl">Service role key <span className="role-tag">secret</span></span><div className="connect-field"><span className={`value ${revealService ? "" : "masked"}`}>{revealService ? `${serviceKey.keyPrefix}XXXXXXXXXXXXXXXXXX` : `${serviceKey.keyPrefix}\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022`}</span><span className="field-btns"><button className="field-btn" onClick={() => setRevealService(!revealService)} aria-label="Reveal key"><IconEye /></button><CopyBtn text={serviceKey.keyPrefix} /></span></div></div>
                  )}
                  <details className="snippet">
                    <summary><svg className="chev" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><polyline points="9 18 15 12 9 6"/></svg>Quick start &mdash; JavaScript</summary>
                    <pre><span className="tok-com">{"// install: npm i @etalbaas/js"}</span>{"\n"}<span className="tok-kw">import</span>{" { "}<span className="tok-fn">createClient</span>{" } "}<span className="tok-kw">from</span>{" "}<span className="tok-str">{`'@etalbaas/js'`}</span>{"\n\n"}<span className="tok-kw">const</span>{" client = "}<span className="tok-fn">createClient</span>{"(\n  "}<span className="tok-str">{`'${apiEndpoint}'`}</span>{",\n  "}<span className="tok-str">{`'your-anon-key'`}</span>{"\n)\n\n"}<span className="tok-kw">const</span>{" { data, error } = "}<span className="tok-kw">await</span>{" client\n  ."}<span className="tok-fn">from</span>{"("}<span className="tok-str">{`'users'`}</span>{")\n  ."}<span className="tok-fn">select</span>{"("}<span className="tok-str">{`'*'`}</span>{")\n  ."}<span className="tok-fn">limit</span>{"("}<span className="tok-str">10</span>{")"}</pre>
                  </details>
                </div>
              </section>
            </>
          )}

          {/* ====== Functions tab ====== */}
          {activeTab === "functions" && (
            <>
              {fnCount > 0 ? (
                <>
                  <header className="tab-head">
                    <div><h2>Functions</h2><p>Deploy and manage serverless functions for this project.</p></div>
                    <div className="actions"><button className="btn btn-primary" onClick={() => setDeployFnModalOpen(true)}><IconRocket />Deploy Function<span className="kbd-inline">D</span></button></div>
                  </header>
                  <div className="pd-filters">
                    <label className="search-input"><IconSearch /><input type="text" placeholder="Search functions\u2026" aria-label="Search functions" /><span className="kbd-hint">/</span></label>
                    <div className="select-wrap"><select aria-label="Kind filter"><option>All kinds</option><option>Heavy Job</option><option>Heavy Deploy</option><option>Light Deploy</option></select></div>
                    <div className="select-wrap"><select aria-label="Status filter"><option>All statuses</option><option>Ready</option><option>Building</option><option>Pending</option><option>Failed</option></select></div>
                  </div>
                  <div className="panel">
                    <div className="fn-table" role="table">
                      <div className="row header" role="row">
                        <input type="checkbox" className="checkbox" aria-label="Select all" />
                        <button className="sortable sorted">Name <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round"><polyline points="6 9 12 15 18 9"/></svg></button>
                        <span>Kind</span><span>Mode</span><span>Status</span><span>Triggers</span><span>Last built</span><span style={{ textAlign: "right" }}>Actions</span>
                      </div>
                      {functions?.map(fn => (
                        <div key={fn.name} className={`row body-row ${fn.status === "failed" ? "failed-row" : ""}`} role="row" tabIndex={0} onClick={() => router.push(`/projects/${projectId}/functions/${fn.id}`)} style={{ cursor: "pointer" }}>
                          <input type="checkbox" className="checkbox row-check" aria-label="Select function" />
                          <div className="cell-name"><span className="nm">{fn.name}</span><span className="disp">{fn.displayName || fn.name}</span></div>
                          <span className={`kind-badge ${fn.kind === "heavy_job" ? "heavy-job" : fn.kind === "heavy_deployment" ? "heavy-deployment" : "light-deployment"}`}><span className="dt" />{fn.kind === "heavy_job" ? "Heavy Job" : fn.kind === "heavy_deployment" ? "Heavy Deploy" : "Light Deploy"}</span>
                          <span className="mode-cell">{fn.mode || "async"}</span>
                          <span className="status-cell"><span className={`badge-s ${fn.status === "ready" ? "delivered" : fn.status === "building" ? "created" : fn.status === "failed" ? "failed" : "retrying"}`} style={{ fontWeight: 500 }}><span className="bd" style={{ width: 6, height: 6, borderRadius: "50%", background: "currentColor", display: "inline-block" }} />{fn.status.charAt(0).toUpperCase() + fn.status.slice(1)}</span></span>
                          <span className="triggers-cell"><span className="tg-mute">&mdash;</span></span>
                          <span className={`built-cell ${fn.status === "building" ? "building" : fn.status === "failed" ? "failed-text" : ""}`}>
                            {fn.status === "building" ? (<><span className="mini-spinner" /><span className="progress-bar" /></>) : fn.lastBuiltAt ? formatRelative(fn.lastBuiltAt) : (<span className="mono" style={{ color: "var(--fg-mute)" }}>Never</span>)}
                          </span>
                          <span className="actions-cell">
                            <button className="action-btn" aria-label="View logs"><IconLogs /></button>
                            <button className="action-btn" aria-label="Rebuild"><IconRefresh /></button>
                            <button className="action-btn danger" aria-label="Delete"><IconTrash /></button>
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                </>
              ) : (
                <>
                  <header className="tab-head"><div><h2>Functions</h2><p>Deploy and manage serverless functions for this project.</p></div><div className="actions"><button className="btn btn-primary" onClick={() => setDeployFnModalOpen(true)}><IconRocket /> Deploy Function <span className="kbd-inline">D</span></button></div></header>
                  <div className="fn-empty">
                    <div className="bolt"><svg viewBox="0 0 24 24" fill="currentColor"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg></div>
                    <h2>No functions deployed</h2><p>Deploy your first function to handle events, process data, or serve APIs.</p>
                    <button className="btn btn-primary" onClick={() => setDeployFnModalOpen(true)}><IconRocket /> Deploy Function</button>
                  </div>
                </>
              )}
            </>
          )}

          {/* ====== Events tab ====== */}
          {activeTab === "events" && (
            <>
              <header className="tab-head">
                <div><h2>Event history</h2><p>Track trigger events and function invocations.</p></div>
                <div className="actions">
                  <label className={`live-toggle ${liveToggle ? "on" : ""}`}>
                    <input type="checkbox" className="switch" checked={liveToggle} onChange={() => setLiveToggle(!liveToggle)} />
                    <span className="dot-live"></span>
                    <span className="live-label">Live</span>
                  </label>
                  <button className="btn btn-ghost"><IconDownload /> Export CSV</button>
                </div>
              </header>
              <div className="pd-filters">
                <div className="select-wrap"><select aria-label="Date range"><option>Last 1h</option><option>Last 24h</option><option>Last 7d</option><option>Last 30d</option><option>Custom range&hellip;</option></select></div>
                <div className="select-wrap"><select aria-label="Function filter"><option>All functions</option><option>process-upload</option><option>send-welcome-email</option><option>sync-inventory</option><option>cleanup-tokens</option><option>generate-report</option><option>process-payment</option></select></div>
                <div className="select-wrap"><select aria-label="Source filter"><option>All sources</option><option>Database</option><option>Object Storage</option></select></div>
                <span style={{ display: "inline-flex", gap: 6, flexWrap: "wrap" }}>
                  <button className={`pd-chip ${evtFilter === "all" ? "active" : ""}`} onClick={() => setEvtFilter("all")}>All <span className="ch-count">1,247</span></button>
                  <button className={`pd-chip ${evtFilter === "delivered" ? "active" : ""}`} onClick={() => setEvtFilter("delivered")}><span className="ch-dot delivered" style={{ background: "var(--ok)" }}></span>Delivered <span className="ch-count">1,201</span></button>
                  <button className={`pd-chip ${evtFilter === "retrying" ? "active" : ""}`} onClick={() => setEvtFilter("retrying")}><span className="ch-dot retrying" style={{ background: "var(--warn)" }}></span>Retrying <span className="ch-count">12</span></button>
                  <button className={`pd-chip ${evtFilter === "failed" ? "active" : ""}`} onClick={() => setEvtFilter("failed")}><span className="ch-dot failed" style={{ background: "var(--err)" }}></span>Failed <span className="ch-count">34</span></button>
                </span>
                <label className="search-input" style={{ marginLeft: "auto" }}>
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></svg>
                  <input type="text" placeholder="Search by trace ID\u2026" className="mono" style={{ fontFamily: "'JetBrains Mono',monospace" }} aria-label="Search by trace ID" />
                </label>
              </div>
              <div className="panel">
                <div className="evt-table" role="table">
                  <div className="row header" role="row">
                    <span>Status</span><span>Source</span><span>Function</span><span>Attempts</span><span>Error</span><span>Trace ID</span><span>Fired at</span>
                  </div>
                  {HC_EVENTS.filter(e => evtFilter === "all" || e.status === evtFilter).map(evt => (
                    <div key={evt.trace} className={`row body-row ${evt.status === "retrying" ? "retrying-row" : ""} ${evt.status === "failed" ? "failed-row" : ""} ${evtExpanded.has(evt.trace) ? "expanded" : ""}`} onClick={() => toggleEvtExpanded(evt.trace)}>
                      <span><span className={`badge-s ${evt.status}`} style={{ fontWeight: 500 }}><span className="bd" style={{ width: 6, height: 6, borderRadius: "50%", background: "currentColor", display: "inline-block" }}></span>{evt.status.charAt(0).toUpperCase() + evt.status.slice(1)}</span></span>
                      <span className="evt-source">
                        {evt.sourceIcon === "db" ? (
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v14c0 1.66 4.03 3 9 3s9-1.34 9-3V5"/><path d="M3 12c0 1.66 4.03 3 9 3s9-1.34 9-3"/></svg>
                        ) : (
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
                        )}
                        <span className="truncate"><span className="verb">{evt.verb}</span> on <span className="tbl">{evt.tbl}</span></span>
                      </span>
                      <a className="evt-fn" onClick={(e) => e.stopPropagation()}>{evt.fn}</a>
                      <span className={`evt-attempts ${evt.attCls}`}>{evt.attempts}</span>
                      <span className={`evt-error ${!evt.error ? "no-err" : ""}`} title={evt.error || undefined}>{evt.error || "\u2014"}</span>
                      <span className="evt-trace"><span className="tr-id">{evt.trShort}</span>&hellip;<CopyBtn text={evt.trace} /></span>
                      <span className="evt-when">{evt.when}<svg className="chev" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><polyline points="9 18 15 12 9 6"/></svg></span>
                      {evtExpanded.has(evt.trace) && (
                        <div className="evt-detail"><div className="evt-detail-inner">
                          <div className="evt-detail-section">
                            <span className="lbl">{evt.resultMsg ? "Result" : (evt.status === "failed" ? "Final error" : "Last error")}</span>
                            {evt.resultMsg ? (
                              <div className="err-msg" style={evt.resultStyle}>{evt.resultMsg}</div>
                            ) : (
                              <div className="err-msg">{(evt as any).errMsg}</div>
                            )}
                          </div>
                          <div className="evt-detail-section">
                            <span className="lbl">Retry timeline</span>
                            <div className="retry-timeline">
                              {evt.retries.map((r, ri) => (
                                <div key={ri} className={`retry-step ${r.cls}`}>
                                  <span className="num">{r.num}</span>
                                  <span className="what" style={(r as any).muted ? { color: "var(--fg-mute)" } : undefined}>{r.what}</span>
                                  <span className="when">{r.when}</span>
                                </div>
                              ))}
                            </div>
                          </div>
                          {evt.payload && (
                            <details className="payload-block">
                              <summary><svg className="chev" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><polyline points="9 18 15 12 9 6"/></svg>Payload preview</summary>
                              <pre>{evt.payload}</pre>
                            </details>
                          )}
                          <div className="evt-links">
                            <a><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>View function</a>
                            <a><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>Open trace in Jaeger</a>
                          </div>
                        </div></div>
                      )}
                    </div>
                  ))}
                </div>
                <div className="pg-foot">
                  <span>Showing <span className="mono">1&ndash;25</span> of <span className="mono">1,247</span> events</span>
                  <button className="btn btn-ghost">Load more</button>
                </div>
              </div>
            </>
          )}

          {/* ====== Storage tab ====== */}
          {activeTab === "storage" && (
            <>
              <header className="tab-head">
                <div><h2>Storage</h2><p>Manage buckets and objects.</p></div>
                <div className="actions"><button className="btn btn-primary"><IconPlus /> Create Bucket<span className="kbd-inline">B</span></button></div>
              </header>
              <div className="storage-split">
                <div className="bucket-list">
                  {HC_BUCKETS_MOCK.map((b, i) => (
                    <div key={b.name} className={`item ${activeBucket === i ? "active" : ""}`} onClick={() => setActiveBucket(i)}>
                      <div className="row1"><span className="nm">{b.name}</span><span className={`access-badge ${b.access}`}>{b.access}</span></div>
                      <span className="meta">{b.meta}</span>
                    </div>
                  ))}
                  <button className="create"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 13, height: 13 }}><path d="M12 5v14M5 12h14"/></svg> Create bucket</button>
                </div>
                <div className="file-browser">
                  <div className="browser-head">
                    <div className="left">
                      <span className="bkt-name">{HC_BUCKETS_MOCK[activeBucket].name}</span>
                      <span className={`access-badge ${HC_BUCKETS_MOCK[activeBucket].access}`}>{HC_BUCKETS_MOCK[activeBucket].access}</span>
                      <span className="pill-meta">50 MB max</span>
                      <span className="pill-meta">image/*, video/*</span>
                    </div>
                    <div className="actions">
                      <button className="btn btn-ghost" onClick={() => setUploadOverlayOpen(true)}>
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 14, height: 14 }}><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
                        Upload
                      </button>
                      <button className="icon-only-btn" aria-label="Bucket settings">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .34 1.86l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.7 1.7 0 0 0-1.86-.34 1.7 1.7 0 0 0-1.04 1.56V21a2 2 0 1 1-4 0v-.09A1.7 1.7 0 0 0 9 19.4a1.7 1.7 0 0 0-1.86.34l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.7 1.7 0 0 0 4.6 15a1.7 1.7 0 0 0-1.56-1.04H3a2 2 0 1 1 0-4h.09A1.7 1.7 0 0 0 4.6 9a1.7 1.7 0 0 0-.34-1.86l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.7 1.7 0 0 0 9 4.6c.43-.18.72-.6.72-1.06V3a2 2 0 1 1 4 0v.09c0 .46.29.88.72 1.06a1.7 1.7 0 0 0 1.86-.34l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.7 1.7 0 0 0 19.4 9c.18.43.6.72 1.06.72H21a2 2 0 1 1 0 4h-.09A1.7 1.7 0 0 0 19.4 15z"/></svg>
                      </button>
                    </div>
                  </div>
                  <div className="browser-crumbs">
                    <a>{HC_BUCKETS_MOCK[activeBucket].name}</a><span className="sep">/</span><a>images</a><span className="sep">/</span><span className="here">products</span>
                  </div>
                  <div className="browser-tools">
                    <label className="search-input"><IconSearch /><input type="text" placeholder="Search files\u2026" aria-label="Search files" /><span className="kbd-hint">/</span></label>
                    <div className="view-toggle" role="tablist" aria-label="View mode">
                      <button className={storView === "list" ? "on" : ""} onClick={() => setStorView("list")} aria-label="List view">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><circle cx="4" cy="6" r="0.8" fill="currentColor"/><circle cx="4" cy="12" r="0.8" fill="currentColor"/><circle cx="4" cy="18" r="0.8" fill="currentColor"/></svg>
                      </button>
                      <button className={storView === "grid" ? "on" : ""} onClick={() => setStorView("grid")} aria-label="Grid view">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>
                      </button>
                    </div>
                    <div className="select-wrap"><select aria-label="Sort by"><option>Name</option><option>Size</option><option>Modified</option><option>Type</option></select></div>
                  </div>
                  {/* LIST VIEW */}
                  {storView === "list" && (
                    <div className="file-table">
                      <div className="row header"><span></span><span></span><span>Name</span><span>Size</span><span>Type</span><span>Modified</span><span style={{ textAlign: "right" }}>Actions</span></div>
                      {HC_FILES_LIST.map(f => (
                        <div key={f.file} className="row body-row" onClick={() => { if (!f.folder) setFileDetailOpen(true); }}>
                          <input type="checkbox" className="checkbox" aria-label="Select" onClick={e => e.stopPropagation()} />
                          {f.folder ? (
                            <span className="file-icon folder"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg></span>
                          ) : f.iconCls === "thumb" ? (
                            <span className="file-icon thumb"></span>
                          ) : f.iconCls === "thumb green" ? (
                            <span className="file-icon thumb green"></span>
                          ) : f.iconCls === "doc" ? (
                            <span className="file-icon doc"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg></span>
                          ) : f.iconCls === "video" ? (
                            <span className="file-icon video"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2" ry="2"/></svg></span>
                          ) : (
                            <span className="file-icon archive"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg></span>
                          )}
                          <span className={`file-name ${f.folder ? "folder" : ""}`}>{f.file}{f.folder && <span className="sub">{f.sub}</span>}</span>
                          <span className={`file-size ${f.sizeEmpty ? "no-val" : ""}`}>{f.size}</span>
                          <span className="file-type">{f.type}</span>
                          <span className="file-mod">{f.mod}</span>
                          <span className="file-actions">
                            <button className="action-btn" aria-label="Download"><IconDownload /></button>
                            {f.hasUrl && <button className="action-btn" aria-label="Copy URL"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.72-1.71"/></svg></button>}
                            <button className="action-btn danger" aria-label="Delete"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-2 14a2 2 0 0 1-2 2H9a2 2 0 0 1-2-2L5 6"/></svg></button>
                          </span>
                        </div>
                      ))}
                    </div>
                  )}
                  {/* GRID VIEW */}
                  {storView === "grid" && (
                    <div className="file-grid">
                      {HC_FILES_GRID.map(f => (
                        <div key={f.file} className={`grid-tile ${f.folder ? "folder" : ""}`} onClick={() => { if (!f.folder) setFileDetailOpen(true); }}>
                          <div className={`preview ${f.previewCls}`}>
                            {f.folder ? (
                              <svg className="ph-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/></svg>
                            ) : (f as any).docIcon ? (
                              <><svg className="ph-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" style={{ color: "var(--warn)" }}><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg><span className="size-tag">{f.sizeBadge}</span></>
                            ) : (f as any).videoIcon ? (
                              <><svg className="ph-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" style={{ color: "#fff", zIndex: 1 }}><polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2" ry="2"/></svg><span className="size-tag">{f.sizeBadge}</span></>
                            ) : (f as any).archiveIcon ? (
                              <><svg className="ph-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg><span className="size-tag">{f.sizeBadge}</span></>
                            ) : f.sizeBadge ? (
                              <span className="size-tag">{f.sizeBadge}</span>
                            ) : null}
                          </div>
                          <div className="meta"><div className="nm">{f.file}</div></div>
                        </div>
                      ))}
                    </div>
                  )}
                  {/* Upload overlay */}
                  {uploadOverlayOpen && (
                    <div className="upload-overlay">
                      <button className="close-btn" onClick={() => setUploadOverlayOpen(false)} aria-label="Close upload">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 14, height: 14 }}><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                      </button>
                      <div className="drop">
                        <span className="cloud"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M16 16l-4-4-4 4"/><path d="M12 12v9"/><path d="M20.39 18.39A5 5 0 0 0 18 9h-1.26A8 8 0 1 0 3 16.3"/><polyline points="16 16 12 12 8 16"/></svg></span>
                        <h3>Drop files here or click to browse</h3>
                        <p>Max file size: 50 MB &middot; Allowed types: image/*, video/*</p>
                      </div>
                      <div className="progress-list">
                        <div className="prog-row">
                          <span className="file-icon image" style={{ width: 22, height: 22 }}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 11, height: 11 }}><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg></span>
                          <span className="nm">product-hero-2026.webp</span>
                          <span className="bar"><i style={{ width: "78%" }}></i></span>
                          <span className="pct">78%</span>
                          <button className="cancel" aria-label="Cancel"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 11, height: 11 }}><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
                          <span className="sz" style={{ gridColumn: "2/3", gridRow: 1, justifySelf: "end" }}>3.1 MB</span>
                        </div>
                        <div className="prog-row">
                          <span className="file-icon video" style={{ width: 22, height: 22 }}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 11, height: 11 }}><polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2"/></svg></span>
                          <span className="nm">walkthrough-may.mp4</span>
                          <span className="bar"><i style={{ width: "34%" }}></i></span>
                          <span className="pct">34%</span>
                          <button className="cancel" aria-label="Cancel"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 11, height: 11 }}><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
                          <span className="sz" style={{ gridColumn: "2/3", gridRow: 1, justifySelf: "end" }}>28.4 MB</span>
                        </div>
                        <div className="prog-row">
                          <span className="file-icon image" style={{ width: 22, height: 22 }}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 11, height: 11 }}><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg></span>
                          <span className="nm">og-image-spring.png</span>
                          <span className="bar"><i style={{ width: "100%", background: "var(--ok)" }}></i></span>
                          <span className="pct" style={{ color: "var(--ok)" }}>Done</span>
                          <span></span>
                          <span className="sz" style={{ gridColumn: "2/3", gridRow: 1, justifySelf: "end" }}>812 KB</span>
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </>
          )}

          {/* ====== Secrets tab ====== */}
          {activeTab === "secrets" && (
            <>
              <header className="tab-head">
                <div><h2>Secrets</h2><p>Environment variables for your functions. Values are write-only and never exposed.</p></div>
                <div className="actions"><button className="btn btn-primary" onClick={() => setAddSecretModalOpen(true)}><IconPlus /> Add Secret<span className="kbd-inline">S</span></button></div>
              </header>
              {!secBannerDismissed && (
                <div className="info-banner">
                  <span className="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/><path d="M9 12l2 2 4-4"/></svg></span>
                  <p><strong>Secret values are encrypted</strong> and stored in Kubernetes. Once saved, values cannot be read back &mdash; only rotated or deleted.</p>
                  <button className="dismiss" onClick={() => setSecBannerDismissed(true)} aria-label="Dismiss"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
                </div>
              )}
              <div className="panel">
                <div className="sec-table" role="table">
                  <div className="row header" role="row">
                    <button className="sortable sorted" style={{ background:"none",border:0,padding:0,font:"inherit",color:"inherit",textAlign:"left",textTransform:"uppercase",letterSpacing:"0.07em",cursor:"pointer",display:"inline-flex",alignItems:"center",gap:4 }}>Name<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" style={{ width:9,height:9 }}><polyline points="6 9 12 15 18 9"/></svg></button>
                    <span>Description</span><span>Value</span>
                    <button className="sortable" style={{ background:"none",border:0,padding:0,font:"inherit",color:"inherit",textAlign:"left",textTransform:"uppercase",letterSpacing:"0.07em",cursor:"pointer",display:"inline-flex",alignItems:"center",gap:4 }}>Last rotated<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" style={{ width:9,height:9 }}><polyline points="8 18 12 22 16 18"/><polyline points="8 6 12 2 16 6"/></svg></button>
                    <button className="sortable" style={{ background:"none",border:0,padding:0,font:"inherit",color:"inherit",textAlign:"left",textTransform:"uppercase",letterSpacing:"0.07em",cursor:"pointer",display:"inline-flex",alignItems:"center",gap:4 }}>Created<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" style={{ width:9,height:9 }}><polyline points="8 18 12 22 16 18"/><polyline points="8 6 12 2 16 6"/></svg></button>
                    <span style={{ textAlign: "right" }}>Actions</span>
                  </div>
                  {HC_SECRETS.map(s => (
                    <div key={s.name} className="row body-row">
                      <span className="sec-name">{s.name}</span>
                      <span className="sec-desc">{s.desc}</span>
                      <span className="sec-value"><IconLock />&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;</span>
                      <span className={`sec-when ${s.rotatedNever ? "never" : ""}`}>{s.rotated}</span>
                      <span className="sec-when">{s.created}</span>
                      <span className="sec-actions">
                        <button onClick={() => { setTargetSecretName(s.name); setRotateSecretValue(""); setRotateValueRevealed(false); setRotateSecretModalOpen(true); }}>Rotate</button>
                        <button className="danger" onClick={() => { setTargetSecretName(s.name); setDeleteSecretConfirmInput(""); setDeleteSecretModalOpen(true); }}>Delete</button>
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            </>
          )}

          {/* ====== API Keys tab ====== */}
          {activeTab === "apikeys" && (
            <>
              <header className="tab-head">
                <div><h2>API Keys</h2><p>Authenticate requests to your project{"'"}s API.</p></div>
                <div className="actions"><button className="btn btn-primary" onClick={() => { setCreateKeyStep(1); setCreateKeyName(""); setCreateKeyRole("anon"); setCreateKeyExp("90"); setCreateKeyModalOpen(true); }}><IconPlus /> Create Key<span className="kbd-inline">K</span></button></div>
              </header>
              <div className="role-cards">
                <div className="role-card anon">
                  <span className="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 9.9-1"/></svg></span>
                  <div className="body"><div className="top"><span className="label">anon</span></div><p className="desc">Safe for client-side use. Respects Row Level Security policies.</p><span className="note">Can be exposed in browser code</span></div>
                </div>
                <div className="role-card role-svc">
                  <span className="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg></span>
                  <div className="body"><div className="top"><span className="label">service_role</span></div><p className="desc">Full access, bypasses RLS. Use only in server-side environments.</p><span className="note">Never expose in client-side code</span></div>
                </div>
              </div>
              <div className="panel">
                <div className="ak-table" role="table">
                  <div className="row header" role="row">
                    <button className="sortable sorted" style={{ background:"none",border:0,padding:0,font:"inherit",color:"inherit",textAlign:"left",textTransform:"uppercase",letterSpacing:"0.07em",cursor:"pointer",display:"inline-flex",alignItems:"center",gap:4 }}>Name<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" style={{ width:9,height:9 }}><polyline points="6 9 12 15 18 9"/></svg></button>
                    <span>Prefix</span>
                    <button className="sortable" style={{ background:"none",border:0,padding:0,font:"inherit",color:"inherit",textAlign:"left",textTransform:"uppercase",letterSpacing:"0.07em",cursor:"pointer",display:"inline-flex",alignItems:"center",gap:4 }}>Role<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" style={{ width:9,height:9 }}><polyline points="8 18 12 22 16 18"/><polyline points="8 6 12 2 16 6"/></svg></button>
                    <button className="sortable" style={{ background:"none",border:0,padding:0,font:"inherit",color:"inherit",textAlign:"left",textTransform:"uppercase",letterSpacing:"0.07em",cursor:"pointer",display:"inline-flex",alignItems:"center",gap:4 }}>Created<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" style={{ width:9,height:9 }}><polyline points="8 18 12 22 16 18"/><polyline points="8 6 12 2 16 6"/></svg></button>
                    <button className="sortable" style={{ background:"none",border:0,padding:0,font:"inherit",color:"inherit",textAlign:"left",textTransform:"uppercase",letterSpacing:"0.07em",cursor:"pointer",display:"inline-flex",alignItems:"center",gap:4 }}>Expires<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" style={{ width:9,height:9 }}><polyline points="8 18 12 22 16 18"/><polyline points="8 6 12 2 16 6"/></svg></button>
                    <span>Status</span><span style={{ textAlign: "right" }}>Actions</span>
                  </div>
                  {HC_APIKEYS.map(k => (
                    <div key={k.prefix} className={`row body-row ${k.rowCls}`} role="row">
                      <span className="ak-name">{k.name}</span>
                      <span className="ak-prefix">{k.prefix}<CopyBtn text={k.prefix} /></span>
                      <span><span className={`role-badge ${k.role === "anon" ? "anon" : "svc"}`}>{k.role}</span></span>
                      <span className="ak-when">{k.created}</span>
                      <span className={`ak-expire ${k.expCls}`}>{k.expires}</span>
                      <span className={`ak-status ${k.status}`}><span className="dt" />{k.status === "revoked" ? "Revoked" : "Active"}</span>
                      <span className="ak-actions">
                        {k.status !== "revoked" && <button onClick={() => { setRevokeKeyName(k.name); setRevokeKeyPrefix(k.prefix); setRevokeKeyRole(k.role); setRevokeConfirmInput(""); setRevokeKeyModalOpen(true); }}>Revoke</button>}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            </>
          )}

          {/* ====== Settings tab ====== */}
          {activeTab === "settings" && (
            <div className="settings-split">
              <SettingsNav active={settingsSection} onNav={setSettingsSection} onPause={() => setPauseModalOpen(true)} onDelete={() => setDeleteProjectModalOpen(true)} />
              <div className="settings-main">
                <div className="stg-panel" id="stg-general">
                  <div className="stg-panel-head"><h3>Project information</h3><p className="sub">Basic information about your project.</p></div>
                  <div className="stg-panel-body">
                    <div className="stg-field"><label>Display name</label><input type="text" defaultValue={displayName} maxLength={100} /><span className="helper">Used in the dashboard and notifications.</span></div>
                    <div className="stg-field"><label>Description</label><div className="counter-wrap"><textarea defaultValue={description} maxLength={500} /><span className="char-counter">{description.length} / 500</span></div></div>
                    <div className="stg-field"><label>Project ID</label><div className="readonly-wrap"><input type="text" className="mono" value={projectId} readOnly /><button className="copy" onClick={() => navigator.clipboard.writeText(projectId)}><IconCopy /></button></div><span className="helper">Cannot be changed. Used as subdomain for API endpoints.</span></div>
                  </div>
                  <div className="stg-panel-foot"><button className="btn btn-ghost">Reset</button><button className="btn btn-primary" disabled>Save Changes</button></div>
                </div>
                <div className="stg-panel" id="stg-services">
                  <div className="stg-panel-head"><h3>Enabled services</h3><p className="sub">Disabling a service may destroy data. Review carefully.</p></div>
                  <div className="stg-panel-body">
                    <div className="svc-row"><div className="toggle-col"><input type="checkbox" className="stg-switch" defaultChecked={project?.postgresEnabled} /></div><div className="info"><div className="ttl">PostgreSQL <span className="ver">v16</span></div><div className="sub">Managed database cluster with point-in-time recovery.</div>
                      {project?.postgresEnabled && (<div className="svc-expand"><div className="exp-title">Extensions</div><div className="ext-list">
                        {["pgvector","pgcrypto","pg_stat_statements","postgis"].map(ext => (
                          <label key={ext} className="ext-item"><input type="checkbox" className="checkbox" defaultChecked={project?.postgresExtensions?.includes(ext)} /><div><div className="nm">{ext}</div><span className="desc">{ext==="pgvector"&&"Vector similarity search"}{ext==="pgcrypto"&&"Cryptographic functions"}{ext==="pg_stat_statements"&&"Track execution statistics"}{ext==="postgis"&&"Geographic objects"}</span></div></label>
                        ))}
                      </div><div className="note"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>Extension changes require database restart</div></div>)}
                    </div></div>
                    <div className="svc-row"><div className="toggle-col"><input type="checkbox" className="stg-switch" defaultChecked={project?.redisEnabled} /></div><div className="info"><div className="ttl">Redis <span className="ver">v7.2</span></div><div className="sub">In-memory key-value cache with persistence.</div></div></div>
                    <div className="svc-row"><div className="toggle-col"><input type="checkbox" className="stg-switch" defaultChecked={project?.postgrestEnabled} /></div><div className="info"><div className="ttl">PostgREST <span className="ver">v12</span></div><div className="sub">Instant RESTful API from your database schema.</div>
                      {project?.postgrestEnabled && (<div className="endpoint">{apiEndpoint}/rest/v1<button onClick={() => navigator.clipboard.writeText(`${apiEndpoint}/rest/v1`)}><IconCopy /></button></div>)}
                    </div></div>
                  </div>
                </div>
                <div className="danger-panel" id="stg-danger">
                  <div className="stg-panel-head"><h3>Danger zone</h3><p className="sub">Irreversible operations on this project.</p></div>
                  <div className="stg-panel-body">
                    <div className="danger-row"><div className="info"><div className="ttl">Pause project</div><div className="sub">Temporarily stop all services. Data is preserved but endpoints become unavailable.</div></div><div className="right-actions"><button className="btn btn-warn" onClick={() => setPauseModalOpen(true)}>Pause Project</button></div></div>
                    <div className="danger-row"><div className="info"><div className="ttl">Transfer ownership</div><div className="sub">Transfer this project to another tenant.</div></div><div className="right-actions"><span className="coming-soon">Coming soon</span><button className="btn btn-ghost" disabled>Transfer</button></div></div>
                    <div className="danger-row"><div className="info"><div className="ttl err">Delete project</div><div className="sub">Permanently delete this project and <strong style={{ color: "var(--err)" }}>all its data</strong>. This cannot be undone.</div></div><div className="right-actions"><button className="btn btn-danger" onClick={() => setDeleteProjectModalOpen(true)}>Delete Project</button></div></div>
                  </div>
                </div>
              </div>
            </div>
          )}

        </div>
      </div>

      {/* ===== MODALS ===== */}

      {/* 1. Pause Project Modal */}
      {pauseModalOpen && (
        <div className="modal-scrim open" role="dialog" aria-modal="true" onClick={() => setPauseModalOpen(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <div className="modal-head">
              <h3>Pause project</h3>
              <button className="close" onClick={() => setPauseModalOpen(false)} aria-label="Close"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width:14,height:14 }}><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>
            <div className="modal-body">
              <div className="modal-warn">
                <span className="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/></svg></span>
                <p>All services will be stopped. Your data (database, storage, secrets) will be <strong>preserved</strong>. You can resume at any time.</p>
              </div>
              <div className="panel" style={{ padding:"14px 16px",borderRadius:10,fontSize:"12.5px" }}>
                <span style={{ fontSize:11,color:"var(--fg-mute)",textTransform:"uppercase",letterSpacing:"0.06em" }}>Currently running</span>
                <div style={{ marginTop:6,display:"flex",gap:6,flexWrap:"wrap" }}>
                  <span className="role-badge" style={{ color:"var(--ok)",background:"rgba(134,239,172,0.08)",borderColor:"rgba(134,239,172,0.22)" }}>PostgreSQL</span>
                  <span className="role-badge" style={{ color:"var(--ok)",background:"rgba(134,239,172,0.08)",borderColor:"rgba(134,239,172,0.22)" }}>Redis</span>
                  <span className="role-badge" style={{ color:"var(--ok)",background:"rgba(134,239,172,0.08)",borderColor:"rgba(134,239,172,0.22)" }}>PostgREST</span>
                  <span className="role-badge" style={{ color:"var(--accent)",background:"var(--accent-soft)",borderColor:"rgba(196,181,253,0.22)" }}>5 Functions</span>
                </div>
              </div>
            </div>
            <div className="modal-foot">
              <button className="btn btn-ghost" onClick={() => setPauseModalOpen(false)}>Cancel</button>
              <button className="btn btn-warn" style={{ borderColor:"rgba(252,211,77,0.5)" }}>Pause Project</button>
            </div>
          </div>
        </div>
      )}

      {/* 2. Delete Project Modal */}
      {deleteProjectModalOpen && (
        <div className="modal-scrim open" role="dialog" aria-modal="true" onClick={() => setDeleteProjectModalOpen(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <div className="modal-head">
              <h3 style={{ color:"var(--err)" }}>Delete project</h3>
              <button className="close" onClick={() => setDeleteProjectModalOpen(false)} aria-label="Close"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width:14,height:14 }}><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>
            <div className="modal-body">
              <div className="modal-err">
                <span className="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg></span>
                <div>
                  <p><strong>This will permanently delete:</strong></p>
                  <ul className="danger-bullets">
                    <li>PostgreSQL database and all data</li>
                    <li>3 storage buckets (18.8 GB)</li>
                    <li>5 deployed functions</li>
                    <li>6 secrets</li>
                    <li>4 API keys</li>
                    <li>All event history</li>
                  </ul>
                </div>
              </div>
              <div className="field">
                <label htmlFor="delete-project-input">Type <span className="mono" style={{ color:"var(--err)",fontFamily:"'JetBrains Mono',monospace" }}>{displayName}</span> to confirm</label>
                <input type="text" id="delete-project-input" className="mono" placeholder="Type the project name" autoComplete="off" value={deleteProjectInput} onChange={e => setDeleteProjectInput(e.target.value)} />
              </div>
              <label className="ack-check"><input type="checkbox" checked={deleteProjectAck} onChange={() => setDeleteProjectAck(!deleteProjectAck)} />I understand this action cannot be undone</label>
            </div>
            <div className="modal-foot">
              <button className="btn btn-ghost" onClick={() => setDeleteProjectModalOpen(false)}>Cancel</button>
              <button className="btn btn-danger" disabled={deleteProjectInput !== displayName || !deleteProjectAck}>Delete Project</button>
            </div>
          </div>
        </div>
      )}

      {/* 3. Add Secret Modal */}
      {addSecretModalOpen && (
        <div className="modal-scrim open" role="dialog" aria-modal="true" onClick={() => setAddSecretModalOpen(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <div className="modal-head">
              <h3>Add secret</h3>
              <button className="close" onClick={() => setAddSecretModalOpen(false)} aria-label="Close"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width:14,height:14 }}><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>
            <div className="modal-body">
              <div className="field"><label>Name</label><input type="text" className="mono" placeholder="e.g. OPENAI_API_KEY" autoComplete="off" spellCheck={false} value={addSecretName} onChange={e => setAddSecretName(e.target.value)} /><span className="hint">UPPERCASE letters, numbers, and underscores only.</span></div>
              <div className="field">
                <label>Value</label>
                <div className="value-field">
                  <textarea className={`mono ${secValueRevealed ? "" : "masked"}`} placeholder="Paste your secret value\u2026" rows={3} autoComplete="off" spellCheck={false} value={addSecretValue} onChange={e => setAddSecretValue(e.target.value)} />
                  <button type="button" className="reveal" onClick={() => setSecValueRevealed(!secValueRevealed)}>{secValueRevealed ? "Hide" : "Show"}</button>
                </div>
                <span className="footnote"><IconLock />This value will be encrypted and cannot be retrieved after saving.</span>
              </div>
              <div className="field"><label>Description <span style={{ color:"var(--fg-mute)",textTransform:"none",letterSpacing:0,fontWeight:400 }}>&mdash; optional</span></label><input type="text" placeholder="What is this secret used for?" value={addSecretDesc} onChange={e => setAddSecretDesc(e.target.value)} /></div>
            </div>
            <div className="modal-foot">
              <button className="btn btn-ghost" onClick={() => setAddSecretModalOpen(false)}>Cancel</button>
              <button className="btn btn-primary">Add Secret</button>
            </div>
          </div>
        </div>
      )}

      {/* 4. Rotate Secret Modal */}
      {rotateSecretModalOpen && (
        <div className="modal-scrim open" role="dialog" aria-modal="true" onClick={() => setRotateSecretModalOpen(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <div className="modal-head">
              <h3>Rotate <span className="mono">{targetSecretName}</span></h3>
              <button className="close" onClick={() => setRotateSecretModalOpen(false)} aria-label="Close"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width:14,height:14 }}><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>
            <div className="modal-body">
              <div className="modal-warn">
                <span className="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg></span>
                <p>This will <strong>immediately replace</strong> the current value. Functions using this secret will pick up the new value on next restart.</p>
              </div>
              <div className="field">
                <label>New value</label>
                <div className="value-field">
                  <textarea className={`mono ${rotateValueRevealed ? "" : "masked"}`} placeholder="Paste the new secret value\u2026" rows={3} autoComplete="off" spellCheck={false} value={rotateSecretValue} onChange={e => setRotateSecretValue(e.target.value)} />
                  <button type="button" className="reveal" onClick={() => setRotateValueRevealed(!rotateValueRevealed)}>{rotateValueRevealed ? "Hide" : "Show"}</button>
                </div>
              </div>
            </div>
            <div className="modal-foot">
              <button className="btn btn-ghost" onClick={() => setRotateSecretModalOpen(false)}>Cancel</button>
              <button className="btn btn-primary">Rotate Value</button>
            </div>
          </div>
        </div>
      )}

      {/* 5. Delete Secret Modal */}
      {deleteSecretModalOpen && (
        <div className="modal-scrim open" role="dialog" aria-modal="true" onClick={() => setDeleteSecretModalOpen(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <div className="modal-head">
              <h3>Delete secret</h3>
              <button className="close" onClick={() => setDeleteSecretModalOpen(false)} aria-label="Close"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width:14,height:14 }}><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>
            <div className="modal-body">
              <div className="modal-err">
                <span className="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg></span>
                <p>Are you sure you want to delete <strong className="mono">{targetSecretName}</strong>? Functions referencing this secret will <strong>fail on next restart.</strong></p>
              </div>
              <div className="field">
                <label>Type <span className="mono" style={{ color:"var(--err)",textTransform:"none",letterSpacing:0 }}>{targetSecretName}</span> to confirm</label>
                <input type="text" className="mono" placeholder="Type the secret name" autoComplete="off" spellCheck={false} value={deleteSecretConfirmInput} onChange={e => setDeleteSecretConfirmInput(e.target.value)} />
              </div>
            </div>
            <div className="modal-foot">
              <button className="btn btn-ghost" onClick={() => setDeleteSecretModalOpen(false)}>Cancel</button>
              <button className="btn btn-danger" disabled={deleteSecretConfirmInput !== targetSecretName}>Delete Secret</button>
            </div>
          </div>
        </div>
      )}

      {/* 6. Create Key Modal (2-step) */}
      {createKeyModalOpen && (
        <div className="modal-scrim open" role="dialog" aria-modal="true" onClick={() => setCreateKeyModalOpen(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            {createKeyStep === 1 ? (
              <>
                <div className="modal-head">
                  <h3>Create API Key</h3>
                  <button className="close" onClick={() => setCreateKeyModalOpen(false)} aria-label="Close"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width:14,height:14 }}><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
                </div>
                <div className="modal-body">
                  <div className="field"><label>Name</label><input type="text" placeholder="e.g. Frontend App, CI Pipeline" maxLength={63} autoComplete="off" value={createKeyName} onChange={e => setCreateKeyName(e.target.value)} /><span className="hint">Max 63 characters.</span></div>
                  <div className="field">
                    <label>Role</label>
                    <div className="role-radio-group">
                      <label className={`role-radio anon ${createKeyRole === "anon" ? "checked" : ""}`}>
                        <input type="radio" name="key-role" value="anon" checked={createKeyRole === "anon"} onChange={() => setCreateKeyRole("anon")} />
                        <div className="top"><span className="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 9.9-1"/></svg></span><span className="nm" style={{ color:"var(--ok)" }}>anon</span></div>
                        <div className="ttl">Public / Client-safe</div>
                        <div className="sub">Respects RLS</div>
                      </label>
                      <label className={`role-radio svc ${createKeyRole === "service_role" ? "checked" : ""}`}>
                        <input type="radio" name="key-role" value="service_role" checked={createKeyRole === "service_role"} onChange={() => setCreateKeyRole("service_role")} />
                        <div className="top"><span className="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg></span><span className="nm" style={{ color:"var(--err)" }}>service_role</span></div>
                        <div className="ttl">Admin / Server-only</div>
                        <div className="sub">Bypasses RLS</div>
                      </label>
                    </div>
                    {createKeyRole === "service_role" && (
                      <div className="modal-err" style={{ marginTop: 8 }}>
                        <span className="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg></span>
                        <p>This key has <strong>full database access</strong>. Keep it secret.</p>
                      </div>
                    )}
                  </div>
                  <div className="field">
                    <label>Expiration</label>
                    <div className="exp-group">
                      <label className={`exp-option ${createKeyExp === "90" ? "checked" : ""}`}><input type="radio" name="key-exp" value="90" checked={createKeyExp === "90"} onChange={() => setCreateKeyExp("90")} /><span className="radio"></span><span className="lbl">90 days</span><span className="recommended">Recommended</span></label>
                      <label className={`exp-option ${createKeyExp === "30" ? "checked" : ""}`}><input type="radio" name="key-exp" value="30" checked={createKeyExp === "30"} onChange={() => setCreateKeyExp("30")} /><span className="radio"></span><span className="lbl">30 days</span></label>
                      <label className={`exp-option ${createKeyExp === "365" ? "checked" : ""}`}><input type="radio" name="key-exp" value="365" checked={createKeyExp === "365"} onChange={() => setCreateKeyExp("365")} /><span className="radio"></span><span className="lbl">1 year</span></label>
                      <label className={`exp-option ${createKeyExp === "never" ? "checked" : ""}`}><input type="radio" name="key-exp" value="never" checked={createKeyExp === "never"} onChange={() => setCreateKeyExp("never")} /><span className="radio"></span><span className="lbl">No expiration</span><span className="sub" style={{ color:"var(--warn)" }}>Security risk</span></label>
                    </div>
                  </div>
                </div>
                <div className="modal-foot">
                  <button className="btn btn-ghost" onClick={() => setCreateKeyModalOpen(false)}>Cancel</button>
                  <button className="btn btn-primary" onClick={() => setCreateKeyStep(2)}>Create Key</button>
                </div>
              </>
            ) : (
              <>
                <div className="modal-head">
                  <h3>API Key Created</h3>
                  <button className="close" onClick={() => setCreateKeyModalOpen(false)} aria-label="Close"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width:14,height:14 }}><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
                </div>
                <div className="modal-body">
                  <div className="key-success">
                    <div className="head-ok"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="9 12 11 14 15 9"/></svg>Your new key is ready</div>
                    <div className="key-block"><span className="tag">Full key</span>etbs_a1b2_R3yJ9pX2qN7vTfL5cE0wM8aZkD4sH6</div>
                    <button className="btn btn-primary copy-full" onClick={() => navigator.clipboard.writeText("etbs_a1b2_R3yJ9pX2qN7vTfL5cE0wM8aZkD4sH6")}>
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width:13,height:13 }}><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
                      Copy key
                    </button>
                    <div className="modal-err">
                      <span className="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg></span>
                      <p><strong>Copy this key now.</strong> It will not be shown again.</p>
                    </div>
                  </div>
                </div>
                <div className="modal-foot"><button className="btn btn-primary" onClick={() => setCreateKeyModalOpen(false)}>Done</button></div>
              </>
            )}
          </div>
        </div>
      )}

      {/* 7. Revoke Key Modal */}
      {revokeKeyModalOpen && (
        <div className="modal-scrim open" role="dialog" aria-modal="true" onClick={() => setRevokeKeyModalOpen(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <div className="modal-head">
              <h3>Revoke API Key</h3>
              <button className="close" onClick={() => setRevokeKeyModalOpen(false)} aria-label="Close"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width:14,height:14 }}><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>
            <div className="modal-body">
              <div className="modal-err">
                <span className="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg></span>
                <p>Revoking this key will <strong>immediately break</strong> any application using it.</p>
              </div>
              <div className="panel" style={{ padding:"14px 16px",borderRadius:10 }}>
                <div style={{ display:"flex",alignItems:"center",gap:12,flexWrap:"wrap" }}>
                  <div style={{ display:"flex",flexDirection:"column",gap:3 }}>
                    <span style={{ fontSize:11,color:"var(--fg-mute)",textTransform:"uppercase",letterSpacing:"0.06em" }}>Key</span>
                    <span style={{ fontSize:13,fontWeight:600 }}>{revokeKeyName}</span>
                  </div>
                  <span style={{ display:"flex",alignItems:"center",gap:6,fontFamily:"'JetBrains Mono',monospace",fontSize:12,color:"var(--fg-dim)" }}><span>{revokeKeyPrefix}</span></span>
                  <span className={`role-badge ${revokeKeyRole === "anon" ? "anon" : "svc"}`}>{revokeKeyRole}</span>
                </div>
              </div>
              <div className="field">
                <label>Type <span className="mono" style={{ color:"var(--err)",textTransform:"none",letterSpacing:0,fontFamily:"'JetBrains Mono',monospace" }}>{revokeKeyName}</span> to confirm</label>
                <input type="text" placeholder="Type the key name" autoComplete="off" value={revokeConfirmInput} onChange={e => setRevokeConfirmInput(e.target.value)} />
              </div>
            </div>
            <div className="modal-foot">
              <button className="btn btn-ghost" onClick={() => setRevokeKeyModalOpen(false)}>Cancel</button>
              <button className="btn btn-danger" disabled={revokeConfirmInput !== revokeKeyName}>Revoke Key</button>
            </div>
          </div>
        </div>
      )}

      {/* 8. Deploy Function Modal */}
      {deployFnModalOpen && (
        <div className="dfm" role="dialog" aria-modal="true" onClick={() => setDeployFnModalOpen(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            {/* Header */}
            <div className="head">
              <div className="title">
                <h2>Deploy function</h2>
                <p className="sub">Configure source, runtime, triggers, and environment for your new serverless function.</p>
              </div>
              <button className="close" onClick={() => setDeployFnModalOpen(false)} aria-label="Close">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
              </button>
            </div>

            {/* Body */}
            <div className="body">
              {/* Basic */}
              <section className="section">
                <h3 className="section-label">Basic</h3>
                <div className="field-row">
                  <div className="field">
                    <label htmlFor="dfm-fn-name">Function name</label>
                    <input type="text" id="dfm-fn-name" className="mono" placeholder="process-upload" autoComplete="off" value={fnName} onChange={e => { const cleaned = e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""); setFnName(cleaned); }} />
                    <span className="hint">Lowercase, numbers, hyphens. 3-40 chars. Used in URLs.</span>
                  </div>
                  <div className="field">
                    <label htmlFor="dfm-fn-display">Display name <span className="opt">— optional</span></label>
                    <input type="text" id="dfm-fn-display" placeholder="Process Upload" autoComplete="off" value={fnDisplayName} onChange={e => setFnDisplayName(e.target.value)} />
                    <span className="hint">Shown in dashboard only.</span>
                  </div>
                </div>

                <div className="field">
                  <label>Kind</label>
                  <div className="radio-cards">
                    <label className={`radio-card heavy-job${fnKind === "heavy-job" ? " checked" : ""}`} onClick={() => handleDfmKindChange("heavy-job")}>
                      <input type="radio" name="kind" value="heavy-job" checked={fnKind === "heavy-job"} readOnly />
                      <span className="dot"></span>
                      <span className="info">
                        <span className="ttl">Heavy Job</span>
                        <span className="desc">One-off execution. Scales to zero. Batch processing, data pipelines.</span>
                      </span>
                    </label>
                    <label className={`radio-card heavy-deploy${fnKind === "heavy-deployment" ? " checked" : ""}`} onClick={() => handleDfmKindChange("heavy-deployment")}>
                      <input type="radio" name="kind" value="heavy-deployment" checked={fnKind === "heavy-deployment"} readOnly />
                      <span className="dot"></span>
                      <span className="info">
                        <span className="ttl">Heavy Deployment</span>
                        <span className="desc">Long-running service. Custom container. GPU workloads, ML inference.</span>
                      </span>
                    </label>
                    <label className={`radio-card light-deploy${fnKind === "light-deployment" ? " checked" : ""}`} onClick={() => handleDfmKindChange("light-deployment")}>
                      <input type="radio" name="kind" value="light-deployment" checked={fnKind === "light-deployment"} readOnly />
                      <span className="dot"></span>
                      <span className="info">
                        <span className="ttl">Light Deployment</span>
                        <span className="desc">Fast cold start. ≤30s timeout. Webhooks, API handlers.</span>
                      </span>
                    </label>
                  </div>
                </div>

                <div className="field">
                  <label>Mode</label>
                  <div className="segmented" role="tablist">
                    {["sync", "async", "stream"].map(m => (
                      <button key={m} type="button" className={fnMode === m ? "active" : ""} onClick={() => setFnMode(m)}>{m.charAt(0).toUpperCase() + m.slice(1)}</button>
                    ))}
                  </div>
                </div>
              </section>

              {/* Source */}
              <section className="section">
                <h3 className="section-label">Source</h3>
                <div className="segmented">
                  {(["inline", "git", "zip"] as const).map(t => (
                    <button key={t} type="button" className={sourceTab === t ? "active" : ""} onClick={() => setSourceTab(t)}>{t === "inline" ? "Inline" : t === "git" ? "Git" : "Zip"}</button>
                  ))}
                </div>

                {/* Inline pane */}
                <div className={`pane${sourceTab === "inline" ? " active" : ""}`}>
                  <div>
                    <div className="filename-row">
                      <span className="tag">file</span>
                      <input type="text" value={inlineFilename} onChange={e => setInlineFilename(e.target.value)} placeholder="main.py" autoComplete="off" />
                    </div>
                    <textarea className="code-editor" rows={8} spellCheck={false} placeholder={"def handler(event, context):\n    return {'ok': True}"} value={inlineCode} onChange={e => setInlineCode(e.target.value)} />
                    <span className="hint" style={{ marginTop: 8, display: "inline-flex", alignItems: "center", gap: 6 }}>
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 11, height: 11, flexShrink: 0 }}><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>
                      Inline is for quick snippets. For real projects, use the <strong style={{ color: "var(--fg-dim)", fontWeight: 500 }}>Git</strong> tab.
                    </span>
                  </div>
                </div>

                {/* Git pane */}
                <div className={`pane${sourceTab === "git" ? " active" : ""}`}>
                  <div className="field">
                    <label htmlFor="dfm-git-url">Repository URL</label>
                    <input type="text" id="dfm-git-url" className="mono" placeholder="https://github.com/acme/functions.git" autoComplete="off" value={gitUrl} onChange={e => setGitUrl(e.target.value)} />
                  </div>
                  <div className="field-row">
                    <div className="field">
                      <label htmlFor="dfm-git-branch">Branch</label>
                      <input type="text" id="dfm-git-branch" className="mono" value={gitBranch} onChange={e => setGitBranch(e.target.value)} autoComplete="off" />
                    </div>
                    <div className="field">
                      <label htmlFor="dfm-git-subpath">Subpath</label>
                      <input type="text" id="dfm-git-subpath" className="mono" value={gitSubpath} onChange={e => setGitSubpath(e.target.value)} autoComplete="off" />
                      <span className="hint">Path inside the repo to use as the build root.</span>
                    </div>
                  </div>
                </div>

                {/* Zip pane */}
                <div className={`pane${sourceTab === "zip" ? " active" : ""}`}>
                  <div className="field">
                    <label htmlFor="dfm-zip-path">Source path in object storage</label>
                    <input type="text" id="dfm-zip-path" className="mono" placeholder="s3://builds/process-upload-v1.zip" autoComplete="off" value={zipPath} onChange={e => setZipPath(e.target.value)} />
                    <span className="hint">Upload a tarball or zip to a bucket and reference it here. File picker coming soon.</span>
                  </div>
                </div>
              </section>

              {/* Runtime */}
              <section className="section">
                <h3 className="section-label">Runtime</h3>
                <div className="segmented">
                  <button type="button" className={runtimeTab === "preset" ? "active" : ""} onClick={() => setRuntimeTab("preset")}>Preset</button>
                  <button type="button" className={runtimeTab === "custom" ? "active" : ""} onClick={() => setRuntimeTab("custom")}>Custom Dockerfile</button>
                </div>

                {/* Preset pane */}
                <div className={`pane${runtimeTab === "preset" ? " active" : ""}`}>
                  <div className="field">
                    <label htmlFor="dfm-runtime-preset">Base image</label>
                    <select id="dfm-runtime-preset" value={runtimePreset} onChange={e => setRuntimePreset(e.target.value)}>
                      <option value="python-3.11">python-3.11</option>
                      <option value="python-3.12">python-3.12</option>
                      <option value="python-3.11-ml">python-3.11-ml</option>
                      <option value="python-3.12-ml">python-3.12-ml</option>
                      <option value="node-20">node-20</option>
                      <option value="node-22">node-22</option>
                      <option value="go-1.22">go-1.22</option>
                    </select>
                  </div>
                  <div className="field">
                    <label htmlFor="dfm-runtime-deps">Dependencies <span className="opt">— requirements.txt / package.json deps</span></label>
                    <textarea id="dfm-runtime-deps" className="mono" rows={4} spellCheck={false} placeholder={"numpy\npandas\nrequests"} value={runtimeDeps} onChange={e => setRuntimeDeps(e.target.value)} />
                  </div>
                </div>

                {/* Custom Dockerfile pane */}
                <div className={`pane${runtimeTab === "custom" ? " active" : ""}`}>
                  <div className="field">
                    <label htmlFor="dfm-custom-dockerfile">Dockerfile</label>
                    <textarea id="dfm-custom-dockerfile" className="code-editor" rows={10} spellCheck={false} placeholder={'FROM python:3.12-slim\nWORKDIR /app\nCOPY . .\nRUN pip install --no-cache-dir -r requirements.txt\nCMD ["python", "main.py"]'} value={customDockerfile} onChange={e => setCustomDockerfile(e.target.value)} />
                    <span className="hint">Must expose port 8080 for Sync/Stream modes. Async functions run to completion.</span>
                  </div>
                </div>
              </section>

              {/* Timeout & GPU */}
              <section className="section">
                <h3 className="section-label">Timeout &amp; GPU</h3>
                <div className="field-row">
                  <div className="field">
                    <label htmlFor="dfm-timeout">Timeout</label>
                    <div className="with-suffix">
                      <input type="number" id="dfm-timeout" min={1} max={dfmTimeoutHint.max} value={fnTimeout} onChange={e => setFnTimeout(Number(e.target.value))} />
                      <span className="suffix">seconds</span>
                    </div>
                    <span className={dfmTimeoutHint.cls}>{dfmTimeoutHint.text}</span>
                  </div>
                  <div className="field">
                    <label>GPU acceleration</label>
                    <label className="toggle-row" style={{ padding: 0, marginTop: 4 }}>
                      <input type="checkbox" className="switch" checked={gpuEnabled} onChange={e => setGpuEnabled(e.target.checked)} />
                      <span className="info">
                        <span className="ttl">{dfmGpuStatusText}</span>
                        <span className="desc">Enable GPU-backed runtime</span>
                      </span>
                    </label>
                  </div>
                </div>

                <div className={`gpu-config${gpuEnabled ? " open" : ""}`}>
                  <div className="field-row" style={{ marginTop: 10 }}>
                    <div className="field">
                      <label htmlFor="dfm-gpu-type">GPU Type</label>
                      <select id="dfm-gpu-type" value={gpuType} onChange={e => setGpuType(e.target.value)}>
                        <option value="H100">H100</option>
                        <option value="A100">A100</option>
                        <option value="A10G">A10G</option>
                        <option value="T4">T4</option>
                      </select>
                    </div>
                    <div className="field">
                      <label htmlFor="dfm-gpu-provider">Provider</label>
                      <select id="dfm-gpu-provider" value={gpuProvider} onChange={e => setGpuProvider(e.target.value)}>
                        <option value="runpod">RunPod</option>
                        <option value="lambda">Lambda Labs</option>
                        <option value="sakura">Sakura</option>
                        <option value="self">Self-managed</option>
                      </select>
                    </div>
                  </div>
                  <div className="field-row" style={{ marginTop: 12 }}>
                    <div className="field">
                      <label htmlFor="dfm-gpu-product">Product</label>
                      <select id="dfm-gpu-product" value={gpuProduct} onChange={e => setGpuProduct(e.target.value)}>
                        <option value="serverless">Serverless</option>
                        <option value="pods">Pods</option>
                        <option value="on-demand">On-demand</option>
                      </select>
                    </div>
                    <div className="field"></div>
                  </div>
                  <div className="gpu-note">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
                    <span><strong>GPU removes timeout limits</strong> for heavy-* kinds. Cold-start cost and billing depend on the selected provider.</span>
                  </div>
                </div>
              </section>

              {/* Triggers */}
              <section className="section">
                <h3 className="section-label">Triggers</h3>
                <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
                  {dfmTriggers.map(trig => (
                    <div key={trig.id} className="trigger-card">
                      <button className="remove" onClick={() => removeDfmTrigger(trig.id)} aria-label="Remove trigger">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                      </button>
                      <div className="field">
                        <label>Type</label>
                        <div className="segmented">
                          <button type="button" className={trig.type === "db" ? "active" : ""} onClick={() => updateDfmTrigger(trig.id, { type: "db" })}>Database change</button>
                          <button type="button" className={trig.type === "storage" ? "active" : ""} onClick={() => updateDfmTrigger(trig.id, { type: "storage" })}>Object storage</button>
                        </div>
                      </div>
                      {trig.type === "db" ? (
                        <>
                          <div className="field" style={{ marginTop: 10 }}>
                            <label>Table</label>
                            <input type="text" className="mono" placeholder="public.orders" autoComplete="off" value={trig.table} onChange={e => updateDfmTrigger(trig.id, { table: e.target.value })} />
                          </div>
                          <div className="field" style={{ marginTop: 10 }}>
                            <label>Events</label>
                            <div className="check-group">
                              {["INSERT", "UPDATE", "DELETE"].map(evt => (
                                <label key={evt} className={`check-pill${trig.dbEvents.includes(evt) ? " checked" : ""}`} onClick={e => { e.preventDefault(); toggleDfmTriggerEvent(trig.id, "dbEvents", evt); }}>
                                  <input type="checkbox" checked={trig.dbEvents.includes(evt)} readOnly /><span className="ck"></span>{evt}
                                </label>
                              ))}
                            </div>
                          </div>
                          <div className="field" style={{ marginTop: 10 }}>
                            <label>Filter <span className="opt">— optional WHERE clause</span></label>
                            <input type="text" className="mono" placeholder="status = &#39;paid&#39;" autoComplete="off" value={trig.filter} onChange={e => updateDfmTrigger(trig.id, { filter: e.target.value })} />
                          </div>
                        </>
                      ) : (
                        <>
                          <div className="field" style={{ marginTop: 10 }}>
                            <label>Bucket</label>
                            <input type="text" className="mono" placeholder="assets" autoComplete="off" value={trig.bucket} onChange={e => updateDfmTrigger(trig.id, { bucket: e.target.value })} />
                          </div>
                          <div className="field" style={{ marginTop: 10 }}>
                            <label>Events</label>
                            <div className="check-group">
                              {["ObjectCreated", "ObjectRemoved"].map(evt => (
                                <label key={evt} className={`check-pill${trig.storageEvents.includes(evt) ? " checked" : ""}`} onClick={e => { e.preventDefault(); toggleDfmTriggerEvent(trig.id, "storageEvents", evt); }}>
                                  <input type="checkbox" checked={trig.storageEvents.includes(evt)} readOnly /><span className="ck"></span>{evt}
                                </label>
                              ))}
                            </div>
                          </div>
                          <div className="field" style={{ marginTop: 10 }}>
                            <label>Prefix <span className="opt">— optional</span></label>
                            <input type="text" className="mono" placeholder="uploads/" autoComplete="off" value={trig.prefix} onChange={e => updateDfmTrigger(trig.id, { prefix: e.target.value })} />
                          </div>
                        </>
                      )}
                    </div>
                  ))}
                </div>
                <button className="add-btn" onClick={addDfmTrigger}>
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 5v14M5 12h14"/></svg>
                  Add trigger
                </button>
              </section>

              {/* Environment variables */}
              <section className="section">
                <h3 className="section-label">Environment variables</h3>
                <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
                  {dfmEnvVars.map(ev => (
                    <div key={ev.id} className="env-row">
                      <button className="remove" onClick={() => removeDfmEnvVar(ev.id)} aria-label="Remove variable">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                      </button>
                      <div className="top">
                        <div className="field">
                          <label>Name</label>
                          <input type="text" className="mono" placeholder="MY_VAR" autoComplete="off" value={ev.name} onChange={e => updateDfmEnvVar(ev.id, { name: e.target.value })} />
                        </div>
                        <div className="field">
                          <label>Source</label>
                          <div className="source-seg">
                            <button type="button" className={ev.source === "value" ? "active" : ""} onClick={() => updateDfmEnvVar(ev.id, { source: "value" })}>Value</button>
                            <button type="button" className={ev.source === "secret" ? "active" : ""} onClick={() => updateDfmEnvVar(ev.id, { source: "secret" })}>Secret</button>
                          </div>
                        </div>
                      </div>
                      <div className="field value-wrap">
                        {ev.source === "value" ? (
                          <>
                            <label>Value</label>
                            <input type="text" placeholder="my-value" autoComplete="off" value={ev.value} onChange={e => updateDfmEnvVar(ev.id, { value: e.target.value })} />
                          </>
                        ) : (
                          <>
                            <label>Secret reference</label>
                            <select value={ev.secretName} onChange={e => updateDfmEnvVar(ev.id, { secretName: e.target.value })}>
                              {["OPENAI_API_KEY", "DATABASE_URL", "STRIPE_SECRET_KEY", "SMTP_PASSWORD", "REDIS_URL", "WEBHOOK_SIGNING_SECRET"].map(s => (
                                <option key={s} value={s}>{s}</option>
                              ))}
                            </select>
                            <span className="secret-list">{"\u2192"} resolved from project secret at runtime</span>
                          </>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
                <button className="add-btn" onClick={addDfmEnvVar}>
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 5v14M5 12h14"/></svg>
                  Add variable
                </button>
                <span className="hint" style={{ color: "var(--fg-mute)", fontSize: "11.5px" }}>Secrets are referenced by name and resolved at runtime.</span>
              </section>
            </div>

            {/* Footer */}
            <div className="foot">
              <button className="btn btn-ghost" onClick={() => setDeployFnModalOpen(false)}>Cancel</button>
              <button className="btn btn-primary" disabled={!dfmNameValid || fnDeploying} onClick={handleDfmDeploy}>
                {fnDeploying && <span className="spinner"></span>}
                <span className="label">{fnDeploying ? "Deploying\u2026" : "Deploy"}</span>
              </button>
            </div>
          </div>
        </div>
      )}

      {/* File detail side panel */}
      {fileDetailOpen && (
        <aside className="file-detail open" aria-hidden="false">
          <div className="head">
            <span className="nm">hero-banner.webp</span>
            <button className="close" onClick={() => setFileDetailOpen(false)} aria-label="Close"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width:14,height:14 }}><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
          </div>
          <div className="body">
            <div className="preview-box">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" style={{ width:36,height:36,zIndex:1 }}><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg>
              <span className="dim">1920 &times; 1080</span>
            </div>
            <div className="meta-grid">
              <div><div className="lbl">Size</div><div className="val">2.4 MB</div></div>
              <div><div className="lbl">Type</div><div className="val">image/webp</div></div>
              <div><div className="lbl">Modified</div><div className="val">May 15, 2026 &middot; 14:32</div></div>
              <div><div className="lbl">Path</div><div className="val">assets / images / products</div></div>
            </div>
            <div>
              <div className="lbl" style={{ fontSize:"10.5px",color:"var(--fg-mute)",textTransform:"uppercase",letterSpacing:"0.07em",marginBottom:6 }}>Public URL</div>
              <div className="url-field">
                <input type="text" readOnly value={`${apiEndpoint}/storage/assets/hero-banner.webp`} />
                <button onClick={() => navigator.clipboard.writeText(`${apiEndpoint}/storage/assets/hero-banner.webp`)} aria-label="Copy URL"><IconCopy /></button>
              </div>
            </div>
            <div className="btns">
              <button className="btn btn-ghost"><IconDownload /> Download</button>
              <button className="btn btn-ghost"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width:13,height:13 }}><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.72-1.71"/></svg> Copy URL</button>
              <button className="btn btn-ghost full"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width:13,height:13 }}><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="9" y1="15" x2="15" y2="15"/></svg> Replace file</button>
              <button className="btn btn-danger-ghost full"><IconTrash /> Delete</button>
            </div>
          </div>
        </aside>
      )}
    </>
  );
}

/* Settings sidebar navigation */
function SettingsNav({ active, onNav, onPause, onDelete }: { active: string; onNav: (s: "general"|"services"|"danger") => void; onPause: () => void; onDelete: () => void }) {
  return (
    <nav className="settings-nav">
      <a className={active === "general" ? "active" : ""} href="#stg-general" onClick={e => { e.preventDefault(); onNav("general"); }}>General</a>
      <a className={active === "services" ? "active" : ""} href="#stg-services" onClick={e => { e.preventDefault(); onNav("services"); }}>Services</a>
      <a className={`${active === "danger" ? "active" : ""} danger`} href="#stg-danger" onClick={e => { e.preventDefault(); onNav("danger"); }}>Danger zone</a>
    </nav>
  );
}

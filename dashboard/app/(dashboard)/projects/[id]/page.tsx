"use client";
import { useState, useEffect, useRef, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { useProject, usePauseProject, useResumeProject, useDeleteProject } from "@/features/projects/hooks";
import { useFunctions, useDeleteFunction } from "@/features/functions/hooks";
import { functionClient } from "@/lib/api/clients";
import { useBuckets, useListObjects, useUploadObject, useDeleteObject } from "@/features/storage/hooks";
import { storageRestClient } from "@/lib/api/clients";
import type { StorageObject, PgTable, PgColumn, PgPolicy } from "@/lib/api/clients";
import { useApiKeys, useCreateApiKey, useRevokeApiKey } from "@/features/api-keys/hooks";
import { useSecrets, useCreateSecret, useDeleteSecret, useUpdateSecretValue } from "@/features/secrets/hooks";
import {
  useTables, useColumns, usePolicies,
  useCreateTable, useDeleteTable,
  useCreateColumn, useDeleteColumn,
  useCreatePolicy, useDeletePolicy,
  useTableData, useRowCount, useInsertRow, useDeleteRow,
  useExecuteQuery,
  useUpdateTable,
} from "@/features/database/hooks";
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

type TabName = "overview" | "database" | "functions" | "events" | "storage" | "secrets" | "apikeys" | "settings";

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
  const { data: secrets } = useSecrets(projectId);
  const createApiKeyMut = useCreateApiKey();
  const revokeApiKeyMut = useRevokeApiKey();
  const createSecretMut = useCreateSecret();
  const deleteSecretMut = useDeleteSecret();
  const updateSecretValueMut = useUpdateSecretValue();
  const pauseProjectMut = usePauseProject();
  const resumeProjectMut = useResumeProject();
  const deleteProjectMut = useDeleteProject();
  const deleteFunctionMut = useDeleteFunction();

  const [activeTab, setActiveTab] = useState<TabName>("overview");
  const [menuOpen, setMenuOpen] = useState(false);
  const [tableMenuOpen, setTableMenuOpen] = useState(false);
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
  const [deleteFnModalOpen, setDeleteFnModalOpen] = useState(false);
  const [deleteFnTarget, setDeleteFnTarget] = useState<{ id: string; name: string } | null>(null);

  /* Delete project */
  const [deleteProjectInput, setDeleteProjectInput] = useState("");
  const [deleteProjectAck, setDeleteProjectAck] = useState(false);

  /* Create key */
  const [createKeyStep, setCreateKeyStep] = useState<1 | 2>(1);
  const [createKeyName, setCreateKeyName] = useState("");
  const [createKeyRole, setCreateKeyRole] = useState<"anon" | "service_role">("anon");
  const [createKeyExp, setCreateKeyExp] = useState("90");
  const [createdRawKey, setCreatedRawKey] = useState("");

  /* Revoke key */
  const [revokeKeyName, setRevokeKeyName] = useState("");
  const [revokeKeyPrefix, setRevokeKeyPrefix] = useState("");
  const [revokeKeyRole, setRevokeKeyRole] = useState("");
  const [revokeConfirmInput, setRevokeConfirmInput] = useState("");
  const [revokeKeyId, setRevokeKeyId] = useState("");

  /* Secrets */
  const [targetSecretName, setTargetSecretName] = useState("");
  const [targetSecretId, setTargetSecretId] = useState("");
  const [addSecretName, setAddSecretName] = useState("");
  const [addSecretValue, setAddSecretValue] = useState("");
  const [addSecretDesc, setAddSecretDesc] = useState("");
  const [rotateSecretValue, setRotateSecretValue] = useState("");
  const [deleteSecretConfirmInput, setDeleteSecretConfirmInput] = useState("");
  const [secValueRevealed, setSecValueRevealed] = useState(false);
  const [rotateValueRevealed, setRotateValueRevealed] = useState(false);

  /* Functions search/filter */
  const [fnSearch, setFnSearch] = useState("");
  const [fnKindFilter, setFnKindFilter] = useState("");
  const [fnStatusFilter, setFnStatusFilter] = useState("");
  const [fnSortAsc, setFnSortAsc] = useState(true);
  const [selectedFnNames, setSelectedFnNames] = useState<Set<string>>(new Set());
  const filteredFunctions = (functions ?? []).filter(fn => {
    if (fnSearch && !fn.name.toLowerCase().includes(fnSearch.toLowerCase()) && !(fn.displayName || "").toLowerCase().includes(fnSearch.toLowerCase())) return false;
    if (fnKindFilter && fn.kind !== fnKindFilter) return false;
    if (fnStatusFilter && fn.status !== fnStatusFilter) return false;
    return true;
  }).sort((a, b) => {
    const cmp = a.name.localeCompare(b.name);
    return fnSortAsc ? cmp : -cmp;
  });

  /* Events */
  const [evtFilter, setEvtFilter] = useState("all");
  const [evtExpanded, setEvtExpanded] = useState<Set<string>>(new Set());
  const [liveToggle, setLiveToggle] = useState(true);

  /* Storage */
  const [activeBucketIdx, setActiveBucketIdx] = useState(0);
  const [storView, setStorView] = useState<"list" | "grid">("list");
  const [uploadOverlayOpen, setUploadOverlayOpen] = useState(false);
  const [fileDetailOpen, setFileDetailOpen] = useState(false);
  const [selectedObject, setSelectedObject] = useState<StorageObject | null>(null);
  const [storSearch, setStorSearch] = useState("");
  const activeBucketName = buckets?.[activeBucketIdx]?.name ?? "";
  const activeBucketData = buckets?.[activeBucketIdx];
  const { data: objects, isLoading: objectsLoading } = useListObjects(projectId, activeBucketName);
  const uploadMutation = useUploadObject();
  const deleteMutation = useDeleteObject();
  const [uploadFiles, setUploadFiles] = useState<Array<{ file: File; progress: number; status: "uploading" | "done" | "error" }>>([]);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const filteredObjects = (objects ?? []).filter(obj =>
    !storSearch || obj.name.toLowerCase().includes(storSearch.toLowerCase())
  );

  const handleFileUpload = useCallback(async (files: FileList) => {
    if (!activeBucketName) return;
    const entries = Array.from(files).map(file => ({ file, progress: 0, status: "uploading" as const }));
    setUploadFiles(entries);
    for (let i = 0; i < entries.length; i++) {
      try {
        await uploadMutation.mutateAsync({
          projectId,
          bucket: activeBucketName,
          path: entries[i].file.name,
          file: entries[i].file,
        });
        setUploadFiles(prev => prev.map((e, j) => j === i ? { ...e, progress: 100, status: "done" } : e));
      } catch {
        setUploadFiles(prev => prev.map((e, j) => j === i ? { ...e, status: "error" } : e));
      }
    }
  }, [activeBucketName, projectId, uploadMutation]);

  const handleFileDownload = useCallback(async (objName: string) => {
    if (!activeBucketName) return;
    try {
      const response = await storageRestClient.downloadObject(projectId, activeBucketName, objName);
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = objName.split("/").pop() || objName;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (err) {
      console.error("Download failed:", err);
    }
  }, [activeBucketName, projectId]);

  const handleFileDelete = useCallback(async (objName: string) => {
    if (!activeBucketName || !confirm(`Delete "${objName}"?`)) return;
    try {
      await deleteMutation.mutateAsync({ projectId, bucket: activeBucketName, path: objName });
      if (selectedObject?.name === objName) {
        setSelectedObject(null);
        setFileDetailOpen(false);
      }
    } catch (err) {
      console.error("Delete failed:", err);
    }
  }, [activeBucketName, projectId, deleteMutation, selectedObject]);

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
      else if (sourceTab === "zip") req.zipSource = { path: zipPath };
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

  /* Database */
  const [dbSub, setDbSub] = useState<"tables" | "sql" | "conn">("tables");
  const [dbView, setDbView] = useState<"columns" | "data" | "rls" | "indexes">("columns");
  const [selectedTableId, setSelectedTableId] = useState<number | null>(null);
  const [collapsedSchemas, setCollapsedSchemas] = useState<Set<string>>(new Set(["storage", "auth"]));
  const [sqlResultTab, setSqlResultTab] = useState<"results" | "messages">("results");
  const [sqlQuery, setSqlQuery] = useState("-- Write your SQL here\nSELECT 1;");
  const [dataPage, setDataPage] = useState(0);
  const dataPageSize = 50;

  // Database hooks
  const { data: dbTables, isLoading: dbTablesLoading } = useTables(projectId);
  const selectedTable = dbTables?.find(t => t.id === selectedTableId) ?? null;
  const selectedTableName = selectedTable?.name ?? "";
  const selectedTableSchema = selectedTable?.schema ?? "public";
  const { data: dbColumns } = useColumns(projectId, selectedTableId ?? 0);
  const { data: dbPolicies } = usePolicies(projectId);
  const { data: dbTableData, isLoading: dbDataLoading } = useTableData(projectId, selectedTableName, dataPageSize, dataPage * dataPageSize);
  const { data: dbRowCount } = useRowCount(projectId, selectedTableName);
  const createTableMut = useCreateTable(projectId);
  const deleteTableMut = useDeleteTable(projectId);
  const createColumnMut = useCreateColumn(projectId);
  const deleteColumnMut = useDeleteColumn(projectId);
  const createPolicyMut = useCreatePolicy(projectId);
  const deletePolicyMut = useDeletePolicy(projectId);
  const insertRowMut = useInsertRow(projectId, selectedTableName);
  const deleteRowMut = useDeleteRow(projectId, selectedTableName);
  const executeQueryMut = useExecuteQuery(projectId);
  const updateTableMut = useUpdateTable(projectId);

  // DB modal states
  const [newTableModalOpen, setNewTableModalOpen] = useState(false);
  const [newTableName, setNewTableName] = useState("");
  const [addColumnModalOpen, setAddColumnModalOpen] = useState(false);
  const [newColName, setNewColName] = useState("");
  const [newColType, setNewColType] = useState("text");
  const [newColNullable, setNewColNullable] = useState(true);
  const [insertRowModalOpen, setInsertRowModalOpen] = useState(false);
  const [insertRowData, setInsertRowData] = useState<Record<string, string>>({});
  const [newPolicyModalOpen, setNewPolicyModalOpen] = useState(false);
  const [newPolicyName, setNewPolicyName] = useState("");
  const [newPolicyDefinition, setNewPolicyDefinition] = useState("true");
  const [sqlMessages, setSqlMessages] = useState<string[]>([]);

  // Auto-select first table when tables load
  useEffect(() => {
    if (dbTables && dbTables.length > 0 && selectedTableId === null) {
      const publicTables = dbTables.filter(t => t.schema === "public");
      if (publicTables.length > 0) setSelectedTableId(publicTables[0].id);
      else setSelectedTableId(dbTables[0].id);
    }
  }, [dbTables, selectedTableId]);

  const toggleSchema = (schema: string) => {
    setCollapsedSchemas(prev => {
      const next = new Set(prev);
      if (next.has(schema)) next.delete(schema); else next.add(schema);
      return next;
    });
  };

  // Group tables by schema
  const tablesBySchema = (dbTables ?? []).reduce<Record<string, PgTable[]>>((acc, t) => {
    (acc[t.schema] ??= []).push(t);
    return acc;
  }, {});
  const schemaOrder = Object.keys(tablesBySchema).sort((a, b) => {
    if (a === "public") return -1;
    if (b === "public") return 1;
    return a.localeCompare(b);
  });
  const systemSchemas = new Set(["auth", "storage", "extensions", "pgbouncer", "realtime", "pgsodium", "vault", "_realtime"]);

  const tablePolicies = (dbPolicies ?? []).filter(
    p => selectedTable && p.table === selectedTable.name && p.schema === selectedTable.schema
  );

  const totalPages = dbRowCount ? Math.ceil(dbRowCount / dataPageSize) : 1;

  const primaryKeyCol = selectedTable?.primary_keys?.[0]?.name ?? dbColumns?.[0]?.name ?? "id";

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
  const activeApiKeys = apiKeys?.filter((k) => !k.revokedAt);
  const akCount = activeApiKeys?.length ?? 0;

  // Close kebab on outside click (skip clicks on the toggle buttons themselves)
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      const t = e.target as HTMLElement;
      if (!t.closest?.('[aria-label="More actions"]')) setMenuOpen(false);
      if (!t.closest?.(".icon-menu-btn")) setTableMenuOpen(false);
    };
    document.addEventListener("click", handler);
    return () => document.removeEventListener("click", handler);
  }, []);

  const toggleEvtExpanded = (trace: string) => {
    setEvtExpanded(prev => {
      const next = new Set(prev);
      if (next.has(trace)) next.delete(trace); else next.add(trace);
      return next;
    });
  };

  const tabs: { key: TabName; label: string; count?: number }[] = [
    { key: "overview", label: "Overview" },
    { key: "database", label: "Database" },
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
              onClick={() => setMenuOpen(!menuOpen)}
            >
              <IconKebab />
            </button>
            {menuOpen && (
              <div className="menu open" role="menu">
                <div className="menu-item"><IconDuplicate /> Duplicate project</div>
                <div className="menu-item"><IconDownload /> Export config</div>
                <div className="menu-divider" />
                <div className="menu-item danger" onClick={() => { setMenuOpen(false); setDeleteProjectModalOpen(true); }}><IconTrash /> Delete project</div>
              </div>
            )}
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
                  <div><span className="lbl">Environment</span><span className="val mono" style={{ fontSize: 12 }}>self-hosted</span></div>
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
                    <div className="row"><span className="row-lbl">Anon key <span className="role-tag anon">public</span></span><div className="connect-field"><span className={`value mono ${revealAnon ? "" : "masked"}`}>{revealAnon ? `${anonKey.keyPrefix}XXXXXXXXXXXXXXXXXX` : `\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf`}</span><span className="field-btns"><button className="field-btn" onClick={() => setRevealAnon(!revealAnon)} aria-label="Reveal key"><IconEye /></button><CopyBtn text={anonKey.keyPrefix} /></span></div></div>
                  )}
                  {serviceKey && (
                    <div className="row"><span className="row-lbl">Service role key <span className="role-tag">secret</span></span><div className="connect-field"><span className={`value mono ${revealService ? "" : "masked"}`}>{revealService ? `${serviceKey.keyPrefix}XXXXXXXXXXXXXXXXXX` : `\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf\u25cf`}</span><span className="field-btns"><button className="field-btn" onClick={() => setRevealService(!revealService)} aria-label="Reveal key"><IconEye /></button><CopyBtn text={serviceKey.keyPrefix} /></span></div></div>
                  )}
                  <details className="snippet">
                    <summary><svg className="chev" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><polyline points="9 18 15 12 9 6"/></svg>Quick start &mdash; JavaScript</summary>
                    <pre><span className="tok-com">{"// install: npm i @etalbaas/js"}</span>{"\n"}<span className="tok-kw">import</span>{" { "}<span className="tok-fn">createClient</span>{" } "}<span className="tok-kw">from</span>{" "}<span className="tok-str">{`'@etalbaas/js'`}</span>{"\n\n"}<span className="tok-kw">const</span>{" client = "}<span className="tok-fn">createClient</span>{"(\n  "}<span className="tok-str">{`'${apiEndpoint}'`}</span>{",\n  "}<span className="tok-str">{`'your-anon-key'`}</span>{"\n)\n\n"}<span className="tok-kw">const</span>{" { data, error } = "}<span className="tok-kw">await</span>{" client\n  ."}<span className="tok-fn">from</span>{"("}<span className="tok-str">{`'users'`}</span>{")\n  ."}<span className="tok-fn">select</span>{"("}<span className="tok-str">{`'*'`}</span>{")\n  ."}<span className="tok-fn">limit</span>{"("}<span className="tok-str">10</span>{")"}</pre>
                  </details>
                </div>
              </section>
            </>
          )}

          {/* ====== Database tab ====== */}
          {activeTab === "database" && (
            <>
              <header className="tab-head">
                <div>
                  <h2>Database</h2>
                  <p>Inspect tables, run queries, and grab connection strings for this project&apos;s PostgreSQL cluster.</p>
                </div>
                <div className="actions">
                  <nav className="sub-tabs" role="tablist">
                    <button className={dbSub === "tables" ? "active" : ""} onClick={() => setDbSub("tables")} role="tab" aria-selected={dbSub === "tables"}>
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><ellipse cx="12" cy="5" rx="9" ry="3"></ellipse><path d="M3 5v14c0 1.66 4.03 3 9 3s9-1.34 9-3V5"></path><path d="M3 12c0 1.66 4.03 3 9 3s9-1.34 9-3"></path></svg>
                      Tables
                    </button>
                    <button className={dbSub === "sql" ? "active" : ""} onClick={() => setDbSub("sql")} role="tab" aria-selected={dbSub === "sql"}>
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="16 18 22 12 16 6"></polyline><polyline points="8 6 2 12 8 18"></polyline></svg>
                      SQL Editor
                    </button>
                    <button className={dbSub === "conn" ? "active" : ""} onClick={() => setDbSub("conn")} role="tab" aria-selected={dbSub === "conn"}>
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path></svg>
                      Connection
                    </button>
                  </nav>
                </div>
              </header>

              {/* ===== Sub: Tables ===== */}
              {dbSub === "tables" && (
                <div className="tables-split">
                  {/* Schema sidebar */}
                  <aside className="schema-sidebar">
                    <div className="schema-sidebar-head">
                      <h4>Schemas</h4>
                      <span className="search-icon" title="Search tables">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="11" cy="11" r="7"></circle><path d="m20 20-3.5-3.5"></path></svg>
                      </span>
                    </div>
                    <div className="schema-list">
                      {dbTablesLoading && <div style={{ padding: "12px 16px", color: "var(--fg-mute)", fontSize: 12 }}>Loading tables...</div>}
                      {schemaOrder.map(schema => (
                        <div key={schema} className={`schema-group${systemSchemas.has(schema) ? " system" : ""}${collapsedSchemas.has(schema) ? " collapsed" : ""}`} data-schema={schema}>
                          <button className="schema-group-head" onClick={() => toggleSchema(schema)}>
                            <svg className="chev" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><polyline points="6 9 12 15 18 9"></polyline></svg>
                            <span className="schema-name">{schema}</span>
                            {systemSchemas.has(schema) && <span className="schema-tag">System</span>}
                          </button>
                          <div className="schema-tables">
                            {(tablesBySchema[schema] ?? []).map(tbl => (
                              <button key={tbl.id} className={`schema-table${selectedTableId === tbl.id ? " active" : ""}`} onClick={() => { setSelectedTableId(tbl.id); setDataPage(0); }}>
                                <svg className="tbl-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><ellipse cx="12" cy="5" rx="9" ry="3"></ellipse><path d="M3 5v14c0 1.66 4.03 3 9 3s9-1.34 9-3V5"></path><path d="M3 12c0 1.66 4.03 3 9 3s9-1.34 9-3"></path></svg>
                                {tbl.name}
                              </button>
                            ))}
                          </div>
                        </div>
                      ))}
                    </div>
                    <div className="schema-sidebar-foot">
                      <button className="new-table-btn" onClick={() => { setNewTableName(""); setNewTableModalOpen(true); }}>
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 5v14M5 12h14"></path></svg>
                        New Table
                      </button>
                    </div>
                  </aside>

                  {/* Table main */}
                  <div className="table-main">
                    <div className="table-main-head">
                      <div className="title-wrap">
                        <h2><span className="schema-prefix">{selectedTableSchema}.</span><span>{selectedTableName}</span></h2>
                        {selectedTable && (
                          <span className={`rls-badge${selectedTable.rls_enabled ? " on" : ""}`}>
                            <span className="dt"></span>{selectedTable.rls_enabled ? "RLS Enabled" : "RLS Disabled"}
                          </span>
                        )}
                      </div>
                      <div className="actions">
                        <button className="toolbar-btn" onClick={() => { setNewColName(""); setNewColType("text"); setNewColNullable(true); setAddColumnModalOpen(true); }}>
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 5v14M5 12h14"></path></svg>
                          Add Column
                        </button>
                        <div className={`icon-menu-btn${tableMenuOpen ? " open" : ""}`} onClick={() => setTableMenuOpen(!tableMenuOpen)}>
                          <svg viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="5" r="1.5"></circle><circle cx="12" cy="12" r="1.5"></circle><circle cx="12" cy="19" r="1.5"></circle></svg>
                          <div className="menu-dropdown">
                            <button onClick={() => {
                              if (!selectedTableName) return;
                              executeQueryMut.mutate(`SELECT * FROM ${selectedTableSchema}.${selectedTableName} LIMIT 0`);
                            }}>
                              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                              Duplicate Structure
                            </button>
                            <button onClick={() => {
                              if (!selectedTableName) return;
                              setSqlQuery(`-- Export: ${selectedTableSchema}.${selectedTableName}\nSELECT * FROM ${selectedTableSchema}.${selectedTableName};`);
                              setDbSub("sql");
                            }}>
                              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg>
                              Export as SQL
                            </button>
                            <div className="sep"></div>
                            <button className="danger" onClick={() => {
                              if (!selectedTableName || !confirm(`Truncate table ${selectedTableName}? This will delete all rows.`)) return;
                              executeQueryMut.mutate(`TRUNCATE TABLE ${selectedTableSchema}.${selectedTableName}`);
                            }}>
                              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M3 6h18"></path><path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"></path></svg>
                              Truncate
                            </button>
                            <button className="danger" onClick={() => {
                              if (!selectedTableId || !confirm(`Drop table ${selectedTableName}? This cannot be undone.`)) return;
                              deleteTableMut.mutate(selectedTableId, {
                                onSuccess: () => setSelectedTableId(null),
                              });
                            }}>
                              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6l-2 14a2 2 0 0 1-2 2H9a2 2 0 0 1-2-2L5 6"></path></svg>
                              Drop Table
                            </button>
                          </div>
                        </div>
                      </div>
                    </div>

                    {/* Mini tabs */}
                    <div className="mini-tabs" role="tablist">
                      {(["columns", "data", "rls", "indexes"] as const).map(v => (
                        <button key={v} className={`mini-tab${dbView === v ? " active" : ""}`} onClick={() => setDbView(v)} role="tab" aria-selected={dbView === v}>
                          {v === "columns" ? "Columns" : v === "data" ? "Data" : v === "rls" ? "RLS Policies" : "Indexes"}
                        </button>
                      ))}
                    </div>

                    {/* Columns view */}
                    <div className={`mini-pane${dbView === "columns" ? " active" : ""}`}>
                      <div className="col-table">
                        <div className="row head">
                          <div className="cell">Name</div>
                          <div className="cell">Type</div>
                          <div className="cell">Default</div>
                          <div className="cell">Nullable</div>
                          <div className="cell">Primary</div>
                          <div className="cell">Unique</div>
                          <div className="cell"></div>
                        </div>
                        {(dbColumns ?? []).map(col => {
                          const isPk = selectedTable?.primary_keys?.some(pk => pk.name === col.name);
                          return (
                            <div key={col.id} className="row body">
                              <div className="cell"><span className="col-name">{isPk && <svg className="pk-key" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="8" cy="15" r="4"></circle><path d="M10.85 12.15L19 4"></path><path d="M18 5l3 3"></path><path d="M15 8l3 3"></path></svg>}{col.name}</span></div>
                              <div className="cell"><span className="col-type">{col.format}</span></div>
                              <div className="cell">{col.default_value ? <span className="col-default">{col.default_value}</span> : <span className="col-default empty">&mdash;</span>}</div>
                              <div className="cell"><span className={`col-check${col.is_nullable ? " yes" : ""}`}>{col.is_nullable ? <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg> : <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>}</span></div>
                              <div className="cell"><span className={`col-check${isPk ? " yes" : ""}`}>{isPk ? <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg> : <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>}</span></div>
                              <div className="cell"><span className={`col-check${col.is_unique ? " yes" : ""}`}>{col.is_unique ? <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg> : <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>}</span></div>
                              <div className="cell"><div className="col-actions"><button title="Edit"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path><path d="M18.5 2.5a2.12 2.12 0 0 1 3 3L12 15l-4 1 1-4z"></path></svg></button><button className="del" title="Delete" onClick={() => { if (confirm(`Delete column ${col.name}?`)) deleteColumnMut.mutate(col.id); }}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6l-2 14a2 2 0 0 1-2 2H9a2 2 0 0 1-2-2L5 6"></path></svg></button></div></div>
                            </div>
                          );
                        })}
                        {(dbColumns ?? []).length === 0 && <div className="row body"><div className="cell" style={{ gridColumn: "1 / -1", color: "var(--fg-mute)" }}>No columns</div></div>}
                      </div>
                    </div>

                    {/* Data view */}
                    <div className={`mini-pane${dbView === "data" ? " active" : ""}`}>
                      <div className="data-toolbar">
                        <div className="left">
                          <span className="row-count">{dbRowCount ?? 0} rows</span>
                        </div>
                        <div className="right">
                          <button className="toolbar-btn" onClick={() => { /* TODO: filter UI */ }}>
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"></polygon></svg>
                            Filter
                          </button>
                          <button className="toolbar-btn" onClick={() => setDataPage(p => p)}>
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="23 4 23 10 17 10"></polyline><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path></svg>
                            Refresh
                          </button>
                          <button className="toolbar-btn primary" onClick={() => { setInsertRowData({}); setInsertRowModalOpen(true); }}>
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round"><path d="M12 5v14M5 12h14"></path></svg>
                            Insert Row
                          </button>
                        </div>
                      </div>
                      <div className="data-table-wrap">
                        {dbDataLoading ? (
                          <div style={{ padding: "24px", color: "var(--fg-mute)", textAlign: "center" }}>Loading data...</div>
                        ) : (
                        <table className="data-table">
                          <thead>
                            <tr>
                              {(dbColumns ?? []).map(col => <th key={col.id}>{col.name}</th>)}
                              <th className="actions"></th>
                            </tr>
                          </thead>
                          <tbody>
                            {(dbTableData ?? []).map((row, ri) => (
                              <tr key={ri}>
                                {(dbColumns ?? []).map(col => {
                                  const val = row[col.name];
                                  const isNull = val === null || val === undefined;
                                  const isUuid = col.format === "uuid" && typeof val === "string";
                                  return (
                                    <td key={col.id} className={isNull ? "null" : isUuid ? "uuid" : ""}>
                                      {isNull ? "NULL" : isUuid ? `${String(val).slice(0, 8)}...` : String(val)}
                                    </td>
                                  );
                                })}
                                <td><div className="row-actions"><button title="Edit row"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path><path d="M18.5 2.5a2.12 2.12 0 0 1 3 3L12 15l-4 1 1-4z"></path></svg></button><button className="del" onClick={() => { if (confirm("Delete this row?")) deleteRowMut.mutate({ pkCol: primaryKeyCol, pkVal: row[primaryKeyCol] }); }}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6l-2 14a2 2 0 0 1-2 2H9a2 2 0 0 1-2-2L5 6"></path></svg></button></div></td>
                              </tr>
                            ))}
                            {(dbTableData ?? []).length === 0 && <tr><td colSpan={(dbColumns?.length ?? 1) + 1} style={{ color: "var(--fg-mute)", textAlign: "center" }}>No data</td></tr>}
                          </tbody>
                        </table>
                        )}
                      </div>
                      <div className="pagination-bar">
                        <span>Showing {dataPage * dataPageSize + 1}&ndash;{Math.min((dataPage + 1) * dataPageSize, dbRowCount ?? 0)} of {dbRowCount ?? 0} rows</span>
                        <div className="pages">
                          <button className="pg-btn" disabled={dataPage === 0} onClick={() => setDataPage(p => Math.max(0, p - 1))}>&larr;</button>
                          {Array.from({ length: Math.min(totalPages, 5) }, (_, i) => (
                            <button key={i} className={`pg-btn${dataPage === i ? " active" : ""}`} onClick={() => setDataPage(i)}>{i + 1}</button>
                          ))}
                          <button className="pg-btn" disabled={dataPage >= totalPages - 1} onClick={() => setDataPage(p => p + 1)}>&rarr;</button>
                        </div>
                      </div>
                    </div>

                    {/* RLS Policies view */}
                    <div className={`mini-pane${dbView === "rls" ? " active" : ""}`}>
                      <div className="rls-toolbar">
                        <div className="toggle-wrap">
                          <input type="checkbox" className="rls-switch" checked={selectedTable?.rls_enabled ?? false} onChange={(e) => {
                            if (selectedTableId) updateTableMut.mutate({ tableId: selectedTableId, rls_enabled: e.target.checked });
                          }} />
                          <span>RLS: <strong style={{ color: selectedTable?.rls_enabled ? "var(--ok)" : "var(--warn)" }}>{selectedTable?.rls_enabled ? "Enabled" : "Disabled"}</strong></span>
                        </div>
                        <button className="toolbar-btn primary" onClick={() => { setNewPolicyName(""); setNewPolicyDefinition("true"); setNewPolicyModalOpen(true); }}>
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round"><path d="M12 5v14M5 12h14"></path></svg>
                          New Policy
                        </button>
                      </div>
                      {tablePolicies.map(policy => (
                        <div key={policy.id} className="policy-card">
                          <div className="pc-head">
                            <span className="pc-name"><span className="ind"></span>{policy.name}</span>
                            <div className="pc-actions">
                              <button>Edit</button>
                              <button className="del" onClick={() => { if (confirm(`Drop policy ${policy.name}?`)) deletePolicyMut.mutate(policy.id); }}>Drop</button>
                            </div>
                          </div>
                          <div className="pc-row">
                            <span className="lbl">Roles</span>
                            <span className="val">{policy.roles.map(r => <span key={r} className="pill">{r}</span>)}</span>
                          </div>
                          <div className="pc-row">
                            <span className="lbl">Command</span>
                            <span className="val"><span className="pill">{policy.command}</span></span>
                          </div>
                          {policy.definition && (
                            <div className="pc-row" style={{ display: "block" }}>
                              <span className="lbl">USING</span>
                              <div className="pc-sql" style={{ marginTop: 4 }}>{policy.definition}</div>
                            </div>
                          )}
                          {policy.check && (
                            <div className="pc-row" style={{ display: "block" }}>
                              <span className="lbl">CHECK</span>
                              <div className="pc-sql" style={{ marginTop: 4 }}>{policy.check}</div>
                            </div>
                          )}
                        </div>
                      ))}
                      {tablePolicies.length === 0 && <div style={{ padding: "24px", color: "var(--fg-mute)", textAlign: "center" }}>No policies defined for this table</div>}
                    </div>

                    {/* Indexes view */}
                    <div className={`mini-pane${dbView === "indexes" ? " active" : ""}`}>
                      <div className="data-toolbar">
                        <div className="left" style={{ color: "var(--fg-mute)", fontSize: 12 }}>2 indexes</div>
                        <div className="right">
                          <button className="toolbar-btn primary">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round"><path d="M12 5v14M5 12h14"></path></svg>
                            Create Index
                          </button>
                        </div>
                      </div>
                      <div className="idx-table">
                        <div className="row head">
                          <div className="cell">Name</div>
                          <div className="cell">Columns</div>
                          <div className="cell">Type</div>
                          <div className="cell">Unique</div>
                          <div className="cell"></div>
                        </div>
                        <div className="row body">
                          <div className="cell"><span className="idx-name">profiles_pkey</span></div>
                          <div className="cell"><span className="idx-cols">id</span></div>
                          <div className="cell"><span className="idx-type">btree</span></div>
                          <div className="cell"><span className="col-check yes"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg></span></div>
                          <div className="cell"><div className="col-actions" style={{ opacity: 1 }}><button className="del"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6l-2 14a2 2 0 0 1-2 2H9a2 2 0 0 1-2-2L5 6"></path></svg></button></div></div>
                        </div>
                        <div className="row body">
                          <div className="cell"><span className="idx-name">profiles_email_idx</span></div>
                          <div className="cell"><span className="idx-cols">email</span></div>
                          <div className="cell"><span className="idx-type">btree</span></div>
                          <div className="cell"><span className="col-check yes"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg></span></div>
                          <div className="cell"><div className="col-actions" style={{ opacity: 1 }}><button className="del"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6l-2 14a2 2 0 0 1-2 2H9a2 2 0 0 1-2-2L5 6"></path></svg></button></div></div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              )}

              {/* ===== Sub: SQL Editor ===== */}
              {dbSub === "sql" && (
                <div className="sql-editor-frame">
                  <div className="sql-editor-toolbar">
                    <div className="left">
                      <button className="toolbar-btn primary" disabled={executeQueryMut.isPending} onClick={() => {
                        const now = new Date().toLocaleTimeString("en-US", { hour12: false });
                        executeQueryMut.mutate(sqlQuery, {
                          onSuccess: (result) => {
                            const count = Array.isArray(result) ? result.length : 0;
                            setSqlMessages(prev => [...prev, `[${now}] Query executed successfully. ${count} rows returned.`]);
                            setSqlResultTab("results");
                          },
                          onError: (err) => {
                            setSqlMessages(prev => [...prev, `[${now}] ERROR: ${err.message}`]);
                            setSqlResultTab("messages");
                          },
                        });
                      }}>
                        <svg viewBox="0 0 24 24" fill="currentColor"><polygon points="6 4 20 12 6 20 6 4"></polygon></svg>
                        {executeQueryMut.isPending ? "Running..." : "Run"}
                      </button>
                      <span className="kbd">Ctrl + Enter</span>
                    </div>
                    <div className="right">
                      <button className="toolbar-btn" onClick={() => { setSqlQuery(""); setSqlMessages([]); executeQueryMut.reset(); }}>
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6l-2 14a2 2 0 0 1-2 2H9a2 2 0 0 1-2-2L5 6"></path></svg>
                        Clear
                      </button>
                    </div>
                  </div>
                  <div className="sql-editor-area">
                    <div className="sql-line-numbers">{sqlQuery.split("\n").map((_, i) => i + 1).join("\n")}</div>
                    <textarea
                      className="sql-code"
                      spellCheck="false"
                      value={sqlQuery}
                      onChange={(e) => setSqlQuery(e.target.value)}
                      onKeyDown={(e) => {
                        if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
                          e.preventDefault();
                          const now = new Date().toLocaleTimeString("en-US", { hour12: false });
                          executeQueryMut.mutate(sqlQuery, {
                            onSuccess: (result) => {
                              const count = Array.isArray(result) ? result.length : 0;
                              setSqlMessages(prev => [...prev, `[${now}] Query executed successfully. ${count} rows returned.`]);
                              setSqlResultTab("results");
                            },
                            onError: (err) => {
                              setSqlMessages(prev => [...prev, `[${now}] ERROR: ${err.message}`]);
                              setSqlResultTab("messages");
                            },
                          });
                        }
                      }}
                      style={{ width: "100%", minHeight: 120, resize: "vertical", background: "transparent", color: "var(--fg)", border: "none", outline: "none", fontFamily: "inherit", fontSize: "inherit", lineHeight: "inherit", padding: 0 }}
                    />
                  </div>
                  <div className="sql-results-frame">
                    <div className="sql-results-tabs" role="tablist">
                      <button className={sqlResultTab === "results" ? "active" : ""} onClick={() => setSqlResultTab("results")} role="tab" aria-selected={sqlResultTab === "results"}>Results</button>
                      <button className={sqlResultTab === "messages" ? "active" : ""} onClick={() => setSqlResultTab("messages")} role="tab" aria-selected={sqlResultTab === "messages"}>Messages</button>
                    </div>
                    <div className={`sql-results-pane${sqlResultTab === "results" ? " active" : ""}`}>
                      {executeQueryMut.data ? (() => {
                        const rows = executeQueryMut.data;
                        const cols = rows.length > 0 ? Object.keys(rows[0]) : [];
                        return (
                        <>
                          <div className="data-table-wrap" style={{ maxHeight: "none" }}>
                            <table className="data-table">
                              <thead>
                                <tr>
                                  {cols.map((c, i) => <th key={i}>{c}</th>)}
                                </tr>
                              </thead>
                              <tbody>
                                {rows.map((row, ri) => (
                                  <tr key={ri}>
                                    {cols.map((c, ci) => {
                                      const val = row[c];
                                      return <td key={ci} className={val === null ? "null" : ""}>{val === null ? "NULL" : String(val)}</td>;
                                    })}
                                  </tr>
                                ))}
                              </tbody>
                            </table>
                          </div>
                          <div className="sql-result-meta">
                            <span className="dot-ok"></span>
                            <span>{rows.length} rows returned</span>
                          </div>
                        </>
                        );
                      })() : executeQueryMut.isError ? (
                        <div className="sql-message" style={{ color: "var(--err)" }}>{executeQueryMut.error?.message}</div>
                      ) : (
                        <div style={{ padding: "24px", color: "var(--fg-mute)", textAlign: "center" }}>Run a query to see results</div>
                      )}
                    </div>
                    <div className={`sql-results-pane${sqlResultTab === "messages" ? " active" : ""}`}>
                      {sqlMessages.length > 0 ? sqlMessages.map((msg, i) => (
                        <div key={i} className={`sql-message${msg.includes("ERROR") ? "" : " success"}`}>{msg}</div>
                      )) : (
                        <div style={{ padding: "24px", color: "var(--fg-mute)", textAlign: "center" }}>No messages</div>
                      )}
                    </div>
                  </div>
                </div>
              )}

              {/* ===== Sub: Connection ===== */}
              {dbSub === "conn" && (
                <>
                  <div className="conn-panels">
                    {/* Direct connection */}
                    <div className="conn-panel">
                      <div className="conn-panel-head">
                        <div>
                          <h3>Direct connection</h3>
                          <span className="sub">Connect from any PostgreSQL client (psql, DBeaver, TablePlus).</span>
                        </div>
                        <span className="badge-ok"><span className="dt"></span>Healthy</span>
                      </div>
                      <div className="conn-row">
                        <span className="label">Host</span>
                        <span className="val">
                          <span className="vtext">xk7a9bc2.db.local.etalbaas.dev</span>
                          <button className="icon-btn" title="Copy" aria-label="Copy"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg></button>
                        </span>
                      </div>
                      <div className="conn-row">
                        <span className="label">Port</span>
                        <span className="val">
                          <span className="vtext">5432</span>
                          <button className="icon-btn" title="Copy" aria-label="Copy"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg></button>
                        </span>
                      </div>
                      <div className="conn-row">
                        <span className="label">Database</span>
                        <span className="val">
                          <span className="vtext">postgres</span>
                          <button className="icon-btn" title="Copy" aria-label="Copy"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg></button>
                        </span>
                      </div>
                      <div className="conn-row">
                        <span className="label">User</span>
                        <span className="val">
                          <span className="vtext">app</span>
                          <button className="icon-btn" title="Copy" aria-label="Copy"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg></button>
                        </span>
                      </div>
                      <div className="conn-row">
                        <span className="label">Password</span>
                        <span className="val">
                          <span className="vtext masked">••••••••••••••••</span>
                          <button className="icon-btn" title="Show / hide" aria-label="Toggle visibility"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg></button>
                          <button className="icon-btn" title="Copy" aria-label="Copy"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg></button>
                        </span>
                      </div>
                      <div className="conn-row">
                        <span className="label">Connection String</span>
                        <span className="val">
                          <span className="vtext">postgresql://app:****@xk7a9bc2.db.local.etalbaas.dev:5432/postgres?sslmode=require</span>
                          <button className="icon-btn" title="Copy" aria-label="Copy"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg></button>
                        </span>
                      </div>
                    </div>

                    {/* PostgREST API */}
                    <div className="conn-panel">
                      <div className="conn-panel-head">
                        <div>
                          <h3>PostgREST API</h3>
                          <span className="sub">Instant REST API. Tables and views are auto-exposed under <span className="mono" style={{ color: "var(--fg-dim)" }}>/rest/{"{table}"}</span>.</span>
                        </div>
                      </div>
                      <div className="conn-row">
                        <span className="label">REST URL</span>
                        <span className="val">
                          <span className="vtext">https://xk7a9bc2.api.local.etalbaas.dev/rest</span>
                          <button className="icon-btn" title="Copy" aria-label="Copy"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg></button>
                        </span>
                      </div>
                      <div className="conn-row">
                        <span className="label">Anon Key</span>
                        <span className="val">
                          <span className="vtext masked">eyJhbGciOi••••••••••••••••</span>
                          <button className="icon-btn" title="Show / hide" aria-label="Toggle visibility"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg></button>
                          <button className="icon-btn" title="Copy" aria-label="Copy"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg></button>
                        </span>
                      </div>
                      <div className="conn-row">
                        <span className="label">Service Role</span>
                        <span className="val">
                          <span className="vtext masked">eyJhbGciOi••••••••••••••••</span>
                          <button className="icon-btn" title="Show / hide" aria-label="Toggle visibility"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg></button>
                          <button className="icon-btn" title="Copy" aria-label="Copy"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg></button>
                        </span>
                      </div>
                    </div>

                    {/* Connection pooling */}
                    <div className="conn-panel">
                      <div className="conn-panel-head">
                        <div>
                          <h3>Connection pooling</h3>
                          <span className="sub">Use the pooler endpoint for serverless and short-lived connections.</span>
                        </div>
                        <span className="badge-info">PgBouncer</span>
                      </div>
                      <div className="conn-row">
                        <span className="label">Mode</span>
                        <span className="val"><span className="vtext">Transaction</span></span>
                      </div>
                      <div className="conn-row">
                        <span className="label">Pool size</span>
                        <span className="val"><span className="vtext">100</span></span>
                      </div>
                      <div className="conn-row">
                        <span className="label">Pooler host</span>
                        <span className="val">
                          <span className="vtext">db-pooler-rw.project-xk7a9bc2.svc:5432</span>
                          <button className="icon-btn" title="Copy" aria-label="Copy"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg></button>
                        </span>
                      </div>
                    </div>
                  </div>

                  {/* Quick start snippets */}
                  <div className="quickstart-section">
                    <h3>Quick start</h3>
                    <details className="qs-block" open>
                      <summary>
                        <svg className="chev" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><polyline points="9 18 15 12 9 6"></polyline></svg>
                        <span className="label">
                          <svg className="lang-icon" viewBox="0 0 24 24" fill="none" stroke="#fcd34d" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="16 18 22 12 16 6"></polyline><polyline points="8 6 2 12 8 18"></polyline></svg>
                          JavaScript <span style={{ color: "var(--fg-mute)", fontWeight: 400, fontSize: "11.5px" }}>— Supabase Client</span>
                        </span>
                        <button className="copy-btn">
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                          Copy
                        </button>
                      </summary>
                      <pre><span className="tok-kw">import</span>{" { createClient } "}<span className="tok-kw">from</span>{" "}<span className="tok-str">{`'@supabase/supabase-js'`}</span>{"\n\n"}<span className="tok-kw">const</span>{" supabase = "}<span className="tok-fn">createClient</span>{"(\n  "}<span className="tok-str">{`'https://xk7a9bc2.api.local.etalbaas.dev'`}</span>{",\n  "}<span className="tok-str">{`'your-anon-key'`}</span>{"\n)\n\n"}<span className="tok-kw">const</span>{" { data, error } = "}<span className="tok-kw">await</span>{" supabase\n  ."}<span className="tok-fn">from</span>{"("}<span className="tok-str">{`'profiles'`}</span>{")\n  ."}<span className="tok-fn">select</span>{"("}<span className="tok-str">{`'*'`}</span>{")\n  ."}<span className="tok-fn">eq</span>{"("}<span className="tok-str">{`'role'`}</span>{", "}<span className="tok-str">{`'admin'`}</span>{")"}</pre>
                    </details>
                    <details className="qs-block">
                      <summary>
                        <svg className="chev" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><polyline points="9 18 15 12 9 6"></polyline></svg>
                        <span className="label">
                          <svg className="lang-icon" viewBox="0 0 24 24" fill="none" stroke="#93c5fd" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="16 18 22 12 16 6"></polyline><polyline points="8 6 2 12 8 18"></polyline></svg>
                          Python <span style={{ color: "var(--fg-mute)", fontWeight: 400, fontSize: "11.5px" }}>— psycopg2</span>
                        </span>
                        <button className="copy-btn">
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                          Copy
                        </button>
                      </summary>
                      <pre><span className="tok-kw">import</span>{" psycopg2\n\nconn = psycopg2."}<span className="tok-fn">connect</span>{"(\n    host="}<span className="tok-str">{`"xk7a9bc2.db.local.etalbaas.dev"`}</span>{",\n    port="}<span className="tok-num">5432</span>{",\n    dbname="}<span className="tok-str">{`"postgres"`}</span>{",\n    user="}<span className="tok-str">{`"app"`}</span>{",\n    password="}<span className="tok-str">{`"your-password"`}</span>{",\n    sslmode="}<span className="tok-str">{`"require"`}</span>{",\n)\ncur = conn."}<span className="tok-fn">cursor</span>{"()\ncur."}<span className="tok-fn">execute</span>{"("}<span className="tok-str">{`"SELECT id, email FROM public.profiles LIMIT 10"`}</span>{")\n"}<span className="tok-kw">for</span>{" row "}<span className="tok-kw">in</span>{" cur."}<span className="tok-fn">fetchall</span>{"():\n    "}<span className="tok-fn">print</span>{"(row)"}</pre>
                    </details>
                    <details className="qs-block">
                      <summary>
                        <svg className="chev" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round"><polyline points="9 18 15 12 9 6"></polyline></svg>
                        <span className="label">
                          <svg className="lang-icon" viewBox="0 0 24 24" fill="none" stroke="#86efac" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="4 17 10 11 4 5"></polyline><line x1="12" y1="19" x2="20" y2="19"></line></svg>
                          psql <span style={{ color: "var(--fg-mute)", fontWeight: 400, fontSize: "11.5px" }}>— command line</span>
                        </span>
                        <button className="copy-btn">
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                          Copy
                        </button>
                      </summary>
                      <pre>{"psql "}<span className="tok-str">{`"postgresql://app:****@xk7a9bc2.db.local.etalbaas.dev:5432/postgres?sslmode=require"`}</span></pre>
                    </details>
                  </div>
                </>
              )}
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
                    <label className="search-input"><IconSearch /><input type="text" placeholder={"Search functions\u2026"} aria-label="Search functions" value={fnSearch} onChange={(e) => setFnSearch(e.target.value)} /><span className="kbd-hint">/</span></label>
                    <div className="select-wrap"><select aria-label="Kind filter" value={fnKindFilter} onChange={(e) => setFnKindFilter(e.target.value)}><option value="">All kinds</option><option value="heavy-job">Heavy Job</option><option value="heavy-deployment">Heavy Deploy</option><option value="light-deployment">Light Deploy</option></select></div>
                    <div className="select-wrap"><select aria-label="Status filter" value={fnStatusFilter} onChange={(e) => setFnStatusFilter(e.target.value)}><option value="">All statuses</option><option value="ready">Ready</option><option value="building">Building</option><option value="pending">Pending</option><option value="failed">Failed</option></select></div>
                  </div>
                  <div className="panel">
                    <div className="fn-table" role="table">
                      <div className="row header" role="row">
                        <input type="checkbox" className="checkbox" aria-label="Select all" checked={filteredFunctions.length > 0 && filteredFunctions.every(fn => selectedFnNames.has(fn.name))} onChange={(e) => { if (e.target.checked) setSelectedFnNames(new Set(filteredFunctions.map(fn => fn.name))); else setSelectedFnNames(new Set()); }} />
                        <button className="sortable sorted" onClick={() => setFnSortAsc(!fnSortAsc)}>Name <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" style={{ transform: fnSortAsc ? "rotate(0)" : "rotate(180deg)", transition: "transform .15s" }}><polyline points="6 9 12 15 18 9"/></svg></button>
                        <span>Kind</span><span>Mode</span><span>Status</span><span>Triggers</span><span>Last built</span><span style={{ textAlign: "right" }}>Actions</span>
                      </div>
                      {filteredFunctions.map(fn => (
                        <div key={fn.name} className={`row body-row ${fn.status === "failed" ? "failed-row" : ""}`} role="row" tabIndex={0} onClick={() => router.push(`/projects/${projectId}/functions/${fn.id}`)} style={{ cursor: "pointer" }}>
                          <input type="checkbox" className="checkbox row-check" aria-label="Select function" checked={selectedFnNames.has(fn.name)} onChange={(e) => { e.stopPropagation(); const next = new Set(selectedFnNames); if (e.target.checked) next.add(fn.name); else next.delete(fn.name); setSelectedFnNames(next); }} />
                          <div className="cell-name"><span className="nm">{fn.name}</span><span className="disp">{fn.displayName || fn.name}</span></div>
                          <span className={`kind-badge ${fn.kind === "heavy-job" ? "heavy-job" : fn.kind === "heavy-deployment" ? "heavy-deployment" : "light-deployment"}`}><span className="dt" />{fn.kind === "heavy-job" ? "Heavy Job" : fn.kind === "heavy-deployment" ? "Heavy Deploy" : "Light Deploy"}</span>
                          <span className="mode-cell">{fn.mode || "async"}</span>
                          <span className="status-cell"><span className={`badge-s ${fn.status === "ready" ? "delivered" : fn.status === "building" ? "created" : fn.status === "failed" ? "failed" : "retrying"}`} style={{ fontWeight: 500 }}><span className="bd" style={{ width: 6, height: 6, borderRadius: "50%", background: "currentColor", display: "inline-block" }} />{fn.status.charAt(0).toUpperCase() + fn.status.slice(1)}</span></span>
                          <span className="triggers-cell"><span className="tg-mute">&mdash;</span></span>
                          <span className={`built-cell ${fn.status === "building" ? "building" : fn.status === "failed" ? "failed-text" : ""}`}>
                            {fn.status === "building" ? (<><span className="mini-spinner" /><span className="progress-bar" /></>) : fn.lastBuiltAt ? formatRelative(fn.lastBuiltAt) : (<span className="mono" style={{ color: "var(--fg-mute)" }}>Never</span>)}
                          </span>
                          <span className="actions-cell">
                            <button className="action-btn" aria-label="View logs"><IconLogs /></button>
                            <button className="action-btn" aria-label="Rebuild"><IconRefresh /></button>
                            <button className="action-btn danger" aria-label="Delete" onClick={e => { e.stopPropagation(); setDeleteFnTarget({ id: fn.id, name: fn.name }); setDeleteFnModalOpen(true); }}><IconTrash /></button>
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
                  <input type="text" placeholder={"Search by trace ID\u2026"} className="mono" style={{ fontFamily: "'JetBrains Mono',monospace" }} aria-label="Search by trace ID" />
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
                            <details className="payload-block" onClick={(e) => e.stopPropagation()}>
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
                  {(buckets ?? []).map((b, i) => (
                    <div key={b.id} className={`item ${activeBucketIdx === i ? "active" : ""}`} onClick={() => { setActiveBucketIdx(i); setSelectedObject(null); setFileDetailOpen(false); setStorSearch(""); }}>
                      <div className="row1"><span className="nm">{b.name}</span><span className={`access-badge ${b.accessLevel}`}>{b.accessLevel}</span></div>
                      <span className="meta">{b.createdAt ? formatRelative(b.createdAt) : ""}</span>
                    </div>
                  ))}
                  <button className="create"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 13, height: 13 }}><path d="M12 5v14M5 12h14"/></svg> Create bucket</button>
                </div>
                <div className="file-browser">
                  {activeBucketData ? (
                    <>
                      <div className="browser-head">
                        <div className="left">
                          <span className="bkt-name">{activeBucketData.name}</span>
                          <span className={`access-badge ${activeBucketData.accessLevel}`}>{activeBucketData.accessLevel}</span>
                          {activeBucketData.fileSizeLimit > 0 && <span className="pill-meta">{Math.round(activeBucketData.fileSizeLimit / 1048576)} MB max</span>}
                          {activeBucketData.allowedMimeTypes?.length > 0 && <span className="pill-meta">{activeBucketData.allowedMimeTypes.join(", ")}</span>}
                        </div>
                        <div className="actions">
                          <input type="file" ref={fileInputRef} style={{ display: "none" }} multiple onChange={e => { if (e.target.files?.length) { handleFileUpload(e.target.files); setUploadOverlayOpen(true); } }} />
                          <button className="btn btn-ghost" onClick={() => { setUploadOverlayOpen(true); fileInputRef.current?.click(); }}>
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 14, height: 14 }}><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
                            Upload
                          </button>
                          <button className="icon-only-btn" aria-label="Bucket settings">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .34 1.86l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.7 1.7 0 0 0-1.86-.34 1.7 1.7 0 0 0-1.04 1.56V21a2 2 0 1 1-4 0v-.09A1.7 1.7 0 0 0 9 19.4a1.7 1.7 0 0 0-1.86.34l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.7 1.7 0 0 0 4.6 15a1.7 1.7 0 0 0-1.56-1.04H3a2 2 0 1 1 0-4h.09A1.7 1.7 0 0 0 4.6 9a1.7 1.7 0 0 0-.34-1.86l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.7 1.7 0 0 0 9 4.6c.43-.18.72-.6.72-1.06V3a2 2 0 1 1 4 0v.09c0 .46.29.88.72 1.06a1.7 1.7 0 0 0 1.86-.34l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.7 1.7 0 0 0 19.4 9c.18.43.6.72 1.06.72H21a2 2 0 1 1 0 4h-.09A1.7 1.7 0 0 0 19.4 15z"/></svg>
                          </button>
                        </div>
                      </div>
                      <div className="browser-crumbs">
                        <span className="here">{activeBucketData.name}</span>
                      </div>
                      <div className="browser-tools">
                        <label className="search-input"><IconSearch /><input type="text" placeholder={"Search files\u2026"} aria-label="Search files" value={storSearch} onChange={e => setStorSearch(e.target.value)} /><span className="kbd-hint">/</span></label>
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
                      {objectsLoading && <div style={{ padding: 24, textAlign: "center", color: "var(--fg-mute)" }}>Loading objects...</div>}
                      {!objectsLoading && filteredObjects.length === 0 && (
                        <div style={{ padding: 48, textAlign: "center", color: "var(--fg-mute)" }}>
                          {storSearch ? "No files match your search." : "This bucket is empty."}
                          {!storSearch && <div style={{ marginTop: 12 }}><button className="btn btn-ghost" onClick={() => fileInputRef.current?.click()}>Upload files</button></div>}
                        </div>
                      )}
                      {/* LIST VIEW */}
                      {!objectsLoading && filteredObjects.length > 0 && storView === "list" && (
                        <div className="file-table">
                          <div className="row header"><span></span><span></span><span>Name</span><span>Size</span><span>Type</span><span>Modified</span><span style={{ textAlign: "right" }}>Actions</span></div>
                          {filteredObjects.map(obj => (
                            <div key={obj.id} className="row body-row" onClick={() => { setSelectedObject(obj); setFileDetailOpen(true); }}>
                              <input type="checkbox" className="checkbox" aria-label="Select" onClick={e => e.stopPropagation()} />
                              <span className="file-icon doc"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg></span>
                              <span className="file-name">{obj.name}</span>
                              <span className="file-size">{obj.size != null ? (obj.size > 1048576 ? `${(obj.size / 1048576).toFixed(1)} MB` : obj.size > 1024 ? `${(obj.size / 1024).toFixed(0)} KB` : `${obj.size} B`) : "\u2014"}</span>
                              <span className="file-type">{obj.mime_type || "\u2014"}</span>
                              <span className="file-mod">{obj.updated_at ? formatRelative(obj.updated_at) : "\u2014"}</span>
                              <span className="file-actions">
                                <button className="action-btn" aria-label="Download" onClick={e => { e.stopPropagation(); handleFileDownload(obj.name); }}><IconDownload /></button>
                                <button className="action-btn" aria-label="Copy URL" onClick={e => { e.stopPropagation(); navigator.clipboard.writeText(`${activeBucketData.name}/${obj.name}`); }}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.72-1.71"/></svg></button>
                                <button className="action-btn danger" aria-label="Delete" onClick={e => { e.stopPropagation(); handleFileDelete(obj.name); }}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-2 14a2 2 0 0 1-2 2H9a2 2 0 0 1-2-2L5 6"/></svg></button>
                              </span>
                            </div>
                          ))}
                        </div>
                      )}
                      {/* GRID VIEW */}
                      {!objectsLoading && filteredObjects.length > 0 && storView === "grid" && (
                        <div className="file-grid active">
                          {filteredObjects.map(obj => (
                            <div key={obj.id} className="grid-tile" onClick={() => { setSelectedObject(obj); setFileDetailOpen(true); }}>
                              <div className="preview">
                                <svg className="ph-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
                                {obj.size != null && <span className="size-tag">{obj.size > 1048576 ? `${(obj.size / 1048576).toFixed(1)} MB` : obj.size > 1024 ? `${(obj.size / 1024).toFixed(0)} KB` : `${obj.size} B`}</span>}
                              </div>
                              <div className="meta"><div className="nm">{obj.name}</div></div>
                            </div>
                          ))}
                        </div>
                      )}
                      {/* Upload overlay */}
                      {uploadOverlayOpen && (
                        <div className="upload-overlay show">
                          <button className="close-btn" onClick={() => { setUploadOverlayOpen(false); setUploadFiles([]); }} aria-label="Close upload">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 14, height: 14 }}><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
                          </button>
                          <div className="drop" onClick={() => fileInputRef.current?.click()} style={{ cursor: "pointer" }}>
                            <span className="cloud"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M16 16l-4-4-4 4"/><path d="M12 12v9"/><path d="M20.39 18.39A5 5 0 0 0 18 9h-1.26A8 8 0 1 0 3 16.3"/><polyline points="16 16 12 12 8 16"/></svg></span>
                            <h3>Drop files here or click to browse</h3>
                            <p>{activeBucketData.fileSizeLimit > 0 ? `Max file size: ${Math.round(activeBucketData.fileSizeLimit / 1048576)} MB` : "No file size limit"}{activeBucketData.allowedMimeTypes?.length > 0 ? ` \u00b7 Allowed types: ${activeBucketData.allowedMimeTypes.join(", ")}` : ""}</p>
                          </div>
                          {uploadFiles.length > 0 && (
                            <div className="progress-list">
                              {uploadFiles.map((uf, i) => (
                                <div key={i} className="prog-row">
                                  <span className="file-icon doc" style={{ width: 22, height: 22 }}><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 11, height: 11 }}><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg></span>
                                  <span className="nm">{uf.file.name}</span>
                                  <span className="bar"><i style={{ width: uf.status === "done" ? "100%" : uf.status === "error" ? "100%" : "50%", background: uf.status === "done" ? "var(--ok)" : uf.status === "error" ? "var(--err)" : undefined }}></i></span>
                                  <span className="pct" style={{ color: uf.status === "done" ? "var(--ok)" : uf.status === "error" ? "var(--err)" : undefined }}>{uf.status === "done" ? "Done" : uf.status === "error" ? "Error" : "Uploading..."}</span>
                                  <span></span>
                                  <span className="sz" style={{ gridColumn: "2/3", gridRow: 1, justifySelf: "end" }}>{uf.file.size > 1048576 ? `${(uf.file.size / 1048576).toFixed(1)} MB` : `${(uf.file.size / 1024).toFixed(0)} KB`}</span>
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      )}
                      {/* File detail panel */}
                      {fileDetailOpen && selectedObject && (
                        <div className="file-detail-panel" style={{ position: "absolute", right: 0, top: 0, bottom: 0, width: 320, background: "var(--card)", borderLeft: "1px solid var(--border)", padding: 20, zIndex: 10, display: "flex", flexDirection: "column", gap: 12 }}>
                          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                            <h3 style={{ margin: 0, fontSize: 14 }}>{selectedObject.name}</h3>
                            <button className="icon-only-btn" onClick={() => { setFileDetailOpen(false); setSelectedObject(null); }} aria-label="Close"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 14, height: 14 }}><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
                          </div>
                          <div style={{ fontSize: 12, color: "var(--fg-mute)", display: "flex", flexDirection: "column", gap: 8 }}>
                            <div><span style={{ fontWeight: 500 }}>Size:</span> {selectedObject.size != null ? (selectedObject.size > 1048576 ? `${(selectedObject.size / 1048576).toFixed(1)} MB` : `${(selectedObject.size / 1024).toFixed(0)} KB`) : "Unknown"}</div>
                            <div><span style={{ fontWeight: 500 }}>Type:</span> {selectedObject.mime_type || "Unknown"}</div>
                            <div><span style={{ fontWeight: 500 }}>Modified:</span> {selectedObject.updated_at ? formatRelative(selectedObject.updated_at) : "Unknown"}</div>
                            {selectedObject.etag && <div><span style={{ fontWeight: 500 }}>ETag:</span> <span className="mono">{selectedObject.etag}</span></div>}
                          </div>
                          <div style={{ display: "flex", gap: 8, flexWrap: "wrap", marginTop: 8 }}>
                            <button className="btn btn-ghost" onClick={() => navigator.clipboard.writeText(`${activeBucketData.name}/${selectedObject.name}`)}>Copy URL</button>
                            <button className="btn btn-ghost" onClick={() => handleFileDownload(selectedObject.name)}><IconDownload /> Download</button>
                            <button className="btn btn-ghost" onClick={() => fileInputRef.current?.click()}>Replace</button>
                            <button className="btn btn-ghost" style={{ color: "var(--err)" }} onClick={() => handleFileDelete(selectedObject.name)}>Delete</button>
                          </div>
                        </div>
                      )}
                    </>
                  ) : (
                    <div style={{ padding: 48, textAlign: "center", color: "var(--fg-mute)" }}>
                      {(buckets ?? []).length === 0 ? "No buckets yet. Create one to start uploading files." : "Select a bucket from the sidebar."}
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
              {secrets && secrets.length > 0 ? (
              <div className="panel">
                <table>
                  <thead>
                    <tr>
                      <th>Name</th>
                      <th>Description</th>
                      <th>Value</th>
                      <th>Last rotated</th>
                      <th>Created</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {secrets.map(s => {
                      const rotated = s.updatedAt && s.updatedAt !== s.createdAt;
                      return (
                        <tr key={s.id}>
                          <td><code>{s.name}</code></td>
                          <td><span className="sec-desc">{s.description || "\u2014"}</span></td>
                          <td><span className="sec-value">&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;&bull;</span></td>
                          <td>{rotated ? formatRelative(s.updatedAt) : "Never rotated"}</td>
                          <td>{formatRelative(s.createdAt)}</td>
                          <td>
                            <div className="sec-actions">
                              <button onClick={() => { setTargetSecretId(s.id); setTargetSecretName(s.name); setRotateSecretValue(""); setRotateValueRevealed(false); setRotateSecretModalOpen(true); }}>Rotate</button>
                              <button className="danger" onClick={() => { setTargetSecretId(s.id); setTargetSecretName(s.name); setDeleteSecretConfirmInput(""); setDeleteSecretModalOpen(true); }}>Delete</button>
                            </div>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
              ) : (
              <div className="fn-empty">
                <div className="bolt"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg></div>
                <h2>No secrets configured</h2><p>Add environment variables that your functions can access at runtime.</p>
                <button className="btn btn-primary" onClick={() => setAddSecretModalOpen(true)}><IconPlus /> Add Secret</button>
              </div>
              )}
            </>
          )}

          {/* ====== API Keys tab ====== */}
          {activeTab === "apikeys" && (
            <>
              <header className="tab-head">
                <div><h2>API Keys</h2><p>Authenticate requests to your project{"'"}s API.</p></div>
                {akCount > 0 && <div className="actions"><button className="btn btn-primary" onClick={() => { setCreateKeyStep(1); setCreateKeyName(""); setCreateKeyRole("anon"); setCreateKeyExp("90"); setCreateKeyModalOpen(true); }}><IconPlus /> Create Key<span className="kbd-inline">K</span></button></div>}
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
              {activeApiKeys && activeApiKeys.length > 0 ? (
              <div className="panel">
                <div className="ak-table" role="table">
                  <div className="row header" role="row">
                    <span>Name</span>
                    <span>Prefix</span>
                    <span>Role</span>
                    <span>Created</span>
                    <span style={{ textAlign: "right" }}>Actions</span>
                  </div>
                  {activeApiKeys.map(k => (
                    <div key={k.id} className={`row body-row`} role="row">
                      <span className="ak-name">{k.name}</span>
                      <span className="ak-prefix">{k.keyPrefix}<CopyBtn text={k.keyPrefix} /></span>
                      <span><span className={`role-badge ${k.role === "anon" ? "anon" : "svc"}`}>{k.role}</span></span>
                      <span className="ak-when">{formatRelative(k.createdAt)}</span>
                      <span className="ak-actions">
                        <button onClick={() => { setRevokeKeyId(k.id); setRevokeKeyName(k.name); setRevokeKeyPrefix(k.keyPrefix); setRevokeKeyRole(k.role); setRevokeConfirmInput(""); setRevokeKeyModalOpen(true); }}>Revoke</button>
                      </span>
                    </div>
                  ))}
                </div>
              </div>
              ) : (
              <div className="fn-empty">
                <div className="bolt"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg></div>
                <h2>No API keys</h2><p>Create API keys to authenticate requests to your project{"'"}s endpoints.</p>
                <button className="btn btn-primary" onClick={() => { setCreateKeyStep(1); setCreateKeyName(""); setCreateKeyRole("anon"); setCreateKeyExp("90"); setCreateKeyModalOpen(true); }}><IconPlus /> Create Key</button>
              </div>
              )}
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
                    <div className="stg-field"><label>Display name</label><input type="text" key={`dn-${project?.displayName}`} defaultValue={displayName} maxLength={100} /><span className="helper">Used in the dashboard and notifications.</span></div>
                    <div className="stg-field"><label>Description</label><div className="counter-wrap"><textarea key={`desc-${project?.description}`} defaultValue={description} maxLength={500} /><span className="char-counter">{description.length} / 500</span></div></div>
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
                    <div className="danger-row"><div className="info"><div className="ttl">{status === "paused" ? "Resume project" : "Pause project"}</div><div className="sub">{status === "paused" ? "Resume all services. Data has been preserved." : "Temporarily stop all services. Data is preserved but endpoints become unavailable."}</div></div><div className="right-actions"><button className="btn btn-warn" onClick={() => setPauseModalOpen(true)}>{status === "paused" ? "Resume Project" : "Pause Project"}</button></div></div>
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
              <h3>{status === "paused" ? "Resume project" : "Pause project"}</h3>
              <button className="close" onClick={() => setPauseModalOpen(false)} aria-label="Close"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width:14,height:14 }}><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>
            <div className="modal-body">
              <div className="modal-warn">
                <span className="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="6" y="4" width="4" height="16"/><rect x="14" y="4" width="4" height="16"/></svg></span>
                <p>{status === "paused" ? "All services will be restarted." : <>All services will be stopped. Your data (database, storage, secrets) will be <strong>preserved</strong>. You can resume at any time.</>}</p>
              </div>
              {status !== "paused" && (
              <div className="panel" style={{ padding:"14px 16px",borderRadius:10,fontSize:"12.5px" }}>
                <span style={{ fontSize:11,color:"var(--fg-mute)",textTransform:"uppercase",letterSpacing:"0.06em" }}>Currently running</span>
                <div style={{ marginTop:6,display:"flex",gap:6,flexWrap:"wrap" }}>
                  {project?.postgresEnabled && <span className="role-badge" style={{ color:"var(--ok)",background:"rgba(134,239,172,0.08)",borderColor:"rgba(134,239,172,0.22)" }}>PostgreSQL</span>}
                  {project?.redisEnabled && <span className="role-badge" style={{ color:"var(--ok)",background:"rgba(134,239,172,0.08)",borderColor:"rgba(134,239,172,0.22)" }}>Redis</span>}
                  {project?.postgrestEnabled && <span className="role-badge" style={{ color:"var(--ok)",background:"rgba(134,239,172,0.08)",borderColor:"rgba(134,239,172,0.22)" }}>PostgREST</span>}
                  {fnCount > 0 && <span className="role-badge" style={{ color:"var(--accent)",background:"var(--accent-soft)",borderColor:"rgba(196,181,253,0.22)" }}>{fnCount} Functions</span>}
                </div>
              </div>
              )}
            </div>
            <div className="modal-foot">
              <button className="btn btn-ghost" onClick={() => setPauseModalOpen(false)}>Cancel</button>
              <button className="btn btn-warn" style={{ borderColor:"rgba(252,211,77,0.5)" }} onClick={async () => { try { if (status === "paused") { await resumeProjectMut.mutateAsync(projectId); } else { await pauseProjectMut.mutateAsync(projectId); } } catch {} finally { setPauseModalOpen(false); } }}>{status === "paused" ? "Resume" : "Pause Project"}</button>
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
              <button className="btn btn-danger" disabled={deleteProjectInput !== displayName || !deleteProjectAck} onClick={async () => { try { await deleteProjectMut.mutateAsync(projectId); router.push("/projects"); } catch {} finally { setDeleteProjectModalOpen(false); } }}>Delete Project</button>
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
              <div className="field"><label>Name</label><input id="secret-name" type="text" className="mono" placeholder="e.g. OPENAI_API_KEY" autoComplete="off" spellCheck={false} value={addSecretName} onChange={e => setAddSecretName(e.target.value.toUpperCase().replace(/[^A-Z0-9_]/g, ""))} /><span className="hint">UPPERCASE letters, numbers, and underscores only.</span></div>
              <div className="field">
                <label>Value</label>
                <input id="secret-value" type="password" className="mono" placeholder="Paste your secret value" autoComplete="off" spellCheck={false} value={addSecretValue} onChange={e => setAddSecretValue(e.target.value)} />
                <span className="footnote"><IconLock />This value will be encrypted and cannot be retrieved after saving.</span>
              </div>
              <div className="field"><label>Description <span style={{ color:"var(--fg-mute)",textTransform:"none",letterSpacing:0,fontWeight:400 }}>&mdash; optional</span></label><input id="secret-desc" type="text" placeholder="What is this secret used for?" value={addSecretDesc} onChange={e => setAddSecretDesc(e.target.value)} /></div>
            </div>
            <div className="modal-foot">
              <button className="btn btn-ghost" onClick={() => setAddSecretModalOpen(false)}>Cancel</button>
              <button className="btn btn-primary" disabled={!addSecretName || !addSecretValue} onClick={async () => { await createSecretMut.mutateAsync({ projectId, name: addSecretName, value: addSecretValue, description: addSecretDesc || undefined }); setAddSecretName(""); setAddSecretValue(""); setAddSecretDesc(""); setAddSecretModalOpen(false); }}>Add Secret</button>
            </div>
          </div>
        </div>
      )}

      {/* 4. Rotate Secret Modal */}
      {rotateSecretModalOpen && (
        <div className="modal-scrim open" role="dialog" aria-modal="true" onClick={() => setRotateSecretModalOpen(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <div className="modal-head">
              <h3>Rotate <code>{targetSecretName}</code></h3>
              <button className="close" onClick={() => setRotateSecretModalOpen(false)} aria-label="Close"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width:14,height:14 }}><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>
            <div className="modal-body">
              <div className="modal-warn">
                <span className="ic"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg></span>
                <p>This will <strong>immediately replace</strong> the current value. Functions using this secret will pick up the new value on next restart.</p>
              </div>
              <div className="field">
                <label>New value</label>
                <input type="password" className="mono" placeholder="Paste the new secret value" autoComplete="off" spellCheck={false} value={rotateSecretValue} onChange={e => setRotateSecretValue(e.target.value)} />
              </div>
            </div>
            <div className="modal-foot">
              <button className="btn btn-ghost" onClick={() => setRotateSecretModalOpen(false)}>Cancel</button>
              <button className="btn btn-primary" disabled={!rotateSecretValue} onClick={async () => { await updateSecretValueMut.mutateAsync({ projectId, secretId: targetSecretId, value: rotateSecretValue }); setRotateSecretValue(""); setRotateSecretModalOpen(false); }}>Rotate Value</button>
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
              <button className="btn btn-danger" disabled={deleteSecretConfirmInput !== targetSecretName} onClick={async () => { await deleteSecretMut.mutateAsync({ projectId, secretId: targetSecretId }); setDeleteSecretModalOpen(false); }}>Delete Secret</button>
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
                  <button className="btn btn-primary" disabled={!createKeyName || createApiKeyMut.isPending} onClick={async () => { const expDays = createKeyExp === "never" ? undefined : parseInt(createKeyExp); const result = await createApiKeyMut.mutateAsync({ projectId, name: createKeyName, role: createKeyRole, expiresInDays: expDays }); setCreatedRawKey(result.rawKey || ""); setCreateKeyStep(2); }}>Create Key</button>
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
                    <div className="key-block"><span className="tag">Full key</span>{createdRawKey}</div>
                    <button className="btn btn-primary copy-full" onClick={() => navigator.clipboard.writeText(createdRawKey)}>
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
              <button className="btn btn-danger" disabled={revokeConfirmInput !== revokeKeyName} onClick={async () => { try { await revokeApiKeyMut.mutateAsync({ projectId, keyId: revokeKeyId }); } catch {} finally { setRevokeKeyModalOpen(false); } }}>Revoke Key</button>
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

      {/* ===== DB Modals ===== */}
      {newTableModalOpen && (
        <div className="modal-overlay" onClick={() => setNewTableModalOpen(false)}>
          <div className="modal-box" onClick={e => e.stopPropagation()}>
            <h3>New Table</h3>
            <div className="modal-field">
              <label>Table name</label>
              <input type="text" value={newTableName} onChange={e => setNewTableName(e.target.value)} placeholder="e.g. users" autoFocus />
            </div>
            <div className="modal-actions">
              <button className="btn btn-ghost" onClick={() => setNewTableModalOpen(false)}>Cancel</button>
              <button className="btn btn-primary" disabled={!newTableName.trim() || createTableMut.isPending} onClick={() => {
                createTableMut.mutate({ name: newTableName.trim(), schema: "public" }, {
                  onSuccess: (table) => { setNewTableModalOpen(false); setSelectedTableId(table.id); },
                });
              }}>{createTableMut.isPending ? "Creating..." : "Create Table"}</button>
            </div>
            {createTableMut.isError && <div style={{ color: "var(--err)", fontSize: 12, marginTop: 8 }}>{createTableMut.error?.message}</div>}
          </div>
        </div>
      )}

      {addColumnModalOpen && (
        <div className="modal-overlay" onClick={() => setAddColumnModalOpen(false)}>
          <div className="modal-box" onClick={e => e.stopPropagation()}>
            <h3>Add Column to {selectedTableName}</h3>
            <div className="modal-field">
              <label>Column name</label>
              <input type="text" value={newColName} onChange={e => setNewColName(e.target.value)} placeholder="e.g. email" autoFocus />
            </div>
            <div className="modal-field">
              <label>Type</label>
              <select value={newColType} onChange={e => setNewColType(e.target.value)}>
                {["text", "integer", "bigint", "boolean", "uuid", "timestamptz", "jsonb", "numeric", "float8", "date", "bytea"].map(t => <option key={t} value={t}>{t}</option>)}
              </select>
            </div>
            <div className="modal-field" style={{ flexDirection: "row", alignItems: "center", gap: 8 }}>
              <input type="checkbox" checked={newColNullable} onChange={e => setNewColNullable(e.target.checked)} id="col-nullable" />
              <label htmlFor="col-nullable" style={{ marginBottom: 0 }}>Nullable</label>
            </div>
            <div className="modal-actions">
              <button className="btn btn-ghost" onClick={() => setAddColumnModalOpen(false)}>Cancel</button>
              <button className="btn btn-primary" disabled={!newColName.trim() || !selectedTableId || createColumnMut.isPending} onClick={() => {
                createColumnMut.mutate({ table_id: selectedTableId!, name: newColName.trim(), type: newColType, is_nullable: newColNullable }, {
                  onSuccess: () => setAddColumnModalOpen(false),
                });
              }}>{createColumnMut.isPending ? "Adding..." : "Add Column"}</button>
            </div>
            {createColumnMut.isError && <div style={{ color: "var(--err)", fontSize: 12, marginTop: 8 }}>{createColumnMut.error?.message}</div>}
          </div>
        </div>
      )}

      {insertRowModalOpen && (
        <div className="modal-overlay" onClick={() => setInsertRowModalOpen(false)}>
          <div className="modal-box" onClick={e => e.stopPropagation()} style={{ maxWidth: 520 }}>
            <h3>Insert Row into {selectedTableName}</h3>
            {(dbColumns ?? []).filter(c => !c.is_identity && !c.is_generated).map(col => (
              <div key={col.id} className="modal-field">
                <label>{col.name} <span style={{ color: "var(--fg-mute)", fontSize: 11 }}>({col.format}{col.is_nullable ? ", nullable" : ""})</span></label>
                <input type="text" value={insertRowData[col.name] ?? ""} onChange={e => setInsertRowData(prev => ({ ...prev, [col.name]: e.target.value }))} placeholder={col.default_value ? `Default: ${col.default_value}` : col.is_nullable ? "NULL" : ""} />
              </div>
            ))}
            <div className="modal-actions">
              <button className="btn btn-ghost" onClick={() => setInsertRowModalOpen(false)}>Cancel</button>
              <button className="btn btn-primary" disabled={insertRowMut.isPending} onClick={() => {
                const data: Record<string, unknown> = {};
                for (const [k, v] of Object.entries(insertRowData)) {
                  if (v !== "") data[k] = v;
                }
                insertRowMut.mutate(data, { onSuccess: () => setInsertRowModalOpen(false) });
              }}>{insertRowMut.isPending ? "Inserting..." : "Insert Row"}</button>
            </div>
            {insertRowMut.isError && <div style={{ color: "var(--err)", fontSize: 12, marginTop: 8 }}>{insertRowMut.error?.message}</div>}
          </div>
        </div>
      )}

      {newPolicyModalOpen && (
        <div className="modal-overlay" onClick={() => setNewPolicyModalOpen(false)}>
          <div className="modal-box" onClick={e => e.stopPropagation()}>
            <h3>New RLS Policy for {selectedTableName}</h3>
            <div className="modal-field">
              <label>Policy name</label>
              <input type="text" value={newPolicyName} onChange={e => setNewPolicyName(e.target.value)} placeholder="e.g. enable_read_for_all" autoFocus />
            </div>
            <div className="modal-field">
              <label>USING expression</label>
              <input type="text" value={newPolicyDefinition} onChange={e => setNewPolicyDefinition(e.target.value)} placeholder="true" />
            </div>
            <div className="modal-actions">
              <button className="btn btn-ghost" onClick={() => setNewPolicyModalOpen(false)}>Cancel</button>
              <button className="btn btn-primary" disabled={!newPolicyName.trim() || !selectedTableId || createPolicyMut.isPending} onClick={() => {
                createPolicyMut.mutate({ name: newPolicyName.trim(), table: selectedTableName, schema: selectedTableSchema, definition: newPolicyDefinition, command: "ALL" }, {
                  onSuccess: () => setNewPolicyModalOpen(false),
                });
              }}>{createPolicyMut.isPending ? "Creating..." : "Create Policy"}</button>
            </div>
            {createPolicyMut.isError && <div style={{ color: "var(--err)", fontSize: 12, marginTop: 8 }}>{createPolicyMut.error?.message}</div>}
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

      {/* Delete Function Confirm Dialog */}
      {deleteFnModalOpen && deleteFnTarget && (
        <div className="modal-scrim open" role="dialog" aria-modal="true" onClick={() => setDeleteFnModalOpen(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <div className="modal-head">
              <h3 style={{ color: "var(--err)" }}>Delete function</h3>
              <button className="close" onClick={() => setDeleteFnModalOpen(false)} aria-label="Close"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" style={{ width: 14, height: 14 }}><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg></button>
            </div>
            <div className="modal-body">
              <p>Are you sure you want to delete <strong className="mono">{deleteFnTarget.name}</strong>? This action cannot be undone.</p>
            </div>
            <div className="modal-foot">
              <button className="btn btn-ghost" onClick={() => setDeleteFnModalOpen(false)}>Cancel</button>
              <button className="btn btn-danger" onClick={async () => { try { await deleteFunctionMut.mutateAsync({ projectId, functionId: deleteFnTarget.id }); } catch {} finally { setDeleteFnModalOpen(false); setDeleteFnTarget(null); } }}>Delete</button>
            </div>
          </div>
        </div>
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

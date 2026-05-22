"use client";

import { useRouter, usePathname } from "next/navigation";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const router = useRouter();
  const pathname = usePathname();
  const isProjects = pathname.startsWith("/projects");
  const isSettings = pathname.startsWith("/settings");

  return (
    <>
      <div className="scene" aria-hidden="true">
        <div className="grid"></div>
        <div className="glow"></div>
        <div className="noise"></div>
      </div>

      <div className="app">
        <aside className="sidebar" aria-label="Primary navigation">
          <div className="logo" aria-hidden="true">
            <svg viewBox="0 0 32 32" fill="none">
              <path d="M16 3 L28 9 L16 15 L4 9 Z" fill="var(--accent)" opacity="0.95" />
              <path d="M4 16 L16 22 L28 16" stroke="var(--accent)" strokeWidth="2" strokeLinejoin="round" opacity="0.7" />
              <path d="M4 23 L16 29 L28 23" stroke="var(--accent)" strokeWidth="2" strokeLinejoin="round" opacity="0.4" />
            </svg>
          </div>
          <nav className="nav">
            <button className={`nav-btn${isProjects ? " active" : ""}`} aria-current={isProjects ? "page" : undefined} aria-label="Projects" onClick={() => router.push("/projects")}>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="3" width="7" height="7" rx="1.5" /><rect x="14" y="3" width="7" height="7" rx="1.5" /><rect x="3" y="14" width="7" height="7" rx="1.5" /><rect x="14" y="14" width="7" height="7" rx="1.5" /></svg>
              <span className="tip">Projects</span>
            </button>
            <button className={`nav-btn${isSettings ? " active" : ""}`} aria-current={isSettings ? "page" : undefined} aria-label="Settings" onClick={() => router.push("/settings")}>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.7 1.7 0 0 0 .34 1.86l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.7 1.7 0 0 0-1.86-.34 1.7 1.7 0 0 0-1.04 1.56V21a2 2 0 1 1-4 0v-.09A1.7 1.7 0 0 0 9 19.4a1.7 1.7 0 0 0-1.86.34l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.7 1.7 0 0 0 4.6 15a1.7 1.7 0 0 0-1.56-1.04H3a2 2 0 1 1 0-4h.09A1.7 1.7 0 0 0 4.6 9a1.7 1.7 0 0 0-.34-1.86l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.7 1.7 0 0 0 9 4.6a1.7 1.7 0 0 0 1.04-1.56V3a2 2 0 1 1 4 0v.09A1.7 1.7 0 0 0 15 4.6a1.7 1.7 0 0 0 1.86-.34l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.7 1.7 0 0 0 19.4 9c.18.43.6.72 1.06.72H21a2 2 0 1 1 0 4h-.09A1.7 1.7 0 0 0 19.4 15z" /></svg>
              <span className="tip">Settings</span>
            </button>
            <button className="nav-btn" aria-label="Documentation">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" /><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z" /></svg>
              <span className="ext"></span>
              <span className="tip">Docs</span>
            </button>
          </nav>
          <button className="avatar" aria-label="Account menu">EB</button>
        </aside>

        <div className="main">
          {children}
        </div>
      </div>
    </>
  );
}

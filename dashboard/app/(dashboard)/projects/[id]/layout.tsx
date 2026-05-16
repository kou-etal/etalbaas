"use client";

import Link from "next/link";
import { usePathname, useParams } from "next/navigation";
import { cn } from "@/lib/utils/cn";
import {
  LayoutDashboard,
  Zap,
  Radio,
  HardDrive,
  KeyRound,
  Key,
  Settings,
} from "lucide-react";

const tabs = [
  { href: "", icon: LayoutDashboard, label: "Overview" },
  { href: "/functions", icon: Zap, label: "Functions" },
  { href: "/events", icon: Radio, label: "Events" },
  { href: "/storage", icon: HardDrive, label: "Storage" },
  { href: "/secrets", icon: KeyRound, label: "Secrets" },
  { href: "/api-keys", icon: Key, label: "API Keys" },
  { href: "/settings", icon: Settings, label: "Settings" },
];

export default function ProjectDetailLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const params = useParams();
  const projectId = params.id as string;
  const basePath = `/projects/${projectId}`;

  return (
    <div>
      {/* Tab navigation */}
      <div className="border-b border-border px-6">
        <nav className="flex gap-1 -mb-px overflow-x-auto">
          {tabs.map((tab) => {
            const href = `${basePath}${tab.href}`;
            const isActive =
              tab.href === ""
                ? pathname === basePath
                : pathname.startsWith(href);

            return (
              <Link
                key={tab.href}
                href={href}
                className={cn(
                  "flex items-center gap-2 border-b-2 px-4 py-3 text-sm font-medium transition-colors whitespace-nowrap",
                  isActive
                    ? "border-primary text-primary"
                    : "border-transparent text-muted-foreground hover:text-foreground hover:border-border"
                )}
              >
                <tab.icon className="h-4 w-4" />
                {tab.label}
              </Link>
            );
          })}
        </nav>
      </div>

      {/* Tab content */}
      {children}
    </div>
  );
}

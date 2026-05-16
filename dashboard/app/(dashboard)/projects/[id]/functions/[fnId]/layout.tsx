"use client";

import Link from "next/link";
import { usePathname, useParams } from "next/navigation";
import { cn } from "@/lib/utils/cn";
import { FunctionDetailHeader } from "@/features/functions/function-detail-header";
import { useFunction } from "@/features/functions/hooks";
import { DetailSkeleton } from "@/components/shared/loading-skeleton";

const subTabs = [
  { href: "", label: "Overview" },
  { href: "/triggers", label: "Triggers" },
  { href: "/invocations", label: "Invocations" },
  { href: "/logs", label: "Logs" },
  { href: "/settings", label: "Settings" },
];

export default function FunctionDetailLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const params = useParams();
  const projectId = params.id as string;
  const fnId = params.fnId as string;
  const { data: fn, isLoading } = useFunction(fnId);
  const basePath = `/projects/${projectId}/functions/${fnId}`;

  if (isLoading) return <DetailSkeleton />;

  return (
    <div>
      <FunctionDetailHeader
        name={fn?.name || "Function"}
        runtime={fn?.runtime || "node20"}
        kind={fn?.kind || "http"}
        mode={fn?.mode || "on-demand"}
        status={fn?.status || "running"}
      />

      {/* Sub-tabs */}
      <div className="border-b border-border px-6">
        <nav className="flex gap-1 -mb-px">
          {subTabs.map((tab) => {
            const href = `${basePath}${tab.href}`;
            const isActive =
              tab.href === "" ? pathname === basePath : pathname === href;

            return (
              <Link
                key={tab.href}
                href={href}
                className={cn(
                  "border-b-2 px-4 py-2.5 text-sm font-medium transition-colors",
                  isActive
                    ? "border-primary text-primary"
                    : "border-transparent text-muted-foreground hover:text-foreground hover:border-border"
                )}
              >
                {tab.label}
              </Link>
            );
          })}
        </nav>
      </div>

      {children}
    </div>
  );
}

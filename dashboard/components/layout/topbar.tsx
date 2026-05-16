"use client";

import { usePathname } from "next/navigation";
import Link from "next/link";
import { ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils/cn";

interface BreadcrumbItem {
  label: string;
  href?: string;
}

function useBreadcrumbs(): BreadcrumbItem[] {
  const pathname = usePathname();
  const segments = pathname.split("/").filter(Boolean);

  const breadcrumbs: BreadcrumbItem[] = [];

  if (segments[0] === "projects") {
    breadcrumbs.push({ label: "Projects", href: "/projects" });

    if (segments[1]) {
      breadcrumbs.push({
        label: segments[1].slice(0, 8) + "...",
        href: `/projects/${segments[1]}`,
      });

      if (segments[2]) {
        const tabLabel = segments[2].charAt(0).toUpperCase() + segments[2].slice(1);
        breadcrumbs.push({
          label: tabLabel,
          href: `/projects/${segments[1]}/${segments[2]}`,
        });

        if (segments[3]) {
          breadcrumbs.push({
            label: segments[3].slice(0, 8) + "...",
          });
        }
      }
    }
  } else if (segments[0] === "settings") {
    breadcrumbs.push({ label: "Settings" });
  }

  return breadcrumbs;
}

export function Topbar() {
  const breadcrumbs = useBreadcrumbs();

  return (
    <header className="sticky top-0 z-30 flex h-14 items-center border-b border-border bg-background/80 backdrop-blur-sm px-6">
      <nav className="flex items-center gap-1 text-sm">
        {breadcrumbs.map((item, index) => (
          <div key={index} className="flex items-center gap-1">
            {index > 0 && (
              <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
            )}
            {item.href && index < breadcrumbs.length - 1 ? (
              <Link
                href={item.href}
                className="text-muted-foreground hover:text-foreground transition-colors"
              >
                {item.label}
              </Link>
            ) : (
              <span
                className={cn(
                  index === breadcrumbs.length - 1
                    ? "text-foreground font-medium"
                    : "text-muted-foreground"
                )}
              >
                {item.label}
              </span>
            )}
          </div>
        ))}
      </nav>
    </header>
  );
}

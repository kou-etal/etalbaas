"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  FolderKanban,
  Settings,
  LogOut,
} from "lucide-react";
import { cn } from "@/lib/utils/cn";
import { useAuth } from "@/features/auth/use-auth";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

const navItems = [
  { href: "/projects", icon: FolderKanban, label: "Projects" },
  { href: "/settings", icon: Settings, label: "Settings" },
];

export function Sidebar() {
  const pathname = usePathname();
  const { user, signOut } = useAuth();

  return (
    <TooltipProvider delayDuration={0}>
      <aside className="fixed left-0 top-0 z-40 flex h-screen w-16 flex-col items-center border-r border-sidebar-border bg-sidebar py-4">
        {/* Logo */}
        <Link href="/projects" className="mb-8" aria-label="Home">
          <svg
            viewBox="0 0 240 200"
            className="h-8 w-8"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              d="M120 10 L220 60 L120 110 L20 60 Z"
              fill="hsl(var(--primary))"
              opacity="0.95"
            />
            <path
              d="M20 90 L120 140 L220 90"
              stroke="hsl(var(--primary))"
              strokeWidth="14"
              strokeLinejoin="round"
              opacity="0.7"
            />
            <path
              d="M20 140 L120 190 L220 140"
              stroke="hsl(var(--primary))"
              strokeWidth="14"
              strokeLinejoin="round"
              opacity="0.4"
            />
          </svg>
        </Link>

        {/* Navigation */}
        <nav className="flex flex-1 flex-col items-center gap-2">
          {navItems.map((item) => {
            const isActive =
              pathname === item.href || pathname.startsWith(item.href + "/");
            return (
              <Tooltip key={item.href}>
                <TooltipTrigger asChild>
                  <Link
                    href={item.href}
                    aria-label={item.label}
                    aria-current={isActive ? "page" : undefined}
                    className={cn(
                      "flex h-10 w-10 items-center justify-center rounded-lg transition-colors",
                      isActive
                        ? "bg-primary/15 text-primary"
                        : "text-sidebar-foreground hover:bg-accent hover:text-foreground"
                    )}
                  >
                    <item.icon className="h-5 w-5" />
                  </Link>
                </TooltipTrigger>
                <TooltipContent side="right">{item.label}</TooltipContent>
              </Tooltip>
            );
          })}
        </nav>

        {/* User avatar */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              type="button"
              aria-label="Open account menu"
              className="flex h-9 w-9 items-center justify-center rounded-full bg-primary/20 text-xs font-medium text-primary hover:bg-primary/30 transition-colors"
            >
              {user?.email?.charAt(0).toUpperCase() || "U"}
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent side="right" align="end" className="w-48">
            <div className="px-2 py-1.5">
              <p className="text-sm font-medium truncate">
                {user?.email || "User"}
              </p>
            </div>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={signOut}>
              <LogOut className="mr-2 h-4 w-4" />
              Sign out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </aside>
    </TooltipProvider>
  );
}

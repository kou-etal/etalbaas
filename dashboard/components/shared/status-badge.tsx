"use client";

import { Badge } from "@/components/ui/badge";

type Status = "running" | "stopped" | "building" | "error" | "pending" | "active" | "inactive";

const statusConfig: Record<Status, { label: string; variant: "success" | "destructive" | "warning" | "info" | "secondary" }> = {
  running: { label: "Running", variant: "success" },
  active: { label: "Active", variant: "success" },
  stopped: { label: "Stopped", variant: "secondary" },
  inactive: { label: "Inactive", variant: "secondary" },
  building: { label: "Building", variant: "info" },
  pending: { label: "Pending", variant: "warning" },
  error: { label: "Error", variant: "destructive" },
};

interface StatusBadgeProps {
  status: string;
  className?: string;
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const config = statusConfig[status as Status] || {
    label: status,
    variant: "secondary" as const,
  };

  return (
    <Badge variant={config.variant} className={className}>
      <span className="mr-1.5 h-1.5 w-1.5 rounded-full bg-current" />
      {config.label}
    </Badge>
  );
}

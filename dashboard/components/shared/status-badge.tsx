"use client";

import { Badge } from "@/components/ui/badge";

type Status =
  // Project
  | "pending" | "provisioning" | "ready" | "paused" | "failed" | "deleted"
  // Function
  | "building"
  // Event
  | "received" | "retrying" | "delivered"
  // Legacy (for backward compat)
  | "running" | "active" | "stopped" | "inactive" | "error";

const statusConfig: Record<Status, { label: string; variant: "success" | "destructive" | "warning" | "info" | "secondary" }> = {
  // Project statuses
  ready: { label: "Ready", variant: "success" },
  provisioning: { label: "Provisioning", variant: "info" },
  paused: { label: "Paused", variant: "warning" },
  deleted: { label: "Deleted", variant: "secondary" },
  // Function statuses
  building: { label: "Building", variant: "info" },
  // Event statuses
  received: { label: "Received", variant: "info" },
  retrying: { label: "Retrying", variant: "warning" },
  delivered: { label: "Delivered", variant: "success" },
  // Shared
  pending: { label: "Pending", variant: "warning" },
  failed: { label: "Failed", variant: "destructive" },
  // Legacy
  running: { label: "Running", variant: "success" },
  active: { label: "Active", variant: "success" },
  stopped: { label: "Stopped", variant: "secondary" },
  inactive: { label: "Inactive", variant: "secondary" },
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

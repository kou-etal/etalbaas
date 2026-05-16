"use client";

import { Activity } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { EmptyState } from "@/components/shared/empty-state";

export default function InvocationsPage() {
  // Phase 1: Simple display with polling
  const invocations: Array<{
    id: string;
    timestamp: string;
    duration: number;
    status: number;
  }> = [];

  if (invocations.length === 0) {
    return (
      <div className="p-6">
        <EmptyState
          icon={Activity}
          title="No invocations"
          description="Invocation history will appear here once your function starts receiving requests."
        />
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="rounded-lg border">
        <table className="w-full">
          <thead>
            <tr className="border-b bg-muted/50">
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Time</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Duration</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Status</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Request ID</th>
            </tr>
          </thead>
          <tbody>
            {invocations.map((inv) => (
              <tr key={inv.id} className="border-b last:border-0 hover:bg-muted/30">
                <td className="px-4 py-3 text-sm">{inv.timestamp}</td>
                <td className="px-4 py-3 text-sm font-mono">{inv.duration}ms</td>
                <td className="px-4 py-3">
                  <Badge variant={inv.status < 400 ? "success" : "destructive"}>
                    {inv.status}
                  </Badge>
                </td>
                <td className="px-4 py-3 text-xs font-mono text-muted-foreground">{inv.id}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

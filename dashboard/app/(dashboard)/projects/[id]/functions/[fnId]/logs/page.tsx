"use client";

import { ScrollText } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { EmptyState } from "@/components/shared/empty-state";

export default function LogsPage() {
  // Phase 1: Simple log viewer placeholder
  return (
    <div className="p-6 space-y-4">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Function Logs</CardTitle>
        </CardHeader>
        <CardContent>
          <EmptyState
            icon={ScrollText}
            title="Logs coming soon"
            description="Real-time log streaming will be available in a future update. For now, check your cluster's logging solution (Loki/Grafana)."
          />
        </CardContent>
      </Card>
    </div>
  );
}

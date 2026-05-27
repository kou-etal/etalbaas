"use client";

import { useParams } from "next/navigation";
import { Radio, ChevronDown, AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { StatusBadge } from "@/components/shared/status-badge";
import { EmptyState } from "@/components/shared/empty-state";
import { TableSkeleton } from "@/components/shared/loading-skeleton";
import { useEvents } from "@/features/events/hooks";
import { useFunctions } from "@/features/functions/hooks";
import { formatDate } from "@/lib/utils/format";
import React, { useState, useMemo } from "react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

function getTriggerLabel(trigger?: { databaseChange?: { table: string; event: string }; objectStorage?: { bucket: string; event: string } }): string {
  if (!trigger) return "-";
  if (trigger.databaseChange) return `db:${trigger.databaseChange.table}`;
  if (trigger.objectStorage) return `storage:${trigger.objectStorage.bucket}`;
  return "-";
}

function getTriggerSource(trigger?: { databaseChange?: unknown; objectStorage?: unknown }): string {
  if (!trigger) return "unknown";
  if (trigger.databaseChange) return "database";
  if (trigger.objectStorage) return "storage";
  return "unknown";
}

export default function EventsPage() {
  const params = useParams();
  const projectId = params.id as string;
  const { data: events, isLoading, isError } = useEvents(projectId);
  const { data: functions } = useFunctions(projectId);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [functionFilter, setFunctionFilter] = useState("all");
  const [sourceFilter, setSourceFilter] = useState("all");

  // Status summary
  const summary = useMemo(() => {
    if (!events) return { all: 0, delivered: 0, retrying: 0, failed: 0 };
    return {
      all: events.length,
      delivered: events.filter((e) => e.status === "delivered").length,
      retrying: events.filter((e) => e.status === "retrying").length,
      failed: events.filter((e) => e.status === "failed").length,
    };
  }, [events]);

  // Filtered events
  const filtered = useMemo(() => {
    if (!events) return [];
    return events.filter((e) => {
      if (functionFilter !== "all" && e.functionId !== functionFilter) return false;
      if (sourceFilter !== "all" && getTriggerSource(e.trigger) !== sourceFilter) return false;
      return true;
    });
  }, [events, functionFilter, sourceFilter]);

  if (isLoading) return <div className="p-6"><TableSkeleton /></div>;

  if (isError) {
    return (
      <div className="p-6">
        <EmptyState
          icon={AlertCircle}
          title="Failed to load events"
          description="An error occurred while fetching event history. Please try again."
        />
      </div>
    );
  }

  if (!events || events.length === 0) {
    return (
      <div className="p-6">
        <div className="mb-4">
          <h3 className="text-lg font-medium">Event history</h3>
          <p className="text-sm text-muted-foreground">
            Track trigger events and function invocations.
          </p>
        </div>
        <EmptyState
          icon={Radio}
          title="No events recorded"
          description="Events will appear here when triggers fire and invoke your functions."
        />
      </div>
    );
  }

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-medium">Event history</h3>
          <p className="text-sm text-muted-foreground">
            Track trigger events and function invocations.
          </p>
        </div>
      </div>

      {/* Status summary */}
      <div className="flex gap-3">
        <Badge variant="secondary">All {summary.all}</Badge>
        <Badge variant="success">{summary.delivered}</Badge>
        <Badge variant="warning">{summary.retrying}</Badge>
        <Badge variant="destructive">{summary.failed}</Badge>
      </div>

      {/* Filters */}
      <div className="flex gap-3">
        <Select value={functionFilter} onValueChange={setFunctionFilter}>
          <SelectTrigger className="w-[200px]">
            <SelectValue placeholder="All functions" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All functions</SelectItem>
            {functions?.map((fn) => (
              <SelectItem key={fn.id} value={fn.id}>
                {fn.displayName || fn.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={sourceFilter} onValueChange={setSourceFilter}>
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="All sources" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All sources</SelectItem>
            <SelectItem value="database">Database</SelectItem>
            <SelectItem value="storage">Object Storage</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="rounded-lg border">
        <table className="w-full">
          <thead>
            <tr className="border-b bg-muted/50">
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground w-8"></th>
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Function</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Trigger</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Status</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Attempts</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Time</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((event) => {
              const fnName = functions?.find((f) => f.id === event.functionId);
              return (
                <React.Fragment key={event.id}>
                  <tr
                    className="border-b last:border-0 hover:bg-muted/30 transition-colors cursor-pointer"
                    onClick={() => setExpandedId(expandedId === event.id ? null : event.id)}
                  >
                    <td className="px-4 py-3">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6"
                        aria-expanded={expandedId === event.id}
                        aria-label="Toggle details"
                      >
                        <ChevronDown
                          className={`h-3 w-3 transition-transform ${expandedId === event.id ? "rotate-180" : ""}`}
                        />
                      </Button>
                    </td>
                    <td className="px-4 py-3">
                      <code className="text-xs font-mono">
                        {fnName ? (fnName.displayName || fnName.name) : (event.functionId || "-")}
                      </code>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-sm text-muted-foreground">{getTriggerLabel(event.trigger)}</span>
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={event.status} />
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-sm text-muted-foreground">
                        {event.attemptCount}/{event.attemptCount}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-xs text-muted-foreground">
                        {event.createdAt ? formatDate(event.createdAt) : "-"}
                      </span>
                    </td>
                  </tr>
                  {expandedId === event.id && (
                    <tr className="border-b">
                      <td colSpan={6} className="px-8 py-4 bg-muted/20">
                        <div className="space-y-2">
                          {event.lastError && (
                            <div>
                              <p className="text-xs font-medium text-destructive">Error</p>
                              <p className="text-xs text-muted-foreground font-mono whitespace-pre-wrap">{event.lastError}</p>
                            </div>
                          )}
                          {event.traceId && (
                            <div>
                              <p className="text-xs font-medium">Trace ID</p>
                              <code className="text-xs font-mono text-muted-foreground">{event.traceId}</code>
                            </div>
                          )}
                          {event.invocationId && (
                            <div>
                              <p className="text-xs font-medium">Invocation ID</p>
                              <code className="text-xs font-mono text-muted-foreground">{event.invocationId}</code>
                            </div>
                          )}
                        </div>
                      </td>
                    </tr>
                  )}
                </React.Fragment>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}

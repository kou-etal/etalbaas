"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { Plus, Search, Zap } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FunctionTable } from "@/features/functions/function-table";
import { CreateFunctionDialog } from "@/features/functions/create-function-dialog";
import { EmptyState } from "@/components/shared/empty-state";
import { TableSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { useFunctions, useDeleteFunction } from "@/features/functions/hooks";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

export default function FunctionsPage() {
  const params = useParams();
  const projectId = params.id as string;
  const [search, setSearch] = useState("");
  const [kindFilter, setKindFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState("all");
  const [createOpen, setCreateOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const { data: functions, isLoading } = useFunctions(projectId);
  const deleteFunction = useDeleteFunction();

  const filtered = functions?.filter((fn) => {
    const matchSearch = fn.name.toLowerCase().includes(search.toLowerCase()) ||
      (fn.displayName || "").toLowerCase().includes(search.toLowerCase());
    const matchKind = kindFilter === "all" || fn.kind === kindFilter;
    const matchStatus = statusFilter === "all" || fn.status === statusFilter;
    return matchSearch && matchKind && matchStatus;
  });

  if (isLoading) return <div className="p-6"><TableSkeleton /></div>;

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          Deploy and manage serverless functions for this project.
        </p>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Deploy Function
        </Button>
      </div>

      {/* Filters */}
      {functions && functions.length > 0 && (
        <div className="flex items-center gap-3">
          <div className="relative flex-1 max-w-sm">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search functions..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9"
            />
          </div>
          <Select value={kindFilter} onValueChange={setKindFilter}>
            <SelectTrigger className="w-[160px]">
              <SelectValue placeholder="All kinds" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All kinds</SelectItem>
              <SelectItem value="heavy-job">Heavy Job</SelectItem>
              <SelectItem value="heavy-deployment">Heavy Deploy</SelectItem>
              <SelectItem value="light-deployment">Light Deploy</SelectItem>
            </SelectContent>
          </Select>
          <Select value={statusFilter} onValueChange={setStatusFilter}>
            <SelectTrigger className="w-[160px]">
              <SelectValue placeholder="All statuses" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All statuses</SelectItem>
              <SelectItem value="ready">Ready</SelectItem>
              <SelectItem value="building">Building</SelectItem>
              <SelectItem value="pending">Pending</SelectItem>
              <SelectItem value="failed">Failed</SelectItem>
            </SelectContent>
          </Select>
        </div>
      )}

      {!filtered || filtered.length === 0 ? (
        <EmptyState
          icon={Zap}
          title="No functions deployed"
          description="Deploy your first function to handle events, process data, or serve APIs."
          action={{ label: "Deploy Function", onClick: () => setCreateOpen(true) }}
        />
      ) : (
        <FunctionTable
          projectId={projectId}
          functions={filtered.map((fn) => ({
            id: fn.id,
            name: fn.name,
            displayName: fn.displayName,
            runtime: fn.presetRuntime?.preset || fn.customRuntime?.dockerfile || "-",
            kind: fn.kind,
            mode: fn.mode,
            status: fn.status,
            lastBuiltAt: fn.lastBuiltAt || "",
          }))}
          onDelete={setDeleteTarget}
        />
      )}

      <CreateFunctionDialog
        projectId={projectId}
        open={createOpen}
        onOpenChange={setCreateOpen}
      />

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title="Delete Function"
        description="This will permanently delete the function and all its triggers. This action cannot be undone."
        confirmLabel="Delete"
        variant="destructive"
        isLoading={deleteFunction.isPending}
        onConfirm={async () => {
          if (deleteTarget) {
            await deleteFunction.mutateAsync({
              functionId: deleteTarget,
              projectId,
            });
            setDeleteTarget(null);
          }
        }}
      />
    </div>
  );
}

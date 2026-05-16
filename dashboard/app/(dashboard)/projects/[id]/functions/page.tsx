"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { Plus, Search, Zap } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FunctionTable } from "@/features/functions/function-table";
import { EmptyState } from "@/components/shared/empty-state";
import { TableSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { useFunctions, useDeleteFunction } from "@/features/functions/hooks";

export default function FunctionsPage() {
  const params = useParams();
  const projectId = params.id as string;
  const [search, setSearch] = useState("");
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const { data: functions, isLoading } = useFunctions(projectId);
  const deleteFunction = useDeleteFunction();

  const filtered = functions?.filter((fn) =>
    fn.name.toLowerCase().includes(search.toLowerCase())
  );

  if (isLoading) return <div className="p-6"><TableSkeleton /></div>;

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search functions..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
        <Button>
          <Plus className="mr-2 h-4 w-4" />
          Create Function
        </Button>
      </div>

      {!filtered || filtered.length === 0 ? (
        <EmptyState
          icon={Zap}
          title="No functions"
          description="Deploy your first function to handle HTTP requests, events, or scheduled tasks."
        />
      ) : (
        <FunctionTable
          projectId={projectId}
          functions={filtered.map((fn) => ({
            id: fn.id,
            name: fn.name,
            runtime: fn.runtime || "node20",
            kind: fn.kind || "http",
            mode: fn.mode || "on-demand",
            status: fn.status || "running",
            updatedAt: fn.updatedAt || "",
          }))}
          onDelete={setDeleteTarget}
        />
      )}

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

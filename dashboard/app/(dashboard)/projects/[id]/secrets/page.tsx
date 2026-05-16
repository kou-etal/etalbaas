"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { KeyRound, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { EmptyState } from "@/components/shared/empty-state";
import { TableSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useSecrets, useCreateSecret, useDeleteSecret } from "@/features/secrets/hooks";
import { formatDate } from "@/lib/utils/format";

export default function SecretsPage() {
  const params = useParams();
  const projectId = params.id as string;
  const { data: secrets, isLoading } = useSecrets(projectId);
  const createSecret = useCreateSecret();
  const deleteSecret = useDeleteSecret();
  const [createOpen, setCreateOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [key, setKey] = useState("");
  const [value, setValue] = useState("");

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    await createSecret.mutateAsync({ projectId, key, value });
    setKey("");
    setValue("");
    setCreateOpen(false);
  };

  if (isLoading) return <div className="p-6"><TableSkeleton /></div>;

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">Secrets</h3>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Add Secret
        </Button>
      </div>

      {!secrets || secrets.length === 0 ? (
        <EmptyState
          icon={KeyRound}
          title="No secrets"
          description="Add environment secrets that your functions can access securely at runtime."
          action={{ label: "Add Secret", onClick: () => setCreateOpen(true) }}
        />
      ) : (
        <div className="rounded-lg border">
          <table className="w-full">
            <thead>
              <tr className="border-b bg-muted/50">
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Key</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Value</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Created</th>
                <th className="px-4 py-3 w-10"></th>
              </tr>
            </thead>
            <tbody>
              {secrets.map((secret) => (
                <tr key={secret.key} className="border-b last:border-0 hover:bg-muted/30">
                  <td className="px-4 py-3">
                    <code className="text-sm font-mono">{secret.key}</code>
                  </td>
                  <td className="px-4 py-3">
                    <span className="text-sm text-muted-foreground">••••••••</span>
                  </td>
                  <td className="px-4 py-3">
                    <span className="text-xs text-muted-foreground">
                      {secret.createdAt ? formatDate(secret.createdAt) : "-"}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 text-destructive"
                      onClick={() => setDeleteTarget(secret.key)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="sm:max-w-md">
          <form onSubmit={handleCreate}>
            <DialogHeader>
              <DialogTitle>Add Secret</DialogTitle>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="secret-key">Key</Label>
                <Input
                  id="secret-key"
                  placeholder="MY_SECRET_KEY"
                  value={key}
                  onChange={(e) => setKey(e.target.value)}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="secret-value">Value</Label>
                <Input
                  id="secret-value"
                  type="password"
                  placeholder="secret value"
                  value={value}
                  onChange={(e) => setValue(e.target.value)}
                  required
                />
              </div>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setCreateOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={!key || !value || createSecret.isPending}>
                {createSecret.isPending ? "Adding..." : "Add Secret"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title="Delete Secret"
        description={`This will permanently delete the secret "${deleteTarget}". Functions using this secret will no longer have access to it.`}
        confirmLabel="Delete"
        variant="destructive"
        isLoading={deleteSecret.isPending}
        onConfirm={async () => {
          if (deleteTarget) {
            await deleteSecret.mutateAsync({ projectId, key: deleteTarget });
            setDeleteTarget(null);
          }
        }}
      />
    </div>
  );
}

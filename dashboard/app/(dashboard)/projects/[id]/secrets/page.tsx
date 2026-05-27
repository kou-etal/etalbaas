"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { KeyRound, Plus, Trash2, RotateCw, Lock } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { EmptyState } from "@/components/shared/empty-state";
import { TableSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useSecrets, useCreateSecret, useDeleteSecret, useUpdateSecretValue } from "@/features/secrets/hooks";
import { formatDate, formatRelative } from "@/lib/utils/format";

export default function SecretsPage() {
  const params = useParams();
  const projectId = params.id as string;
  const { data: secrets, isLoading } = useSecrets(projectId);
  const createSecret = useCreateSecret();
  const deleteSecret = useDeleteSecret();
  const updateSecretValue = useUpdateSecretValue();

  const [createOpen, setCreateOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; name: string } | null>(null);
  const [rotateTarget, setRotateTarget] = useState<{ id: string; name: string } | null>(null);

  // Create form
  const [name, setName] = useState("");
  const [value, setValue] = useState("");
  const [description, setDescription] = useState("");

  // Rotate form
  const [rotateValue, setRotateValue] = useState("");

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    await createSecret.mutateAsync({ projectId, name, value, description: description || undefined });
    setName("");
    setValue("");
    setDescription("");
    setCreateOpen(false);
  };

  const handleRotate = async () => {
    if (!rotateTarget) return;
    await updateSecretValue.mutateAsync({
      projectId,
      secretId: rotateTarget.id,
      value: rotateValue,
    });
    setRotateValue("");
    setRotateTarget(null);
  };

  if (isLoading) return <div className="p-6"><TableSkeleton /></div>;

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-medium">Secrets</h3>
          <p className="text-sm text-muted-foreground">
            Environment variables for your functions. Values are write-only and never exposed.
          </p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Add Secret
        </Button>
      </div>

      {/* Info banner */}
      <div className="flex items-start gap-2 rounded-lg border bg-muted/30 p-3">
        <Lock className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
        <p className="text-xs text-muted-foreground">
          Secret values are encrypted and stored in Kubernetes. Once saved, values cannot be read back — only rotated or deleted.
        </p>
      </div>

      {!secrets || secrets.length === 0 ? (
        <EmptyState
          icon={KeyRound}
          title="No secrets configured"
          description="Add environment variables that your functions can access at runtime."
          action={{ label: "Add Secret", onClick: () => setCreateOpen(true) }}
        />
      ) : (
        <div className="rounded-lg border">
          <table className="w-full">
            <thead>
              <tr className="border-b bg-muted/50">
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Name</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Description</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Value</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Last rotated</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Created</th>
                <th className="px-4 py-3 w-20"></th>
              </tr>
            </thead>
            <tbody>
              {secrets.map((secret) => {
                const rotated = secret.updatedAt && secret.updatedAt !== secret.createdAt;
                return (
                  <tr key={secret.id} className="border-b last:border-0 hover:bg-muted/30">
                    <td className="px-4 py-3">
                      <code className="text-sm font-mono">{secret.name}</code>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-xs text-muted-foreground line-clamp-1">
                        {secret.description || "—"}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-sm text-muted-foreground">••••••••••••</span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-xs text-muted-foreground">
                        {rotated ? formatRelative(secret.updatedAt) : "Never rotated"}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-xs text-muted-foreground">
                        {secret.createdAt ? formatDate(secret.createdAt) : "-"}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex gap-1">
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8"
                          onClick={() => setRotateTarget({ id: secret.id, name: secret.name })}
                          title="Rotate value"
                        >
                          <RotateCw className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-8 w-8 text-destructive"
                          onClick={() => setDeleteTarget({ id: secret.id, name: secret.name })}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Create Secret Dialog */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="sm:max-w-md">
          <form onSubmit={handleCreate}>
            <DialogHeader>
              <DialogTitle>Add secret</DialogTitle>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="secret-name">Name</Label>
                <Input
                  id="secret-name"
                  placeholder="MY_SECRET_KEY"
                  value={name}
                  onChange={(e) => setName(e.target.value.toUpperCase().replace(/[^A-Z0-9_]/g, ""))}
                  required
                />
                <p className="text-xs text-muted-foreground">
                  UPPERCASE letters, numbers, and underscores only.
                </p>
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
                <p className="text-xs text-muted-foreground">
                  This value will be encrypted and cannot be retrieved after saving.
                </p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="secret-desc">
                  Description <span className="text-muted-foreground font-normal">— optional</span>
                </Label>
                <Input
                  id="secret-desc"
                  placeholder="What is this secret for?"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                />
              </div>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setCreateOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={!name || !value || createSecret.isPending}>
                {createSecret.isPending ? "Adding..." : "Add Secret"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Rotate Dialog */}
      <Dialog open={!!rotateTarget} onOpenChange={(open) => !open && setRotateTarget(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Rotate secret</DialogTitle>
            <DialogDescription>
              This will immediately replace the current value of <code className="font-mono text-foreground">{rotateTarget?.name}</code>.
              Functions using this secret will pick up the new value on next restart.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2 py-4">
            <Label>New value</Label>
            <Input
              type="password"
              placeholder="new secret value"
              value={rotateValue}
              onChange={(e) => setRotateValue(e.target.value)}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRotateTarget(null)}>
              Cancel
            </Button>
            <Button
              onClick={handleRotate}
              disabled={!rotateValue || updateSecretValue.isPending}
            >
              {updateSecretValue.isPending ? "Rotating..." : "Rotate Value"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Dialog */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title="Delete secret"
        description={`Are you sure you want to delete "${deleteTarget?.name}"? Functions referencing this secret will fail on next restart.`}
        confirmLabel="Delete Secret"
        variant="destructive"
        isLoading={deleteSecret.isPending}
        onConfirm={async () => {
          if (deleteTarget) {
            await deleteSecret.mutateAsync({ projectId, secretId: deleteTarget.id });
            setDeleteTarget(null);
          }
        }}
      />
    </div>
  );
}

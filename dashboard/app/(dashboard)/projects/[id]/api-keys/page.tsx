"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { Key, Plus, Trash2, Copy, Check, AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { useApiKeys, useCreateApiKey, useRevokeApiKey } from "@/features/api-keys/hooks";
import { formatDate } from "@/lib/utils/format";

export default function ApiKeysPage() {
  const params = useParams();
  const projectId = params.id as string;
  const { data: apiKeys, isLoading } = useApiKeys(projectId);
  const createApiKey = useCreateApiKey();
  const revokeApiKey = useRevokeApiKey();
  const [createOpen, setCreateOpen] = useState(false);
  const [revokeTarget, setRevokeTarget] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [role, setRole] = useState("anon");
  const [expiration, setExpiration] = useState("90");
  const [newKey, setNewKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    const expiresInDays = expiration === "0" ? 0 : parseInt(expiration);
    const result = await createApiKey.mutateAsync({
      projectId,
      name,
      role,
      expiresInDays,
    });
    setNewKey(result.rawKey || "");
    setName("");
    setRole("anon");
    setExpiration("90");
  };

  const handleCopy = async () => {
    if (newKey) {
      await navigator.clipboard.writeText(newKey);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  if (isLoading) return <div className="p-6"><TableSkeleton /></div>;

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-medium">API Keys</h3>
          <p className="text-sm text-muted-foreground">
            Authenticate requests to your project&apos;s API.
          </p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Create Key
        </Button>
      </div>

      {!apiKeys || apiKeys.length === 0 ? (
        <EmptyState
          icon={Key}
          title="No API keys"
          description="Create API keys to authenticate requests to your project's endpoints."
          action={{ label: "Create Key", onClick: () => setCreateOpen(true) }}
        />
      ) : (
        <div className="rounded-lg border">
          <table className="w-full">
            <thead>
              <tr className="border-b bg-muted/50">
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Name</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Key</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Role</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">Created</th>
                <th className="px-4 py-3 w-10"></th>
              </tr>
            </thead>
            <tbody>
              {apiKeys.map((apiKey) => (
                <tr key={apiKey.id} className="border-b last:border-0 hover:bg-muted/30">
                  <td className="px-4 py-3">
                    <span className="text-sm font-medium">{apiKey.name}</span>
                  </td>
                  <td className="px-4 py-3">
                    <code className="text-xs font-mono text-muted-foreground">
                      {apiKey.keyPrefix || "••••"}••••••••
                    </code>
                  </td>
                  <td className="px-4 py-3">
                    <Badge variant={apiKey.role === "service_role" ? "destructive" : "secondary"} className="text-xs">
                      {apiKey.role === "service_role" ? "service_role" : "anon"}
                    </Badge>
                  </td>
                  <td className="px-4 py-3">
                    <span className="text-xs text-muted-foreground">
                      {apiKey.createdAt ? formatDate(apiKey.createdAt) : "-"}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 text-destructive"
                      onClick={() => setRevokeTarget(apiKey.id)}
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

      {/* Create API Key Dialog */}
      <Dialog open={createOpen} onOpenChange={(open) => { setCreateOpen(open); if (!open) setNewKey(null); }}>
        <DialogContent className="sm:max-w-md">
          {newKey ? (
            <>
              <DialogHeader>
                <DialogTitle>API Key Created</DialogTitle>
                <DialogDescription>Your new key is ready</DialogDescription>
              </DialogHeader>
              <div className="space-y-3 py-4">
                <Label>Full key</Label>
                <div className="flex items-center gap-2">
                  <code className="flex-1 rounded-md bg-muted p-3 text-xs font-mono break-all">
                    {newKey}
                  </code>
                  <Button variant="outline" size="icon" onClick={handleCopy}>
                    {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                  </Button>
                </div>
                <p className="text-xs text-muted-foreground">
                  Copy this key now. It will not be shown again.
                </p>
              </div>
              <DialogFooter>
                <Button onClick={() => { setCreateOpen(false); setNewKey(null); }}>
                  Done
                </Button>
              </DialogFooter>
            </>
          ) : (
            <form onSubmit={handleCreate}>
              <DialogHeader>
                <DialogTitle>Create API Key</DialogTitle>
              </DialogHeader>
              <div className="space-y-4 py-4">
                {/* Name */}
                <div className="space-y-2">
                  <Label htmlFor="key-name">Key Name</Label>
                  <Input
                    id="key-name"
                    placeholder="Production API Key"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    maxLength={63}
                    required
                  />
                  <p className="text-xs text-muted-foreground">Max 63 characters.</p>
                </div>

                {/* Role */}
                <div className="space-y-3">
                  <Label>Role</Label>
                  <RadioGroup value={role} onValueChange={setRole} className="space-y-2">
                    <label className="flex items-start gap-3 rounded-lg border p-3 cursor-pointer hover:bg-muted/50">
                      <RadioGroupItem value="anon" className="mt-0.5" />
                      <div>
                        <p className="text-sm font-medium">anon</p>
                        <p className="text-xs text-muted-foreground">Public / Client-safe · Respects RLS</p>
                      </div>
                    </label>
                    <label className="flex items-start gap-3 rounded-lg border p-3 cursor-pointer hover:bg-muted/50">
                      <RadioGroupItem value="service_role" className="mt-0.5" />
                      <div>
                        <p className="text-sm font-medium">service_role</p>
                        <p className="text-xs text-muted-foreground">Admin / Server-only · Bypasses RLS</p>
                      </div>
                    </label>
                  </RadioGroup>
                  {role === "service_role" && (
                    <div className="flex items-center gap-2 rounded-md bg-destructive/10 p-2">
                      <AlertTriangle className="h-4 w-4 text-destructive shrink-0" />
                      <p className="text-xs text-destructive">
                        This key has full database access. Keep it secret.
                      </p>
                    </div>
                  )}
                </div>

                {/* Expiration */}
                <div className="space-y-2">
                  <Label>Expiration</Label>
                  <Select value={expiration} onValueChange={setExpiration}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="30">30 days</SelectItem>
                      <SelectItem value="90">90 days (Recommended)</SelectItem>
                      <SelectItem value="365">1 year</SelectItem>
                      <SelectItem value="0">No expiration</SelectItem>
                    </SelectContent>
                  </Select>
                  {expiration === "0" && (
                    <p className="text-xs text-destructive">Security risk — key never expires.</p>
                  )}
                </div>
              </div>
              <DialogFooter>
                <Button type="button" variant="outline" onClick={() => setCreateOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit" disabled={!name || createApiKey.isPending}>
                  {createApiKey.isPending ? "Creating..." : "Create Key"}
                </Button>
              </DialogFooter>
            </form>
          )}
        </DialogContent>
      </Dialog>

      {/* Revoke Dialog */}
      <ConfirmDialog
        open={!!revokeTarget}
        onOpenChange={(open) => !open && setRevokeTarget(null)}
        title="Revoke API Key"
        description="Revoking this key will immediately break any application using it."
        confirmLabel="Revoke Key"
        variant="destructive"
        isLoading={revokeApiKey.isPending}
        onConfirm={async () => {
          if (revokeTarget) {
            await revokeApiKey.mutateAsync({ projectId, keyId: revokeTarget });
            setRevokeTarget(null);
          }
        }}
      />
    </div>
  );
}

"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { HardDrive, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { EmptyState } from "@/components/shared/empty-state";
import { PageSkeleton } from "@/components/shared/loading-skeleton";
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
import { useBuckets, useCreateBucket } from "@/features/storage/hooks";

const accessLevelVariant: Record<string, "warning" | "secondary" | "info"> = {
  public: "warning",
  protected: "info",
  private: "secondary",
};

export default function StoragePage() {
  const params = useParams();
  const projectId = params.id as string;
  const { data: buckets, isLoading } = useBuckets(projectId);
  const createBucket = useCreateBucket();
  const [createOpen, setCreateOpen] = useState(false);
  const [bucketName, setBucketName] = useState("");
  const [accessLevel, setAccessLevel] = useState("private");

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    await createBucket.mutateAsync({ projectId, name: bucketName, accessLevel });
    setBucketName("");
    setAccessLevel("private");
    setCreateOpen(false);
  };

  if (isLoading) return <PageSkeleton />;

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-medium">Buckets</h3>
          <p className="text-sm text-muted-foreground">
            Manage buckets and objects.
          </p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Create Bucket
        </Button>
      </div>

      {!buckets || buckets.length === 0 ? (
        <EmptyState
          icon={HardDrive}
          title="No buckets yet"
          description="Create a bucket to start storing files for this project."
          action={{ label: "Create Bucket", onClick: () => setCreateOpen(true) }}
        />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {buckets.map((bucket) => (
            <Card key={bucket.id} className="hover:border-primary/30 transition-colors">
              <CardHeader className="pb-2">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <HardDrive className="h-4 w-4 text-muted-foreground" />
                    <CardTitle className="text-sm font-medium">{bucket.name}</CardTitle>
                  </div>
                  <Badge variant={accessLevelVariant[bucket.accessLevel] || "secondary"}>
                    {bucket.accessLevel}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent>
                {(bucket.fileSizeLimit > 0 || (bucket.allowedMimeTypes && bucket.allowedMimeTypes.length > 0)) && (
                  <div className="text-xs text-muted-foreground space-y-0.5">
                    {bucket.fileSizeLimit > 0 && (
                      <p>{Math.round(bucket.fileSizeLimit / (1024 * 1024))} MB max</p>
                    )}
                    {bucket.allowedMimeTypes && bucket.allowedMimeTypes.length > 0 && (
                      <p>{bucket.allowedMimeTypes.join(", ")}</p>
                    )}
                  </div>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="sm:max-w-md">
          <form onSubmit={handleCreate}>
            <DialogHeader>
              <DialogTitle>Create Bucket</DialogTitle>
              <DialogDescription>
                Create a new storage bucket for this project.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="bucket-name">Bucket Name</Label>
                <Input
                  id="bucket-name"
                  placeholder="my-bucket"
                  value={bucketName}
                  onChange={(e) => setBucketName(e.target.value)}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="access-level">Access Level</Label>
                <Select value={accessLevel} onValueChange={setAccessLevel}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="private">Private</SelectItem>
                    <SelectItem value="protected">Protected</SelectItem>
                    <SelectItem value="public">Public</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setCreateOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={!bucketName || createBucket.isPending}>
                {createBucket.isPending ? "Creating..." : "Create"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}

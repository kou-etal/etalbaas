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
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useBuckets, useCreateBucket } from "@/features/storage/hooks";

export default function StoragePage() {
  const params = useParams();
  const projectId = params.id as string;
  const { data: buckets, isLoading } = useBuckets(projectId);
  const createBucket = useCreateBucket();
  const [createOpen, setCreateOpen] = useState(false);
  const [bucketName, setBucketName] = useState("");
  const [isPublic, setIsPublic] = useState(false);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    await createBucket.mutateAsync({ projectId, name: bucketName, isPublic });
    setBucketName("");
    setIsPublic(false);
    setCreateOpen(false);
  };

  if (isLoading) return <PageSkeleton />;

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">Buckets</h3>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Create Bucket
        </Button>
      </div>

      {!buckets || buckets.length === 0 ? (
        <EmptyState
          icon={HardDrive}
          title="No buckets"
          description="Create a storage bucket to store files for your project."
          action={{ label: "Create Bucket", onClick: () => setCreateOpen(true) }}
        />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {buckets.map((bucket) => (
            <Card key={bucket.id}>
              <CardHeader className="pb-2">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-sm font-medium">{bucket.name}</CardTitle>
                  <Badge variant={bucket.isPublic ? "warning" : "secondary"}>
                    {bucket.isPublic ? "Public" : "Private"}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent>
                <p className="text-xs text-muted-foreground">
                  {bucket.objectCount || 0} objects
                </p>
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
              <div className="flex items-center gap-2">
                <input
                  type="checkbox"
                  id="is-public"
                  checked={isPublic}
                  onChange={(e) => setIsPublic(e.target.checked)}
                  className="rounded"
                />
                <Label htmlFor="is-public">Public access</Label>
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

"use client";

import { useParams, useRouter } from "next/navigation";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { useProject, useDeleteProject } from "@/features/projects/hooks";
import { DetailSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { useState } from "react";

export default function ProjectSettingsPage() {
  const params = useParams();
  const router = useRouter();
  const projectId = params.id as string;
  const { data: project, isLoading } = useProject(projectId);
  const deleteProject = useDeleteProject();
  const [deleteOpen, setDeleteOpen] = useState(false);

  if (isLoading) return <DetailSkeleton />;

  return (
    <div className="p-6 space-y-6 max-w-2xl">
      {/* General */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">General</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <p className="text-sm font-medium">Project Name</p>
            <p className="text-sm text-muted-foreground">{project?.name || "-"}</p>
          </div>
          <div>
            <p className="text-sm font-medium">Description</p>
            <p className="text-sm text-muted-foreground">{project?.description || "No description"}</p>
          </div>
          <div>
            <p className="text-sm font-medium">Region</p>
            <p className="text-sm text-muted-foreground">{project?.region || "ap-northeast-1"}</p>
          </div>
          <div>
            <p className="text-sm font-medium">Project ID</p>
            <code className="text-sm font-mono text-muted-foreground">{projectId}</code>
          </div>
        </CardContent>
      </Card>

      {/* Danger Zone */}
      <Card className="border-destructive/50">
        <CardHeader>
          <CardTitle className="text-base text-destructive">Danger Zone</CardTitle>
          <CardDescription>
            Irreversible and destructive actions
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Separator className="mb-4" />
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">Delete this project</p>
              <p className="text-xs text-muted-foreground">
                Once deleted, all data will be permanently removed.
              </p>
            </div>
            <Button variant="destructive" onClick={() => setDeleteOpen(true)}>
              Delete Project
            </Button>
          </div>
        </CardContent>
      </Card>

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete Project"
        description={`This will permanently delete "${project?.name}" and all associated resources. This action cannot be undone.`}
        confirmLabel="Delete"
        variant="destructive"
        isLoading={deleteProject.isPending}
        onConfirm={async () => {
          await deleteProject.mutateAsync(projectId);
          router.push("/projects");
        }}
      />
    </div>
  );
}

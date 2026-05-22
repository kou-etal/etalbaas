"use client";

import { useParams, useRouter } from "next/navigation";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { useProject, useDeleteProject, usePauseProject, useResumeProject } from "@/features/projects/hooks";
import { DetailSkeleton } from "@/components/shared/loading-skeleton";
import { TypeToConfirmDialog } from "@/components/shared/type-to-confirm-dialog";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { useState } from "react";
import { Database, Server, Globe, Pause, Play, ArrowRightLeft, Trash2 } from "lucide-react";

export default function ProjectSettingsPage() {
  const params = useParams();
  const router = useRouter();
  const projectId = params.id as string;
  const { data: project, isLoading } = useProject(projectId);
  const deleteProject = useDeleteProject();
  const pauseProject = usePauseProject();
  const resumeProject = useResumeProject();

  const [deleteOpen, setDeleteOpen] = useState(false);
  const [pauseOpen, setPauseOpen] = useState(false);

  if (isLoading) return <DetailSkeleton />;

  const isPaused = project?.status === "paused";

  return (
    <div className="p-6 space-y-6 max-w-2xl">
      <div>
        <h2 className="text-lg font-semibold">Configure your project and its services.</h2>
      </div>

      {/* General */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">General</CardTitle>
          <CardDescription>Project information</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>Display name</Label>
            <Input
              value={project?.displayName || ""}
              disabled
              className="bg-muted"
            />
            <p className="text-xs text-muted-foreground">
              Used in the dashboard and notifications.
            </p>
          </div>
          <div className="space-y-2">
            <div className="flex items-baseline justify-between">
              <Label>Description</Label>
              <span className="text-xs text-muted-foreground">
                {project?.description?.length || 0} / 500
              </span>
            </div>
            <Textarea
              value={project?.description || ""}
              disabled
              className="bg-muted"
              rows={3}
            />
          </div>
          <div className="space-y-2">
            <Label>Project ID</Label>
            <div className="flex items-center gap-2">
              <code className="rounded bg-muted px-3 py-2 text-sm font-mono flex-1">
                {projectId}
              </code>
            </div>
            <p className="text-xs text-muted-foreground">
              Cannot be changed. Used as the subdomain for API endpoints.
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Services */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Services</CardTitle>
          <CardDescription>
            Turn services on or off. Disabling a service may destroy associated data.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* PostgreSQL */}
          <div className="rounded-lg border p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <Switch checked={project?.postgresEnabled ?? false} disabled />
                <div className="flex items-center gap-2">
                  <Database className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm font-medium">PostgreSQL</span>
                  <span className="text-xs text-muted-foreground">v16</span>
                </div>
              </div>
            </div>
            <p className="text-xs text-muted-foreground ml-[56px] mt-1">
              Managed database cluster with point-in-time recovery.
            </p>
            {project?.postgresEnabled && project.postgresExtensions && project.postgresExtensions.length > 0 && (
              <div className="ml-[56px] mt-2 flex flex-wrap gap-1.5">
                <span className="text-xs text-muted-foreground">Extensions:</span>
                {project.postgresExtensions.map((ext) => (
                  <Badge key={ext} variant="outline" className="text-xs">
                    {ext}
                  </Badge>
                ))}
              </div>
            )}
          </div>

          {/* Redis */}
          <div className="rounded-lg border p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <Switch checked={project?.redisEnabled ?? false} disabled />
                <div className="flex items-center gap-2">
                  <Server className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm font-medium">Redis</span>
                  <span className="text-xs text-muted-foreground">v7.2</span>
                </div>
              </div>
            </div>
            <p className="text-xs text-muted-foreground ml-[56px] mt-1">
              In-memory key-value cache with persistence.
            </p>
          </div>

          {/* PostgREST */}
          <div className="rounded-lg border p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <Switch checked={project?.postgrestEnabled ?? false} disabled />
                <div className="flex items-center gap-2">
                  <Globe className="h-4 w-4 text-muted-foreground" />
                  <span className="text-sm font-medium">PostgREST</span>
                  <span className="text-xs text-muted-foreground">v12</span>
                </div>
              </div>
            </div>
            <p className="text-xs text-muted-foreground ml-[56px] mt-1">
              Instant RESTful API from your database schema.
            </p>
            {project?.postgrestEnabled && (
              <p className="text-xs text-muted-foreground ml-[56px] mt-1 font-mono">
                https://{projectId}.etalbaas.io/rest/v1
              </p>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Danger Zone */}
      <Card className="border-destructive/50">
        <CardHeader>
          <CardTitle className="text-base text-destructive">Danger zone</CardTitle>
          <CardDescription>Irreversible operations on this project.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Pause / Resume */}
          <div className="flex items-center justify-between">
            <div className="flex-1">
              <div className="flex items-center gap-2">
                {isPaused ? (
                  <Play className="h-4 w-4 text-muted-foreground" />
                ) : (
                  <Pause className="h-4 w-4 text-muted-foreground" />
                )}
                <p className="text-sm font-medium">
                  {isPaused ? "Resume project" : "Pause project"}
                </p>
              </div>
              <p className="text-xs text-muted-foreground mt-1 ml-6">
                {isPaused
                  ? "Resume all services. Data has been preserved."
                  : "Temporarily stop all services. Data is preserved. No compute charges while paused."}
              </p>
            </div>
            <Button
              variant="outline"
              onClick={() => setPauseOpen(true)}
              disabled={pauseProject.isPending || resumeProject.isPending}
            >
              {isPaused ? "Resume Project" : "Pause Project"}
            </Button>
          </div>

          <Separator />

          {/* Transfer */}
          <div className="flex items-center justify-between">
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <ArrowRightLeft className="h-4 w-4 text-muted-foreground" />
                <p className="text-sm font-medium">Transfer ownership</p>
              </div>
              <p className="text-xs text-muted-foreground mt-1 ml-6">
                Transfer this project to another tenant.
              </p>
            </div>
            <Button variant="outline" disabled>
              Transfer
              <Badge variant="secondary" className="ml-2 text-[10px]">Coming soon</Badge>
            </Button>
          </div>

          <Separator />

          {/* Delete */}
          <div className="flex items-center justify-between">
            <div className="flex-1">
              <div className="flex items-center gap-2">
                <Trash2 className="h-4 w-4 text-destructive" />
                <p className="text-sm font-medium">Delete project</p>
              </div>
              <p className="text-xs text-muted-foreground mt-1 ml-6">
                Permanently delete this project and all its data including databases, functions, storage, and secrets. This action cannot be undone.
              </p>
            </div>
            <Button variant="destructive" onClick={() => setDeleteOpen(true)}>
              Delete Project
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Pause/Resume Confirm */}
      <ConfirmDialog
        open={pauseOpen}
        onOpenChange={setPauseOpen}
        title={isPaused ? "Resume project?" : "Pause project?"}
        description={
          isPaused
            ? "All services will be restarted."
            : "All services will be stopped. Your data (database, storage, secrets) will be preserved. You can resume at any time."
        }
        confirmLabel={isPaused ? "Resume" : "Pause Project"}
        variant={isPaused ? "default" : "destructive"}
        isLoading={pauseProject.isPending || resumeProject.isPending}
        onConfirm={async () => {
          if (isPaused) {
            await resumeProject.mutateAsync(projectId);
          } else {
            await pauseProject.mutateAsync(projectId);
          }
          setPauseOpen(false);
        }}
      />

      {/* Delete Confirm (type-to-confirm) */}
      <TypeToConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete this project?"
        description={
          <div className="space-y-2">
            <p>
              This will permanently delete <span className="font-semibold">{project?.displayName}</span> and all of its data — databases, functions, buckets, and keys. This action cannot be undone.
            </p>
          </div>
        }
        confirmText={project?.displayName || ""}
        confirmLabel="Delete Project"
        isLoading={deleteProject.isPending}
        onConfirm={async () => {
          await deleteProject.mutateAsync(projectId);
          router.push("/projects");
        }}
      />
    </div>
  );
}

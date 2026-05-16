"use client";

import { useState } from "react";
import { Plus, Search, FolderKanban } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ProjectCard } from "@/features/projects/project-card";
import { CreateProjectDialog } from "@/features/projects/create-project-dialog";
import { EmptyState } from "@/components/shared/empty-state";
import { PageSkeleton } from "@/components/shared/loading-skeleton";
import { useProjects, useDeleteProject } from "@/features/projects/hooks";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";

export default function ProjectsPage() {
  const [search, setSearch] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const { data: projects, isLoading } = useProjects();
  const deleteProject = useDeleteProject();

  const filtered = projects?.filter(
    (p) =>
      p.name.toLowerCase().includes(search.toLowerCase()) ||
      p.description?.toLowerCase().includes(search.toLowerCase())
  );

  if (isLoading) return <PageSkeleton />;

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Projects</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Manage your backend projects
          </p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          New Project
        </Button>
      </div>

      {/* Search */}
      {projects && projects.length > 0 && (
        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search projects..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
      )}

      {/* Grid */}
      {!filtered || filtered.length === 0 ? (
        <EmptyState
          icon={FolderKanban}
          title="No projects yet"
          description="Create your first project to get started with functions, events, and storage."
          action={{ label: "Create Project", onClick: () => setCreateOpen(true) }}
        />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {filtered.map((project) => (
            <ProjectCard
              key={project.id}
              project={{
                id: project.id,
                name: project.name,
                description: project.description || "",
                status: "active",
                region: project.region || "ap-northeast-1",
                createdAt: project.createdAt || "",
              }}
              onDelete={setDeleteTarget}
            />
          ))}
        </div>
      )}

      <CreateProjectDialog open={createOpen} onOpenChange={setCreateOpen} />
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title="Delete Project"
        description="This action cannot be undone. All functions, events, secrets, and storage within this project will be permanently deleted."
        confirmLabel="Delete"
        variant="destructive"
        isLoading={deleteProject.isPending}
        onConfirm={async () => {
          if (deleteTarget) {
            await deleteProject.mutateAsync(deleteTarget);
            setDeleteTarget(null);
          }
        }}
      />
    </div>
  );
}

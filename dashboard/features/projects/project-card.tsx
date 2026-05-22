"use client";

import Link from "next/link";
import { FolderKanban, MoreVertical, Trash2 } from "lucide-react";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { StatusBadge } from "@/components/shared/status-badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Button } from "@/components/ui/button";
import { formatRelative } from "@/lib/utils/format";

interface ProjectCardProps {
  project: {
    id: string;
    displayName: string;
    description: string;
    status: string;
    postgresEnabled?: boolean;
    redisEnabled?: boolean;
    postgrestEnabled?: boolean;
    createdAt: string;
  };
  onDelete?: (id: string) => void;
}

function ServicePill({ label }: { label: string }) {
  return (
    <span className="inline-flex items-center rounded bg-muted px-1.5 py-0.5 text-[10px] font-semibold text-muted-foreground">
      {label}
    </span>
  );
}

export function ProjectCard({ project, onDelete }: ProjectCardProps) {
  return (
    <Card className="group relative hover:border-primary/30 transition-colors">
      <Link href={`/projects/${project.id}`} className="absolute inset-0 z-10" />
      <CardHeader className="flex flex-row items-start justify-between space-y-0 pb-2">
        <div className="flex items-center gap-3">
          <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10">
            <FolderKanban className="h-4 w-4 text-primary" />
          </div>
          <div>
            <h3 className="font-medium leading-none">{project.displayName}</h3>
            <p className="text-xs text-muted-foreground mt-1">{project.id}</p>
          </div>
        </div>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="relative z-20 h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity"
            >
              <MoreVertical className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem
              className="text-destructive"
              onClick={() => onDelete?.(project.id)}
            >
              <Trash2 className="mr-2 h-4 w-4" />
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </CardHeader>
      <CardContent>
        <div className="flex items-center justify-between mb-3">
          <StatusBadge status={project.status || "pending"} />
        </div>
        <p className="text-sm text-muted-foreground line-clamp-2 mb-4">
          {project.description || "No description"}
        </p>
        <div className="flex items-center justify-between">
          <div className="flex gap-1.5">
            {project.postgresEnabled && <ServicePill label="PG" />}
            {project.redisEnabled && <ServicePill label="RD" />}
            {project.postgrestEnabled && <ServicePill label="API" />}
          </div>
          <span className="text-xs text-muted-foreground">
            {project.createdAt ? formatRelative(project.createdAt) : ""}
          </span>
        </div>
      </CardContent>
    </Card>
  );
}

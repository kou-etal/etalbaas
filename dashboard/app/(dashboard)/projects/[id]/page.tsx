"use client";

import { useParams } from "next/navigation";
import { Activity, Zap, Radio, HardDrive } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { StatusBadge } from "@/components/shared/status-badge";
import { DetailSkeleton } from "@/components/shared/loading-skeleton";
import { useProject } from "@/features/projects/hooks";
import { formatDate } from "@/lib/utils/format";

export default function ProjectOverviewPage() {
  const params = useParams();
  const projectId = params.id as string;
  const { data: project, isLoading } = useProject(projectId);

  if (isLoading) return <DetailSkeleton />;

  return (
    <div className="p-6 space-y-6">
      {/* Project Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-bold">{project?.name || "Project"}</h2>
          <p className="text-sm text-muted-foreground mt-1">
            {project?.description || "No description"}
          </p>
        </div>
        <StatusBadge status="active" />
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Functions</CardTitle>
            <Zap className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">0</div>
            <p className="text-xs text-muted-foreground">deployed</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Events</CardTitle>
            <Radio className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">0</div>
            <p className="text-xs text-muted-foreground">last 24h</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Storage</CardTitle>
            <HardDrive className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">0 MB</div>
            <p className="text-xs text-muted-foreground">used</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Uptime</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">100%</div>
            <p className="text-xs text-muted-foreground">last 7 days</p>
          </CardContent>
        </Card>
      </div>

      {/* Connection Info */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Connection Info</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="grid gap-2 sm:grid-cols-2">
            <div>
              <p className="text-xs text-muted-foreground">Project ID</p>
              <code className="text-sm font-mono">{projectId}</code>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Region</p>
              <code className="text-sm font-mono">{project?.region || "ap-northeast-1"}</code>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">API Endpoint</p>
              <code className="text-sm font-mono break-all">
                {process.env.NEXT_PUBLIC_API_URL || "https://api.etalbaas.io"}/{projectId}
              </code>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">Created</p>
              <code className="text-sm font-mono">
                {project?.createdAt ? formatDate(project.createdAt) : "-"}
              </code>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

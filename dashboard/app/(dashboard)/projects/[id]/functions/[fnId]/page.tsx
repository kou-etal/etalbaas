"use client";

import { useParams } from "next/navigation";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useFunction } from "@/features/functions/hooks";
import { formatDate } from "@/lib/utils/format";

export default function FunctionOverviewPage() {
  const params = useParams();
  const fnId = params.fnId as string;
  const { data: fn } = useFunction(fnId);

  return (
    <div className="p-6 space-y-6">
      {/* Build Info */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Build Info</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2">
          <div>
            <p className="text-xs text-muted-foreground">Runtime</p>
            <p className="text-sm font-mono">{fn?.runtime || "-"}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Entrypoint</p>
            <p className="text-sm font-mono">{fn?.entrypoint || "handler"}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Image</p>
            <p className="text-sm font-mono break-all">{fn?.image || "-"}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Last Build</p>
            <p className="text-sm">{fn?.lastBuildAt ? formatDate(fn.lastBuildAt) : "Never"}</p>
          </div>
        </CardContent>
      </Card>

      {/* Resources */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Resources</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-3">
          <div>
            <p className="text-xs text-muted-foreground">CPU</p>
            <p className="text-sm font-mono">{fn?.resources?.cpu || "100m"}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Memory</p>
            <p className="text-sm font-mono">{fn?.resources?.memory || "128Mi"}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Timeout</p>
            <p className="text-sm font-mono">{fn?.timeout || "30"}s</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Min Replicas</p>
            <p className="text-sm font-mono">{fn?.minReplicas ?? 0}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Max Replicas</p>
            <p className="text-sm font-mono">{fn?.maxReplicas ?? 10}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">GPU</p>
            <p className="text-sm font-mono">{fn?.gpu || "none"}</p>
          </div>
        </CardContent>
      </Card>

      {/* Environment Variables */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Environment Variables</CardTitle>
        </CardHeader>
        <CardContent>
          {fn?.envVars && Object.keys(fn.envVars).length > 0 ? (
            <div className="rounded-md border">
              <table className="w-full">
                <thead>
                  <tr className="border-b bg-muted/50">
                    <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">Key</th>
                    <th className="px-3 py-2 text-left text-xs font-medium text-muted-foreground">Source</th>
                  </tr>
                </thead>
                <tbody>
                  {Object.entries(fn.envVars).map(([key, source]) => (
                    <tr key={key} className="border-b last:border-0">
                      <td className="px-3 py-2 text-sm font-mono">{key}</td>
                      <td className="px-3 py-2 text-sm text-muted-foreground">{source as string}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">No environment variables configured.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

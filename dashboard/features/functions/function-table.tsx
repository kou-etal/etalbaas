"use client";

import Link from "next/link";
import { Zap, MoreVertical, Trash2, RotateCw } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/shared/status-badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { formatRelative } from "@/lib/utils/format";

interface FunctionRow {
  id: string;
  name: string;
  displayName?: string;
  runtime: string;
  kind: string;
  mode: string;
  status: string;
  lastBuiltAt: string;
}

interface FunctionTableProps {
  projectId: string;
  functions: FunctionRow[];
  onDelete?: (id: string) => void;
}

const kindLabel: Record<string, string> = {
  "heavy-job": "Heavy Job",
  "heavy-deployment": "Heavy Deploy",
  "light-deployment": "Light Deploy",
};

export function FunctionTable({ projectId, functions, onDelete }: FunctionTableProps) {
  return (
    <div className="rounded-lg border">
      <table className="w-full">
        <thead>
          <tr className="border-b bg-muted/50">
            <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
              Name
            </th>
            <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
              Kind
            </th>
            <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
              Mode
            </th>
            <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
              Status
            </th>
            <th className="px-4 py-3 text-left text-xs font-medium text-muted-foreground">
              Last built
            </th>
            <th className="px-4 py-3 w-10"></th>
          </tr>
        </thead>
        <tbody>
          {functions.map((fn) => (
            <tr
              key={fn.id}
              className="border-b last:border-0 hover:bg-muted/30 transition-colors"
            >
              <td className="px-4 py-3">
                <Link
                  href={`/projects/${projectId}/functions/${fn.id}`}
                  className="flex items-center gap-2 hover:text-primary transition-colors"
                >
                  <Zap className="h-4 w-4 text-primary" />
                  <div>
                    <span className="font-medium text-sm">{fn.displayName || fn.name}</span>
                    {fn.displayName && fn.displayName !== fn.name && (
                      <p className="text-xs text-muted-foreground font-mono">{fn.name}</p>
                    )}
                  </div>
                </Link>
              </td>
              <td className="px-4 py-3">
                <Badge variant="outline" className="text-xs">
                  {kindLabel[fn.kind] || fn.kind}
                </Badge>
              </td>
              <td className="px-4 py-3">
                <span className="text-sm text-muted-foreground">{fn.mode}</span>
              </td>
              <td className="px-4 py-3">
                <StatusBadge status={fn.status || "pending"} />
              </td>
              <td className="px-4 py-3">
                <span className="text-xs text-muted-foreground">
                  {fn.lastBuiltAt ? formatRelative(fn.lastBuiltAt) : "Never"}
                </span>
              </td>
              <td className="px-4 py-3">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="icon" className="h-8 w-8">
                      <MoreVertical className="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem>
                      <RotateCw className="mr-2 h-4 w-4" />
                      Rebuild
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      className="text-destructive"
                      onClick={() => onDelete?.(fn.id)}
                    >
                      <Trash2 className="mr-2 h-4 w-4" />
                      Delete
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

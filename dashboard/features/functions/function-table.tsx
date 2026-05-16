"use client";

import Link from "next/link";
import { Zap, MoreVertical, Trash2 } from "lucide-react";
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

interface FunctionItem {
  id: string;
  name: string;
  runtime: string;
  kind: string;
  mode: string;
  status: string;
  updatedAt: string;
}

interface FunctionTableProps {
  projectId: string;
  functions: FunctionItem[];
  onDelete?: (id: string) => void;
}

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
              Runtime
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
              Updated
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
                  <span className="font-medium text-sm">{fn.name}</span>
                </Link>
              </td>
              <td className="px-4 py-3">
                <Badge variant="outline" className="text-xs">
                  {fn.runtime}
                </Badge>
              </td>
              <td className="px-4 py-3">
                <span className="text-sm text-muted-foreground">{fn.kind}</span>
              </td>
              <td className="px-4 py-3">
                <span className="text-sm text-muted-foreground">{fn.mode}</span>
              </td>
              <td className="px-4 py-3">
                <StatusBadge status={fn.status || "running"} />
              </td>
              <td className="px-4 py-3">
                <span className="text-xs text-muted-foreground">
                  {fn.updatedAt ? formatRelative(fn.updatedAt) : "-"}
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

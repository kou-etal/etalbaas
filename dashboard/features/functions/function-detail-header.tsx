"use client";

import { Zap } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { StatusBadge } from "@/components/shared/status-badge";

interface FunctionDetailHeaderProps {
  name: string;
  runtime: string;
  kind: string;
  mode: string;
  status: string;
}

export function FunctionDetailHeader({
  name,
  runtime,
  kind,
  mode,
  status,
}: FunctionDetailHeaderProps) {
  return (
    <div className="flex items-center justify-between px-6 py-4 border-b border-border">
      <div className="flex items-center gap-4">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
          <Zap className="h-5 w-5 text-primary" />
        </div>
        <div>
          <h2 className="text-lg font-bold">{name}</h2>
          <div className="flex items-center gap-2 mt-1">
            <Badge variant="outline" className="text-xs">{runtime}</Badge>
            <Badge variant="secondary" className="text-xs">{kind}</Badge>
            <Badge variant="secondary" className="text-xs">{mode}</Badge>
          </div>
        </div>
      </div>
      <StatusBadge status={status} />
    </div>
  );
}

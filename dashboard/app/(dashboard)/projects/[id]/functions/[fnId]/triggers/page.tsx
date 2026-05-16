"use client";

import { Radio, Globe, Clock, Plus } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/shared/empty-state";

// Placeholder data until real API integration
const mockTriggers = [
  {
    id: "1",
    type: "http",
    config: { method: "POST", path: "/invoke" },
  },
];

export default function TriggersPage() {
  const triggers = mockTriggers;

  const getIcon = (type: string) => {
    switch (type) {
      case "http": return Globe;
      case "event": return Radio;
      case "cron": return Clock;
      default: return Radio;
    }
  };

  if (triggers.length === 0) {
    return (
      <div className="p-6">
        <EmptyState
          icon={Radio}
          title="No triggers"
          description="Add triggers to define how your function is invoked."
        />
      </div>
    );
  }

  return (
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">Triggers</h3>
        <Button>
          <Plus className="mr-2 h-4 w-4" />
          Add Trigger
        </Button>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        {triggers.map((trigger) => {
          const Icon = getIcon(trigger.type);
          return (
            <Card key={trigger.id}>
              <CardHeader className="flex flex-row items-center gap-3 pb-2">
                <div className="flex h-8 w-8 items-center justify-center rounded-md bg-primary/10">
                  <Icon className="h-4 w-4 text-primary" />
                </div>
                <div>
                  <CardTitle className="text-sm">{trigger.type.toUpperCase()} Trigger</CardTitle>
                </div>
              </CardHeader>
              <CardContent>
                <div className="flex flex-wrap gap-2">
                  {Object.entries(trigger.config).map(([key, value]) => (
                    <Badge key={key} variant="outline" className="text-xs">
                      {key}: {value}
                    </Badge>
                  ))}
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
}

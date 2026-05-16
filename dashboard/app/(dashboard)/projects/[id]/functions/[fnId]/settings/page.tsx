"use client";

import { useParams, useRouter } from "next/navigation";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { useFunction, useDeleteFunction } from "@/features/functions/hooks";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { useState } from "react";

export default function FunctionSettingsPage() {
  const params = useParams();
  const router = useRouter();
  const projectId = params.id as string;
  const fnId = params.fnId as string;
  const { data: fn } = useFunction(fnId);
  const deleteFunction = useDeleteFunction();
  const [deleteOpen, setDeleteOpen] = useState(false);

  return (
    <div className="p-6 space-y-6 max-w-2xl">
      {/* General Info */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">General</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <div>
            <p className="text-sm font-medium">Function Name</p>
            <p className="text-sm text-muted-foreground">{fn?.name || "-"}</p>
          </div>
          <div>
            <p className="text-sm font-medium">Function ID</p>
            <code className="text-sm font-mono text-muted-foreground">{fnId}</code>
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
              <p className="text-sm font-medium">Delete this function</p>
              <p className="text-xs text-muted-foreground">
                This will remove the function, its triggers, and all invocation history.
              </p>
            </div>
            <Button variant="destructive" onClick={() => setDeleteOpen(true)}>
              Delete Function
            </Button>
          </div>
        </CardContent>
      </Card>

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete Function"
        description={`This will permanently delete "${fn?.name}" and all associated triggers and invocations.`}
        confirmLabel="Delete"
        variant="destructive"
        isLoading={deleteFunction.isPending}
        onConfirm={async () => {
          await deleteFunction.mutateAsync({ functionId: fnId, projectId });
          router.push(`/projects/${projectId}/functions`);
        }}
      />
    </div>
  );
}

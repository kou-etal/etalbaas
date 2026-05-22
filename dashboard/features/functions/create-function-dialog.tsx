"use client";

import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { useCreateFunction } from "./hooks";

interface CreateFunctionDialogProps {
  projectId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CreateFunctionDialog({ projectId, open, onOpenChange }: CreateFunctionDialogProps) {
  const [name, setName] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [kind, setKind] = useState("heavy-job");
  const [mode, setMode] = useState("sync");
  const [runtime, setRuntime] = useState("python3.11");
  const createFunction = useCreateFunction();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await createFunction.mutateAsync({
      projectId,
      name,
      kind,
      mode,
    });
    resetForm();
    onOpenChange(false);
  };

  const resetForm = () => {
    setName("");
    setDisplayName("");
    setKind("heavy-job");
    setMode("sync");
    setRuntime("python3.11");
  };

  const handleOpenChange = (v: boolean) => {
    if (!v) resetForm();
    onOpenChange(v);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[560px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Deploy Function</DialogTitle>
            <DialogDescription>
              Create a new serverless function. You can configure triggers and environment variables later.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-5 py-4">
            {/* Function name */}
            <div className="space-y-2">
              <Label htmlFor="fn-name">Function name</Label>
              <Input
                id="fn-name"
                placeholder="process-upload"
                value={name}
                onChange={(e) => setName(e.target.value)}
                maxLength={40}
                required
              />
              <p className="text-xs text-muted-foreground">
                Lowercase letters, numbers, and hyphens. 3-40 characters. Used in URLs and SDK references.
              </p>
            </div>

            {/* Display name */}
            <div className="space-y-2">
              <Label htmlFor="fn-display">
                Display name <span className="text-muted-foreground font-normal">— optional</span>
              </Label>
              <Input
                id="fn-display"
                placeholder="Process Upload"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                maxLength={100}
              />
            </div>

            <Separator />

            {/* Kind */}
            <div className="space-y-3">
              <Label>Kind</Label>
              <RadioGroup value={kind} onValueChange={setKind} className="space-y-2">
                <label className="flex items-start gap-3 rounded-lg border p-3 cursor-pointer hover:bg-muted/50 transition-colors">
                  <RadioGroupItem value="heavy-job" className="mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Heavy Job</p>
                    <p className="text-xs text-muted-foreground">
                      One-off execution. Scales to zero. Best for batch processing, data pipelines.
                    </p>
                  </div>
                </label>
                <label className="flex items-start gap-3 rounded-lg border p-3 cursor-pointer hover:bg-muted/50 transition-colors">
                  <RadioGroupItem value="heavy-deployment" className="mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Heavy Deployment</p>
                    <p className="text-xs text-muted-foreground">
                      Long-running service. Custom container. Best for GPU workloads, ML inference.
                    </p>
                  </div>
                </label>
                <label className="flex items-start gap-3 rounded-lg border p-3 cursor-pointer hover:bg-muted/50 transition-colors">
                  <RadioGroupItem value="light-deployment" className="mt-0.5" />
                  <div>
                    <p className="text-sm font-medium">Light Deployment</p>
                    <p className="text-xs text-muted-foreground">
                      Fast cold start. 30s max timeout. Best for webhooks, API handlers.
                    </p>
                  </div>
                </label>
              </RadioGroup>
            </div>

            {/* Mode */}
            <div className="space-y-3">
              <Label>Mode</Label>
              <RadioGroup value={mode} onValueChange={setMode} className="flex gap-4">
                <label className="flex items-center gap-2 text-sm cursor-pointer">
                  <RadioGroupItem value="sync" />
                  Sync
                </label>
                <label className="flex items-center gap-2 text-sm cursor-pointer">
                  <RadioGroupItem value="async" />
                  Async
                </label>
                <label className="flex items-center gap-2 text-sm cursor-pointer">
                  <RadioGroupItem value="stream" />
                  Stream
                </label>
              </RadioGroup>
            </div>

            {/* Runtime */}
            <div className="space-y-2">
              <Label>Runtime</Label>
              <Select value={runtime} onValueChange={setRuntime}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="python3.11">Python 3.11</SelectItem>
                  <SelectItem value="node20">Node.js 20</SelectItem>
                  <SelectItem value="go1.22">Go 1.22</SelectItem>
                  <SelectItem value="custom">Custom Dockerfile</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={!name || createFunction.isPending}>
              {createFunction.isPending ? "Deploying…" : "Deploy"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

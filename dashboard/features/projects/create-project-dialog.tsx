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
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Checkbox } from "@/components/ui/checkbox";
import { Separator } from "@/components/ui/separator";
import { useCreateProject } from "./hooks";

interface CreateProjectDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CreateProjectDialog({ open, onOpenChange }: CreateProjectDialogProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [postgresEnabled, setPostgresEnabled] = useState(true);
  const [pgvector, setPgvector] = useState(false);
  const [pgcrypto, setPgcrypto] = useState(false);
  const [redisEnabled, setRedisEnabled] = useState(false);
  const [postgrestEnabled, setPostgrestEnabled] = useState(true);
  const createProject = useCreateProject();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const extensions: string[] = [];
    if (pgvector) extensions.push("pgvector");
    if (pgcrypto) extensions.push("pgcrypto");

    await createProject.mutateAsync({
      displayName: name,
      description,
      postgresEnabled,
      postgresExtensions: extensions,
      redisEnabled,
      postgrestEnabled: postgresEnabled ? postgrestEnabled : false,
    });
    resetForm();
    onOpenChange(false);
  };

  const resetForm = () => {
    setName("");
    setDescription("");
    setPostgresEnabled(true);
    setPgvector(false);
    setPgcrypto(false);
    setRedisEnabled(false);
    setPostgrestEnabled(true);
  };

  const handleOpenChange = (v: boolean) => {
    if (!v) resetForm();
    onOpenChange(v);
  };

  // When PG is disabled, disable PostgREST
  const handlePostgresToggle = (checked: boolean) => {
    setPostgresEnabled(checked);
    if (!checked) {
      setPostgrestEnabled(false);
      setPgvector(false);
      setPgcrypto(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[560px] max-h-[85vh] overflow-y-auto">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Create a new project</DialogTitle>
            <DialogDescription>
              Configure your project&apos;s name and services. You can change these later.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-5 py-4">
            {/* Project name */}
            <div className="space-y-2">
              <Label htmlFor="project-name">Project name</Label>
              <Input
                id="project-name"
                placeholder="my-project"
                value={name}
                onChange={(e) => setName(e.target.value)}
                maxLength={100}
                required
              />
              <p className="text-xs text-muted-foreground">
                Your project will be available at{" "}
                <span className="font-mono">
                  https://<span className="text-foreground">________</span>.etalbaas.io
                </span>
              </p>
              <p className="text-xs text-muted-foreground">
                An 8-character ID will be assigned automatically.
              </p>
            </div>

            {/* Description */}
            <div className="space-y-2">
              <div className="flex items-baseline justify-between">
                <Label htmlFor="project-desc">
                  Description <span className="text-muted-foreground font-normal">— optional</span>
                </Label>
                <span className="text-xs text-muted-foreground">
                  {description.length} / 500
                </span>
              </div>
              <Textarea
                id="project-desc"
                placeholder="What is this project for?"
                value={description}
                onChange={(e) => setDescription(e.target.value.slice(0, 500))}
                maxLength={500}
                rows={3}
              />
            </div>

            <Separator />

            {/* Enable services */}
            <div className="space-y-4">
              <h4 className="text-sm font-medium">Enable services</h4>
              <p className="text-xs text-muted-foreground -mt-2">
                Choose which infrastructure to provision.
              </p>

              {/* PostgreSQL */}
              <div className="rounded-lg border p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <Switch
                      checked={postgresEnabled}
                      onCheckedChange={handlePostgresToggle}
                    />
                    <div>
                      <span className="text-sm font-medium">PostgreSQL</span>
                      <span className="ml-2 text-xs text-muted-foreground">v16</span>
                    </div>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground ml-[56px]">
                  Managed database with automatic backups.
                </p>
                {postgresEnabled && (
                  <div className="ml-[56px] space-y-2">
                    <p className="text-xs text-muted-foreground">
                      Extensions <span className="text-muted-foreground/60">(optional)</span>
                    </p>
                    <div className="space-y-2">
                      <label className="flex items-center gap-2 text-sm cursor-pointer">
                        <Checkbox
                          checked={pgvector}
                          onCheckedChange={(v) => setPgvector(v === true)}
                        />
                        <div>
                          <span className="font-medium">pgvector</span>
                          <p className="text-xs text-muted-foreground">
                            Vector similarity search for AI / ML workloads
                          </p>
                        </div>
                      </label>
                      <label className="flex items-center gap-2 text-sm cursor-pointer">
                        <Checkbox
                          checked={pgcrypto}
                          onCheckedChange={(v) => setPgcrypto(v === true)}
                        />
                        <div>
                          <span className="font-medium">pgcrypto</span>
                          <p className="text-xs text-muted-foreground">
                            Cryptographic functions
                          </p>
                        </div>
                      </label>
                    </div>
                  </div>
                )}
              </div>

              {/* Redis */}
              <div className="rounded-lg border p-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <Switch
                      checked={redisEnabled}
                      onCheckedChange={setRedisEnabled}
                    />
                    <div>
                      <span className="text-sm font-medium">Redis</span>
                      <span className="ml-2 text-xs text-muted-foreground">v7.2</span>
                    </div>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground ml-[56px] mt-1">
                  In-memory cache for high-speed data access.
                </p>
              </div>

              {/* PostgREST */}
              <div className="rounded-lg border p-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <Switch
                      checked={postgrestEnabled}
                      onCheckedChange={setPostgrestEnabled}
                      disabled={!postgresEnabled}
                    />
                    <div>
                      <span className="text-sm font-medium">PostgREST</span>
                      <span className="ml-2 text-xs text-muted-foreground">v12</span>
                    </div>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground ml-[56px] mt-1">
                  Auto-generated REST API from your database schema.
                </p>
                {!postgresEnabled && (
                  <p className="text-xs text-muted-foreground ml-[56px] mt-1 italic">
                    Requires PostgreSQL
                  </p>
                )}
              </div>
            </div>
          </div>

          <DialogFooter className="flex items-center justify-between sm:justify-between">
            <span className="text-xs text-muted-foreground">Free plan</span>
            <div className="flex gap-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => handleOpenChange(false)}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={!name || createProject.isPending}>
                {createProject.isPending ? "Creating…" : "Create Project"}
              </Button>
            </div>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

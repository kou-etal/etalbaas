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

interface TypeToConfirmDialogProps {
  open: boolean;
  onOpenChange: (_open: boolean) => void;
  title: string;
  description: React.ReactNode;
  confirmText: string;
  confirmLabel?: string;
  variant?: "default" | "destructive";
  onConfirm: () => void;
  isLoading?: boolean;
  children?: React.ReactNode;
}

export function TypeToConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmText,
  confirmLabel = "Delete",
  variant = "destructive",
  onConfirm,
  isLoading,
  children,
}: TypeToConfirmDialogProps) {
  const [typed, setTyped] = useState("");

  const handleOpenChange = (v: boolean) => {
    if (!v) setTyped("");
    onOpenChange(v);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription asChild>
            <div>{description}</div>
          </DialogDescription>
        </DialogHeader>
        {children}
        <div className="space-y-2">
          <Label>
            Type <span className="font-mono font-semibold">{confirmText}</span> to confirm
          </Label>
          <Input
            value={typed}
            onChange={(e) => setTyped(e.target.value)}
            placeholder={confirmText}
          />
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => handleOpenChange(false)}
            disabled={isLoading}
          >
            Cancel
          </Button>
          <Button
            variant={variant === "destructive" ? "destructive" : "default"}
            onClick={onConfirm}
            disabled={typed !== confirmText || isLoading}
          >
            {isLoading ? "Processing..." : confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

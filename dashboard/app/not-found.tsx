import Link from "next/link";
import { FileQuestion } from "lucide-react";
import { Button } from "@/components/ui/button";

export default function NotFound() {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center space-y-4">
        <div className="flex justify-center">
          <div className="rounded-full bg-muted p-4">
            <FileQuestion className="h-8 w-8 text-muted-foreground" />
          </div>
        </div>
        <h2 className="text-lg font-semibold">Page not found</h2>
        <p className="text-sm text-muted-foreground">
          The page you&apos;re looking for doesn&apos;t exist.
        </p>
        <Button asChild>
          <Link href="/projects">Go to Projects</Link>
        </Button>
      </div>
    </div>
  );
}

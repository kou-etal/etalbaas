import { Sidebar } from "@/components/layout/sidebar";
import { Topbar } from "@/components/layout/topbar";
import { BackgroundScene } from "@/components/layout/background-scene";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen">
      <BackgroundScene />
      <Sidebar />
      <div className="pl-16">
        <Topbar />
        <main className="min-h-[calc(100vh-3.5rem)]">{children}</main>
      </div>
    </div>
  );
}

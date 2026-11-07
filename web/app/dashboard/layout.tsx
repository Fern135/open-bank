import type { Metadata } from "next";
import { Suspense } from "react";
import DashboardShell from "./DashboardShell";

export const metadata: Metadata = {
  title: "Dashboard",
  robots: { index: false },
};

export default function DashboardLayout({ children }: LayoutProps<"/dashboard">) {
  return (
    <Suspense>
      <DashboardShell>{children}</DashboardShell>
    </Suspense>
  );
}

import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar';
import { Sidebar } from '@/components/sidebar';
import { Outlet } from 'react-router-dom';
import { Toaster } from '@/components/ui/sonner';

export default function Layout() {
  return (
    <SidebarProvider>
      <Sidebar />
      <SidebarInset>
        <main className="relative w-full flex-1">
          <Outlet />
        </main>
        <Toaster />
      </SidebarInset>
    </SidebarProvider>
  );
}

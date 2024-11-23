import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar';
import { Sidebar } from '@/components/sidebar';
import { Outlet } from 'react-router-dom';
import { Header } from '@/components/header';
import { Toaster } from '@/components/ui/sonner';

export default function Layout() {
  return (
    <SidebarProvider>
      <Sidebar />
      <SidebarInset>
        <main className="relative w-full flex-1">
          <Header className="absolute left-0 right-0 top-0 z-10" />
          <div className="flex h-screen w-full">
            <Outlet />
          </div>
        </main>
        <Toaster />
      </SidebarInset>
    </SidebarProvider>
  );
}

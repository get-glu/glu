import { SidebarProvider } from '@/components/ui/sidebar';
import { Sidebar } from '@/components/sidebar';
import { ThemeToggle } from '@/components/theme-toggle';
import { Outlet } from 'react-router-dom';
import { Header } from '@/components/header';

export default function Layout() {
  return (
    <SidebarProvider>
      <Sidebar />
      <main className="relative w-full flex-1">
        <Header />
        <div className="flex h-screen w-full">
          <Outlet />
        </div>
      </main>
    </SidebarProvider>
  );
}

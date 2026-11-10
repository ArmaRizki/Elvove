import { useEffect, useRef, useState } from "react";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import {
  LayoutDashboard,
  Image,
  FolderOpen,
  Package,
  CalendarCheck,
  Info,
  Phone,
  Share2,
  FileText,
  LogOut,
  Menu,
  X,
  Mail,
} from "lucide-react";

const navItems = [
  { to: "/admin/dashboard", label: "Dasbor", icon: LayoutDashboard },
  { to: "/admin/photos", label: "Foto", icon: Image },
  { to: "/admin/categories", label: "Kategori", icon: FolderOpen },
  { to: "/admin/packages", label: "Paket", icon: Package },
  { to: "/admin/bookings", label: "Pemesanan", icon: CalendarCheck },
  { to: "/admin/about", label: "Tentang Kami", icon: Info },
  { to: "/admin/contact", label: "Informasi Kontak", icon: Phone },
  { to: "/admin/social-links", label: "Tautan Media Sosial", icon: Share2 },
  { to: "/admin/terms", label: "Syarat dan Ketentuan", icon: FileText },
  { to: "/admin/smtp", label: "Pengaturan Email", icon: Mail },
];

function SidebarContent({ admin, onLogout, onNavigate }) {
  return (
    <div className="flex min-h-full flex-col px-4 py-6">
      <p className="mb-6 px-2 font-display text-lg text-gray-900">ELVOVE</p>

      <nav className="flex-1 space-y-1">
        {navItems.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            onClick={onNavigate}
            className={({ isActive }) =>
              `flex items-center gap-3 rounded px-3 py-2.5 text-sm transition-colors ${
                isActive
                  ? "bg-white text-[#E11D48]"
                  : "text-gray-700 hover:bg-gray-50 hover:text-gray-900"
              }`
            }
          >
            <Icon size={16} />
            {label}
          </NavLink>
        ))}
      </nav>

      <div className="mt-4 border-t border-gray-200 px-2 pt-4">
        <p className="mb-3 truncate text-xs text-gray-700">{admin?.username}</p>

        <button
          onClick={onLogout}
          className="flex items-center gap-2 text-sm text-gray-700 hover:text-gray-900"
        >
          <LogOut size={16} />
          Keluar
        </button>
      </div>
    </div>
  );
}

export default function AdminLayout() {
  const { admin, logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [desktopOpen, setDesktopOpen] = useState(true);
  const [mobileOpen, setMobileOpen] = useState(false);
  const closeButtonRef = useRef(null);

  const handleLogout = () => {
    logout();
    navigate("/admin/login");
  };

  // Tapping a navigation item closes the mobile drawer.
  useEffect(() => {
    setMobileOpen(false);
  }, [location.pathname]);

  // Drawer: lock page scroll, close on Escape, move focus into the drawer.
  useEffect(() => {
    if (!mobileOpen) return undefined;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    const onKeyDown = (event) => {
      if (event.key === "Escape") setMobileOpen(false);
    };
    window.addEventListener("keydown", onKeyDown);
    closeButtonRef.current?.focus();
    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener("keydown", onKeyDown);
    };
  }, [mobileOpen]);

  return (
    <div className="min-h-screen overflow-x-clip bg-white">
      <div className="flex">
        {/* Desktop sidebar: fixed to the viewport so page scrolling never
            moves it. It scrolls on its own when the menu is taller than the
            screen. Collapsing slides it out and removes it from the tab order. */}
        <aside
          aria-label="Navigasi admin"
          inert={!desktopOpen}
          className={`fixed top-0 left-0 z-40 hidden h-screen w-64 overflow-y-auto overscroll-contain border-r border-gray-200 bg-white transition-transform duration-300 lg:block ${
            desktopOpen ? "translate-x-0" : "-translate-x-full"
          }`}
        >
          <SidebarContent admin={admin} onLogout={handleLogout} />
        </aside>

        {/* Mobile drawer */}
        {mobileOpen && (
          <div className="fixed inset-0 z-50 lg:hidden">
            <button
              type="button"
              aria-label="Tutup menu"
              className="absolute inset-0 h-full w-full bg-black/40"
              onClick={() => setMobileOpen(false)}
            />
            <aside
              role="dialog"
              aria-modal="true"
              aria-label="Navigasi admin"
              className="absolute inset-y-0 left-0 w-72 max-w-[85vw] border-r border-gray-200 bg-white shadow-xl"
            >
              <button
                ref={closeButtonRef}
                type="button"
                onClick={() => setMobileOpen(false)}
                className="absolute right-3 top-4 p-2 text-gray-700 hover:bg-gray-100"
                aria-label="Tutup menu"
              >
                <X size={20} />
              </button>
              <SidebarContent
                admin={admin}
                onLogout={handleLogout}
                onNavigate={() => setMobileOpen(false)}
              />
            </aside>
          </div>
        )}

        {/* Offset the content by the fixed sidebar width; the padding follows
            the collapse toggle so no space is left behind. */}
        <div
          className={`min-w-0 flex-1 transition-[padding] duration-300 ${
            desktopOpen ? "lg:pl-64" : "lg:pl-0"
          }`}
        >
          <header className="sticky top-0 z-30 flex h-14 items-center gap-2 border-b border-gray-200 bg-white px-4 sm:px-6">
            <button
              type="button"
              onClick={() => setMobileOpen(true)}
              className="inline-flex h-10 w-10 items-center justify-center rounded text-gray-700 transition-colors hover:bg-gray-100 lg:hidden"
              aria-label="Buka menu"
              aria-expanded={mobileOpen}
            >
              <Menu size={20} />
            </button>

            <button
              type="button"
              onClick={() => setDesktopOpen((open) => !open)}
              className="hidden h-9 w-9 items-center justify-center rounded text-gray-700 transition-colors hover:bg-gray-100 lg:inline-flex"
              title={desktopOpen ? "Sembunyikan bilah samping" : "Tampilkan bilah samping"}
              aria-label={desktopOpen ? "Sembunyikan bilah samping" : "Tampilkan bilah samping"}
            >
              {desktopOpen ? <X size={20} /> : <Menu size={20} />}
            </button>

            <span className="font-display text-sm text-gray-900 lg:hidden">ELVOVE Admin</span>

            <span className="ml-auto max-w-[45%] truncate text-xs text-gray-500">
              {admin?.username}
            </span>
          </header>

          <main className="min-w-0 overflow-x-hidden p-4 sm:p-6 lg:p-8">
            <Outlet />
          </main>
        </div>
      </div>
    </div>
  );
}

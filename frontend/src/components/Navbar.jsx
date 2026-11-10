import { useState, useEffect } from "react";
import { Link, NavLink } from "react-router-dom";
import { Menu, X } from "lucide-react";

const links = [
  { to: "/", label: "Beranda" },
  { to: "/portfolio", label: "Portofolio" },
  { to: "/packages", label: "Paket" },
  { to: "/contact", label: "Kontak" },
  { to: "/about", label: "Tentang" },
];

export default function Navbar() {
  const [open, setOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 24);
    window.addEventListener("scroll", onScroll);
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  useEffect(() => {
    if (!open) return;
    const onKeyDown = (e) => {
      if (e.key === "Escape") setOpen(false);
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [open]);

  return (
    <header
      className={`fixed top-0 inset-x-0 z-50 transition-colors duration-300 ${
        scrolled || open ? "bg-white/95 backdrop-blur-md border-b border-gray-200" : "bg-transparent"
      }`}
    >
      <div className="max-w-7xl mx-auto flex items-center justify-between gap-4 px-4 sm:px-6 py-4">
        <Link
          to="/"
          onClick={() => setOpen(false)}
          className="flex min-w-0 items-center gap-2 font-display text-xl tracking-wide text-black"
        >
          <img src="/logo.png" alt="Logo ELVOVE" className="h-7 w-7 shrink-0 object-contain" />
          <span className="text-[#E11D48]">ELVOVE</span>
        </Link>

        <nav aria-label="Navigasi utama" className="hidden lg:flex items-center gap-8">
          {links.map((l) => (
            <NavLink
              key={l.to}
              to={l.to}
              className={({ isActive }) =>
                `text-sm tracking-wide transition-colors ${
                  isActive ? "text-[#E11D48]" : "text-black hover:text-[#E11D48]"
                }`
              }
            >
              {l.label}
            </NavLink>
          ))}
        </nav>

        <Link
          to="/booking"
          className="hidden lg:inline-block border border-[#E11D48] text-[#E11D48] px-5 py-2 text-sm tracking-wide hover:bg-[#E11D48] hover:text-white transition-colors"
        >
          Pesan Sekarang
        </Link>

        <button
          type="button"
          className="lg:hidden -mr-2 p-2 rounded-md bg-white/90 text-black shadow-sm"
          onClick={() => setOpen((v) => !v)}
          aria-expanded={open}
          aria-controls="menu-mobile"
          aria-label={open ? "Tutup menu" : "Buka menu"}
        >
          {open ? <X size={24} aria-hidden="true" /> : <Menu size={24} aria-hidden="true" />}
        </button>
      </div>

      {open && (
        <nav
          id="menu-mobile"
          aria-label="Navigasi mobile"
          className="lg:hidden bg-white border-t border-gray-200 px-4 sm:px-6 py-4 flex flex-col gap-1 max-h-[calc(100vh-4.5rem)] overflow-y-auto"
        >
          {links.map((l) => (
            <NavLink
              key={l.to}
              to={l.to}
              onClick={() => setOpen(false)}
              className={({ isActive }) =>
                `py-3 text-base border-b border-gray-100 ${isActive ? "text-[#E11D48]" : "text-black"}`
              }
            >
              {l.label}
            </NavLink>
          ))}
          <Link
            to="/booking"
            onClick={() => setOpen(false)}
            className="mt-4 border border-[#E11D48] bg-[#E11D48] text-white px-5 py-3 text-sm tracking-wide text-center"
          >
            Pesan Sekarang
          </Link>
        </nav>
      )}
    </header>
  );
}

import { Link } from "react-router-dom";
import SectionHeading from "../components/SectionHeading";

/**
 * The page a visitor lands on when the address does not exist.
 *
 * It answers 200, because the site is a single page served by nginx as
 * index.html; making the status a real 404 needs nginx to answer it with
 * `error_page 404`, which is a separate change. The wording here avoids saying
 * "tidak ditemukan" about the status and only describes the address.
 */
export default function PageNotFound() {
  return (
    <div className="pt-28 sm:pt-32 pb-20 sm:pb-24 max-w-3xl mx-auto px-5 sm:px-6">
      <p className="text-xs tracking-[0.3em] text-[#E11D48]">404</p>
      <SectionHeading as="h1" title="Halaman tidak ada" />
      <div className="space-y-4 text-gray-700 text-sm leading-relaxed">
        <p>
          Alamat yang Anda buka tidak ada di situs ini. Mungkin tautannya sudah
          berubah, atau salah ketik.
        </p>
        <p>
          Foto pilihan klien dibuka dari tautan di email Elvove, jadi tautan itu
          tidak bisa dicari dari halaman ini.
        </p>
      </div>
      <div className="mt-8 flex flex-wrap gap-3">
        <Link
          to="/"
          className="inline-flex items-center gap-2 bg-[#111111] px-5 py-3 text-xs tracking-[0.2em] text-white transition-colors hover:bg-[#E11D48]"
        >
          KE BERANDA
        </Link>
        <Link
          to="/contact"
          className="inline-flex items-center gap-2 border border-gray-300 px-5 py-3 text-xs tracking-[0.2em] text-gray-900 transition-colors hover:border-[#E11D48] hover:text-[#E11D48]"
        >
          HUBUNGI KAMI
        </Link>
      </div>
    </div>
  );
}

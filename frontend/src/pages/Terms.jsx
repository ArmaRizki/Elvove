import SectionHeading from "../components/SectionHeading";

export default function Terms() {
  return (
    <div className="pt-28 sm:pt-32 pb-20 sm:pb-24 max-w-3xl mx-auto px-5 sm:px-6">
      <SectionHeading as="h1" title="Ketentuan Layanan" />
      <div className="space-y-4 text-gray-700 text-sm leading-relaxed">
        <p>
          Pemesanan sesi memerlukan pembayaran uang muka untuk mengonfirmasi
          tanggal Anda, sesuai kesepakatan pada saat pemesanan. Ketentuan
          pembayaran lengkap akan dibagikan sebelum acara Anda.
        </p>
        <p>
          Foto dan video yang telah diedit diserahkan dalam jangka waktu yang
          tercantum pada paket Anda. Pengiriman kilat mungkin tersedia dengan
          biaya tambahan.
        </p>
        <p>
          Pembatalan yang dilakukan dalam 7 hari sebelum tanggal acara mungkin
          tidak memenuhi syarat untuk pengembalian uang muka.
        </p>
      </div>
    </div>
  );
}

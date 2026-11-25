import { useEffect, useState } from "react";
import { Image, CalendarCheck, Package, Copy, Check, RotateCw } from "lucide-react";
import { adminApi } from "../../services/adminApi";
import LoadingState from "../../components/LoadingState";
import AdminPageHeader from "../../components/admin/AdminPageHeader";
import BookingCalendar from "../../components/admin/BookingCalendar";
import { adminInput, adminSecondaryButton } from "../../components/admin/adminStyles";

const cards = [
  { key: "totalPhotos", label: "Total Foto", icon: Image },
  { key: "totalBookings", label: "Total Pemesanan", icon: CalendarCheck },
  { key: "totalPackages", label: "Total Paket", icon: Package },
];

export default function Dashboard() {
  const [summary, setSummary] = useState(null);
  const [error, setError] = useState("");

  useEffect(() => {
    adminApi
      .getDashboardSummary()
      .then((res) => setSummary(res.data))
      .catch(() => setError("Statistik dasbor gagal dimuat."));
  }, []);

  return (
    <div className="min-w-0">
      <AdminPageHeader
        title="Dasbor"
        description="Ringkasan portofolio dan seluruh pemesanan."
      />

      {error && (
        <p className="mb-6 border border-red-200 bg-red-50 p-3 text-sm text-red-700" role="alert">
          {error}
        </p>
      )}

      {!summary && !error ? (
        <LoadingState label="Memuat dasbor" />
      ) : (
        summary && (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            {cards.map(({ key, label, icon: Icon }) => (
              <div
                key={key}
                className="flex items-center gap-4 border border-gray-200 p-4 sm:p-6"
              >
                <div className="bg-white p-2 text-[#E11D48] sm:p-3">
                  <Icon size={22} />
                </div>
                <div className="min-w-0">
                  <p className="font-display text-2xl text-gray-900 sm:text-3xl">
                    {summary[key]}
                  </p>
                  <p className="mt-1 text-[10px] tracking-widest text-gray-700 sm:text-xs">
                    {label.toUpperCase()}
                  </p>
                </div>
              </div>
            ))}
          </div>
        )
      )}

      <CalendarFeedCard />

      <section className="mt-8 min-w-0" aria-labelledby="dashboard-calendar-heading">
        <div className="mb-4">
          <h2 id="dashboard-calendar-heading" className="font-display text-xl text-gray-900">
            Kalender Pemesanan
          </h2>
          <p className="mt-1 text-sm text-gray-600">
            Menampilkan semua pemesanan yang memiliki tanggal acara, termasuk yang menunggu,
            meminta ubah jadwal, diterima, dan selesai. Pilih pemesanan untuk melihat detail;
            jadwal hanya dapat diubah selama pemesanan belum diterima.
          </p>
        </div>
        <BookingCalendar />
      </section>
    </div>
  );
}

// The address a calendar app subscribes to. It doubles as a key: whoever holds
// it can read the client names, hours and places in the feed, so the panel says
// so plainly and offers a way to invalidate it.
function CalendarFeedCard() {
  const [url, setUrl] = useState("");
  const [error, setError] = useState("");
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    adminApi
      .getCalendarFeed()
      .then((res) => setUrl(res.data.url))
      .catch(() => setError("Alamat kalender gagal dimuat."));
  }, []);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      setError("Tidak bisa menyalin otomatis. Pilih alamatnya lalu salin manual.");
    }
  };

  const rotate = async () => {
    const sure = window.confirm(
      "Buat alamat baru? Alamat lama langsung berhenti bekerja, dan kalender yang sudah berlangganan harus dipasang ulang.",
    );
    if (!sure) return;
    try {
      const res = await adminApi.rotateCalendarFeed();
      setUrl(res.data.url);
      setCopied(false);
      setError("");
    } catch {
      setError("Alamat kalender gagal dibuat ulang.");
    }
  };

  return (
    <section className="mt-8 min-w-0" aria-labelledby="dashboard-feed-heading">
      <div className="border border-gray-200 p-4 sm:p-6">
        <h2 id="dashboard-feed-heading" className="font-display text-xl text-gray-900">
          Langganan Kalender
        </h2>
        <p className="mt-1 text-sm text-gray-600">
          Tempel alamat ini sekali di Google Calendar (Other calendars → From URL), Apple
          Calendar, atau Outlook. Pemesanan muncul sendiri dan ikut berubah saat jadwalnya
          diubah. Alamat ini berlaku seperti kunci — siapa pun yang memilikinya dapat melihat
          nama dan jadwal klien, jadi jangan dibagikan.
        </p>
        <div className="mt-4 flex flex-col gap-2 sm:flex-row">
          <input
            readOnly
            value={url}
            onFocus={(event) => event.target.select()}
            aria-label="Alamat langganan kalender"
            placeholder="Memuat alamat…"
            className={`${adminInput} min-w-0 sm:flex-1`}
          />
          <div className="flex shrink-0 gap-2">
            <button
              type="button"
              onClick={copy}
              disabled={!url}
              className={`${adminSecondaryButton} inline-flex items-center gap-2`}
            >
              {copied ? <Check size={15} /> : <Copy size={15} />}
              {copied ? "Tersalin" : "Salin"}
            </button>
            <button
              type="button"
              onClick={rotate}
              className={`${adminSecondaryButton} inline-flex items-center gap-2`}
            >
              <RotateCw size={15} /> Buat ulang
            </button>
          </div>
        </div>
        {error && (
          <p role="alert" className="mt-2 text-sm text-burgundy">
            {error}
          </p>
        )}
      </div>
    </section>
  );
}

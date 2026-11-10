import { useEffect, useState } from "react";
import { Image, CalendarCheck, Package } from "lucide-react";
import { adminApi } from "../../services/adminApi";
import LoadingState from "../../components/LoadingState";
import AdminPageHeader from "../../components/admin/AdminPageHeader";
import BookingCalendar from "../../components/admin/BookingCalendar";

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

      <section className="mt-8 min-w-0" aria-labelledby="dashboard-calendar-heading">
        <div className="mb-4">
          <h2 id="dashboard-calendar-heading" className="font-display text-xl text-gray-900">
            Kalender Pemesanan
          </h2>
          <p className="mt-1 text-sm text-gray-600">
            Menampilkan semua pemesanan yang memiliki tanggal acara, termasuk yang menunggu,
            meminta ubah jadwal, diterima, dan selesai. Pilih pemesanan untuk melihat detail
            atau mengubah jadwal.
          </p>
        </div>
        <BookingCalendar />
      </section>
    </div>
  );
}

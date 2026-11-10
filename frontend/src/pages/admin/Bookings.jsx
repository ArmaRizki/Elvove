import { useCallback, useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { LoaderCircle, Trash2, X } from "lucide-react";
import { adminApi } from "../../services/adminApi";
import LoadingState from "../../components/LoadingState";
import EmptyState from "../../components/EmptyState";
import AdminPageHeader from "../../components/admin/AdminPageHeader";
import BookingDetails from "../../components/admin/BookingDetails";
import BookingConfirmationEmail from "../../components/admin/BookingConfirmationEmail";
import {
  adminLabel,
  adminTableWide,
  adminTableWrap,
} from "../../components/admin/adminStyles";
import {
  STATUS_OPTIONS,
  formatDate,
  formatPrice,
  formatTime,
  statusBadge,
  statusLabel,
} from "../../components/admin/adminFormat";

function StatusBadge({ status }) {
  return (
    <span className={`inline-flex whitespace-nowrap border px-2 py-1 text-xs ${statusBadge(status)}`}>
      {statusLabel(status)}
    </span>
  );
}

export default function Bookings() {
  const [bookings, setBookings] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selected, setSelected] = useState(null);
  const [statusSaving, setStatusSaving] = useState(false);
  const [notice, setNotice] = useState({ type: "", text: "" });
  const [searchParams, setSearchParams] = useSearchParams();
  const [deepLinkMessage, setDeepLinkMessage] = useState("");

  const load = async () => {
    setLoading(true);
    try {
      const { data } = await adminApi.getBookings();
      setBookings(Array.isArray(data) ? data : []);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const openDetail = (booking) => {
    setSelected(booking);
    setNotice({ type: "", text: "" });
  };

  const openBookingById = useCallback((id) => {
    const match = bookings.find((item) => String(item.id) === String(id));
    if (match) {
      setDeepLinkMessage("");
      setSelected(match);
      setNotice({ type: "", text: "" });
      return true;
    }
    setDeepLinkMessage("Pemesanan tidak ditemukan. Mungkin sudah dihapus.");
    return false;
  }, [bookings]);

  // Opens the booking named in ?booking=ID. Only IDs that are digits are used,
  // and the booking must exist in the list the admin is already allowed to load.
  const bookingParam = searchParams.get("booking");
  useEffect(() => {
    if (loading || !bookingParam) return;
    if (!/^\d+$/.test(bookingParam)) {
      setDeepLinkMessage("Tautan pemesanan tidak valid.");
    } else {
      openBookingById(bookingParam);
    }
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.delete("booking");
      return next;
    }, { replace: true });
    // The link is handled once per page load, after the bookings have loaded.
  }, [loading, bookingParam, openBookingById, setSearchParams]);

  const closeDetail = () => {
    if (statusSaving) return;
    setSelected(null);
    setNotice({ type: "", text: "" });
  };

  const changeStatus = async (nextStatus) => {
    if (!selected || nextStatus === selected.status || statusSaving) return;

    setStatusSaving(true);
    setNotice({ type: "", text: "" });

    try {
      let status = nextStatus;
      if (nextStatus === "ACCEPTED") {
        const { data } = await adminApi.acceptBooking(selected.id);
        status = data?.status || nextStatus;
      } else {
        const { data } = await adminApi.updateBookingStatus(selected.id, nextStatus);
        status = data?.status || nextStatus;
      }

      setSelected((current) => (current ? { ...current, status } : current));
      setBookings((current) =>
        current.map((item) => (item.id === selected.id ? { ...item, status } : item))
      );
      setNotice({
        type: "success",
        text:
          status === "ACCEPTED"
            ? "Status diubah menjadi Diterima. Email konfirmasi belum dikirim."
            : `Status diubah menjadi ${statusLabel(status)}.`,
      });
    } catch (error) {
      setNotice({
        type: "error",
        text: error.response?.data?.message || "Status pemesanan gagal diperbarui.",
      });
    } finally {
      setStatusSaving(false);
    }
  };

  const handleDelete = async (booking) => {
    if (
      !window.confirm(
        `Hapus pemesanan dari ${booking.name}? Tindakan ini tidak dapat dibatalkan.`
      )
    ) {
      return;
    }

    await adminApi.deleteBooking(booking.id);
    if (selected?.id === booking.id) setSelected(null);
    load();
  };

  return (
    <div className="min-w-0">
      <AdminPageHeader
        title="Pemesanan"
        description="Tinjau permintaan pemesanan, atur status, dan kirim konfirmasi ke klien."
      />

      {deepLinkMessage && (
        <p role="status" className="mb-4 border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900">
          {deepLinkMessage}
        </p>
      )}

      {loading ? (
        <LoadingState label="Memuat pemesanan" />
      ) : bookings.length === 0 ? (
        <EmptyState message="Belum ada pemesanan." />
      ) : (
        <>
          {/* Tabel desktop */}
          <div className={`hidden lg:block ${adminTableWrap}`}>
            <table className={adminTableWide}>
              <thead>
                <tr className="border-b border-gray-200 text-left text-gray-700">
                  <th className="px-4 py-3 font-normal">Nama</th>
                  <th className="px-4 py-3 font-normal">Kontak</th>
                  <th className="px-4 py-3 font-normal">Lokasi</th>
                  <th className="px-4 py-3 font-normal">Paket</th>
                  <th className="px-4 py-3 font-normal">Jadwal</th>
                  <th className="px-4 py-3 font-normal">Status</th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>

              <tbody>
                {bookings.map((b) => (
                  <tr key={b.id} className="border-b border-gray-200 align-top last:border-0">
                    <td className="px-4 py-3 text-gray-900">{b.name}</td>

                    <td className="px-4 py-3 text-gray-700">
                      <p className="break-all">{b.email}</p>
                      <p>{b.phone}</p>
                    </td>

                    <td className="px-4 py-3 text-gray-700">
                      <p className="text-xs text-gray-500">{b.eventLocation || "Belum diisi"}</p>
                    </td>

                    <td className="px-4 py-3 text-gray-700">
                      {b.packageTitle ? (
                        <>
                          <p className="text-gray-900">{b.packageTitle}</p>
                          <p className="mt-1 text-xs text-[#E11D48]">
                            {formatPrice(b.packagePrice) || "Harga belum tercatat"}
                          </p>
                        </>
                      ) : (
                        <span className="text-xs text-gray-500">Belum tercatat</span>
                      )}
                    </td>

                    <td className="px-4 py-3 text-gray-700">
                      <p className="whitespace-nowrap">{formatDate(b.eventDate) || "Belum tercatat"}</p>
                      <p className="mt-1 text-xs text-gray-500">
                        {formatTime(b.eventTime) || "Belum ditentukan"}
                      </p>
                    </td>

                    <td className="px-4 py-3">
                      <StatusBadge status={b.status} />
                    </td>

                    <td className="px-4 py-3 text-right whitespace-nowrap">
                      <button
                        type="button"
                        onClick={() => openDetail(b)}
                        className="mr-3 text-xs text-[#E11D48] hover:underline"
                      >
                        Detail
                      </button>
                      <button
                        type="button"
                        onClick={() => handleDelete(b)}
                        className="text-gray-700 hover:text-burgundy"
                        aria-label={`Hapus pemesanan dari ${b.name}`}
                      >
                        <Trash2 size={15} />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Kartu untuk layar kecil */}
          <div className="space-y-3 lg:hidden">
            {bookings.map((b) => (
              <article key={b.id} className="border border-gray-200 p-4">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="break-words font-medium text-gray-900">{b.name}</p>
                    <p className="break-all text-xs text-gray-500">{b.email}</p>
                    <p className="text-xs text-gray-500">{b.phone}</p>
                  </div>
                  <button
                    type="button"
                    onClick={() => handleDelete(b)}
                    className="shrink-0 p-1 text-gray-700 hover:text-burgundy"
                    aria-label={`Hapus pemesanan dari ${b.name}`}
                  >
                    <Trash2 size={16} />
                  </button>
                </div>

                <dl className="mt-3 grid grid-cols-2 gap-3 text-xs">
                  <div className="min-w-0">
                    <dt className="text-gray-500">Lokasi</dt>
                    <dd className="text-gray-900">{b.eventLocation || "Belum diisi"}</dd>
                  </div>
                  <div className="min-w-0">
                    <dt className="text-gray-500">Jadwal</dt>
                    <dd className="text-gray-900">{formatDate(b.eventDate) || "Belum tercatat"}</dd>
                    <dd className="text-gray-500">{formatTime(b.eventTime) || "Belum ditentukan"}</dd>
                  </div>
                  <div className="col-span-2 min-w-0">
                    <dt className="text-gray-500">Paket</dt>
                    <dd className="text-gray-900">
                      {b.packageTitle || "Belum tercatat"}
                      {b.packagePrice != null && (
                        <span className="ml-2 text-[#E11D48]">{formatPrice(b.packagePrice)}</span>
                      )}
                    </dd>
                  </div>
                </dl>

                <div className="mt-3 flex items-center justify-between gap-3">
                  <StatusBadge status={b.status} />
                  <button
                    type="button"
                    onClick={() => openDetail(b)}
                    className="border border-gray-200 px-3 py-2 text-xs text-gray-700 transition-colors hover:border-gray-300 hover:text-gray-900"
                  >
                    Lihat Detail
                  </button>
                </div>
              </article>
            ))}
          </div>
        </>
      )}

      {selected && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-3 sm:p-4"
          onMouseDown={(event) => {
            if (event.target === event.currentTarget) closeDetail();
          }}
        >
          <div
            role="dialog"
            aria-modal="true"
            aria-label={`Detail pemesanan ${selected.name}`}
            className="max-h-[92vh] w-full max-w-2xl overflow-y-auto bg-white shadow-xl"
          >
            <div className="flex items-start justify-between gap-3 border-b border-gray-200 p-4 sm:p-5">
              <div className="min-w-0">
                <p className="text-xs tracking-widest text-[#E11D48]">DETAIL PEMESANAN</p>
                <h2 className="mt-1 break-words font-display text-xl text-gray-900">
                  {selected.name}
                </h2>
              </div>
              <button
                type="button"
                onClick={closeDetail}
                className="p-2 text-gray-700 hover:bg-gray-100"
                aria-label="Tutup detail pemesanan"
              >
                <X size={18} />
              </button>
            </div>

            <BookingDetails booking={selected} />

            <div className="space-y-4 border-t border-gray-200 p-4 sm:p-5">
              <div>
                <label className={adminLabel} htmlFor="booking-status">
                  UBAH STATUS PEMESANAN
                </label>
                <select
                  id="booking-status"
                  value={selected.status}
                  disabled={statusSaving}
                  onChange={(event) => changeStatus(event.target.value)}
                  className="mt-1 w-full border border-gray-200 bg-transparent px-3 py-2.5 text-sm text-gray-900 focus:border-[#E11D48] outline-none disabled:opacity-60 sm:max-w-xs"
                >
                  {STATUS_OPTIONS.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
                {statusSaving && (
                  <p className="mt-2 flex items-center gap-2 text-xs text-gray-500">
                    <LoaderCircle size={14} className="animate-spin" />
                    Menyimpan status...
                  </p>
                )}
              </div>

              {notice.text && (
                <p
                  className={`border p-3 text-sm ${
                    notice.type === "success"
                      ? "border-green-200 bg-green-50 text-green-800"
                      : "border-red-200 bg-red-50 text-red-700"
                  }`}
                  role="status"
                >
                  {notice.text}
                </p>
              )}

              <BookingConfirmationEmail
                key={selected.id}
                booking={selected}
                onSent={(tracking) => {
                  setSelected((current) => (current ? { ...current, ...tracking } : current));
                  setBookings((current) =>
                    current.map((item) =>
                      item.id === selected.id ? { ...item, ...tracking } : item
                    )
                  );
                }}
              />

              {selected.status !== "ACCEPTED" && (
                <p className="text-xs text-gray-500">
                  Tombol kirim email konfirmasi tersedia setelah status pemesanan diubah menjadi
                  Diterima.
                </p>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

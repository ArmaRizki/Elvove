import { useCallback, useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { LoaderCircle, Trash2 } from "lucide-react";
import { adminApi } from "../../services/adminApi";
import LoadingState from "../../components/LoadingState";
import EmptyState from "../../components/EmptyState";
import AdminPageHeader from "../../components/admin/AdminPageHeader";
import useAdminList from "../../hooks/useAdminList";
import Modal from "../../components/Modal";
import BookingDetails from "../../components/admin/BookingDetails";
import BookingConfirmationEmail from "../../components/admin/BookingConfirmationEmail";
import BookingPayment from "../../components/admin/BookingPayment";
import BookingSelection from "../../components/admin/BookingSelection";
import {
  adminLabel,
  adminTableWide,
  adminTableWrap,
} from "../../components/admin/adminStyles";
import {
  STATUS_OPTIONS,
  formatDate,
  formatDateTime,
  formatTime,
  paymentBadge,
  paymentLabel,
  statusBadge,
  statusLabel,
} from "../../components/admin/adminFormat";
import { formatPrice } from "../../utils/format";

const SORT_OPTIONS = [
  { value: "createdAt", label: "Waktu pemesanan" },
  { value: "eventDate", label: "Tanggal book" },
];

// Newest booking first, nearest event first.
const SORT_DIRECTION = { createdAt: -1, eventDate: 1 };

const collator = new Intl.Collator("id", { numeric: true, sensitivity: "base" });

// Raw values, because the visible dates are dd/mm/yyyy and do not order as
// strings. eventDate ("2026-11-25") and createdAt ("2026-11-25T05:30:00") are
// ISO, so comparing them as text is chronological. A missing date sorts last.
const sortValue = {
  createdAt: (booking) => booking.createdAt || "",
  eventDate: (booking) => booking.eventDate || "",
};

function sortBookings(list, key) {
  const value = sortValue[key];
  const factor = SORT_DIRECTION[key];
  return [...list].sort((a, b) => {
    const left = value(a);
    const right = value(b);
    if (!left && !right) return 0;
    if (!left) return 1;
    if (!right) return -1;
    return factor * collator.compare(left, right);
  });
}

// One order for the desktop table and the phone cards, so the two layouts
// never disagree.
function SortOrder({ order, onChange }) {
  return (
    <label className="mb-3 flex items-center gap-2 text-xs text-gray-600">
      <span>Urutkan</span>
      <select
        value={order}
        onChange={(event) => onChange(event.target.value)}
        className="border border-gray-200 bg-transparent px-2 py-1.5 text-xs text-gray-900 outline-none focus:border-[#E11D48]"
      >
        {SORT_OPTIONS.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </label>
  );
}

function StatusBadge({ status }) {
  return (
    <span className={`inline-flex whitespace-nowrap border px-2 py-1 text-xs ${statusBadge(status)}`}>
      {statusLabel(status)}
    </span>
  );
}

// Mirrors the server's transition table (transitionBooking). An ACCEPTED booking
// only moves on to COMPLETED, so the agreed schedule cannot be reopened; a
// COMPLETED one does not change at all. Offering a status the API refuses would
// only produce an error message.
const NEXT_STATUSES = {
  PENDING: ["ACCEPTED", "RESCHEDULE_REQUESTED"],
  ACCEPTED: ["COMPLETED"],
  RESCHEDULE_REQUESTED: ["ACCEPTED"],
  COMPLETED: [],
};

export default function Bookings() {
  const list = useCallback(
    () => adminApi.getBookings().then(({ data }) => (Array.isArray(data) ? data : [])),
    []
  );
  const { data: loadedBookings, loading, reload: load } = useAdminList(list);
  const [bookings, setBookings] = useState([]);
  const [selected, setSelected] = useState(null);
  const [order, setOrder] = useState("createdAt");
  const [statusSaving, setStatusSaving] = useState(false);
  const [notice, setNotice] = useState({ type: "", text: "" });
  const [searchParams, setSearchParams] = useSearchParams();
  const [deepLinkMessage, setDeepLinkMessage] = useState("");

  const sortedBookings = bookings.length ? sortBookings(bookings, order) : bookings;

  // The list is copied into local state because three actions edit one booking
  // in place — status, payment and e-mail tracking — and the row behind the
  // open dialog has to agree with it before the next load confirms it.
  useEffect(() => {
    if (loadedBookings) setBookings(loadedBookings);
  }, [loadedBookings]);

  // The list row has no photo counts or folder ids, so the popup opens with it
  // and swaps in the full record from the detail endpoint when it arrives.
  const openDetail = useCallback(async (booking) => {
    setSelected(booking);
    setNotice({ type: "", text: "" });
    try {
      const { data } = await adminApi.getBooking(booking.id);
      setSelected((current) => (current?.id === booking.id ? data : current));
    } catch {
      setNotice({ type: "error", text: "Detail pemesanan gagal dimuat. Muat ulang halaman ini." });
    }
  }, []);

  const openBookingById = useCallback((id) => {
    const match = bookings.find((item) => String(item.id) === String(id));
    if (match) {
      setDeepLinkMessage("");
      openDetail(match);
      return true;
    }
    setDeepLinkMessage("Pemesanan tidak ditemukan. Mungkin sudah dihapus.");
    return false;
  }, [bookings, openDetail]);

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

  // Applies a booking patch (payment change or e-mail tracking) to the open
  // dialog and to the list behind it, so the two never disagree.
  const applyBookingUpdate = (update) => {
    if (!update) return;
    const patch = { id: selected?.id, ...update };
    setSelected((current) => (current ? { ...current, ...patch } : current));
    setBookings((current) =>
      current.map((item) => (item.id === patch.id ? { ...item, ...patch } : item))
    );
  };

  const handleDelete = async (booking) => {
    if (
      !window.confirm(
        `Hapus pemesanan dari ${booking.name}?\n\nFolder foto klien di Google Drive ikut dipindahkan ke Trash, jadi masih bisa dipulihkan dari sana selama 30 hari.`
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
          <SortOrder order={order} onChange={setOrder} />

          {/* Tabel desktop */}
          <div
            className={`hidden lg:block ${adminTableWrap}`}
            role="region"
            aria-label="Daftar pemesanan"
            tabIndex={0}
          >
            <table className={adminTableWide}>
              <caption className="sr-only">Daftar pemesanan</caption>
              <thead>
                <tr className="border-b border-gray-200 text-left text-gray-700">
                  <th className="px-4 py-3 font-normal">Nama</th>
                  <th className="px-4 py-3 font-normal">Dipesan</th>
                  <th className="px-4 py-3 font-normal">Kontak</th>
                  <th className="px-4 py-3 font-normal">Lokasi</th>
                  <th className="px-4 py-3 font-normal">Paket</th>
                  <th className="px-4 py-3 font-normal">Jadwal</th>
                  <th className="px-4 py-3 font-normal">Status</th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>

              <tbody>
                {sortedBookings.map((b) => (
                  <tr key={b.id} className="border-b border-gray-200 align-top last:border-0">
                    <td className="px-4 py-3 text-gray-900">{b.name}</td>

                    <td className="whitespace-nowrap px-4 py-3 text-gray-700">
                      {formatDateTime(b.createdAt) || "—"}
                    </td>

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
                      <div className="space-y-1">
                        <StatusBadge status={b.status} />
                        {/* Only the recorded payments get a badge; "Belum bayar"
                            on every row would be noise. */}
                        {b.paymentStatus && b.paymentStatus !== "UNPAID" && (
                          <div>
                            <span
                              className={`inline-flex whitespace-nowrap border px-2 py-1 text-xs ${paymentBadge(b.paymentStatus)}`}
                            >
                              {paymentLabel(b.paymentStatus)}
                            </span>
                          </div>
                        )}
                      </div>
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
            {sortedBookings.map((b) => (
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
                  <div className="flex flex-wrap items-center gap-2">
                    <StatusBadge status={b.status} />
                    {b.paymentStatus && b.paymentStatus !== "UNPAID" && (
                      <span className={`inline-flex border px-2 py-1 text-xs ${paymentBadge(b.paymentStatus)}`}>
                        {paymentLabel(b.paymentStatus)}
                      </span>
                    )}
                  </div>
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

      <Modal
        open={Boolean(selected)}
        onClose={closeDetail}
        label={selected ? `Detail pemesanan ${selected.name}` : "Detail pemesanan"}
        eyebrow="DETAIL PEMESANAN"
        title={selected?.name}
        size="lg"
        tone="dim"
      >
        <BookingDetails booking={selected} />

        <div className="space-y-5 border-t border-gray-200 bg-gray-50 p-4 sm:p-5">
          <div className="space-y-4 border border-gray-200 bg-white p-4">
            <h3 className="text-[10px] tracking-widest text-gray-500">STATUS PEMESANAN</h3>
            <div>
              <label className={adminLabel} htmlFor="booking-status">
                UBAH STATUS
              </label>
              <select
                id="booking-status"
                value={selected?.status}
                disabled={statusSaving || (NEXT_STATUSES[selected?.status] || []).length === 0}
                onChange={(event) => changeStatus(event.target.value)}
                className="mt-1 w-full border border-gray-200 bg-transparent px-3 py-2.5 text-sm text-gray-900 focus:border-[#E11D48] outline-none disabled:opacity-60 sm:max-w-xs"
              >
                {STATUS_OPTIONS.filter(
                  (option) =>
                    option.value === selected?.status ||
                    (NEXT_STATUSES[selected?.status] || []).includes(option.value)
                ).map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
              {(NEXT_STATUSES[selected?.status] || []).length === 0 && (
                <p className="mt-2 text-xs text-gray-500">
                  Pemesanan selesai tidak dapat diubah lagi.
                </p>
              )}
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
              key={selected?.id}
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

            {selected?.status !== "ACCEPTED" && selected?.status !== "COMPLETED" && (
              <p className="text-xs text-gray-500">
                Tombol kirim email konfirmasi tersedia setelah status pemesanan diubah menjadi
                Diterima.
              </p>
            )}
          </div>

          <div className="border border-gray-200 bg-white p-4">
            <h3 className="text-[10px] tracking-widest text-gray-500">
              FOTO &amp; PILIHAN KLIEN
            </h3>
            <div className="mt-4">
              <BookingSelection
                key={`selection-${selected?.id}`}
                booking={selected}
                onUpdated={applyBookingUpdate}
              />
            </div>
          </div>

          <div className="border border-gray-200 bg-white p-4">
            <h3 className="text-[10px] tracking-widest text-gray-500">
              PEMBAYARAN &amp; EMAIL PENUTUP
            </h3>
            <div className="mt-4">
              <BookingPayment
                key={`payment-${selected?.id}`}
                booking={selected}
                onUpdated={applyBookingUpdate}
              />
            </div>
          </div>
        </div>
      </Modal>
    </div>
  );
}

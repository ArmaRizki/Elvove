import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ChevronLeft,
  ChevronRight,
  Mail,
  CalendarClock,
  MapPin,
  Phone,
} from "lucide-react";
import { adminApi } from "../../services/adminApi";
import CustomDatePicker from "../CustomDatePicker";
import Modal from "../Modal";
import CustomTimePicker from "../CustomTimePicker";
import BookingPayment from "./BookingPayment";
import BookingSelection from "./BookingSelection";
import { formatDate } from "./adminFormat";
import { pad } from "../../utils/bookingTime";
import { adminLabel, adminInput, adminSecondaryButton } from "./adminStyles";

const keyOf = (date) =>
  `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
const timeOf = (value) => (value || "").slice(0, 5) || "Waktu belum diatur";
const startOfMonth = (date) => new Date(date.getFullYear(), date.getMonth(), 1);
const startOfWeek = (date) => {
  const copy = new Date(date);
  copy.setDate(copy.getDate() - copy.getDay());
  return copy;
};
const addDays = (date, amount) => {
  const copy = new Date(date);
  copy.setDate(copy.getDate() + amount);
  return copy;
};
const sameMonth = (a, b) =>
  a.getMonth() === b.getMonth() && a.getFullYear() === b.getFullYear();
const monthLabel = (date) =>
  date.toLocaleDateString("id-ID", { month: "long", year: "numeric" });

const WEEKDAYS = ["Min", "Sen", "Sel", "Rab", "Kam", "Jum", "Sab"];
const VIEWS = [
  { value: "month", label: "Bulan" },
  { value: "week", label: "Minggu" },
  { value: "day", label: "Hari" },
];

export default function BookingCalendar() {
  const [bookings, setBookings] = useState([]);
  const [loading, setLoading] = useState(true);
  const [cursor, setCursor] = useState(() => startOfMonth(new Date()));
  const [view, setView] = useState("month");
  const [selected, setSelected] = useState(null);
  const [activeDay, setActiveDay] = useState("");
  const [newDate, setNewDate] = useState("");
  const [newTime, setNewTime] = useState("");
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await adminApi.getBookings();
      setBookings(Array.isArray(response.data) ? response.data : []);
    } catch (error) {
      setNotice(error.response?.data?.message || "Gagal memuat pemesanan.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const scheduledBookings = useMemo(
    () => bookings.filter((booking) => booking.eventDate),
    [bookings]
  );

  const byDate = useMemo(() => {
    const result = {};
    scheduledBookings.forEach((booking) => {
      (result[booking.eventDate] ||= []).push(booking);
    });
    Object.values(result).forEach((items) =>
      items.sort((a, b) => (a.eventTime || "").localeCompare(b.eventTime || ""))
    );
    return result;
  }, [scheduledBookings]);

  const todayKey = keyOf(new Date());

  // One cursor drives all three views: month shows the whole grid, week the
  // week around the cursor, day exactly that date.
  const days = useMemo(() => {
    const start =
      view === "month"
        ? startOfWeek(startOfMonth(cursor))
        : view === "week"
          ? startOfWeek(cursor)
          : new Date(cursor.getFullYear(), cursor.getMonth(), cursor.getDate());
    const count = view === "month" ? 42 : view === "week" ? 7 : 1;
    return Array.from({ length: count }, (_, index) => addDays(start, index));
  }, [cursor, view]);

  // Week and day views print dd/mm/yyyy like every other date in the app.
  const headerLabel =
    view === "month"
      ? monthLabel(cursor)
      : view === "week"
        ? `${formatDate(keyOf(days[0]))} – ${formatDate(keyOf(days[days.length - 1]))}`
        : formatDate(keyOf(cursor));

  const openBooking = (booking) => {
    setSelected(booking);
    setNewDate(booking.eventDate || "");
    setNewTime((booking.eventTime || "09:00").slice(0, 5));
    setNotice("");
  };

  const runAction = async (action, successMessage) => {
    setBusy(true);
    setNotice("");
    try {
      const result = await action();
      await load();
      setSelected(null);
      setNotice(result?.message || successMessage);
    } catch (error) {
      setNotice(error.response?.data?.message || "Tindakan gagal. Periksa kembali data Anda.");
    } finally {
      setBusy(false);
    }
  };

  const reschedule = () => {
    if (!newDate || !newTime) {
      setNotice("Pilih tanggal dan waktu baru terlebih dahulu.");
      return;
    }
    if (
      !window.confirm(
        `Ubah jadwal pemesanan ${selected.name} menjadi ${formatDate(newDate)} pukul ${newTime}?`
      )
    ) {
      return;
    }
    runAction(
      async () => {
        const { data } = await adminApi.rescheduleBooking(selected.id, {
          eventDate: newDate,
          eventTime: `${newTime}:00`,
        });
        return data;
      },
      "Jadwal berhasil diperbarui dan email pemberitahuan telah dikirim."
    );
  };

  const sendConfirmation = () =>
    runAction(
      async () => {
        const { data } = await adminApi.sendBookingConfirmation(selected.id);
        return data;
      },
      "Email konfirmasi berhasil dikirim."
    );

  const changeView = (next) => {
    setView(next);
    setActiveDay("");
  };

  const shift = (direction) => {
    setActiveDay("");
    setCursor((current) => {
      if (view === "week") return addDays(current, direction * 7);
      if (view === "day") return addDays(current, direction);
      return new Date(current.getFullYear(), current.getMonth() + direction, 1);
    });
  };

  const goToday = () => {
    setCursor(startOfMonth(new Date()));
    setView("month");
    setActiveDay("");
  };

  const activeDayItems = activeDay ? byDate[activeDay] || [] : [];

  // A booking whose schedule is agreed (ACCEPTED) or already past (COMPLETED) is
  // locked: the date only changes while it is still PENDING or waiting for the
  // customer to agree to a new one. The confirmation e-mail only exists for an
  // accepted booking, so it is the other way round.
  const canReschedule =
    selected?.status === "PENDING" || selected?.status === "RESCHEDULE_REQUESTED";
  const canSendConfirmation = selected?.status === "ACCEPTED";

  return (
    <div className="min-w-0">
      <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-wrap items-center gap-2">
          <div className="flex border border-gray-200">
            {VIEWS.map((item) => (
              <button
                key={item.value}
                type="button"
                onClick={() => changeView(item.value)}
                aria-pressed={view === item.value}
                className={`px-3 py-2 text-xs transition-colors ${
                  view === item.value ? "bg-gray-900 text-white" : "text-gray-600 hover:bg-gray-50"
                }`}
              >
                {item.label}
              </button>
            ))}
          </div>
          <button
            type="button"
            onClick={goToday}
            className="border border-gray-200 px-3 py-2 text-xs text-gray-700 transition-colors hover:bg-gray-50"
          >
            Hari ini
          </button>
        </div>

        <span className="text-xs text-gray-500">
          {scheduledBookings.length} pemesanan terjadwal
        </span>
      </div>

      {notice && (
        <p className="mb-4 text-sm text-[#E11D48]" role="status">
          {notice}
        </p>
      )}

      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2 sm:gap-3">
          <button
            type="button"
            onClick={() => shift(-1)}
            className="border border-gray-200 p-2 text-gray-700 transition-colors hover:bg-gray-50"
            aria-label={
              view === "month" ? "Bulan sebelumnya" : view === "week" ? "Minggu sebelumnya" : "Hari sebelumnya"
            }
          >
            <ChevronLeft size={17} />
          </button>
          <button
            type="button"
            onClick={() => shift(1)}
            className="border border-gray-200 p-2 text-gray-700 transition-colors hover:bg-gray-50"
            aria-label={
              view === "month" ? "Bulan berikutnya" : view === "week" ? "Minggu berikutnya" : "Hari berikutnya"
            }
          >
            <ChevronRight size={17} />
          </button>
          <h2 className="text-sm font-medium text-gray-900 sm:text-base">{headerLabel}</h2>
        </div>
      </div>

      {loading ? (
        <p className="py-10 text-sm text-gray-500">Memuat kalender...</p>
      ) : (
        <div className="overflow-hidden border border-gray-200">
          {view !== "day" && (
            <div className="grid grid-cols-7 border-b border-gray-200 bg-gray-50">
              {WEEKDAYS.map((day) => (
                <div
                  key={day}
                  className="px-1 py-2 text-center text-[10px] font-medium text-gray-500 sm:px-2 sm:py-3 sm:text-xs"
                >
                  {day}
                </div>
              ))}
            </div>
          )}

          <div className={view === "day" ? "grid grid-cols-1" : "grid grid-cols-7"}>
            {days.map((date) => {
              const dateKey = keyOf(date);
              const items = byDate[dateKey] || [];
              const inMonth = view === "month" ? sameMonth(date, cursor) : true;
              const isToday = dateKey === todayKey;
              return (
                <div
                  key={dateKey}
                  className={`border-b border-r border-gray-100 p-1 sm:p-2 ${
                    view === "day" ? "min-h-[14rem]" : view === "week" ? "min-h-[9rem]" : "min-h-[4.5rem] sm:min-h-28"
                  } ${inMonth ? "bg-white" : "bg-gray-50/70"} ${
                    isToday ? "ring-1 ring-inset ring-[#E11D48]/40" : ""
                  }`}
                >
                  <div
                    className={`mb-1 text-[10px] sm:mb-2 sm:text-xs ${
                      inMonth ? "text-gray-700" : "text-gray-300"
                    } ${isToday ? "font-semibold text-[#E11D48]" : ""}`}
                  >
                    {date.getDate()}
                  </div>

                  {/* Booking chips: always shown in day view, where there is
                      room; on phones month/week use the dot summary instead. */}
                  <div
                    className={`${view === "day" ? "block" : "hidden sm:block"} space-y-1`}
                  >
                    {items.map((booking) => (
                      <button
                        key={booking.id}
                        type="button"
                        onClick={() => openBooking(booking)}
                        title={`${timeOf(booking.eventTime)} · ${booking.name}`}
                        className="block w-full truncate border-l-2 border-[#E11D48] bg-rose-50 px-2 py-1 text-left text-[11px] text-rose-900 transition-colors hover:bg-rose-100 sm:truncate"
                      >
                        {timeOf(booking.eventTime)} · {booking.name}
                      </button>
                    ))}
                    {items.length === 0 && view === "day" && (
                      <p className="text-xs text-gray-500">Tidak ada pemesanan pada hari ini.</p>
                    )}
                  </div>

                  {/* Phones: dot summary that opens the day list below the grid */}
                  {view !== "day" && items.length > 0 && (
                    <button
                      type="button"
                      onClick={() => setActiveDay(activeDay === dateKey ? "" : dateKey)}
                      aria-label={`${items.length} pemesanan pada ${dateKey}`}
                      className="flex w-full flex-wrap items-center gap-1 text-[10px] text-rose-900 sm:hidden"
                    >
                      {items.slice(0, 4).map((booking) => (
                        <span
                          key={booking.id}
                          className="h-1.5 w-1.5 rounded-full bg-[#E11D48]"
                        />
                      ))}
                      {items.length > 4 && <span>+{items.length - 4}</span>}
                    </button>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Mobile day list: keeps every booking reachable without a 7-column squeeze */}
      {activeDayItems.length > 0 && (
        <div className="mt-3 border border-gray-200 p-3 sm:hidden">
          <p className="text-[10px] tracking-widest text-gray-700">
            PEMESANAN PADA {activeDay}
          </p>
          <div className="mt-2 space-y-2">
            {activeDayItems.map((booking) => (
              <button
                key={booking.id}
                type="button"
                onClick={() => openBooking(booking)}
                className="block w-full border border-gray-200 px-3 py-2 text-left text-sm text-gray-900 transition-colors hover:border-[#E11D48]/40"
              >
                <span className="font-medium">{timeOf(booking.eventTime)}</span>
                {" · "}
                {booking.name}
              </button>
            ))}
          </div>
        </div>
      )}

      <Modal
        open={Boolean(selected)}
        onClose={() => setSelected(null)}
        label={selected ? `Detail pemesanan ${selected.name}` : "Detail pemesanan"}
        eyebrow={
          selected?.status === "RESCHEDULE_REQUESTED"
            ? "Permintaan ubah jadwal"
            : selected?.status === "COMPLETED"
              ? "Pemesanan selesai"
              : selected?.status === "ACCEPTED"
                ? "Pemesanan diterima"
                : "Pemesanan menunggu"
        }
        title={selected?.name}
        tone="dim"
      >
        <div className="space-y-3 break-words p-4 text-sm sm:p-5">
          <p className="flex items-center gap-2 break-all"><Mail size={15} /> {selected.email}</p>
          <p className="flex items-center gap-2"><Phone size={15} /> {selected.phone}</p>
          <p className="flex items-center gap-2"><MapPin size={15} /> {selected.eventLocation || "-"}</p>
          <p className="text-gray-600">Paket: {selected.packageTitle || "-"}</p>
          <p className="text-gray-600">
            Jadwal saat ini: {formatDate(selected.eventDate) || "-"} · {timeOf(selected.eventTime)}
          </p>
        </div>

      <div className="space-y-3 border-t border-gray-200 p-4 sm:p-5">
        {canReschedule && (
          <>
            <h3 className="flex items-center gap-2 font-medium">
              <CalendarClock size={16} /> Ubah jadwal
            </h3>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div className="min-w-0">
                <label htmlFor="reschedule-date" className={adminLabel}>
                  Tanggal baru
                </label>
                <CustomDatePicker
                  id="reschedule-date"
                  value={newDate}
                  onChange={setNewDate}
                  label="Tanggal baru"
                  className={`${adminInput} form-field--icon flex items-center`}
                />
              </div>
              <div className="min-w-0">
                <label htmlFor="reschedule-time" className={adminLabel}>
                  Waktu baru
                </label>
                <CustomTimePicker
                  id="reschedule-time"
                  value={newTime}
                  onChange={setNewTime}
                  label="Waktu baru"
                  className={`${adminInput} form-field--icon flex items-center`}
                />
              </div>
            </div>
            <button
              type="button"
              disabled={busy}
              onClick={reschedule}
              className={`${adminSecondaryButton} w-full border-[#E11D48] text-[#E11D48] hover:bg-rose-50`}
            >
              Simpan Jadwal & Kirim Email
            </button>
          </>
        )}
        {canSendConfirmation && (
          <button
            type="button"
            disabled={busy}
            onClick={sendConfirmation}
            className={`${adminSecondaryButton} w-full`}
          >
            <Mail size={15} /> Kirim Email Konfirmasi
          </button>
        )}
        <div className="border-t border-gray-200 pt-3">
          <BookingSelection
            key={`selection-${selected.id}`}
            booking={selected}
            onUpdated={(update) => {
              if (!update) return;
              setSelected((current) => (current ? { ...current, ...update } : current));
              load();
            }}
          />
        </div>

        <div className="border-t border-gray-200 pt-3">
          <BookingPayment
            key={`payment-${selected.id}`}
            booking={selected}
            onUpdated={(update) => {
              if (!update) return;
              setSelected((current) => (current ? { ...current, ...update } : current));
              // The list drives the calendar cells, so the payment marker
              // there has to come from the server too.
              load();
            }}
          />
        </div>
        {!canReschedule && !canSendConfirmation && (
          <p className="text-sm text-gray-600">
            Pemesanan ini sudah selesai. Jadwal dan email konfirmasi tidak dapat diubah lagi.
          </p>
        )}
        </div>
      </Modal>
    </div>
  );
}

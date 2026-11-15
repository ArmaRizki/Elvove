import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { CalendarDays, ChevronLeft, ChevronRight } from "lucide-react";
import { formatDateDisplay, todayIso } from "../utils/bookingTime";

const MONTHS = [
  "Januari", "Februari", "Maret", "April", "Mei", "Juni",
  "Juli", "Agustus", "September", "Oktober", "November", "Desember",
];
const WEEKDAYS = ["Sen", "Sel", "Rab", "Kam", "Jum", "Sab", "Min"];

const pad = (value) => String(value).padStart(2, "0");

const parseIso = (value) => {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value || "");
  return match
    ? { year: Number(match[1]), month: Number(match[2]) - 1, day: Number(match[3]) }
    : null;
};

const isoOf = (year, month, day) => `${year}-${pad(month + 1)}-${pad(day)}`;

// Senin sebagai kolom pertama; null menandai sel kosong sebelum tanggal 1.
const monthCells = (year, month) => {
  const offset = (new Date(year, month, 1).getDay() + 6) % 7;
  const days = new Date(year, month + 1, 0).getDate();
  const cells = Array.from({ length: offset }, () => null);
  for (let day = 1; day <= days; day += 1) cells.push(day);
  return cells;
};

/**
 * Single-input date picker: the closed control looks like the form's other
 * fields, and the panel is our own calendar, so the value always reads
 * dd/mm/yyyy. The browser's native date input prints mm/dd/yyyy from the
 * browser locale and cannot be reformatted, which is why it is not used here.
 *
 * The value stays a plain "YYYY-MM-DD" string, so the API payload is unchanged.
 */
export default function CustomDatePicker({
  id,
  value,
  min,
  onChange,
  invalid = false,
  describedBy,
  className = "",
  label = "Tanggal acara",
}) {
  const [today] = useState(todayIso);
  const [open, setOpen] = useState(false);
  const [placement, setPlacement] = useState("bottom");
  const [view, setView] = useState(() => parseIso(value) || parseIso(today));

  const rootRef = useRef(null);
  const panelRef = useRef(null);
  const triggerRef = useRef(null);
  const shouldFocusRef = useRef(false);

  const display = formatDateDisplay(value);
  const cells = monthCells(view.year, view.month);
  const minParts = parseIso(min);
  // Bulan paling awal yang boleh dibuka, supaya tombol "sebelumnya" berhenti
  // di bulan yang memuat tanggal minimum.
  const atFirstMonth =
    Boolean(minParts) &&
    (view.year < minParts.year ||
      (view.year === minParts.year && view.month <= minParts.month));

  // Klik di luar menutup panel.
  useEffect(() => {
    if (!open) return undefined;

    const onPointerDown = (event) => {
      if (rootRef.current && !rootRef.current.contains(event.target)) {
        setOpen(false);
      }
    };

    document.addEventListener("mousedown", onPointerDown);
    document.addEventListener("touchstart", onPointerDown, { passive: true });

    return () => {
      document.removeEventListener("mousedown", onPointerDown);
      document.removeEventListener("touchstart", onPointerDown);
    };
  }, [open]);

  // Buka ke bawah bila ada ruang, atau ke atas bila panel akan terpotong.
  useLayoutEffect(() => {
    if (!open) return;

    const trigger = triggerRef.current;
    const panel = panelRef.current;
    if (!trigger || !panel) return;

    const rect = trigger.getBoundingClientRect();
    const spaceBelow = window.innerHeight - rect.bottom;
    const spaceAbove = rect.top;
    const needed = panel.offsetHeight + 8;

    setPlacement(spaceBelow < needed && spaceAbove > spaceBelow ? "top" : "bottom");
  }, [open]);

  // Fokuskan tanggal terpilih (atau hari pertama yang bisa dipilih).
  useEffect(() => {
    if (!open) return undefined;

    const frame = requestAnimationFrame(() => {
      if (!shouldFocusRef.current) return;
      shouldFocusRef.current = false;
      const panel = panelRef.current;
      const target =
        panel?.querySelector('[aria-current="date"]') ||
        panel?.querySelector("[data-day]:not(:disabled)");
      target?.focus();
    });

    return () => cancelAnimationFrame(frame);
  }, [open, view]);

  const closeAndFocus = () => {
    setOpen(false);
    triggerRef.current?.focus();
  };

  const openPanel = () => {
    shouldFocusRef.current = true;
    setView(parseIso(value) || parseIso(today));
    setPlacement("bottom");
    setOpen(true);
  };

  const togglePanel = () => {
    if (open) {
      setOpen(false);
      return;
    }
    openPanel();
  };

  const shiftMonth = (delta) => {
    setView((current) => {
      const month = current.month + delta;
      return {
        year: current.year + Math.floor(month / 12),
        month: ((month % 12) + 12) % 12,
      };
    });
  };

  const chooseDay = (day) => {
    onChange(isoOf(view.year, view.month, day));
    closeAndFocus();
  };

  const onTriggerKeyDown = (event) => {
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      openPanel();
    }
  };

  const onPanelKeyDown = (event) => {
    if (event.key === "Escape") {
      event.preventDefault();
      closeAndFocus();
    }
  };

  // Panah memindahkan fokus satu hari (atau satu minggu) di dalam kalender.
  const onGridKeyDown = (event) => {
    const step = { ArrowLeft: -1, ArrowRight: 1, ArrowUp: -7, ArrowDown: 7 }[event.key];
    if (!step) return;

    const buttons = [...event.currentTarget.querySelectorAll("[data-day]")];
    const index = buttons.indexOf(document.activeElement);
    if (index === -1) return;

    event.preventDefault();
    const next = buttons[index + step];
    if (next && !next.disabled) next.focus();
  };

  const dayClass = (isSelected) =>
    `flex h-9 items-center justify-center text-sm transition-colors ${
      isSelected
        ? "bg-[#E11D48] font-medium text-white"
        : "text-gray-900 hover:bg-rose-50 disabled:text-gray-300 disabled:hover:bg-transparent"
    }`;

  return (
    <div ref={rootRef} className="relative field-with-icon">
      <button
        id={id}
        ref={triggerRef}
        type="button"
        aria-label={label}
        aria-haspopup="dialog"
        aria-expanded={open}
        aria-invalid={invalid}
        aria-describedby={describedBy}
        onClick={togglePanel}
        onKeyDown={onTriggerKeyDown}
        className={`${className} flex items-center text-left`}
      >
        <span className={display ? "text-gray-900" : "text-gray-500"}>
          {display || "dd/mm/yyyy"}
        </span>
      </button>

      {/* Ikon kalender di luar tombol pemicu, agar ukurannya sama persis
          dengan ikon jam pada field waktu. */}
      <span className="field-icon-button pointer-events-none select-none" aria-hidden="true">
        <CalendarDays size={17} aria-hidden="true" />
      </span>

      {open && (
        <div
          ref={panelRef}
          role="dialog"
          aria-label="Pilih tanggal"
          onKeyDown={onPanelKeyDown}
          className={`absolute left-0 right-0 z-30 border border-gray-300 bg-white shadow-lg ${
            placement === "top" ? "bottom-full mb-1" : "top-full mt-1"
          }`}
        >
          <div className="flex items-center justify-between border-b border-gray-200 px-2 py-2">
            <button
              type="button"
              aria-label="Bulan sebelumnya"
              onClick={() => shiftMonth(-1)}
              disabled={atFirstMonth}
              className="flex h-8 w-8 items-center justify-center text-gray-700 transition-colors hover:text-[#E11D48] disabled:cursor-not-allowed disabled:opacity-40"
            >
              <ChevronLeft size={16} aria-hidden="true" />
            </button>

            <p aria-live="polite" className="text-sm font-medium text-gray-900">
              {MONTHS[view.month]} {view.year}
            </p>

            <button
              type="button"
              aria-label="Bulan berikutnya"
              onClick={() => shiftMonth(1)}
              className="flex h-8 w-8 items-center justify-center text-gray-700 transition-colors hover:text-[#E11D48]"
            >
              <ChevronRight size={16} aria-hidden="true" />
            </button>
          </div>

          <div
            role="grid"
            aria-label={`${MONTHS[view.month]} ${view.year}`}
            onKeyDown={onGridKeyDown}
            className="grid grid-cols-7 gap-y-0.5 px-2 pb-2 pt-2"
          >
            {WEEKDAYS.map((weekday) => (
              <span
                key={weekday}
                className="flex h-7 items-center justify-center text-[10px] tracking-widest text-gray-500"
              >
                {weekday}
              </span>
            ))}

            {cells.map((day, index) => {
              if (day === null) return <span key={`empty-${index}`} aria-hidden="true" />;

              const iso = isoOf(view.year, view.month, day);
              const isSelected = iso === value;
              const isDisabled = Boolean(min) && iso < min;

              return (
                <button
                  key={iso}
                  type="button"
                  data-day={day}
                  aria-current={isSelected ? "date" : undefined}
                  aria-label={`${day} ${MONTHS[view.month]} ${view.year}`}
                  disabled={isDisabled}
                  onClick={() => chooseDay(day)}
                  className={dayClass(isSelected)}
                >
                  {day}
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

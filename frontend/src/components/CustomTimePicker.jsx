import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { Clock } from "lucide-react";

// Sesi foto hanya dimulai pada kelipatan 15 menit.
const HOUR_OPTIONS = Array.from({ length: 24 }, (_, hour) =>
  String(hour).padStart(2, "0")
);
const MINUTE_OPTIONS = ["00", "15", "30", "45"];

const splitTime = (value) => {
  const match = /^([01]\d|2[0-3]):([0-5]\d)/.exec(value || "");
  return match ? { hour: match[1], minute: match[2] } : { hour: "", minute: "" };
};

/**
 * Single-input time picker: the closed control looks like the form's other
 * fields, while the panel offers every hour and only 00/15/30/45 minutes.
 *
 * The value stays a plain "HH:mm" string, so the API payload is unchanged.
 */
export default function CustomTimePicker({
  id,
  value,
  onChange,
  invalid = false,
  describedBy,
  className = "",
  label = "Waktu acara",
}) {
  const [open, setOpen] = useState(false);
  const [placement, setPlacement] = useState("bottom");
  const [draft, setDraft] = useState(() => splitTime(value));

  const rootRef = useRef(null);
  const panelRef = useRef(null);
  const triggerRef = useRef(null);
  const hourListRef = useRef(null);
  const minuteListRef = useRef(null);
  const shouldFocusRef = useRef(false);

  const selected = splitTime(value);
  const display = selected.hour && selected.minute ? `${selected.hour}:${selected.minute}` : "";

  // Nilai dari luar (misalnya form direset) ikut memperbarui pilihan.
  useEffect(() => {
    setDraft(splitTime(value));
  }, [value]);

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

  // Fokuskan pilihan aktif, lalu gulir kedua kolom ke pilihan tersebut.
  useEffect(() => {
    if (!open) return undefined;

    const frame = requestAnimationFrame(() => {
      if (shouldFocusRef.current) {
        shouldFocusRef.current = false;
        const active =
          hourListRef.current?.querySelector('[aria-selected="true"]') ||
          hourListRef.current?.querySelector('[role="option"]');
        active?.focus();
      }

      [hourListRef.current, minuteListRef.current].forEach((list) => {
        const item = list?.querySelector('[aria-selected="true"]');
        if (!list || !item) return;
        list.scrollTop = Math.max(
          0,
          item.offsetTop - list.clientHeight / 2 + item.offsetHeight / 2
        );
      });
    });

    return () => cancelAnimationFrame(frame);
  }, [open]);

  const closeAndFocus = () => {
    setOpen(false);
    triggerRef.current?.focus();
  };

  const commit = (next) => {
    onChange(`${next.hour}:${next.minute}`);
  };

  // Memilih jam menyimpan nilai bila menit sudah ada. Panel tetap terbuka,
  // kecuali pilihan ini melengkapi pasangan yang sebelumnya belum lengkap.
  const chooseHour = (hour) => {
    const wasComplete = Boolean(draft.hour && draft.minute);
    const next = { ...draft, hour };
    setDraft(next);

    if (!next.minute) return;

    commit(next);
    if (!wasComplete) closeAndFocus();
  };

  // Menit adalah pilihan terakhir, jadi panel menutup setelahnya.
  const chooseMinute = (minute) => {
    const next = { ...draft, minute };
    setDraft(next);

    if (!next.hour) return;

    commit(next);
    closeAndFocus();
  };

  const openPanel = () => {
    shouldFocusRef.current = true;
    setDraft(splitTime(value));
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

  const onTriggerKeyDown = (event) => {
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      openPanel();
    }
  };

  const onPanelKeyDown = (event) => {
    if (event.key === "Escape") {
      event.preventDefault();
      setOpen(false);
      triggerRef.current?.focus();
      return;
    }

    if (event.key === "Home" || event.key === "End") {
      const options = [...event.currentTarget.querySelectorAll('[role="option"]')];
      if (options.length === 0) return;
      event.preventDefault();
      (event.key === "Home" ? options[0] : options[options.length - 1]).focus();
    }
  };

  // Panah atas/bawah memindahkan fokus di dalam satu kolom.
  const onListKeyDown = (event) => {
    const keys = ["ArrowDown", "ArrowUp"];
    if (!keys.includes(event.key)) return;

    const options = [...event.currentTarget.querySelectorAll('[role="option"]')];
    const index = options.indexOf(document.activeElement);
    if (index === -1) return;

    event.preventDefault();

    const next =
      event.key === "ArrowDown"
        ? Math.min(options.length - 1, index + 1)
        : Math.max(0, index - 1);

    options[next].focus();
  };

  const optionClass = (isSelected) =>
    `block w-full px-3 py-2 text-left text-sm transition-colors ${
      isSelected
        ? "bg-[#E11D48] font-medium text-white"
        : "text-gray-900 hover:bg-rose-50"
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
          {display || "Pilih waktu"}
        </span>
      </button>

      {/* Ikon jam diletakkan di luar tombol pemicu supaya ukurannya sama
          persis dengan ikon kalender pada field tanggal. Tombol pemicu sendiri
          yang membuka panel, jadi ikon ini hanya dekoratif. */}
      <span className="field-icon-button pointer-events-none select-none" aria-hidden="true">
        <Clock size={17} aria-hidden="true" />
      </span>

      {open && (
        <div
          ref={panelRef}
          role="dialog"
          aria-label="Pilih waktu"
          onKeyDown={onPanelKeyDown}
          className={`absolute left-0 right-0 z-30 border border-gray-300 bg-white shadow-lg ${
            placement === "top" ? "bottom-full mb-1" : "top-full mt-1"
          }`}
        >
          <div className="grid grid-cols-2 divide-x divide-gray-200">
            <div className="relative">
              <p
                id="timepicker-hour-heading"
                className="border-b border-gray-200 px-3 py-2 text-[10px] tracking-widest text-gray-500"
              >
                JAM
              </p>

              <div
                ref={hourListRef}
                role="listbox"
                aria-labelledby="timepicker-hour-heading"
                aria-label="Jam"
                onKeyDown={onListKeyDown}
                className="max-h-56 overflow-y-auto py-1"
              >
                {HOUR_OPTIONS.map((hour) => (
                  <button
                    key={hour}
                    type="button"
                    role="option"
                    aria-selected={draft.hour === hour}
                    tabIndex={draft.hour === hour ? 0 : -1}
                    onClick={() => chooseHour(hour)}
                    className={optionClass(draft.hour === hour)}
                  >
                    {hour}
                  </button>
                ))}
              </div>
            </div>

            <div className="relative">
              <p
                id="timepicker-minute-heading"
                className="border-b border-gray-200 px-3 py-2 text-[10px] tracking-widest text-gray-500"
              >
                MENIT
              </p>

              <div
                ref={minuteListRef}
                role="listbox"
                aria-labelledby="timepicker-minute-heading"
                aria-label="Menit"
                onKeyDown={onListKeyDown}
                className="max-h-56 overflow-y-auto py-1"
              >
                {MINUTE_OPTIONS.map((minute) => (
                  <button
                    key={minute}
                    type="button"
                    role="option"
                    aria-selected={draft.minute === minute}
                    tabIndex={draft.minute === minute ? 0 : -1}
                    onClick={() => chooseMinute(minute)}
                    className={optionClass(draft.minute === minute)}
                  >
                    {minute}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

import { useEffect, useState } from "react";
import { publicApi } from "../services/publicApi";
import SectionHeading from "../components/SectionHeading";
import CustomTimePicker from "../components/CustomTimePicker";
import { formatDateDisplay, isQuarterHour, todayIso } from "../utils/bookingTime";
import { CalendarDays } from "lucide-react";
import { Link } from "react-router-dom";

const initialForm = {
  name: "",
  email: "",
  phone: "",
  eventDate: "",
  eventTime: "",
  packageId: "",
  eventLocation: "",
  budget: "",
  message: "",
};

const formatPrice = (price) =>
  new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0,
  }).format(price);

/**
 * Field tanggal dengan ikon kalender sendiri.
 *
 * Ikon bawaan browser disembunyikan (lihat index.css) supaya ukurannya
 * konsisten dengan ikon jam di sebelahnya. Tombol ini memanggil
 * `showPicker()` bila peramban mendukungnya, sehingga kalender aslinya tetap
 * terbuka seperti biasa.
 */
function DateField({ id, value, min, invalid, describedBy, onChange, className = "" }) {
  const openPicker = (event) => {
    const input = event.currentTarget.parentElement?.querySelector("input");

    // Fokus dulu supaya segmen tanggal bisa langsung diketik, lalu kalender
    // aslinya dibuka.
    input?.focus();

    if (typeof input?.showPicker === "function") {
      try {
        input.showPicker();
      } catch {
        // showPicker ditolak di beberapa konteks; fokus di atas sudah cukup.
      }
    }
  };

  return (
    <div className="relative field-with-icon">
      <input
        id={id}
        type="date"
        aria-label="Tanggal acara"
        aria-invalid={Boolean(invalid)}
        aria-describedby={describedBy}
        min={min}
        value={value}
        onChange={onChange}
        className={`${className} date-input form-field--icon`}
      />

      {/* Chromium memakai format tanggal dari locale browser, bukan atribut
          lang, sehingga selalu tampil mm/dd/yyyy. Teks asli disembunyikan dan
          diganti dd/mm/yyyy. Saat fokus, teks asli kembali tampil agar
          pengetikan tetap terlihat. */}
      <span
        aria-hidden="true"
        className={`date-input-overlay text-base sm:text-sm ${
          value ? "date-input-overlay--filled" : ""
        }`}
      >
        {formatDateDisplay(value) || "dd/mm/yyyy"}
      </span>

      <button
        type="button"
        className="field-icon-button"
        aria-label="Buka kalender"
        aria-haspopup="dialog"
        onClick={openPicker}
      >
        <CalendarDays size={17} aria-hidden="true" />
      </button>
    </div>
  );
}

export default function Booking() {
  const [form, setForm] = useState(initialForm);
  const [packages, setPackages] = useState([]);
  const [packagesLoading, setPackagesLoading] = useState(true);
  const [packagesError, setPackagesError] = useState(false);
  const [errors, setErrors] = useState({});
  const [status, setStatus] = useState("idle");
  const [termsAccepted, setTermsAccepted] = useState(false);
  const [terms, setTerms] = useState(null);
  const [termsLoading, setTermsLoading] = useState(true);
  const [termsError, setTermsError] = useState(false);
  const [submitError, setSubmitError] = useState("");

  // The picker's minimum is the day the page was opened. Keeping it in state
  // means a tab left open overnight does not start rejecting an already chosen
  // today, while a freshly opened page still never offers a past date.
  const [minDate] = useState(todayIso);

  useEffect(() => {
    const loadPackages = async () => {
      try {
        const res = await publicApi.getPackages();

        setPackages(res.data);

        const params = new URLSearchParams(window.location.search);
        const packageId = params.get("packageId");

        if (
          packageId &&
          res.data.some((item) => String(item.id) === packageId)
        ) {
          setForm((prev) => ({
            ...prev,
            packageId,
          }));
        }
      } catch {
        setPackagesError(true);
      } finally {
        setPackagesLoading(false);
      }
    };

    loadPackages();
  }, []);

  useEffect(() => {
    let active = true;

    const loadTerms = async () => {
      try {
        const res = await publicApi.getTerms();
        if (active) setTerms(res.data);
      } catch {
        if (active) setTermsError(true);
      } finally {
        if (active) setTermsLoading(false);
      }
    };

    loadTerms();

    return () => {
      active = false;
    };
  }, []);

  const selectedPackage = packages.find(
    (item) => String(item.id) === form.packageId
  );

  const validate = () => {
    const e = {};

    if (!form.name.trim()) e.name = "Mohon isi nama lengkap Anda";

    if (!form.email.trim()) {
      e.email = "Mohon isi alamat email Anda";
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) {
      e.email = "Masukkan alamat email yang valid";
    }

    if (!form.phone.trim()) {
      e.phone = "Mohon isi nomor telepon Anda";
    } else if (!/^[0-9+\-\s]{7,15}$/.test(form.phone)) {
      e.phone = "Masukkan nomor telepon yang valid";
    }

    if (!form.eventDate) {
      e.eventDate = "Mohon pilih tanggal acara";
    } else if (form.eventDate < minDate) {
      e.eventDate = "Tanggal acara tidak boleh sebelum hari ini";
    }

    if (!form.eventTime) {
      e.eventTime = "Mohon pilih waktu acara";
    } else if (!isQuarterHour(form.eventTime)) {
      e.eventTime = "Pilih waktu pada menit 00, 15, 30, atau 45";
    }

    if (!form.packageId) e.packageId = "Mohon pilih paket";
    if (termsLoading) {
      e.termsAccepted = "Syarat & Ketentuan masih dimuat.";
    } else if (termsError || !terms) {
      e.termsAccepted = "Syarat & Ketentuan tidak dapat dimuat. Silakan muat ulang halaman.";
    } else if (!termsAccepted) {
      e.termsAccepted = "Anda harus menyetujui Syarat & Ketentuan";
    }

    return e;
  };

  const handleChange = (field) => (ev) => {
    const value = ev.target.value;

    setForm((prev) => ({
      ...prev,
      [field]: value,
    }));

    if (errors[field]) {
      setErrors((prev) => ({
        ...prev,
        [field]: undefined,
      }));
    }
  };

  // Jam dan menit dipilih lewat CustomTimePicker; nilainya tetap "HH:mm".
  const handleTimeChange = (value) => {
    setForm((prev) => ({
      ...prev,
      eventTime: value,
    }));

    if (errors.eventTime) {
      setErrors((prev) => ({
        ...prev,
        eventTime: undefined,
      }));
    }
  };

  const handleSubmit = async (ev) => {
    ev.preventDefault();

    const validationErrors = validate();
    setErrors(validationErrors);

    if (Object.keys(validationErrors).length > 0) return;

    setStatus("submitting");
    setSubmitError("");

    try {
      await publicApi.createBooking({
        ...form,
        packageId: Number(form.packageId),
        termsAccepted,
        termsVersion: terms.version,
      });

      setStatus("success");
      setForm(initialForm);
      setTermsAccepted(false);
    } catch (error) {
      if (error.response?.status === 409) {
        try {
          const latestTerms = await publicApi.getTerms();
          setTerms(latestTerms.data);
          setTermsAccepted(false);
          setSubmitError(
            "Syarat & Ketentuan telah diperbarui. Silakan baca dan setujui versi terbaru sebelum mengirim ulang."
          );
        } catch {
          setSubmitError(
            "Syarat & Ketentuan terbaru tidak dapat dimuat. Silakan muat ulang halaman."
          );
        }
      } else {
        setSubmitError(
          error.response?.data?.message ||
          "Terjadi kesalahan saat mengirim permintaan Anda. Silakan coba lagi."
        );
      }

      setStatus("error");
    }
  };

  const inputClass =
    "form-field w-full text-base sm:text-sm";
  const errorClass = "text-[#E11D48] text-xs mt-1.5";

  return (
    <div className="pt-28 sm:pt-32 pb-20 sm:pb-24 max-w-3xl mx-auto px-5 sm:px-6">
      <SectionHeading
        as="h1"
        title="Pesan Sesi Foto"
        subtitle="Ceritakan momen yang ingin Elvove abadikan."
      />

      {status === "success" ? (
        <div className="border border-[#E11D48] p-6 sm:p-8 text-center">
          <p className="font-display text-2xl text-gray-900 mb-2">
            Terima kasih!
          </p>
          <p className="text-gray-700 text-sm leading-relaxed">
            Permintaan Anda telah kami terima. Elvove akan menghubungi Anda
            untuk membahas detailnya.
          </p>
        </div>
      ) : (
        <form onSubmit={handleSubmit} noValidate className="space-y-8">
          {/* Data klien */}
          <fieldset className="space-y-5">
            <legend className="sr-only">Data klien</legend>

            <div className="grid grid-cols-1 items-start gap-5 sm:grid-cols-2">
              <div className="min-w-0">
                <label htmlFor="name" className="field-label">
                  Nama lengkap
                </label>
                <input
                  id="name"
                  aria-label="Nama lengkap"
                  aria-invalid={Boolean(errors.name)}
                  placeholder="Nama lengkap"
                  value={form.name}
                  onChange={handleChange("name")}
                  className={inputClass}
                />
                {errors.name && <p className={errorClass}>{errors.name}</p>}
              </div>

              <div className="min-w-0">
                <label htmlFor="phone" className="field-label">
                  Nomor telepon
                </label>
                <input
                  id="phone"
                  type="tel"
                  aria-label="Nomor telepon"
                  aria-invalid={Boolean(errors.phone)}
                  placeholder="0812 3456 7890"
                  value={form.phone}
                  onChange={handleChange("phone")}
                  className={inputClass}
                />
                {errors.phone && <p className={errorClass}>{errors.phone}</p>}
              </div>

              <div className="min-w-0 sm:col-span-2">
                <label htmlFor="email" className="field-label">
                  Alamat email
                </label>
                <input
                  id="email"
                  type="email"
                  aria-label="Alamat email"
                  aria-invalid={Boolean(errors.email)}
                  placeholder="nama@email.com"
                  value={form.email}
                  onChange={handleChange("email")}
                  className={inputClass}
                />
                {errors.email && <p className={errorClass}>{errors.email}</p>}
              </div>
            </div>
          </fieldset>

          {/* Paket dan jadwal */}
          <fieldset className="space-y-5">
            <legend className="sr-only">Paket dan jadwal acara</legend>

            {/* Pilihan paket */}
            <div>
              <label htmlFor="packageId" className="field-label">
                Paket fotografi
              </label>
              <select
                id="packageId"
                aria-label="Paket fotografi"
                aria-invalid={Boolean(errors.packageId)}
                value={form.packageId}
                onChange={handleChange("packageId")}
                className={`${inputClass} form-select`}
                disabled={packagesLoading || packagesError || packages.length === 0}
              >
                <option value="">
                  {packagesLoading
                    ? "Memuat paket..."
                    : packagesError
                      ? "Paket tidak dapat dimuat"
                      : packages.length === 0
                        ? "Belum ada paket tersedia"
                        : "Pilih paket fotografi"}
                </option>

                {packages.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.title} — {formatPrice(item.price)}
                  </option>
                ))}
              </select>

              {errors.packageId && <p className={errorClass}>{errors.packageId}</p>}

              {selectedPackage && (
                <div className="mt-3 rounded-lg border border-[#E11D48]/25 bg-[#E11D48]/[0.04] p-4">
                  <p className="font-medium text-gray-900">{selectedPackage.title}</p>
                  <p className="mt-1 text-sm text-[#E11D48]">
                    {formatPrice(selectedPackage.price)}
                  </p>

                  {selectedPackage.features?.length > 0 && (
                    <ul className="mt-3 space-y-1">
                      {selectedPackage.features.map((feature, index) => (
                        <li key={index} className="text-sm text-gray-700">
                          • {feature}
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              )}
            </div>

            {/* Tanggal dan waktu berdampingan */}
            <div className="grid grid-cols-1 items-start gap-5 sm:grid-cols-2">
              <div className="min-w-0">
                <label htmlFor="eventDate" className="field-label">
                  Tanggal acara
                </label>
                <DateField
                  id="eventDate"
                  value={form.eventDate}
                  min={minDate}
                  invalid={Boolean(errors.eventDate)}
                  describedBy={errors.eventDate ? "eventDate-error" : undefined}
                  onChange={handleChange("eventDate")}
                  className={inputClass}
                />
                {errors.eventDate && (
                  <p id="eventDate-error" className={errorClass}>
                    {errors.eventDate}
                  </p>
                )}
              </div>

              <div className="min-w-0">
                <label htmlFor="eventTime" className="field-label">
                  Waktu acara
                </label>
                <CustomTimePicker
                  id="eventTime"
                  value={form.eventTime}
                  onChange={handleTimeChange}
                  invalid={Boolean(errors.eventTime)}
                  describedBy={errors.eventTime ? "eventTime-error" : undefined}
                  className={`${inputClass} form-field--icon`}
                />

                {errors.eventTime && (
                  <p id="eventTime-error" className={errorClass}>
                    {errors.eventTime}
                  </p>
                )}
              </div>
            </div>
          </fieldset>

          {/* Lokasi dan catatan */}
          <fieldset className="space-y-5">
            <legend className="sr-only">Detail acara</legend>

            <div className="grid grid-cols-1 items-start gap-5 sm:grid-cols-2">
              <div className="min-w-0 sm:col-span-2">
                <label htmlFor="eventLocation" className="field-label">
                  Lokasi acara
                </label>
                <input
                  id="eventLocation"
                  aria-label="Lokasi acara"
                  placeholder="Universitas, kota, atau nama gedung"
                  value={form.eventLocation}
                  onChange={handleChange("eventLocation")}
                  className={inputClass}
                />
              </div>

              <div className="min-w-0 sm:col-span-2">
                <label htmlFor="message" className="field-label">
                  Ceritakan kebutuhan Anda
                </label>
                <textarea
                  id="message"
                  aria-label="Pesan tambahan"
                  placeholder="Ceritakan lebih lanjut tentang acara Anda..."
                  rows={5}
                  value={form.message}
                  onChange={handleChange("message")}
                  className={`${inputClass} resize-y`}
                />
              </div>
            </div>
          </fieldset>

          {status === "error" && (
            <p className="text-[#E11D48] text-sm" role="alert">
              {submitError || "Terjadi kesalahan saat mengirim permintaan Anda. Silakan coba lagi."}
            </p>
          )}

          {/* Syarat & Ketentuan */}
          <div className="rounded-lg border border-gray-200 bg-gray-50/60 p-5 space-y-4">
            <div>
              <p className="text-gray-900 text-sm font-medium">
                Syarat &amp; Ketentuan
              </p>

              <p className="text-gray-600 text-sm mt-1">
                Mohon baca kebijakan pemesanan kami sebelum mengirim permintaan Anda.
              </p>

              {termsLoading && (
                <p className="text-gray-600 text-xs mt-2">
                  Memuat Syarat &amp; Ketentuan...
                </p>
              )}
              {termsError && (
                <p className="text-[#E11D48] text-xs mt-2">
                  Syarat &amp; Ketentuan tidak dapat dimuat. Silakan muat ulang halaman.
                </p>
              )}
            </div>

            <Link
              to="/terms-and-conditions"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 text-sm text-[#E11D48] hover:underline"
            >
              Baca Syarat &amp; Ketentuan
              <span aria-hidden="true">↗</span>
            </Link>

            <label className="flex items-start gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={termsAccepted}
                onChange={(e) => {
                  setTermsAccepted(e.target.checked);

                  if (e.target.checked) {
                    setErrors((prev) => ({
                      ...prev,
                      termsAccepted: undefined,
                    }));
                  }
                }}
                disabled={termsLoading || termsError || !terms}
                className="mt-1 h-4 w-4 shrink-0 accent-[#E11D48]"
              />

              <span className="text-sm text-gray-700 leading-relaxed">
                Saya telah membaca dan menyetujui Syarat &amp; Ketentuan Layanan
                Fotografi Elvove.
              </span>
            </label>

            {errors.termsAccepted && (
              <p className={errorClass.replace("mt-1.5", "")}>
                {errors.termsAccepted}
              </p>
            )}
          </div>

          <button
            type="submit"
            disabled={
              status === "submitting" ||
              packagesLoading ||
              packagesError ||
              packages.length === 0 ||
              !termsAccepted ||
              termsLoading ||
              termsError ||
              !terms
            }
            className="w-full sm:w-auto rounded-lg bg-[#E11D48] text-white px-8 py-3 text-sm tracking-wide hover:bg-[#BE123C] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {status === "submitting" ? "Mengirim..." : "Kirim Permintaan"}
          </button>
        </form>
      )}
    </div>
  );
}

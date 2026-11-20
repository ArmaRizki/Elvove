import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { Check, ChevronLeft, ChevronRight, LoaderCircle, Lock, Maximize2, X } from "lucide-react";
import { galleryApi } from "../services/galleryApi";
import LoadingState from "../components/LoadingState";
import SectionHeading from "../components/SectionHeading";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "";

/**
 * The page a client opens from the e-mail link to pick the photos to edit.
 *
 * <p>The token in the address is the whole credential, so the page shows only
 * what that one link unlocks. The picked set is saved as the client clicks; the
 * final "kirim" copies the chosen originals into the "foto terpilih" folder on Drive
 * and locks the choice.</p>
 */
export default function PhotoSelection() {
  const { token } = useParams();
  const [state, setState] = useState({ loading: true, error: "" });
  const [gallery, setGallery] = useState(null);
  const [picked, setPicked] = useState([]);
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState({ type: "", text: "" });
  const [submitting, setSubmitting] = useState(false);
  // Index of the photo open in the full view, or null when it is closed.
  const [viewer, setViewer] = useState(null);
  const saveTimer = useRef(null);
  // The save already on the wire, so submit can wait for it instead of racing it.
  const savePending = useRef(null);

  // The gallery is read once per token. Every state change happens in the
  // promise callbacks, never synchronously in the effect.
  useEffect(() => {
    let live = true;
    galleryApi
      .getGallery(token)
      .then(({ data }) => {
        if (!live) return;
        setGallery(data);
        setPicked((data.photos || []).filter((p) => p.selected).map((p) => p.id));
        setState({ loading: false, error: "" });
      })
      .catch((error) => {
        if (!live) return;
        const status = error.response?.status;
        setState({
          loading: false,
          error:
            error.response?.data?.message ||
            (status === 404
              ? "Link ini tidak dikenali. Periksa kembali tautan dari email Anda."
              : "Halaman foto tidak dapat dibuka. Coba muat ulang halaman ini."),
        });
      });
    return () => {
      live = false;
    };
  }, [token]);

  // Drop the debounce when the page unmounts; a request already sent keeps
  // running, but a timer that fires after unmount would only fail.
  useEffect(() => {
    return () => {
      if (saveTimer.current) clearTimeout(saveTimer.current);
    };
  }, []);

  const limit = gallery?.limit || 0;
  const submitted = Boolean(gallery?.submittedAt);
  const locked = submitted || limit < 1;
  const remaining = Math.max(0, limit - picked.length);

  const photos = useMemo(() => gallery?.photos || [], [gallery]);

  const closeViewer = useCallback(() => setViewer(null), []);
  const stepViewer = useCallback(
    (delta) =>
      setViewer((current) => {
        if (current === null) return current;
        const next = current + delta;
        return next < 0 || next >= photos.length ? current : next;
      }),
    [photos.length]
  );

  // Escape closes the full view and the arrows move through the folder, the way
  // a photo viewer is expected to behave. The page behind must not scroll while
  // the overlay is up.
  useEffect(() => {
    if (viewer === null) return undefined;
    const onKey = (event) => {
      if (event.key === "Escape") closeViewer();
      if (event.key === "ArrowLeft") stepViewer(-1);
      if (event.key === "ArrowRight") stepViewer(1);
    };
    window.addEventListener("keydown", onKey);
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      window.removeEventListener("keydown", onKey);
      document.body.style.overflow = previous;
    };
  }, [viewer, closeViewer, stepViewer]);

  const persist = useCallback(
    (ids) => {
      if (saveTimer.current) clearTimeout(saveTimer.current);
      saveTimer.current = setTimeout(() => {
        setSaving(true);
        // Saves run one after another, so they reach the server in the order
        // the photos were clicked and the newest set is the one that stays.
        savePending.current = (savePending.current || Promise.resolve())
          .then(() => galleryApi.saveSelection(token, ids))
          .then(() => setNotice({ type: "", text: "" }))
          .catch((error) =>
            setNotice({
              type: "error",
              text: error.response?.data?.message || "Pilihan belum tersimpan. Coba lagi.",
            })
          )
          .finally(() => setSaving(false));
      }, 350);
    },
    [token]
  );

  const toggle = (id) => {
    if (locked || submitting) return;
    setNotice({ type: "", text: "" });
    setPicked((current) => {
      let next;
      if (current.includes(id)) {
        next = current.filter((item) => item !== id);
      } else {
        if (current.length >= limit) {
          setNotice({
            type: "error",
            text: `Paket Anda dapat memilih ${limit} foto. Batalkan satu pilihan untuk memilih yang lain.`,
          });
          return current;
        }
        next = [...current, id];
      }
      persist(next);
      return next;
    });
  };

  const submit = async () => {
    if (submitting || picked.length === 0) return;
    setSubmitting(true);
    setNotice({ type: "", text: "" });
    try {
      // Flush the debounced save first: submit copies the rows the server has,
      // so a pick made in the last 350 ms would otherwise be left out.
      if (saveTimer.current) {
        clearTimeout(saveTimer.current);
        saveTimer.current = null;
      }
      if (savePending.current) await savePending.current;
      await galleryApi.saveSelection(token, picked);
      const { data } = await galleryApi.submitSelection(token);
      setGallery((current) => ({ ...current, submittedAt: data.submittedAt }));
      setNotice({
        type: "success",
        text: data?.message || "Pilihan foto sudah kami terima. Terima kasih.",
      });
    } catch (error) {
      setNotice({
        type: "error",
        text: error.response?.data?.message || "Pilihan foto gagal dikirim. Coba lagi sebentar lagi.",
      });
    } finally {
      setSubmitting(false);
    }
  };

  if (state.loading) return <LoadingState />;

  if (state.error) {
    return (
      <div className="pt-28 sm:pt-32 pb-24 max-w-3xl mx-auto px-5 sm:px-6">
        <div className="border border-gray-200 p-8 text-center">
          <Lock size={28} className="mx-auto text-gray-400" strokeWidth={1.5} />
          <h1 className="mt-4 text-lg font-semibold text-gray-900">Halaman foto tidak dapat dibuka</h1>
          <p className="mt-2 text-sm text-gray-600">{state.error}</p>
          <Link
            to="/contact"
            className="mt-6 inline-block border border-gray-900 px-5 py-2.5 text-xs font-semibold tracking-widest text-gray-900 hover:bg-gray-900 hover:text-white transition-colors"
          >
            HUBUNGI ELVOVE
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="pt-28 sm:pt-32 pb-24 max-w-7xl mx-auto px-5 sm:px-6">
      <SectionHeading
        as="h1"
        frame="PILIH FOTO"
        title={`Halo, ${gallery.clientName}`}
        subtitle={
          limit > 0
            ? `Pilih maksimal ${limit} foto untuk kami edit dari paket ${gallery.packageTitle || "Anda"}.`
            : "Paket pemesanan ini tidak mencakup pemilihan foto."
        }
      />

      {limit > 0 && !submitted && (
        <div className="mt-8 flex flex-wrap items-center gap-3 border border-gray-200 px-4 py-3 text-sm text-gray-700">
          <span className="font-semibold text-gray-900">
            {picked.length} / {limit} foto dipilih
          </span>
          <span className="text-gray-400">·</span>
          <span>{remaining > 0 ? `Sisa ${remaining} foto` : "Pilihan sudah penuh"}</span>
          {saving && (
            <span className="flex items-center gap-2 text-xs text-gray-500">
              <LoaderCircle size={13} className="animate-spin" />
              Menyimpan pilihan...
            </span>
          )}
        </div>
      )}

      {submitted && (
        <div className="mt-8 border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-800">
          Pilihan foto Anda sudah kami terima pada{" "}
          {new Date(gallery.submittedAt).toLocaleString("id-ID", {
            day: "2-digit",
            month: "long",
            year: "numeric",
          })}
          . Tim kami lanjut mengedit foto yang dipilih.
        </div>
      )}

      {notice.text && (
        <p
          role="status"
          className={`mt-4 border px-4 py-3 text-sm ${
            notice.type === "success"
              ? "border-green-200 bg-green-50 text-green-800"
              : "border-red-200 bg-red-50 text-red-800"
          }`}
        >
          {notice.text}
        </p>
      )}

      <div className="mt-8 grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3 sm:gap-4">
        {photos.map((photo, index) => {
          const chosen = picked.indexOf(photo.id) !== -1;
          return (
            <div
              key={photo.id}
              className={`group relative aspect-square overflow-hidden border-2 bg-gray-100 ${
                chosen ? "border-[#E11D48]" : "border-transparent"
              }`}
            >
              <button
                type="button"
                onClick={() => toggle(photo.id)}
                disabled={locked}
                aria-pressed={chosen}
                className={`block h-full w-full ${locked ? "cursor-default" : "cursor-pointer"}`}
              >
                <PreviewImage
                  token={token}
                  id={photo.id}
                  alt={photo.name}
                  width={400}
                  className="h-full w-full object-cover"
                />
                {chosen && (
                  <span className="absolute left-2 top-2 flex h-7 w-7 items-center justify-center bg-[#E11D48] text-xs font-bold text-white">
                    {picked.indexOf(photo.id) + 1}
                  </span>
                )}
                {!locked && (
                  <span
                    className={`absolute inset-0 flex items-center justify-center bg-black/40 opacity-0 transition-opacity group-hover:opacity-100 ${
                      chosen ? "group-hover:opacity-100" : ""
                    }`}
                  >
                    <span className="flex h-9 w-9 items-center justify-center rounded-full bg-white text-gray-900">
                      {chosen ? <X size={18} /> : <Check size={18} />}
                    </span>
                  </span>
                )}
              </button>
              <button
                type="button"
                onClick={() => setViewer(index)}
                aria-label={`Lihat ${photo.name} ukuran penuh`}
                className="absolute bottom-2 right-2 z-10 flex h-9 w-9 items-center justify-center bg-black/60 text-white transition-colors hover:bg-[#E11D48]"
              >
                <Maximize2 size={16} />
              </button>
            </div>
          );
        })}
      </div>

      {photos.length === 0 && (
        <p className="mt-10 text-center text-sm text-gray-600">
          Foto untuk pemesanan ini belum diunggah. Silakan cek kembali nanti.
        </p>
      )}

      {limit > 0 && !submitted && photos.length > 0 && (
        <div className="mt-10 flex flex-col items-center gap-3 border-t border-gray-200 pt-8">
          <button
            type="button"
            onClick={submit}
            disabled={submitting || picked.length === 0 || saving}
            className="inline-flex items-center gap-2 bg-gray-900 px-8 py-3 text-xs font-semibold tracking-widest text-white transition-colors hover:bg-[#E11D48] disabled:cursor-not-allowed disabled:opacity-40"
          >
            {submitting ? <LoaderCircle size={16} className="animate-spin" /> : <Check size={16} />}
            {submitting ? "MENGIRIM..." : "KIRIM PILIHAN FOTO"}
          </button>
          <p className="max-w-md text-center text-xs text-gray-500">
            Setelah dikirim, pilihan tidak dapat diubah lagi. Pilih {limit} foto untuk hasil terbaik —
            Anda masih bisa mengirim walau belum penuh.
          </p>
        </div>
      )}

      {viewer !== null && photos[viewer] && (
        <div
          role="dialog"
          aria-modal="true"
          aria-label={photos[viewer].name}
          onClick={closeViewer}
          className="fixed inset-0 z-50 flex flex-col bg-black/95"
        >
          <div className="flex items-center justify-between gap-4 px-4 py-3 text-white">
            <span className="min-w-0 truncate text-xs tracking-widest text-white/70">
              {viewer + 1} / {photos.length}
            </span>
            <span className="min-w-0 flex-1 truncate text-center text-sm">{photos[viewer].name}</span>
            <button
              type="button"
              onClick={closeViewer}
              aria-label="Tutup"
              autoFocus
              className="flex h-9 w-9 shrink-0 items-center justify-center text-white/80 transition-colors hover:text-white"
            >
              <X size={20} />
            </button>
          </div>

          <div className="relative flex min-h-0 flex-1 items-center justify-center px-2 pb-4 sm:px-14">
            <PreviewImage
              token={token}
              id={photos[viewer].id}
              alt={photos[viewer].name}
              width={1600}
              className="max-h-full max-w-full object-contain"
              onClick={(event) => event.stopPropagation()}
              fallback={
                <div className="flex h-64 w-64 items-center justify-center text-white/70">
                  <LoaderCircle size={28} className="animate-spin" />
                </div>
              }
            />
            {viewer > 0 && (
              <button
                type="button"
                onClick={(event) => {
                  event.stopPropagation();
                  stepViewer(-1);
                }}
                aria-label="Foto sebelumnya"
                className="absolute left-1 top-1/2 flex h-11 w-11 -translate-y-1/2 items-center justify-center bg-black/40 text-white transition-colors hover:bg-[#E11D48] sm:left-2"
              >
                <ChevronLeft size={24} />
              </button>
            )}
            {viewer < photos.length - 1 && (
              <button
                type="button"
                onClick={(event) => {
                  event.stopPropagation();
                  stepViewer(1);
                }}
                aria-label="Foto berikutnya"
                className="absolute right-1 top-1/2 flex h-11 w-11 -translate-y-1/2 items-center justify-center bg-black/40 text-white transition-colors hover:bg-[#E11D48] sm:right-2"
              >
                <ChevronRight size={24} />
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

/**
 * One photo on the selection page.
 *
 * <p>The picture Drive keeps for the photo is shown first: it is already
 * resized and named by the file id, so a folder of three hundred photos fills in
 * at once and the server converts nothing. A photo whose thumbnail fails falls
 * back to the page's own WebP preview one request at a time, so a broken
 * thumbnail never costs the whole grid.</p>
 */
function PreviewImage({ token, id, alt, width, className, fallback, onClick }) {
  const [status, setStatus] = useState("thumbnail");
  if (status === "both") {
    return fallback ?? <span className={className} aria-hidden="true" />;
  }
  const src =
    status === "thumbnail"
      ? galleryApi.driveThumbnail(id, width)
      : `${API_BASE_URL}${galleryApi.photoUrl(token, id)}`;
  return (
    <img
      src={src}
      alt={alt}
      loading={width > 400 ? undefined : "lazy"}
      decoding="async"
      onClick={onClick}
      className={className}
      onError={() => setStatus(status === "thumbnail" ? "preview" : "both")}
    />
  );
}

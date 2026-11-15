import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { Check, LoaderCircle, Lock, X } from "lucide-react";
import { galleryApi } from "../services/galleryApi";
import LoadingState from "../components/LoadingState";
import SectionHeading from "../components/SectionHeading";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "";

/**
 * The page a client opens from the e-mail link to pick the photos to edit.
 *
 * <p>The token in the address is the whole credential, so the page shows only
 * what that one link unlocks. The picked set is saved as the client clicks; the
 * final "kirim" copies the chosen originals into the mustedit folder on Drive
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
  const saveTimer = useRef(null);

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

  // A pending save has to go out before the tab closes, or the last click is
  // lost when the client closes the window straight after picking.
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

  const persist = useCallback(
    (ids) => {
      if (saveTimer.current) clearTimeout(saveTimer.current);
      saveTimer.current = setTimeout(() => {
        setSaving(true);
        galleryApi
          .saveSelection(token, ids)
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
    if (locked) return;
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
        {photos.map((photo) => {
          const index = picked.indexOf(photo.id);
          const chosen = index !== -1;
          return (
            <button
              key={photo.id}
              type="button"
              onClick={() => toggle(photo.id)}
              disabled={locked}
              aria-pressed={chosen}
              className={`group relative aspect-square overflow-hidden border-2 bg-gray-100 transition-colors ${
                chosen ? "border-[#E11D48]" : "border-transparent"
              } ${locked ? "cursor-default" : "cursor-pointer"}`}
            >
              <img
                src={`${API_BASE_URL}${galleryApi.photoUrl(token, photo.id)}`}
                alt={photo.name}
                loading="lazy"
                decoding="async"
                className="h-full w-full object-cover"
              />
              {chosen && (
                <span className="absolute left-2 top-2 flex h-7 w-7 items-center justify-center bg-[#E11D48] text-xs font-bold text-white">
                  {index + 1}
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
    </div>
  );
}

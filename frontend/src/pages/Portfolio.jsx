import { useEffect, useState } from "react";
import { publicApi } from "../services/publicApi";
import SectionHeading from "../components/SectionHeading";
import LoadingState from "../components/LoadingState";
import EmptyState from "../components/EmptyState";
import { X } from "lucide-react";
import { getImageUrl } from "../utils/imageUrl";

export default function Portfolio() {
  const [categories, setCategories] = useState([]);
  const [photos, setPhotos] = useState([]);
  const [activeCategory, setActiveCategory] = useState(null);
  const [lightboxPhoto, setLightboxPhoto] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([publicApi.getCategories(), publicApi.getPhotos()])
      .then(([catRes, photoRes]) => {
        setCategories(catRes.data);
        setPhotos(photoRes.data);
      })
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (!lightboxPhoto) return;
    const onKeyDown = (e) => {
      if (e.key === "Escape") setLightboxPhoto(null);
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [lightboxPhoto]);

  const filtered = activeCategory
    ? photos.filter((p) => p.categoryId === activeCategory)
    : photos;

  const filterClass = (active) =>
    `px-4 py-2 text-sm border transition-colors ${
      active
        ? "border-[#E11D48] text-[#E11D48]"
        : "border-gray-200 text-gray-700 hover:border-gray-400 hover:text-gray-900"
    }`;

  return (
    <div className="pt-28 sm:pt-32 pb-20 sm:pb-24 max-w-7xl mx-auto px-5 sm:px-6">
      <SectionHeading as="h1" title="Portofolio" subtitle="Mengabadikan cerita, ikatan dan kebersamaan dalam setiap bingkai." />

      <div className="flex flex-wrap gap-2 sm:gap-3 mb-10">
        <button
          type="button"
          onClick={() => setActiveCategory(null)}
          aria-pressed={activeCategory === null}
          className={filterClass(activeCategory === null)}
        >
          Semua
        </button>
        {categories.map((c) => (
          <button
            key={c.id}
            type="button"
            onClick={() => setActiveCategory(c.id)}
            aria-pressed={activeCategory === c.id}
            className={filterClass(activeCategory === c.id)}
          >
            {c.name}
          </button>
        ))}
      </div>

      {loading ? (
        <LoadingState />
      ) : filtered.length === 0 ? (
        <EmptyState message="Belum ada foto dalam kategori ini." />
      ) : (
        <div className="columns-2 md:columns-3 gap-3 sm:gap-4 space-y-3 sm:space-y-4">
          {filtered.map((p) => (
            <button
              key={p.id}
              type="button"
              onClick={() => setLightboxPhoto(p)}
              aria-label={`Perbesar foto: ${p.title}`}
              className="block w-full break-inside-avoid group relative overflow-hidden"
            >
              <img
                src={getImageUrl(p.thumbnailUrl || p.imageUrl)}
                alt={p.title}
                loading="lazy"
                decoding="async"
                className="w-full object-cover group-hover:scale-105 transition-transform duration-500"
              />
              <div className="absolute inset-0 bg-white/0 group-hover:bg-white/30 transition-colors" />
            </button>
          ))}
        </div>
      )}

      {lightboxPhoto && (
        <div
          role="dialog"
          aria-modal="true"
          aria-label="Pratinjau foto"
          className="fixed inset-0 z-[60] bg-white/95 flex items-center justify-center p-4 sm:p-6"
          onClick={() => setLightboxPhoto(null)}
        >
          <button
            type="button"
            className="absolute top-4 right-4 sm:top-6 sm:right-6 p-2 text-gray-900"
            onClick={() => setLightboxPhoto(null)}
            aria-label="Tutup pratinjau"
          >
            <X size={28} aria-hidden="true" />
          </button>
          <img
            src={getImageUrl(lightboxPhoto.previewUrl || lightboxPhoto.thumbnailUrl || lightboxPhoto.imageUrl)}
            alt={lightboxPhoto.title}
            className="max-h-[85vh] max-w-full object-contain"
            onClick={(e) => e.stopPropagation()}
          />
        </div>
      )}
    </div>
  );
}

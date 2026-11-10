import { useEffect, useState } from "react";
import { publicApi } from "../services/publicApi";
import SectionHeading from "../components/SectionHeading";
import LoadingState from "../components/LoadingState";
import { getImageUrl } from "../utils/imageUrl";

export default function LatestGallery() {
  const [photos, setPhotos] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    publicApi.getPhotos({ latest: true }).then((res) => setPhotos(res.data)).finally(() => setLoading(false));
  }, []);

  return (
    <div className="pt-28 sm:pt-32 pb-20 sm:pb-24 max-w-7xl mx-auto px-5 sm:px-6">
      <SectionHeading as="h1" frame="FRAME 070" title="Karya Terbaru" subtitle="Baru diserahkan, langsung dari sesi terakhir." />

      {loading ? (
        <LoadingState />
      ) : (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3 sm:gap-4">
          {photos.map((p) => (
            <div key={p.id} className="aspect-square overflow-hidden group">
              <img
                src={getImageUrl(p.thumbnailUrl || p.imageUrl)}
                alt={p.title}
                loading="lazy"
                decoding="async"
                className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-500"
              />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

import homePhotos from "../assets/home-photos.webp";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { publicApi } from "../services/publicApi";
import SectionHeading from "../components/SectionHeading";
import LoadingState from "../components/LoadingState";
import { getImageUrl } from "../utils/imageUrl";

export default function Home() {
  const [photos, setPhotos] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    publicApi
      .getPhotos({ featured: true })
      .then((photosRes) => setPhotos(photosRes.data.slice(0, 6)))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div>
      {/* Hero */}
      <section className="relative min-h-[560px] h-[88svh] sm:h-screen sm:min-h-[640px] flex items-end overflow-hidden">
        <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-black/40 to-black/50" />

        <div
          className="absolute inset-0 bg-cover bg-center scale-105"
          style={{
            backgroundImage: `url("${homePhotos}")`,
          }}
        />

        <div className="relative z-10 max-w-7xl mx-auto px-5 sm:px-6 pb-14 sm:pb-20 w-full">
          <p
            className="frame-number text-xs text-white"
            data-frame="Elvove"
          />

          <h1 className="font-['IM_Fell_English'] text-4xl sm:text-5xl md:text-7xl leading-[1.08] text-white max-w-3xl break-words">
            Melampaui momen, menjadi{" "}
            <span className="italic text-[#E11D48]">kenangan.</span>
          </h1>

          <p className="mt-5 sm:mt-6 text-white/90 max-w-md text-base leading-relaxed">
            Fotografi yang menangkap suasana, emosi, dan detail dari setiap
            momen.
          </p>

          <div className="mt-7 sm:mt-8 flex flex-col sm:flex-row flex-wrap gap-3 sm:gap-4">
            <Link
              to="/portfolio"
              className="bg-[#E11D48] text-white px-7 py-3 text-sm tracking-wide text-center hover:bg-[#BE123C] transition-colors"
            >
              Lihat Portofolio
            </Link>

            <Link
              to="/booking"
              className="border border-white text-white px-7 py-3 text-sm tracking-wide text-center hover:bg-white hover:text-black transition-colors"
            >
              Pesan Sekarang
            </Link>
          </div>
        </div>
      </section>

      {/* Karya pilihan */}
      <section className="max-w-7xl mx-auto px-5 sm:px-6 py-16 sm:py-24">
        <SectionHeading title="Karya Pilihan" subtitle="Cuplikan dari sesi terbaru kami." />
        {loading ? (
          <LoadingState />
        ) : (
          <div className="grid grid-cols-2 md:grid-cols-3 gap-3 sm:gap-4 auto-rows-[160px] sm:auto-rows-[220px]">
            {photos.map((p, i) => (
              <div
                key={p.id}
                className={`overflow-hidden group relative ${i === 0 ? "col-span-2 row-span-2 md:col-span-2 md:row-span-2" : ""}`}
              >
                <img
                  src={getImageUrl(p.thumbnailUrl || p.imageUrl)}
                  alt={p.title}
                  loading="lazy"
                  decoding="async"
                  className="w-full h-full object-cover group-hover:scale-110 transition-transform duration-700"
                />
                <div className="absolute inset-0 bg-white/0 group-hover:bg-white/40 transition-colors flex items-end p-4 opacity-0 group-hover:opacity-100">
                  <p className="text-gray-900 text-sm">{p.title}</p>
                </div>
              </div>
            ))}
          </div>
        )}
        <div className="text-center mt-10 sm:mt-12">
          <Link to="/portfolio" className="text-[#E11D48] text-sm border-b border-[#E11D48] pb-1 hover:text-[#BE123C] hover:border-[#BE123C] transition-colors">
            Lihat seluruh portofolio
          </Link>
        </div>
      </section>

      {/* CTA */}
      <section className="max-w-5xl mx-auto px-5 sm:px-6 py-16 sm:py-24 text-center">
        <h2 className="font-display text-3xl md:text-4xl text-gray-900 mb-4">
          Sudah punya rencana?
        </h2>
        <p className="text-gray-700 mb-8">Wujudkan konsep foto Anda bersama kami.</p>
        <Link to="/booking" className="bg-[#E11D48] text-white px-8 py-3 text-sm tracking-wide inline-block hover:bg-[#BE123C] transition-colors">
          Mulai Pemesanan
        </Link>
      </section>
    </div>
  );
}

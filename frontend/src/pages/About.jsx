import { useEffect, useState } from "react";
import { publicApi } from "../services/publicApi";
import SectionHeading from "../components/SectionHeading";
import LoadingState from "../components/LoadingState";
import { getImageUrl } from "../utils/imageUrl";

export default function About() {
  const [about, setAbout] = useState(null);

  useEffect(() => {
    publicApi
      .getAbout()
      .then((res) => setAbout(res.data || {}))
      .catch(() => setAbout({}));
  }, []);

  if (!about) return <LoadingState />;

  const story = (about.story || "").trim();
  const imageUrl = about.imageUrl
    ? getImageUrl(about.imageUrl)
    : "";

  return (
    <div className="pt-28 sm:pt-32 pb-20 sm:pb-24 max-w-7xl mx-auto px-5 sm:px-6">
      <SectionHeading as="h1" title="Tentang Kami" />

      <div className="grid md:grid-cols-2 gap-10 md:gap-14 items-start">
        <div>
          {story ? (
            <p className="whitespace-pre-line text-justify text-base leading-7 text-gray-700">
              {story}
            </p>
          ) : (
            <p className="text-justify text-base leading-7 text-gray-700">
              Cerita belum ditambahkan.
            </p>
          )}
        </div>

        <div className="w-full max-w-md">
          {imageUrl ? (
            <img
              src={imageUrl}
              alt="Foto Elvove"
              loading="lazy"
              decoding="async"
              className="aspect-[4/5] w-full border border-gray-200 object-cover"
            />
          ) : (
            <div className="flex aspect-[4/5] w-full items-center justify-center border border-gray-200 bg-gray-50 px-6 text-center text-sm text-gray-700">
              Foto belum tersedia.
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

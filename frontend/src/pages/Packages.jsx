import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Check } from "lucide-react";
import { publicApi } from "../services/publicApi";
import SectionHeading from "../components/SectionHeading";
import LoadingState from "../components/LoadingState";

export default function Packages() {
  const [packages, setPackages] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    publicApi
      .getPackages()
      .then((res) => setPackages(res.data))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="pt-28 sm:pt-32 pb-20 sm:pb-24 max-w-7xl mx-auto px-5 sm:px-6">
      <SectionHeading
        as="h1"
        title="Paket"
        subtitle="Tanpa biaya tersembunyi, semua transparan."
      />

      {loading ? (
        <LoadingState />
      ) : (
        <div className="grid md:grid-cols-3 gap-6 md:gap-8">
          {packages.map((pkg) => (
            <div
              key={pkg.id}
              className={`p-6 sm:p-8 border flex flex-col ${
                pkg.highlighted
                  ? "border-[#E11D48] bg-white relative"
                  : "border-gray-200"
              }`}
            >
              {pkg.highlighted && (
                <span className="absolute -top-3 left-6 sm:left-8 bg-[#E11D48] text-white text-xs px-3 py-1 tracking-widest">
                  PALING DIMINATI
                </span>
              )}

              <h2 className="font-display text-2xl text-gray-900 mb-1 break-words">
                {pkg.title}
              </h2>

              <p className="font-display text-3xl text-[#E11D48] mb-6 break-words">
                Rp{Number(pkg.price).toLocaleString("id-ID")}
              </p>

              <ul className="space-y-3 flex-1 mb-8">
                {pkg.features.map((f, i) => (
                  <li
                    key={i}
                    className="flex items-start gap-2 text-sm text-gray-700 leading-relaxed"
                  >
                    <Check
                      size={16}
                      aria-hidden="true"
                      className="text-[#E11D48] mt-0.5 shrink-0"
                    />
                    <span>{f}</span>
                  </li>
                ))}
              </ul>

              <Link
                to={`/booking?packageId=${pkg.id}`}
                className={`text-center py-3 px-4 text-sm tracking-wide transition-colors ${
                  pkg.highlighted
                    ? "bg-[#E11D48] text-white hover:bg-[#BE123C]"
                    : "border border-gray-400 text-gray-900 hover:border-[#E11D48] hover:text-[#E11D48]"
                }`}
              >
                Pilih {pkg.title}
              </Link>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

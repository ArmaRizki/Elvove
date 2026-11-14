import { useEffect, useMemo, useState } from "react";
import { publicApi } from "../services/publicApi";
import SectionHeading from "../components/SectionHeading";
import { splitHtmlSections, toHtml } from "../utils/richText";

export default function TermsAndConditions() {
  const [terms, setTerms] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    publicApi
      .getTerms()
      .then((res) => {
        setTerms(res.data);
      })
      .catch(() => {
        setError("Syarat & Ketentuan tidak dapat dimuat saat ini.");
      })
      .finally(() => {
        setLoading(false);
      });
  }, []);

  // Each top-level heading starts a new column, as the Markdown sections did.
  const sections = useMemo(() => splitHtmlSections(toHtml(terms?.content)), [terms]);

  return (
    <main className="pt-28 sm:pt-32 pb-20 sm:pb-24 max-w-7xl mx-auto px-5 sm:px-6">
      {loading ? null : error ? (
        <div role="alert" className="max-w-3xl border border-[#E11D48] p-5 sm:p-6 text-sm text-gray-900">
          {error}
          <button
            type="button"
            onClick={() => window.location.reload()}
            className="ml-2 text-[#E11D48] underline underline-offset-4 hover:text-[#BE123C]"
          >
            Coba lagi
          </button>
        </div>
      ) : terms ? (
        <div>
          <SectionHeading as="h1" title={terms.title} />

          {terms.introduction && (
            <p className="mb-10 max-w-3xl text-base leading-7 text-gray-700 break-words">
              {terms.introduction}
            </p>
          )}

          <div className="grid grid-cols-1 gap-x-10 gap-y-6 md:grid-cols-2 md:gap-x-12 md:gap-y-8 xl:gap-x-16">
            {sections.map((section, index) => (
              <section
                key={index}
                className="rich-text min-w-0 border-t border-gray-200 pt-6 md:pt-8"
                dangerouslySetInnerHTML={{ __html: section }}
              />
            ))}
          </div>
        </div>
      ) : null}
    </main>
  );
}

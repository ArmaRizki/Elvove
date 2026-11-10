import SectionHeading from "../components/SectionHeading";

export default function Privacy() {
  return (
    <div className="pt-28 sm:pt-32 pb-20 sm:pb-24 max-w-3xl mx-auto px-5 sm:px-6">
      <SectionHeading as="h1" title="Kebijakan Privasi" />
      <div className="space-y-4 text-gray-700 text-sm leading-relaxed">
        <p>
          Kami hanya mengumpulkan informasi yang Anda kirimkan melalui formulir
          pemesanan — nama, email, nomor telepon, dan detail acara — untuk
          menanggapi permintaan Anda dan merencanakan sesi Anda.
        </p>
        <p>
          Kami tidak pernah menjual atau membagikan informasi Anda kepada pihak
          ketiga. Data Anda disimpan dengan aman dan hanya digunakan untuk
          berkomunikasi dengan Anda mengenai pemesanan Anda.
        </p>
        <p>
          Anda dapat meminta kami menghapus informasi Anda kapan saja dengan
          menghubungi kami secara langsung.
        </p>
      </div>
    </div>
  );
}

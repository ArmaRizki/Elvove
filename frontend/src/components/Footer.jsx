
import { Link } from "react-router-dom";
import { useEffect, useState } from "react";
import {
  Users,
  Play,
  BriefcaseBusiness,
} from "lucide-react";
import { publicApi } from "../services/publicApi";

// The three brand marks lucide does not carry. Paths and viewBox come from the
// icons react-icons used to supply, so the footer looks the same while the
// package and its ~30 MB of icon families stay out of the build.
const BRAND_ICONS = {
  instagram: {
    viewBox: "0 0 448 512",
    path: "M224.1 141c-63.6 0-114.9 51.3-114.9 114.9s51.3 114.9 114.9 114.9S339 319.5 339 255.9 287.7 141 224.1 141zm0 189.6c-41.1 0-74.7-33.5-74.7-74.7s33.5-74.7 74.7-74.7 74.7 33.5 74.7 74.7-33.6 74.7-74.7 74.7zm146.4-194.3c0 14.9-12 26.8-26.8 26.8-14.9 0-26.8-12-26.8-26.8s12-26.8 26.8-26.8 26.8 12 26.8 26.8zm76.1 27.2c-1.7-35.9-9.9-67.7-36.2-93.9-26.2-26.2-58-34.4-93.9-36.2-37-2.1-147.9-2.1-184.9 0-35.8 1.7-67.6 9.9-93.9 36.1s-34.4 58-36.2 93.9c-2.1 37-2.1 147.9 0 184.9 1.7 35.9 9.9 67.7 36.2 93.9s58 34.4 93.9 36.2c37 2.1 147.9 2.1 184.9 0 35.9-1.7 67.7-9.9 93.9-36.2 26.2-26.2 34.4-58 36.2-93.9 2.1-37 2.1-147.8 0-184.8zM398.8 388c-7.8 19.6-22.9 34.7-42.6 42.6-29.5 11.7-99.5 9-132.1 9s-102.7 2.6-132.1-9c-19.6-7.8-34.7-22.9-42.6-42.6-11.7-29.5-9-99.5-9-132.1s-2.6-102.7 9-132.1c7.8-19.6 22.9-34.7 42.6-42.6 29.5-11.7 99.5-9 132.1-9s102.7-2.6 132.1 9c19.6 7.8 34.7 22.9 42.6 42.6 11.7 29.5 9 99.5 9 132.1s2.7 102.7-9 132.1z",
  },
  threads: {
    viewBox: "0 0 24 24",
    path: "M12.186 24h-.007c-3.581-.024-6.334-1.205-8.184-3.509C2.35 18.44 1.5 15.586 1.472 12.01v-.017c.03-3.579.879-6.43 2.525-8.482C5.845 1.205 8.6.024 12.18 0h.014c2.746.02 5.043.725 6.826 2.098 1.677 1.29 2.858 3.13 3.509 5.467l-2.04.569c-1.104-3.96-3.898-5.984-8.304-6.015-2.91.022-5.11.936-6.54 2.717C4.307 6.504 3.616 8.914 3.589 12c.027 3.086.718 5.496 2.057 7.164 1.43 1.783 3.631 2.698 6.54 2.717 2.623-.02 4.358-.631 5.8-2.045 1.647-1.613 1.618-3.593 1.09-4.798-.31-.71-.873-1.3-1.634-1.75-.192 1.352-.622 2.446-1.284 3.272-.886 1.102-2.14 1.704-3.73 1.79-1.202.065-2.361-.218-3.259-.801-1.063-.689-1.685-1.74-1.752-2.964-.065-1.19.408-2.285 1.33-3.082.88-.76 2.119-1.207 3.583-1.291a13.853 13.853 0 0 1 3.02.142c-.126-.742-.375-1.332-.75-1.757-.513-.586-1.308-.883-2.359-.89h-.029c-.844 0-1.992.232-2.721 1.32L7.734 7.847c.98-1.454 2.568-2.256 4.478-2.256h.044c3.194.02 5.097 1.975 5.287 5.388.108.046.216.094.321.142 1.49.7 2.58 1.761 3.154 3.07.797 1.82.871 4.79-1.548 7.158-1.85 1.81-4.094 2.628-7.277 2.65Zm1.003-11.69c-.242 0-.487.007-.739.021-1.836.103-2.98.946-2.916 2.143.067 1.256 1.452 1.839 2.784 1.767 1.224-.065 2.818-.543 3.086-3.71a10.5 10.5 0 0 0-2.215-.221z",
  },
  tiktok: {
    viewBox: "0 0 24 24",
    path: "M12.525.02c1.31-.02 2.61-.01 3.91-.02.08 1.53.63 3.09 1.75 4.17 1.12 1.11 2.7 1.62 4.24 1.79v4.03c-1.44-.05-2.89-.35-4.2-.97-.57-.26-1.1-.59-1.62-.93-.01 2.92.01 5.84-.02 8.75-.08 1.4-.54 2.79-1.35 3.94-1.31 1.92-3.58 3.17-5.91 3.21-1.43.08-2.86-.31-4.08-1.03-2.02-1.19-3.44-3.37-3.65-5.71-.02-.5-.03-1-.01-1.49.18-1.9 1.12-3.72 2.58-4.96 1.66-1.44 3.98-2.13 6.15-1.72.02 1.48-.04 2.96-.04 4.44-.99-.32-2.15-.23-3.02.37-.63.41-1.11 1.04-1.36 1.75-.21.51-.15 1.07-.14 1.61.24 1.64 1.82 3.02 3.5 2.87 1.12-.01 2.19-.66 2.77-1.61.19-.33.4-.67.41-1.06.1-1.79.06-3.57.07-5.36.01-4.03-.01-8.05.02-12.07z",
  },
};

function BrandIcon({ name, size = 19 }) {
  const icon = BRAND_ICONS[name];
  return (
    <svg
      width={size}
      height={size}
      viewBox={icon.viewBox}
      fill="currentColor"
      aria-hidden="true"
    >
      <path d={icon.path} />
    </svg>
  );
}

export default function Footer() {
  const [social, setSocial] = useState(null);

  useEffect(() => {
    publicApi.getSocialLinks()
      .then((res) => setSocial(res.data))
      .catch(() => {});
  }, []);

  const socialItems = [
    {
      name: "Instagram",
      url: social?.instagram,
      icon: <BrandIcon name="instagram" />,
    },
    {
      name: "Threads",
      url: social?.threads,
      icon: <BrandIcon name="threads" />,
    },
    {
      name: "TikTok",
      url: social?.tiktok,
      icon: <BrandIcon name="tiktok" />,
    },
    {
      name: "Facebook",
      url: social?.facebook,
      icon: <Users size={19} />,
    },
    {
      name: "YouTube",
      url: social?.youtube,
      icon: <Play size={19} />,
    },
    {
      name: "LinkedIn",
      url: social?.linkedin,
      icon: <BriefcaseBusiness size={19} />,
    },
  ];

  return (
    <footer className="bg-white border-t border-gray-200">
      <div className="max-w-7xl mx-auto px-6 md:px-10 py-12 md:py-16">
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-10 md:gap-16">

          {/* Brand */}
          <div className="max-w-sm">
            <p className="font-display text-lg text-gray-900 mb-3">
              ELVOVE
            </p>

            <p className="text-sm text-gray-600 leading-relaxed">
              Fotografi yang menangkap suasana, emosi, dan detail dari setiap momen.
            </p>
          </div>

          {/* Tautan Cepat */}
          <div>
            <p className="text-xs tracking-widest text-[#E11D48] mb-5">
              TAUTAN CEPAT
            </p>

            <ul className="space-y-3 text-sm text-gray-600">
              <li>
                <Link to="/portfolio" className="hover:text-[#E11D48] transition-colors">
                  Portofolio
                </Link>
              </li>

              <li>
                <Link to="/about" className="hover:text-[#E11D48] transition-colors">
                  Tentang Kami
                </Link>
              </li>

              <li>
                <Link to="/booking" className="hover:text-[#E11D48] transition-colors">
                  Pesan Sesi
                </Link>
              </li>

              <li>
                <Link to="/terms-and-conditions" className="hover:text-[#E11D48] transition-colors">
                  Syarat &amp; Ketentuan
                </Link>
              </li>
            </ul>
          </div>

          {/* Media Sosial */}
          <div>
            <p className="text-xs tracking-widest text-[#E11D48] mb-5">
              IKUTI KAMI
            </p>

            <div className="flex flex-wrap items-center gap-5">
              {socialItems.map((item) =>
                item.url ? (
                  <a
                    key={item.name}
                    href={item.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    aria-label={item.name}
                    title={item.name}
                    className="text-gray-700 hover:text-[#E11D48] transition-colors duration-200"
                  >
                    {item.icon}
                  </a>
                ) : null
              )}
            </div>
          </div>

        </div>
      </div>

      {/* Divider */}
      <div className="sprocket-rule px-6" />

      {/* Copyright */}
      <div className="max-w-7xl mx-auto px-6 md:px-10 py-6 flex flex-col sm:flex-row justify-between items-center gap-3 text-xs text-gray-500">
        <p>
          &copy; {new Date().getFullYear()} Elvove. Hak cipta dilindungi.
        </p>

        <div className="flex gap-5">
          <Link to="/privacy" className="hover:text-[#E11D48] transition-colors">
            Kebijakan Privasi
          </Link>
        </div>
      </div>
    </footer>
  );
}


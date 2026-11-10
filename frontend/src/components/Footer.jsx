
import { Link } from "react-router-dom";
import { useEffect, useState } from "react";
import {
  Users,
  Play,
  BriefcaseBusiness,
} from "lucide-react";
import { SiThreads, SiTiktok } from "react-icons/si";
import { publicApi } from "../services/publicApi";
import { FaInstagram } from "react-icons/fa";

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
      icon: <FaInstagram size={19} />,
    },
    {
      name: "Threads",
      url: social?.threads,
      icon: <SiThreads size={19} />,
    },
    {
      name: "TikTok",
      url: social?.tiktok,
      icon: <SiTiktok size={19} />,
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


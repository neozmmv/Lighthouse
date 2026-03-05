export default function Hero() {
  return (
    <section
      className="min-h-[82vh] flex items-center px-8 md:px-16 lg:px-24"
      style={{ backgroundColor: "#ffffff", borderBottom: "1px solid #e5e7eb" }}
    >
      <div className="w-full max-w-7xl mx-auto grid grid-cols-1 md:grid-cols-2 gap-12 items-center py-16">

        {/* ── Left: text ── */}
        <div className="flex flex-col gap-6">
          <p className="hero-eyebrow text-xs font-semibold tracking-[0.18em] uppercase text-purple-900">
            You found the LIGHTHOUSE!
          </p>

          <h1
            className="hero-title font-bold leading-tight tracking-tight text-gray-900"
            style={{ fontSize: "clamp(2.4rem, 4.5vw, 3.75rem)" }}
          >
            Your files.<br />
            Your PC.<br />
            Your control.
          </h1>

          <p className="hero-body text-base leading-relaxed text-gray-500 max-w-sm">
            Receive files straight from your own PC. <br/>
            No third parties, no data caps, no compromise.<br/>
            100% private, end-to-end encrypted.
          </p>

          <div className="hero-actions flex items-center gap-3 flex-wrap">
            <a
              href="#upload"
              className="inline-flex items-center gap-2 px-5 py-2.5 text-sm font-medium bg-purple-900 rounded-md hover:bg-purple-950 transition-colors"
              style={{ color: "#ffffff" }}
              onClick={e => {
                e.preventDefault();
                document.getElementById("upload")?.scrollIntoView({ behavior: "smooth" });
              }}
            >
              Get started <span aria-hidden>→</span>
            </a>
            <a
              href="https://github.com/neozmmv/lighthouse"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 px-5 py-2.5 text-sm font-medium rounded-md border border-gray-300 hover:bg-gray-50 transition-colors"
              style={{ color: "#374151" }}
            >
              Host your own
            </a>
          </div>
        </div>

        {/* ── Right: waves ── */}
        <div className="hidden md:flex items-end h-[420px] overflow-hidden rounded-xl" style={{ backgroundColor: "#f5f3ff" }}>
          <svg
            viewBox="0 0 600 420"
            xmlns="http://www.w3.org/2000/svg"
            preserveAspectRatio="none"
            className="w-full h-full"
            aria-hidden="true"
            overflow="hidden"
          >
            {/* Wave 4 — wide, slow, faint (background) */}
            <g>
              <path
                d="M 0 230 C 75 200, 225 260, 300 230 C 375 200, 525 260, 600 230 C 675 200, 825 260, 900 230 C 975 200, 1125 260, 1200 230 L 1200 420 L 0 420 Z"
                fill="#7c3aed" fillOpacity="0.05" stroke="#7c3aed" strokeOpacity="0.12" strokeWidth="1.5"
              >
                <animateTransform attributeName="transform" type="translate" from="0 0" to="-600 0" dur="20s" repeatCount="indefinite" />
              </path>
            </g>
            {/* Wave 1 — medium period */}
            <g>
              <path
                d="M 0 295 C 50 255, 150 335, 200 295 C 250 255, 350 335, 400 295 C 450 255, 550 335, 600 295 C 650 255, 750 335, 800 295 C 850 255, 950 335, 1000 295 C 1050 255, 1150 335, 1200 295 L 1200 420 L 0 420 Z"
                fill="#7c3aed" fillOpacity="0.07" stroke="#7c3aed" strokeOpacity="0.20" strokeWidth="1.5"
              >
                <animateTransform attributeName="transform" type="translate" from="0 0" to="-600 0" dur="10s" repeatCount="indefinite" />
              </path>
            </g>
            {/* Wave 2 — shorter period, faster */}
            <g>
              <path
                d="M 0 355 C 38 327, 112 383, 150 355 C 188 327, 262 383, 300 355 C 338 327, 412 383, 450 355 C 488 327, 562 383, 600 355 C 638 327, 712 383, 750 355 C 788 327, 862 383, 900 355 C 938 327, 1012 383, 1050 355 C 1088 327, 1162 383, 1200 355 L 1200 420 L 0 420 Z"
                fill="#6d28d9" fillOpacity="0.09" stroke="#6d28d9" strokeOpacity="0.22" strokeWidth="1"
              >
                <animateTransform attributeName="transform" type="translate" from="0 0" to="-600 0" dur="7s" repeatCount="indefinite" />
              </path>
            </g>
            {/* Wave 3 — slowest, frontmost */}
            <g>
              <path
                d="M 0 395 C 50 377, 150 413, 200 395 C 250 377, 350 413, 400 395 C 450 377, 550 413, 600 395 C 650 377, 750 413, 800 395 C 850 377, 950 413, 1000 395 C 1050 377, 1150 413, 1200 395 L 1200 420 L 0 420 Z"
                fill="#5b21b6" fillOpacity="0.12" stroke="#5b21b6" strokeOpacity="0.28" strokeWidth="1"
              >
                <animateTransform attributeName="transform" type="translate" from="0 0" to="-600 0" dur="14s" repeatCount="indefinite" />
              </path>
            </g>
          </svg>
        </div>

      </div>
    </section>
  );
}
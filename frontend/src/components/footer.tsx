import { DiGithubBadge } from "react-icons/di";

export default function Footer() {
  return (
    <footer
      className="px-8 md:px-16 lg:px-24 py-8 flex items-center justify-between text-sm text-gray-400"
      style={{ borderTop: "1px solid #e5e7eb" }}
    >
      <span className="text-gray-700">
        <span className="font-bold"><span className="text-purple-900">Light</span><span className="text-gray-800">house</span></span>
        {" "}· Self-hosted · Open source · Private
      </span>

      <div className="flex items-center gap-5">
        <span>© {new Date().getFullYear()}</span>
        <a
          href="https://github.com/neozmmv/lighthouse"
          target="_blank"
          rel="noopener noreferrer"
          className="flex items-center gap-1.5 hover:text-gray-600 transition-colors"
        >
          <DiGithubBadge className="w-6 h-6 text-gray-700" />
          <span>GitHub</span>
        </a>
      </div>
    </footer>
  );
}

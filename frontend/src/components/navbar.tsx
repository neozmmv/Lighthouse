import { Link } from "@tanstack/react-router";
export default function Navbar() {
  return (
    <nav className="flex p-4 justify-between items-center border-b border-gray-300 rounded-md shadow-sm sticky">
      <Link to="/">
        <img
          src="/Lighthouse.svg"
          alt="Lighthouse Logo"
          className="sm:h-20 h-18"
        />
      </Link>
      <div>
        <a
          href="http://localhost:4405/files"
          className="inline-flex items-center gap-2 px-5 py-2.5 text-sm font-medium bg-purple-900 rounded-md hover:bg-purple-950 transition-colors"
          style={{ color: "#ffffff" }}
        >
          Download Files (Host Only)
        </a>
      </div>
      {/* <DiGithubBadge className="h-14 w-14 mr-4 text-gray-800" /> */}
    </nav>
  );
}

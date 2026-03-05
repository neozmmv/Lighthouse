import { DiGithubBadge } from "react-icons/di";
export default function Navbar() {
  return (
    <nav className="flex p-4 justify-between items-center border-b border-gray-300 rounded-md shadow-sm sticky">
      <img
        src="/Lighthouse.svg"
        alt="Lighthouse Logo"
        className="sm:h-20 h-18"
      />
      <DiGithubBadge className="h-14 w-14 mr-4" />
    </nav>
  );
}

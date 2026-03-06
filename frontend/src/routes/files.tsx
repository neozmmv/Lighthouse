import { createFileRoute } from "@tanstack/react-router";
import Navbar from "../components/navbar";
import FileList from "../components/file-list";
import Footer from "../components/footer";

export const Route = createFileRoute("/files")({
  component: RouteComponent,
});

function RouteComponent() {
  return (
    <>
      <Navbar />
      <FileList />
      <Footer />
    </>
  );
}

import { createFileRoute } from "@tanstack/react-router";
import Hero from "../components/hero";
import Navbar from "../components/navbar";
import Dropzone from "../components/dropzone";
import Footer from "../components/footer";

export const Route = createFileRoute("/")({
  component: RouteComponent,
});

function RouteComponent() {
  return (
    <>
      <Navbar />
      <Hero />
      <Dropzone />
      <Footer />
    </>
  );
}

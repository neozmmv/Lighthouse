import { createFileRoute } from "@tanstack/react-router";
import { useState, useEffect } from "react";
import axios from "axios";
import Navbar from "../components/navbar";

export const Route = createFileRoute("/")({
  component: RouteComponent,
});

function RouteComponent() {
  return (
    <>
      <Navbar />
    </>
  );
}

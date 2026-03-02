import { createFileRoute } from "@tanstack/react-router";
import { useState, useEffect } from "react";
import axios from "axios";

export const Route = createFileRoute("/fetch")({
  component: RouteComponent,
});

function RouteComponent() {
  const [data, setData] = useState<{ Hello: string }>({ Hello: "" });

  useEffect(() => {
    axios
      .get("/api/")
      .then((response) => response.data)
      .then((data) => setData(data));
  }, []);
  return (
    <div>
      <h1>Fetch</h1>
      <p>{data.Hello}</p>
    </div>
  );
}

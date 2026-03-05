import { useState, useRef, DragEvent, ChangeEvent } from "react";

type DropState = "idle" | "hovering" | "done";

export default function Dropzone() {
  const [state, setState] = useState<DropState>("idle");
  const [file, setFile] = useState<File | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  function handleFile(f: File) {
    setFile(f);
    setState("done");
  }

  function onDragOver(e: DragEvent) {
    e.preventDefault();
    setState("hovering");
  }

  function onDragLeave() {
    setState("idle");
  }

  function onDrop(e: DragEvent) {
    e.preventDefault();
    const f = e.dataTransfer.files[0];
    if (f) handleFile(f);
  }

  function onChange(e: ChangeEvent<HTMLInputElement>) {
    const f = e.target.files?.[0];
    if (f) handleFile(f);
  }

  function reset() {
    setFile(null);
    setState("idle");
    if (inputRef.current) inputRef.current.value = "";
  }

  const isHovering = state === "hovering";
  const isDone = state === "done";

  return (
    <section
      id="upload"
      className="px-8 md:px-16 lg:px-24 py-16"
      style={{ backgroundColor: "#ffffff", borderBottom: "1px solid #e5e7eb" }}
    >
      <div className="w-full max-w-xl mx-auto">
        <h2 className="text-lg font-semibold text-gray-900 mb-6">
          Upload a file
        </h2>

        <div
          className="relative flex flex-col items-center justify-center gap-4 rounded-xl border-2 border-dashed transition-colors cursor-pointer"
          style={{
            minHeight: "380px",
            borderColor: isHovering ? "#7c3aed" : isDone ? "#7c3aed" : "#d1d5db",
            backgroundColor: isHovering ? "#f5f3ff" : isDone ? "#f5f3ff" : "#fafafa",
          }}
          onMouseEnter={() => setState(s => s === "idle" ? "hovering" : s)}
          onMouseLeave={() => setState(s => s === "hovering" ? "idle" : s)}
          onDragOver={onDragOver}
          onDragLeave={onDragLeave}
          onDrop={onDrop}
          onClick={() => !isDone && inputRef.current?.click()}
        >
          <input
            ref={inputRef}
            type="file"
            className="hidden"
            onChange={onChange}
          />

          {!isDone ? (
            <>
              {/* Upload icon */}
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke={isHovering ? "#7c3aed" : "#9ca3af"}
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="w-10 h-10 transition-colors"
                aria-hidden="true"
              >
                <path d="M4 16v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2" />
                <polyline points="16 12 12 8 8 12" />
                <line x1="12" y1="8" x2="12" y2="20" />
              </svg>

              <div className="text-center">
                <p className="text-sm font-medium text-gray-700">
                  Drop a file here, or{" "}
                  <span className="text-purple-700 underline underline-offset-2">
                    browse
                  </span>
                </p>
                <p className="text-xs text-gray-400 mt-1">Any file type accepted</p>
              </div>
            </>
          ) : (
            <>
              {/* File ready state */}
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="#7c3aed"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="w-10 h-10"
                aria-hidden="true"
              >
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
                <polyline points="14 2 14 8 20 8" />
              </svg>

              <div className="text-center">
                <p className="text-sm font-semibold text-gray-900">{file?.name}</p>
                <p className="text-xs text-gray-400 mt-0.5">
                  {file ? (file.size / 1024).toFixed(1) + " KB" : ""}
                </p>
              </div>

              <button
                className="text-xs text-gray-400 hover:text-gray-600 underline underline-offset-2 transition-colors"
                onClick={(e) => { e.stopPropagation(); reset(); }}
              >
                Remove
              </button>
            </>
          )}
        </div>

        {isDone && (
          <div className="mt-4 flex justify-end">
            <button
              className="px-5 py-2.5 text-sm font-medium text-white rounded-md transition-colors"
              style={{ backgroundColor: "#6d28d9" }}
              onMouseOver={e => (e.currentTarget.style.backgroundColor = "#5b21b6")}
              onMouseOut={e => (e.currentTarget.style.backgroundColor = "#6d28d9")}
            >
              Upload
            </button>
          </div>
        )}
      </div>
    </section>
  );
}

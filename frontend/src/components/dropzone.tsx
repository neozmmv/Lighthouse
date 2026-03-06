import { useState, useRef, DragEvent, ChangeEvent } from "react";

const CHUNK_SIZE = 5 * 1024 * 1024; // 5 MB — S3 minimum part size

type DropState = "idle" | "hovering" | "done" | "uploading" | "success" | "error";

interface UploadCtx {
  file_id: string;
  upload_id: string;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function uploadChunk(
  url: string,
  data: Blob,
  onProgress: (loaded: number) => void,
  signal: AbortSignal
): Promise<string> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();

    signal.addEventListener("abort", () => {
      xhr.abort();
      reject(new DOMException("Aborted", "AbortError"));
    });

    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable) onProgress(e.loaded);
    };

    xhr.onload = () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        const raw = xhr.getResponseHeader("ETag") ?? xhr.getResponseHeader("etag") ?? "";
        resolve(raw.replace(/"/g, ""));
      } else {
        reject(new Error(`Chunk upload failed with status ${xhr.status}`));
      }
    };

    xhr.onerror = () => reject(new Error("Network error during upload"));
    xhr.onabort = () => reject(new DOMException("Aborted", "AbortError"));

    xhr.open("PUT", url);
    xhr.send(data);
  });
}

export default function Dropzone() {
  const [state, setState] = useState<DropState>("idle");
  const [file, setFile] = useState<File | null>(null);
  const [progress, setProgress] = useState(0);
  const [speed, setSpeed] = useState("");
  const [errorMsg, setErrorMsg] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);
  const abortRef = useRef<AbortController | null>(null);
  const ctxRef = useRef<UploadCtx | null>(null);

  function handleFile(f: File) {
    setFile(f);
    setState("done");
  }

  function onDragOver(e: DragEvent) {
    e.preventDefault();
    setState((s) => (s === "idle" ? "hovering" : s));
  }

  function onDragLeave() {
    setState((s) => (s === "hovering" ? "idle" : s));
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
    setProgress(0);
    setSpeed("");
    setErrorMsg("");
    ctxRef.current = null;
    if (inputRef.current) inputRef.current.value = "";
  }

  async function handleCancel() {
    abortRef.current?.abort();
    const ctx = ctxRef.current;
    if (ctx) {
      fetch("http://localhost:8000/api/upload/abort", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ file_id: ctx.file_id, upload_id: ctx.upload_id }),
      }).catch(() => {});
    }
    reset();
  }

  async function handleUpload() {
    if (!file) return;

    setState("uploading");
    setProgress(0);
    setSpeed("");

    const totalChunks = Math.max(1, Math.ceil(file.size / CHUNK_SIZE));
    const chunkLoaded = new Array(totalChunks).fill(0);
    const startTime = Date.now();
    const controller = new AbortController();
    abortRef.current = controller;

    function updateProgress() {
      const totalLoaded = chunkLoaded.reduce((a, b) => a + b, 0);
      setProgress(Math.min(100, Math.round((totalLoaded / file!.size) * 100)));
      const elapsed = (Date.now() - startTime) / 1000;
      if (elapsed > 0.3) {
        const bps = totalLoaded / elapsed;
        setSpeed(
          bps >= 1024 * 1024
            ? `${(bps / (1024 * 1024)).toFixed(1)} MB/s`
            : `${(bps / 1024).toFixed(0)} KB/s`
        );
      }
    }

    let ctx: UploadCtx | null = null;

    try {
      // 1. Init
      const initRes = await fetch("http://localhost:8000/api/upload/init", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ filename: file.name, total_chunks: totalChunks }),
        signal: controller.signal,
      });
      if (!initRes.ok) throw new Error(`Server error: ${initRes.status}`);
      const { file_id, upload_id, urls } = await initRes.json();
      ctx = { file_id, upload_id };
      ctxRef.current = ctx;

      // 2. Upload chunks
      const parts: { part_number: number; etag: string }[] = [];
      for (let i = 0; i < totalChunks; i++) {
        const start = i * CHUNK_SIZE;
        const chunk = file.slice(start, start + CHUNK_SIZE);
        const { part_number, url } = urls[i];

        const etag = await uploadChunk(
          url,
          chunk,
          (loaded) => {
            chunkLoaded[i] = loaded;
            updateProgress();
          },
          controller.signal
        );

        chunkLoaded[i] = chunk.size;
        updateProgress();
        parts.push({ part_number, etag });
      }

      // 3. Finish
      setProgress(100);
      const finishRes = await fetch("http://localhost:8000/api/upload/finish", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ file_id, upload_id, parts }),
        signal: controller.signal,
      });
      if (!finishRes.ok) throw new Error(`Finish failed: ${finishRes.status}`);

      setState("success");
    } catch (e: unknown) {
      if (e instanceof DOMException && e.name === "AbortError") return;

      if (ctx) {
        fetch("http://localhost:8000/api/upload/abort", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ file_id: ctx.file_id, upload_id: ctx.upload_id }),
        }).catch(() => {});
      }

      setErrorMsg(e instanceof Error ? e.message : "Upload failed");
      setState("error");
    }
  }

  const isHovering = state === "hovering";
  const isDone = state === "done";
  const isUploading = state === "uploading";
  const isSuccess = state === "success";
  const isError = state === "error";
  const canInteract = state === "idle" || state === "hovering";

  return (
    <section
      id="upload"
      className="px-8 md:px-16 lg:px-24 py-16"
      style={{ backgroundColor: "#ffffff", borderBottom: "1px solid #e5e7eb" }}
    >
      <div className="w-full max-w-xl mx-auto">
        <h2 className="text-lg font-semibold text-gray-900 mb-6">Upload a file</h2>

        <div
          className="relative flex flex-col items-center justify-center gap-4 rounded-xl border-2 border-dashed transition-colors"
          style={{
            minHeight: "380px",
            cursor: canInteract ? "pointer" : "default",
            borderColor: isHovering
              ? "#7c3aed"
              : isDone || isUploading || isSuccess
              ? "#7c3aed"
              : isError
              ? "#fca5a5"
              : "#d1d5db",
            backgroundColor: isHovering
              ? "#f5f3ff"
              : isDone || isUploading || isSuccess
              ? "#f5f3ff"
              : isError
              ? "#fff7f7"
              : "#fafafa",
          }}
          onMouseEnter={() => setState((s) => (s === "idle" ? "hovering" : s))}
          onMouseLeave={() => setState((s) => (s === "hovering" ? "idle" : s))}
          onDragOver={canInteract ? onDragOver : undefined}
          onDragLeave={canInteract ? onDragLeave : undefined}
          onDrop={canInteract ? onDrop : undefined}
          onClick={() => canInteract && inputRef.current?.click()}
        >
          <input ref={inputRef} type="file" className="hidden" onChange={onChange} />

          {/* ── idle / hovering ── */}
          {canInteract && (
            <>
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
                  <span className="text-purple-700 underline underline-offset-2">browse</span>
                </p>
                <p className="text-xs text-gray-400 mt-1">Any file type accepted</p>
              </div>
            </>
          )}

          {/* ── file selected ── */}
          {isDone && (
            <>
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
                  {file ? formatSize(file.size) : ""}
                </p>
              </div>
              <button
                className="text-xs text-gray-400 hover:text-gray-600 underline underline-offset-2 transition-colors"
                style={{ background: "none", border: "none", padding: 0 }}
                onClick={(e) => { e.stopPropagation(); reset(); }}
              >
                Remove
              </button>
            </>
          )}

          {/* ── uploading ── */}
          {isUploading && (
            <div
              className="w-full px-10 flex flex-col items-center gap-5"
              onClick={(e) => e.stopPropagation()}
            >
              <div className="text-center">
                <p
                  className="text-sm font-semibold text-gray-900 truncate"
                  style={{ maxWidth: "280px" }}
                >
                  {file?.name}
                </p>
                <p className="text-xs text-gray-400 mt-0.5 tabular-nums">
                  {speed || "Starting…"}
                </p>
              </div>

              <div className="w-full">
                <div className="flex justify-between items-baseline mb-2">
                  <span className="text-xs text-gray-500">Uploading</span>
                  <span
                    className="text-sm font-semibold tabular-nums"
                    style={{ color: "#7c3aed" }}
                  >
                    {progress}%
                  </span>
                </div>
                <div
                  className="w-full rounded-full overflow-hidden"
                  style={{ height: "6px", backgroundColor: "#ede9fe" }}
                >
                  <div
                    className="h-full rounded-full"
                    style={{
                      width: `${progress}%`,
                      backgroundColor: "#7c3aed",
                      transition: "width 0.25s ease-out",
                    }}
                  />
                </div>
              </div>

              <button
                className="text-xs text-gray-400 hover:text-gray-600 underline underline-offset-2 transition-colors"
                style={{ background: "none", border: "none", padding: 0 }}
                onClick={handleCancel}
              >
                Cancel
              </button>
            </div>
          )}

          {/* ── success ── */}
          {isSuccess && (
            <div
              className="flex flex-col items-center gap-4"
              onClick={(e) => e.stopPropagation()}
            >
              <div
                className="w-12 h-12 rounded-full flex items-center justify-center"
                style={{ backgroundColor: "#f5f3ff" }}
              >
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="#7c3aed"
                  strokeWidth="2.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="w-6 h-6"
                >
                  <polyline points="20 6 9 17 4 12" />
                </svg>
              </div>
              <div className="text-center">
                <p className="text-sm font-semibold text-gray-900">Upload complete</p>
                <p
                  className="text-xs text-gray-400 mt-0.5 truncate"
                  style={{ maxWidth: "280px" }}
                >
                  {file?.name}
                </p>
              </div>
              <button
                className="text-xs text-gray-400 hover:text-gray-600 underline underline-offset-2 transition-colors"
                style={{ background: "none", border: "none", padding: 0 }}
                onClick={reset}
              >
                Upload another
              </button>
            </div>
          )}

          {/* ── error ── */}
          {isError && (
            <div
              className="flex flex-col items-center gap-4"
              onClick={(e) => e.stopPropagation()}
            >
              <div
                className="w-12 h-12 rounded-full flex items-center justify-center"
                style={{ backgroundColor: "#fef2f2" }}
              >
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="#ef4444"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="w-6 h-6"
                >
                  <line x1="18" y1="6" x2="6" y2="18" />
                  <line x1="6" y1="6" x2="18" y2="18" />
                </svg>
              </div>
              <div className="text-center">
                <p className="text-sm font-semibold text-gray-900">Upload failed</p>
                <p className="text-xs text-gray-400 mt-0.5">{errorMsg}</p>
              </div>
              <button
                className="text-xs text-gray-400 hover:text-gray-600 underline underline-offset-2 transition-colors"
                style={{ background: "none", border: "none", padding: 0 }}
                onClick={() => setState("done")}
              >
                Try again
              </button>
            </div>
          )}
        </div>

        {/* Upload button — only shown in done state */}
        {isDone && (
          <div className="mt-4 flex justify-end">
            <button
              className="px-5 py-2.5 text-sm font-medium text-white rounded-md transition-colors"
              style={{ backgroundColor: "#6d28d9" }}
              onMouseOver={(e) => (e.currentTarget.style.backgroundColor = "#5b21b6")}
              onMouseOut={(e) => (e.currentTarget.style.backgroundColor = "#6d28d9")}
              onClick={handleUpload}
            >
              Upload
            </button>
          </div>
        )}
      </div>
    </section>
  );
}

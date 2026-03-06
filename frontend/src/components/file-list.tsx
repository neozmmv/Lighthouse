import { useState, useEffect } from "react";

interface FileItem {
  file_id: string;
  filename: string;
  size: number;
  uploaded_at: string;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function getExtension(filename: string): string {
  const parts = filename.split(".");
  return parts.length > 1 ? parts[parts.length - 1].toLowerCase() : "file";
}

function SkeletonRow() {
  return (
    <tr>
      <td className="py-4 px-4">
        <div className="flex items-center gap-3">
          <div className="w-10 h-5 rounded bg-gray-100 animate-pulse" />
          <div className="h-4 w-52 bg-gray-100 rounded animate-pulse" />
        </div>
      </td>
      <td className="py-4 px-4"><div className="h-4 w-14 bg-gray-100 rounded animate-pulse" /></td>
      <td className="py-4 px-4"><div className="h-4 w-36 bg-gray-100 rounded animate-pulse" /></td>
      <td className="py-4 px-4 text-right"><div className="h-7 w-24 bg-gray-100 rounded-md animate-pulse ml-auto" /></td>
    </tr>
  );
}

export default function FileList() {
  const [files, setFiles] = useState<FileItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [downloading, setDownloading] = useState<string | null>(null);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    setLoading(true);
    setError(null);
    fetch("http://localhost:8000/api/files")
      .then((r) => {
        if (!r.ok) throw new Error(`Server responded with ${r.status}`);
        return r.json();
      })
      .then((data) => {
        setFiles(data);
        setLoading(false);
      })
      .catch((e: Error) => {
        setError(e.message);
        setLoading(false);
      });
  }, [tick]);

  async function handleDownload(fileId: string) {
    setDownloading(fileId);
    try {
      const r = await fetch(
        `http://localhost:8000/api/files/${encodeURIComponent(fileId)}/download`
      );
      if (!r.ok) throw new Error("Failed to get download link");
      const { url } = await r.json();
      window.open(url, "_blank");
    } catch (e) {
      console.error(e);
    } finally {
      setDownloading(null);
    }
  }

  return (
    <section
      className="px-8 md:px-16 lg:px-24 py-16"
      style={{ minHeight: "calc(100vh - 200px)" }}
    >
      <div className="max-w-5xl mx-auto">
        {/* Header */}
        <div className="flex items-end justify-between mb-8">
          <div>
            <h2 className="text-lg font-semibold text-gray-900">Received files</h2>
            {!loading && !error && (
              <p className="text-sm text-gray-400 mt-0.5">
                {files.length === 0
                  ? "No files yet"
                  : `${files.length} file${files.length !== 1 ? "s" : ""}`}
              </p>
            )}
          </div>
          <button
            onClick={() => setTick((t) => t + 1)}
            className="inline-flex items-center gap-1.5 text-xs text-gray-400 hover:text-gray-600 transition-colors underline underline-offset-2"
            style={{ background: "none", border: "none", padding: 0 }}
          >
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              className="w-3.5 h-3.5"
            >
              <polyline points="23 4 23 10 17 10" />
              <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
            </svg>
            Refresh
          </button>
        </div>

        {/* Error state */}
        {error ? (
          <div
            className="flex flex-col items-center justify-center py-28 text-center rounded-xl border"
            style={{ borderColor: "#e5e7eb" }}
          >
            <div
              className="w-10 h-10 rounded-full flex items-center justify-center mb-4"
              style={{ backgroundColor: "#fef2f2" }}
            >
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="#ef4444"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="w-5 h-5"
              >
                <circle cx="12" cy="12" r="10" />
                <line x1="12" y1="8" x2="12" y2="12" />
                <line x1="12" y1="16" x2="12.01" y2="16" />
              </svg>
            </div>
            <p className="text-sm font-medium text-gray-700">Could not load files</p>
            <p className="text-xs text-gray-400 mt-1">{error}</p>
            <p className="text-xs text-gray-400 mt-0.5">
              Ensure you're accessing from localhost
            </p>
            <button
              onClick={() => setTick((t) => t + 1)}
              className="mt-5 px-4 py-2 text-xs font-medium text-white rounded-md transition-colors"
              style={{ backgroundColor: "#6d28d9" }}
              onMouseOver={(e) => (e.currentTarget.style.backgroundColor = "#5b21b6")}
              onMouseOut={(e) => (e.currentTarget.style.backgroundColor = "#6d28d9")}
            >
              Try again
            </button>
          </div>
        ) : (
          /* Table */
          <div
            className="rounded-xl border overflow-hidden"
            style={{ borderColor: "#e5e7eb" }}
          >
            <table className="w-full text-sm">
              <thead>
                <tr style={{ borderBottom: "1px solid #e5e7eb", backgroundColor: "#fafafa" }}>
                  <th className="py-3 px-4 text-left text-xs font-semibold text-gray-500 tracking-wider uppercase">
                    File
                  </th>
                  <th className="py-3 px-4 text-left text-xs font-semibold text-gray-500 tracking-wider uppercase">
                    Size
                  </th>
                  <th className="py-3 px-4 text-left text-xs font-semibold text-gray-500 tracking-wider uppercase">
                    Uploaded
                  </th>
                  <th className="py-3 px-4 text-right text-xs font-semibold text-gray-500 tracking-wider uppercase">
                    Action
                  </th>
                </tr>
              </thead>
              <tbody>
                {loading ? (
                  Array.from({ length: 5 }).map((_, i) => <SkeletonRow key={i} />)
                ) : files.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="py-28 text-center">
                      <div className="flex flex-col items-center gap-3">
                        <svg
                          viewBox="0 0 24 24"
                          fill="none"
                          stroke="#d1d5db"
                          strokeWidth="1.5"
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          className="w-10 h-10"
                        >
                          <path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z" />
                          <polyline points="13 2 13 9 20 9" />
                        </svg>
                        <p className="text-sm text-gray-400">No files received yet</p>
                      </div>
                    </td>
                  </tr>
                ) : (
                  files.map((file, idx) => (
                    <tr
                      key={file.file_id}
                      style={{
                        borderTop: idx === 0 ? "none" : "1px solid #f3f4f6",
                      }}
                      className="hover:bg-gray-50 transition-colors"
                    >
                      {/* Filename */}
                      <td className="py-3.5 px-4">
                        <div className="flex items-center gap-3 min-w-0">
                          <span
                            className="shrink-0 text-[10px] font-bold tracking-wider font-mono rounded px-1.5 py-0.5"
                            style={{ backgroundColor: "#f5f3ff", color: "#7c3aed" }}
                          >
                            .{getExtension(file.filename)}
                          </span>
                          <span
                            className="text-gray-800 font-medium truncate"
                            style={{ maxWidth: "320px" }}
                            title={file.filename}
                          >
                            {file.filename}
                          </span>
                        </div>
                      </td>

                      {/* Size */}
                      <td className="py-3.5 px-4 text-gray-400 tabular-nums whitespace-nowrap">
                        {formatSize(file.size)}
                      </td>

                      {/* Date */}
                      <td className="py-3.5 px-4 text-gray-400 whitespace-nowrap">
                        {formatDate(file.uploaded_at)}
                      </td>

                      {/* Download */}
                      <td className="py-3.5 px-4 text-right">
                        <button
                          onClick={() => handleDownload(file.file_id)}
                          disabled={downloading === file.file_id}
                          className="inline-flex items-center gap-1.5 px-4 py-1.5 text-xs font-medium text-white rounded-md transition-colors disabled:opacity-60 disabled:cursor-not-allowed"
                          style={{ backgroundColor: "#6d28d9" }}
                          onMouseOver={(e) => {
                            if (downloading !== file.file_id)
                              e.currentTarget.style.backgroundColor = "#5b21b6";
                          }}
                          onMouseOut={(e) => {
                            e.currentTarget.style.backgroundColor = "#6d28d9";
                          }}
                        >
                          {downloading === file.file_id ? (
                            <>
                              <svg
                                className="animate-spin w-3 h-3"
                                viewBox="0 0 24 24"
                                fill="none"
                              >
                                <circle
                                  cx="12"
                                  cy="12"
                                  r="10"
                                  stroke="currentColor"
                                  strokeOpacity="0.25"
                                  strokeWidth="3"
                                />
                                <path
                                  d="M22 12a10 10 0 0 0-10-10"
                                  stroke="currentColor"
                                  strokeWidth="3"
                                  strokeLinecap="round"
                                />
                              </svg>
                              Getting link…
                            </>
                          ) : (
                            <>
                              <svg
                                viewBox="0 0 24 24"
                                fill="none"
                                stroke="currentColor"
                                strokeWidth="2"
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                className="w-3 h-3"
                              >
                                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                                <polyline points="7 10 12 15 17 10" />
                                <line x1="12" y1="15" x2="12" y2="3" />
                              </svg>
                              Download
                            </>
                          )}
                        </button>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </section>
  );
}

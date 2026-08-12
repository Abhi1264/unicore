"use client";

/** Last-resort boundary when the root layout itself fails (owns its own html/body). */
export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <html lang="en">
      <body
        style={{
          fontFamily: "system-ui, sans-serif",
          display: "flex",
          minHeight: "100vh",
          flexDirection: "column",
          alignItems: "center",
          justifyContent: "center",
          gap: "1rem",
          padding: "1.5rem",
          textAlign: "center",
        }}
      >
        <h1 style={{ fontSize: "1.5rem", fontWeight: 600 }}>
          Unicore could not start
        </h1>
        <p style={{ maxWidth: "40ch", color: "#555" }}>
          A problem occurred while loading the application shell.
        </p>
        {error.digest ? (
          <p style={{ fontFamily: "monospace", fontSize: "0.75rem", color: "#777" }}>
            Reference: {error.digest}
          </p>
        ) : null}
        <button
          onClick={reset}
          style={{
            borderRadius: "0.75rem",
            border: "1px solid #0d3d3a",
            background: "#0d3d3a",
            color: "white",
            padding: "0.5rem 1rem",
            cursor: "pointer",
          }}
        >
          Try again
        </button>
      </body>
    </html>
  );
}

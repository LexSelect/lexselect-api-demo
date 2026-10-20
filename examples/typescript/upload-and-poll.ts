/**
 * LexSelect API — Upload & Poll Example
 *
 * Uploads a document via a single multipart request (POST /documents/upload),
 * waits for processing, then prints the parsed result. Uses only the built-in
 * `fetch`/`FormData` APIs (Node.js 18+).
 *
 * Usage:
 *   LEXSELECT_API_KEY=lxs_... npx tsx upload-and-poll.ts ./contract.pdf
 */

const API_KEY = process.env.LEXSELECT_API_KEY;
const API_URL = process.env.LEXSELECT_API_URL || "https://api.lexselect.io/api";

if (!API_KEY) {
  console.error("Set LEXSELECT_API_KEY environment variable");
  process.exit(1);
}

let documentId: string | undefined;

const filePath = process.argv[2];
if (!filePath) {
  console.error("Usage: npx tsx upload-and-poll.ts <file.pdf>");
  process.exit(1);
}

const headers = {
  Authorization: `Bearer ${API_KEY}`,
  "Content-Type": "application/json",
  "X-API-Version": "2026-09-12",
};

// Map file extensions to the MIME type used for the uploaded file part. The
// server also infers the type from the file name when this is omitted.
const CONTENT_TYPES: Record<string, string> = {
  pdf: "application/pdf",
  doc: "application/msword",
  docx: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  rtf: "application/rtf",
  msg: "application/vnd.ms-outlook",
  eml: "message/rfc822",
  odt: "application/vnd.oasis.opendocument.text",
};

/** Infer multipart content type from a supported document extension. */
function contentTypeFor(name: string): string {
  const ext = name.split(".").pop()?.toLowerCase() ?? "";
  return CONTENT_TYPES[ext] ?? "application/octet-stream";
}

/** Preserve HTTP status and retry guidance without treating error text as a status. */
class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public retryAfterMs = 0,
  ) {
    super(status + ": " + message);
  }
}

/** Decode success JSON and preserve status when an error page is not JSON. */
async function responseJson(resp: Response) {
  const text = await resp.text();
  let data: any;
  try {
    data = JSON.parse(text);
  } catch {
    if (resp.ok) throw new Error("API returned invalid JSON");
  }
  if (!resp.ok) {
    const retry = resp.headers.get("Retry-After");
    const seconds = retry === null ? NaN : Number(retry);
    const retryAfterMs = Number.isFinite(seconds)
      ? Math.max(0, seconds * 1000)
      : Math.max(0, Date.parse(retry || "") - Date.now()) || 0;
    throw new ApiError(
      resp.status,
      data?.detail || resp.statusText,
      retryAfterMs,
    );
  }
  return data;
}

/** Retry only reads, with a bounded deadline and the server's minimum delay. */
async function api(path: string, deadline: number) {
  for (let attempt = 0; ; attempt++) {
    const remaining = deadline - Date.now();
    if (remaining <= 0) throw new Error("Read deadline exceeded");
    try {
      const resp = await fetch(API_URL + path, {
        headers,
        signal: AbortSignal.timeout(Math.min(30_000, remaining)),
      });
      return await responseJson(resp);
    } catch (error) {
      const retryable =
        error instanceof ApiError
          ? error.status === 429 || error.status >= 500
          : error instanceof TypeError ||
            (error instanceof Error && error.name === "TimeoutError");
      if (!retryable || attempt >= 3) throw error;
      const delay = Math.max(
        1000 * 2 ** attempt + Math.floor(Math.random() * 250),
        error instanceof ApiError ? error.retryAfterMs : 0,
      );
      await new Promise((resolve) =>
        setTimeout(
          resolve,
          Math.min(delay, Math.max(0, deadline - Date.now())),
        ),
      );
    }
  }
}

/** Upload once, poll within a deadline and fetch the resulting parse. */
async function main() {
  const fs = await import("fs");
  const path = await import("path");

  const fileName = path.basename(filePath);
  const fileData = fs.readFileSync(filePath);

  console.log(`Uploading ${fileName} (${fileData.length} bytes)...`);

  // Single-request multipart upload (POST /documents/upload): the server stores
  // the bytes, verifies the hash, and triggers processing — one round-trip.
  const form = new FormData();
  form.append("name", fileName);
  form.append("size", String(fileData.length));
  form.append(
    "file",
    new Blob([fileData], { type: contentTypeFor(fileName) }),
    fileName,
  );

  // No Content-Type header — fetch sets the multipart boundary automatically.
  const uploadResp = await fetch(`${API_URL}/documents/upload`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${API_KEY}`,
      "X-API-Version": "2026-09-12",
    },
    body: form,
    signal: AbortSignal.timeout(120_000),
  });

  const uploadData = await responseJson(uploadResp);
  const realDocId = uploadData.id;
  documentId = realDocId;
  console.log("Upload complete. Document ID: " + realDocId);

  const deadline = Date.now() + 10 * 60_000;
  let delayMs = 2000;
  while (Date.now() < deadline) {
    const wait = delayMs + Math.floor(Math.random() * 250);
    await new Promise((resolve) =>
      setTimeout(resolve, Math.min(wait, deadline - Date.now())),
    );
    if (Date.now() >= deadline) break;
    delayMs = Math.min(15_000, Math.ceil(delayMs * 1.5));

    let proc;
    try {
      proc = await api(
        "/documents/" + realDocId + "/processing/latest",
        deadline,
      );
    } catch (error) {
      if (!(error instanceof ApiError) || error.status !== 404) throw error;
      process.stdout.write("\rProcessing (waiting for first status)...   ");
      continue;
    }

    if (proc.status === "completed") {
      console.log("\nDone! Pages: " + proc.pages_total);
      const parsed = await api("/documents/" + realDocId + "/parse", deadline);
      console.log(
        "Parsed: " +
          parsed.page_count +
          "/" +
          parsed.total_pages +
          " pages, type=" +
          parsed.document_type,
      );
      return;
    }
    if (proc.status === "failed" || proc.status === "unknown") {
      throw new Error(
        "Processing " +
          proc.status +
          ": " +
          (proc.error_message || "inspect document status"),
      );
    }
    const suffix = proc.total_known ? "" : "+";
    process.stdout.write(
      "\rProcessing [" +
        proc.stage +
        " " +
        proc.pages_done +
        "/" +
        proc.pages_total +
        suffix +
        "]   ",
    );
  }
  throw new Error("Timed out waiting for processing");
}

main().catch((error) => {
  console.error(error.message);
  console.error(
    documentId
      ? "Reconcile document " + documentId + " before re-uploading."
      : "If upload acceptance is uncertain, list documents before re-uploading.",
  );
  process.exit(1);
});

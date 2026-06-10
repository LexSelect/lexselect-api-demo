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

const filePath = process.argv[2];
if (!filePath) {
  console.error("Usage: npx tsx upload-and-poll.ts <file.pdf>");
  process.exit(1);
}

const headers = {
  Authorization: `Bearer ${API_KEY}`,
  "Content-Type": "application/json",
  "X-API-Version": "2026-06-07",
};

// Map file extensions to the MIME type used for the uploaded file part. The
// server also infers the type from the file name when this is omitted.
const CONTENT_TYPES: Record<string, string> = {
  pdf: "application/pdf",
  doc: "application/msword",
  docx: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  rtf: "application/rtf",
  html: "text/html",
  msg: "application/vnd.ms-outlook",
  eml: "message/rfc822",
  odt: "application/vnd.oasis.opendocument.text",
};

function contentTypeFor(name: string): string {
  const ext = name.split(".").pop()?.toLowerCase() ?? "";
  return CONTENT_TYPES[ext] ?? "application/octet-stream";
}

async function api(method: string, path: string, body?: object) {
  const resp = await fetch(`${API_URL}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  const data = await resp.json();
  if (!resp.ok) {
    throw new Error(`${resp.status}: ${data.detail || JSON.stringify(data)}`);
  }
  return data;
}

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
    fileName
  );

  // No Content-Type header — fetch sets the multipart boundary automatically.
  const uploadResp = await fetch(`${API_URL}/documents/upload`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${API_KEY}`,
      "X-API-Version": "2026-06-07",
    },
    body: form,
  });

  const uploadData = await uploadResp.json();
  if (!uploadResp.ok) {
    throw new Error(
      `${uploadResp.status}: ${uploadData.detail || JSON.stringify(uploadData)}`
    );
  }

  const realDocId = uploadData.id;
  console.log(`Upload complete. Document ID: ${realDocId}`);

  // Poll processing status — a single status line rewritten in place.
  for (let i = 0; i < 120; i++) {
    await new Promise((r) => setTimeout(r, 3000));

    try {
      const proc = await api("GET", `/documents/${realDocId}/processing/latest`);

      if (proc.status === "completed") {
        console.log(`\nDone! Pages: ${proc.pages_total}`);

        // Fetch the parsed structure (the actual product value).
        const parsed = await api("GET", `/documents/${realDocId}/parse`);
        console.log(
          `Parsed: ${parsed.page_count}/${parsed.total_pages} pages, type=${parsed.document_type}`
        );
        return;
      }

      if (proc.status === "failed") {
        console.log();
        throw new Error(`Processing failed: ${proc.error_message}`);
      }

      // Show honest progress, rewriting the same line in place (\r): when
      // total_known is false the total may still grow, so render
      // "done/total+" instead of a percentage. Trailing spaces clear any
      // leftover characters from a longer previous render.
      const suffix = proc.total_known ? "" : "+";
      process.stdout.write(
        `\rProcessing [${proc.stage} ${proc.pages_done}/${proc.pages_total}${suffix}]   `
      );
    } catch (e: any) {
      if (!e.message.includes("404")) throw e;
      // Not started yet, keep polling
      process.stdout.write("\rProcessing (waiting for first status)...   ");
    }
  }

  console.log("\nTimed out waiting for processing");
  process.exit(1);
}

main().catch((e) => {
  console.error(e.message);
  process.exit(1);
});

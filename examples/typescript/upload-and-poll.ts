/**
 * LexSelect API — Upload & Poll Example
 *
 * Uploads a PDF document via the 3-step flow, waits for processing, then prints
 * the parsed result. Uses only the built-in `fetch` API (Node.js 18+).
 *
 * Usage:
 *   LEXSELECT_API_KEY=lxs_... npx tsx upload-and-poll.ts ./contract.pdf
 *
 * Local dev note: when pointing LEXSELECT_API_URL at a local stack whose S3
 * (LocalStack) serves a self-signed cert, Node's fetch rejects the presigned
 * PUT. Run with NODE_TLS_REJECT_UNAUTHORIZED=0 for local testing only.
 */

import { createHash } from "node:crypto";

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
  "X-API-Version": "2026-03-06",
};

// Map file extensions to the MIME type the presigned URL is signed for. The S3
// PUT Content-Type MUST match the type inferred at create time (step 1), or S3
// rejects the upload with SignatureDoesNotMatch.
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

  // SHA-256 of the file: hex for the completion call, base64 for the S3
  // checksum header so the server verifies integrity without re-downloading.
  const digest = createHash("sha256").update(fileData).digest();
  const hashHex = digest.toString("hex");
  const hashB64 = digest.toString("base64");

  console.log(`Uploading ${fileName} (${fileData.length} bytes)...`);

  // Step 1: Create upload session
  const createResp = await api("POST", "/documents", {
    name: fileName,
    size: fileData.length,
  });

  const docId = createResp.id;
  console.log(`Document created: ${docId}`);

  // Step 2: Upload to S3 — Content-Type must match the inferred type, and the
  // optional x-amz-checksum-sha256 lets the server verify integrity via metadata.
  const s3Resp = await fetch(createResp.upload_url, {
    method: "PUT",
    body: fileData,
    headers: {
      "Content-Type": contentTypeFor(fileName),
      "x-amz-checksum-sha256": hashB64,
    },
  });

  if (!s3Resp.ok) throw new Error(`S3 upload failed: ${s3Resp.status}`);
  console.log("File uploaded to S3");

  // Step 3: Complete upload — content_hash_sha256 (hex) is required and is
  // verified against the uploaded bytes.
  const updateResp = await api("PUT", `/documents/${docId}`, {
    status: "uploaded",
    content_hash_sha256: hashHex,
  });

  const realDocId = updateResp.id;
  console.log(`Upload complete. Document ID: ${realDocId}`);

  // Step 4: Poll processing status
  process.stdout.write("Processing");
  for (let i = 0; i < 120; i++) {
    await new Promise((r) => setTimeout(r, 3000));
    process.stdout.write(".");

    try {
      const proc = await api("GET", `/documents/${realDocId}/processing/latest`);

      if (proc.status === "completed") {
        console.log(`\nDone! Pages: ${proc.page_count}`);

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
    } catch (e: any) {
      if (!e.message.includes("404")) throw e;
      // Not started yet, keep polling
    }
  }

  console.log("\nTimed out waiting for processing");
  process.exit(1);
}

main().catch((e) => {
  console.error(e.message);
  process.exit(1);
});

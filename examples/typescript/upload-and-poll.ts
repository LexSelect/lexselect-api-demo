/**
 * LexSelect API — Upload & Poll Example
 *
 * Uploads a PDF document, waits for processing to complete, then prints the result.
 * Uses only the built-in `fetch` API (Node.js 18+).
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
  "X-API-Version": "2026-03-06",
};

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

  // Step 1: Create document
  const createResp = await api("POST", "/documents", {
    name: fileName,
    size: fileData.length,
  });

  const docId = createResp.id;
  console.log(`Document created: ${docId}`);

  // Step 2: Upload to S3
  const s3Resp = await fetch(createResp.upload_url, {
    method: "PUT",
    body: fileData,
    headers: { "Content-Type": "application/octet-stream" },
  });

  if (!s3Resp.ok) throw new Error(`S3 upload failed: ${s3Resp.status}`);
  console.log("File uploaded to S3");

  // Step 3: Mark as uploaded
  const updateResp = await api("PUT", `/documents/${docId}`, {
    status: "uploaded",
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
        console.log(JSON.stringify(proc, null, 2));
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

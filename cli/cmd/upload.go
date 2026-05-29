package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/LexSelect/lexselect-api-demo/cli/api"
	"github.com/LexSelect/lexselect-api-demo/cli/output"
)

const (
	pollInterval    = 3 * time.Second
	maxPollAttempts = 120
)

func init() {
	uploadCmd := &cobra.Command{
		Use:   "upload <file>",
		Short: "Upload document, wait for processing, print result",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			single, _ := cmd.Flags().GetBool("single-request")
			return runUpload(cmd.Context(), args[0], single)
		},
	}
	uploadCmd.Flags().Bool("single-request", false,
		"Use the single-request multipart endpoint (POST /documents/upload) instead of the 3-step flow")
	rootCmd.AddCommand(uploadCmd)
}

func runUpload(ctx context.Context, filePath string, single bool) error {
	client, err := api.New(cfg)
	if err != nil {
		return err
	}

	fileName := filepath.Base(filePath)
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	contentType := output.ContentType(fileName)

	fmt.Printf("Uploading %s (%d bytes)...\n", fileName, len(fileData))

	var realDocID string

	if single {
		// Single-request multipart upload — the server runs all three steps,
		// computes the hash, and triggers processing.
		doc, err := client.UploadMultipart(ctx, fileName, "", contentType, fileData)
		if err != nil {
			return fmt.Errorf("upload failed: %w", err)
		}
		realDocID, _ = doc["id"].(string)
		fmt.Printf("Upload complete. Document ID: %s\n", realDocID)
	} else {
		// SHA-256 of the file: hex for the completion call, base64 for the S3
		// checksum header so the server can verify integrity without a download.
		sum := sha256.Sum256(fileData)
		hexHash := hex.EncodeToString(sum[:])
		b64Hash := base64.StdEncoding.EncodeToString(sum[:])

		// Step 1: Create upload session
		createResp, err := client.Request(ctx, "POST", "/documents", map[string]interface{}{
			"name": fileName,
			"size": len(fileData),
		})
		if err != nil {
			return fmt.Errorf("create failed: %w", err)
		}
		docID, ok := createResp["id"].(string)
		if !ok {
			return fmt.Errorf("unexpected create response: missing 'id'")
		}
		uploadURL, ok := createResp["upload_url"].(string)
		if !ok {
			return fmt.Errorf("unexpected create response: missing 'upload_url'")
		}
		fmt.Printf("Document created: %s\n", docID)

		// Step 2: Upload to S3 (Content-Type must match what was inferred in
		// step 1; the checksum lets the server verify the stored bytes).
		if err := client.UploadToS3(ctx, uploadURL, fileData, contentType, b64Hash); err != nil {
			return err
		}
		fmt.Println("File uploaded to S3")

		// Step 3: Complete — content_hash_sha256 (hex) is required and the
		// server verifies it against the uploaded object.
		updateResp, err := client.Request(ctx, "PUT", "/documents/"+docID, map[string]interface{}{
			"status":              "uploaded",
			"content_hash_sha256": hexHash,
		})
		if err != nil {
			return fmt.Errorf("update failed: %w", err)
		}
		realDocID = docID
		if id, ok := updateResp["id"].(string); ok {
			realDocID = id
		}
		fmt.Printf("Upload complete. Document ID: %s\n", realDocID)
	}

	// Step 4: Poll processing
	fmt.Print("Processing")
	for i := 0; i < maxPollAttempts; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
		fmt.Print(".")

		procResp, err := client.Request(ctx, "GET", "/documents/"+realDocID+"/processing/latest", nil)
		if err != nil {
			if apiErr, ok := err.(*api.APIError); ok && apiErr.StatusCode == 404 {
				continue
			}
			fmt.Println()
			return fmt.Errorf("processing check failed: %w", err)
		}

		switch procResp["status"] {
		case "completed":
			pageCount, _ := procResp["page_count"].(float64)
			fmt.Printf("\nDone! Pages: %d\n", int(pageCount))
			if cfg.JSONOutput {
				output.JSON(procResp)
			}
			return nil
		case "failed":
			errMsg, _ := procResp["error_message"].(string)
			fmt.Println()
			return fmt.Errorf("processing failed: %s", errMsg)
		}
	}

	fmt.Println()
	return fmt.Errorf("processing timed out after %d attempts", maxPollAttempts)
}

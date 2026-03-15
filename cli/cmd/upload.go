package cmd

import (
	"context"
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
	rootCmd.AddCommand(&cobra.Command{
		Use:   "upload <file>",
		Short: "Upload document, wait for processing, print result",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpload(cmd.Context(), args[0])
		},
	})
}

func runUpload(ctx context.Context, filePath string) error {
	client, err := api.New(cfg)
	if err != nil {
		return err
	}

	fileName := filepath.Base(filePath)
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	fmt.Printf("Uploading %s (%d bytes)...\n", fileName, len(fileData))

	// Step 1: Create document
	createResp, err := client.Request(ctx, "POST", "/documents", map[string]interface{}{
		"name": fileName,
		"size": len(fileData),
	})
	if err != nil {
		return fmt.Errorf("create failed: %w", err)
	}

	docID := createResp["id"].(string)
	uploadURL := createResp["upload_url"].(string)
	fmt.Printf("Document created: %s\n", docID)

	// Step 2: Upload to S3
	contentType := output.ContentType(fileName)
	if err := client.UploadToS3(ctx, uploadURL, fileData, contentType); err != nil {
		return err
	}
	fmt.Println("File uploaded to S3")

	// Step 3: Mark as uploaded
	updateResp, err := client.Request(ctx, "PUT", "/documents/"+docID, map[string]interface{}{
		"status": "uploaded",
	})
	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	realDocID := docID
	if id, ok := updateResp["id"].(string); ok {
		realDocID = id
	}
	fmt.Printf("Upload complete. Document ID: %s\n", realDocID)

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

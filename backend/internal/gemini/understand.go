package gemini

import (
	"context"
	"fmt"
	"io"
	"time"

	"google.golang.org/genai"
)

const (
	fileReadyPollAttempts = 5
	fileReadyPollInterval = time.Second
)

// UnderstandDocument uploads a file to the Gemini Files API, waits for it to
// become ready, and asks for a short tutoring-context summary.
func (c *Client) UnderstandDocument(ctx context.Context, r io.Reader, mimeType, filename string) (fileURI, summary string, err error) {
	file, err := c.genai.Files.Upload(ctx, r, &genai.UploadFileConfig{
		MIMEType:    mimeType,
		DisplayName: filename,
	})
	if err != nil {
		return "", "", fmt.Errorf("upload file: %w", err)
	}

	file, err = c.waitUntilActive(ctx, file)
	if err != nil {
		return "", "", err
	}

	content := genai.NewContentFromParts([]*genai.Part{
		{Text: "Summarize this document in 2-3 sentences for a tutoring context."},
		{FileData: &genai.FileData{FileURI: file.URI, MIMEType: mimeType}},
	}, genai.RoleUser)

	resp, err := c.genai.Models.GenerateContent(ctx, c.textModel, []*genai.Content{content}, nil)
	if err != nil {
		return "", "", fmt.Errorf("summarize: %w", err)
	}

	return file.URI, resp.Text(), nil
}

func (c *Client) waitUntilActive(ctx context.Context, file *genai.File) (*genai.File, error) {
	for i := 0; i < fileReadyPollAttempts; i++ {
		switch file.State {
		case genai.FileStateActive:
			return file, nil
		case genai.FileStateFailed:
			return nil, fmt.Errorf("file processing failed")
		}

		time.Sleep(fileReadyPollInterval)

		var err error
		file, err = c.genai.Files.Get(ctx, file.Name, nil)
		if err != nil {
			return nil, fmt.Errorf("poll file state: %w", err)
		}
	}
	return nil, fmt.Errorf("file did not become active in time")
}

package email

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v3"
)

type Client struct {
	client    *resend.Client
	fromEmail string
}

func NewClient(apiKey, fromEmail string) *Client {
	return &Client{
		client:    resend.NewClient(apiKey),
		fromEmail: fromEmail,
	}
}

func (c *Client) SendVerificationEmail(ctx context.Context, toEmail, toName, verifyURL string) error {
	html := fmt.Sprintf(`
		<p>Hi %s,</p>
		<p>Welcome to Theresa. Click the link below to verify your email address:</p>
		<p><a href="%s">%s</a></p>
		<p>This link expires in 24 hours.</p>
	`, toName, verifyURL, verifyURL)

	_, err := c.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    c.fromEmail,
		To:      []string{toEmail},
		Subject: "Verify your Theresa account",
		Html:    html,
	})
	return err
}

func (c *Client) SendPasswordResetEmail(ctx context.Context, toEmail, toName, resetURL string) error {
	html := fmt.Sprintf(`
		<p>Hi %s,</p>
		<p>We received a request to reset your Theresa password. Click the link below to choose a new one:</p>
		<p><a href="%s">%s</a></p>
		<p>This link expires in 1 hour. If you didn't request this, you can safely ignore this email.</p>
	`, toName, resetURL, resetURL)

	_, err := c.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    c.fromEmail,
		To:      []string{toEmail},
		Subject: "Reset your Theresa password",
		Html:    html,
	})
	return err
}

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
	text := fmt.Sprintf(
		"Hi %s,\n\nWelcome to Theresa. Verify your email address:\n%s\n\nThis link expires in 24 hours.",
		toName, verifyURL,
	)

	_, err := c.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    c.fromEmail,
		To:      []string{toEmail},
		Subject: "Verify your Theresa account",
		Html:    html,
		Text:    text,
	})
	return err
}

// SendLowCreditsEmail notifies a user that they've crossed a usage
// threshold (50/75/95% of their current credit cycle) - a nudge to top up
// before they hit zero mid-conversation, not just a silent cutoff.
func (c *Client) SendLowCreditsEmail(ctx context.Context, toEmail, toName string, percentUsed int, remainingNaira float64) error {
	html := fmt.Sprintf(`
		<p>Hi %s,</p>
		<p>You've used %d%% of your Theresa voice credits - about ₦%.2f left.</p>
		<p>Top up anytime to keep your voice sessions running without interruption.</p>
	`, toName, percentUsed, remainingNaira)
	text := fmt.Sprintf(
		"Hi %s,\n\nYou've used %d%% of your Theresa voice credits - about ₦%.2f left.\n\nTop up anytime to keep your voice sessions running without interruption.",
		toName, percentUsed, remainingNaira,
	)

	_, err := c.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    c.fromEmail,
		To:      []string{toEmail},
		Subject: fmt.Sprintf("You've used %d%% of your Theresa credits", percentUsed),
		Html:    html,
		Text:    text,
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
	text := fmt.Sprintf(
		"Hi %s,\n\nWe received a request to reset your Theresa password. Reset it here:\n%s\n\nThis link expires in 1 hour. If you didn't request this, you can safely ignore this email.",
		toName, resetURL,
	)

	_, err := c.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    c.fromEmail,
		To:      []string{toEmail},
		Subject: "Reset your Theresa password",
		Html:    html,
		Text:    text,
	})
	return err
}

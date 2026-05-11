package resend

import (
	"fmt"
	"os"

	"github.com/resend/resend-go/v2"
)

// Client is the global Resend email client
var Client *resend.Client

// Init initialises the Resend client using the API key from .env
// Call this once when the server starts
func Init() {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		panic("RESEND_API_KEY is not set in .env")
	}
	Client = resend.NewClient(apiKey)
}

// SendOTP sends a 6-digit OTP code to the given email address
func SendOTP(toEmail string, code string) error {
	from := os.Getenv("RESEND_FROM")
	if from == "" {
		from = "onboarding@resend.dev"
	}

	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{toEmail},
		Subject: "Your NFT Platform OTP Code",
		Html: fmt.Sprintf(`
			<div style="font-family: Arial, sans-serif; max-width: 400px; margin: auto;">
				<h2>Your One-Time Password</h2>
				<p>Use this code to log in to your NFT Minting Platform account:</p>
				<h1 style="letter-spacing: 8px; color: #534AB7;">%s</h1>
				<p>This code expires in 10 minutes.</p>
				<p>If you did not request this, ignore this email.</p>
			</div>
		`, code),
	}

	_, err := Client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send OTP email: %w", err)
	}

	return nil
}

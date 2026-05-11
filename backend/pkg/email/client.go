package email

import (
	"fmt"
	"os"
	"strconv"

	gomail "gopkg.in/gomail.v2"
)

// SendOTP sends a 6-digit OTP code via Gmail SMTP
func SendOTP(toEmail string, code string) error {
	from := os.Getenv("GMAIL_USER")
	password := os.Getenv("GMAIL_APP_PASSWORD")

	if from == "" || password == "" {
		return fmt.Errorf("GMAIL_USER or GMAIL_APP_PASSWORD not set in .env")
	}

	// Build the email
	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "Your NFT Platform OTP Code")
	m.SetBody("text/html", fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 400px; margin: auto;">
			<h2>Your One-Time Password</h2>
			<p>Use this code to log in to your NFT Minting Platform account:</p>
			<h1 style="letter-spacing: 8px; color: #534AB7;">%s</h1>
			<p>This code expires in 10 minutes.</p>
			<p>If you did not request this, ignore this email.</p>
		</div>
	`, code))

	// Gmail SMTP settings
	port, _ := strconv.Atoi("587")
	d := gomail.NewDialer("smtp.gmail.com", port, from, password)

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

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

// SendSaleNotification notifies the seller their NFT has sold
func SendSaleNotification(sellerEmail, nftName, buyerEmail string, priceLovelace int64, txHash string) error {
	from := os.Getenv("GMAIL_USER")
	password := os.Getenv("GMAIL_APP_PASSWORD")
	if from == "" || password == "" {
		return fmt.Errorf("GMAIL_USER or GMAIL_APP_PASSWORD not set in .env")
	}
	priceADA := float64(priceLovelace) / 1_000_000
	cardanoscanURL := fmt.Sprintf("https://preprod.cardanoscan.io/transaction/%s", txHash)
	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", sellerEmail)
	m.SetHeader("Subject", fmt.Sprintf("Your NFT \"%s\" has sold!", nftName))
	m.SetBody("text/html", fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; max-width: 500px; margin: auto;">
			<h2 style="color: #534AB7;">Your NFT Sold!</h2>
			<p>Great news — your NFT has been purchased.</p>
			<table style="width:100%%;border-collapse:collapse;margin:16px 0;">
				<tr><td style="padding:8px;color:#666;">NFT</td><td style="padding:8px;font-weight:bold;">%s</td></tr>
				<tr><td style="padding:8px;color:#666;">Sale Price</td><td style="padding:8px;font-weight:bold;">%.2f ADA</td></tr>
				<tr><td style="padding:8px;color:#666;">Buyer</td><td style="padding:8px;">%s</td></tr>
			</table>
			<a href="%s" style="display:inline-block;padding:12px 24px;background:#534AB7;color:#fff;text-decoration:none;border-radius:8px;">
				View Transaction on Cardanoscan →
			</a>
			<p style="margin-top:24px;font-size:12px;color:#888;">
				This is an automated notification from NFT Minting Platform.
			</p>
		</div>
	`, nftName, priceADA, buyerEmail, cardanoscanURL))
	port, _ := strconv.Atoi("587")
	d := gomail.NewDialer("smtp.gmail.com", port, from, password)
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send sale notification: %w", err)
	}
	return nil
}

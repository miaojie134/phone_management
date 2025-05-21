package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
	"strconv"
	"strings"
)

// SMTPConfig holds the SMTP server configuration
type SMTPConfig struct {
	Host     string
	Port     int // Should always be 465 with this simplified version
	Username string
	Password string
	Sender   string
}

// LoadSMTPConfigFromEnv loads SMTP configuration from environment variables.
// It now ensures that the configured port is 465.
func LoadSMTPConfigFromEnv() (*SMTPConfig, error) {
	host := os.Getenv("SMTP_HOST")
	portStr := os.Getenv("SMTP_PORT")
	username := os.Getenv("SMTP_USERNAME")
	password := os.Getenv("SMTP_PASSWORD")
	sender := os.Getenv("SMTP_SENDER_EMAIL")

	if host == "" || portStr == "" || sender == "" {
		return nil, fmt.Errorf("SMTP_HOST, SMTP_PORT, and SMTP_SENDER_EMAIL must be set")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_PORT: %v", err)
	}

	// Since we are simplifying to only support port 465 directly
	if port != 465 {
		return nil, fmt.Errorf("configuration error: SMTP_PORT must be 465 for this email implementation")
	}

	return &SMTPConfig{
		Host:     host,
		Port:     port, // Will be 465
		Username: username,
		Password: password,
		Sender:   sender,
	}, nil
}

// SendVerificationEmail sends a verification email to the user using port 465 with TLS.
func SendVerificationEmail(toEmail string, employeeName string, verificationLink string) error {
	config, err := LoadSMTPConfigFromEnv()
	if err != nil {
		return fmt.Errorf("failed to load SMTP config: %w", err)
	}

	subject := "【虚拟资产】手机号码使用情况确认" // As per user's last modification
	body := fmt.Sprintf(`
<html>
<body>
    <p>%s老师,</p>
    <p>您好！为了确保公司手机号码资源得到有效管理和准确记录，我们需要您配合完成当前名下登记手机号码的使用情况确认。</p>
    <p>请点击以下专属链接，查看您名下登记使用的号码并进行确认：</p>
    <p><a href="%s">%s</a></p>
    <p>此链接有效期为7天，请尽快处理。如果您在操作过程中遇到任何问题，或对名下号码信息有疑问，请及时联系苗杰。</p>
    <p>感谢您的理解与配合！</p>
    <p><small>（这是一封自动发送的邮件，请勿直接回复。）</small></p>
</body>
</html>
`, employeeName, verificationLink, verificationLink)

	msgHeaders := []string{
		"To: " + toEmail,
		"From: " + config.Sender,
		"Subject: " + subject,
		"MIME-version: 1.0",
		"Content-Type: text/html; charset=\"UTF-8\"",
		"", // Empty line separating headers from body
	}
	fullMsg := []byte(strings.Join(msgHeaders, "\r\n") + body)

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port) // Port will be 465
	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)

	// SSL/TLS connection from the start for port 465
	tlsconfig := &tls.Config{
		ServerName: config.Host,
		MinVersion: tls.VersionTLS12, // Explicitly set minimum TLS version
	}

	conn, err := tls.Dial("tcp", addr, tlsconfig)
	if err != nil {
		return fmt.Errorf("failed to dial TLS (%s): %w", addr, err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client after TLS dial: %w", err)
	}
	defer client.Close()

	if config.Username != "" && config.Password != "" {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	if err = client.Mail(config.Sender); err != nil {
		return fmt.Errorf("SMTP mail from failed: %w", err)
	}
	if err = client.Rcpt(toEmail); err != nil {
		return fmt.Errorf("SMTP rcpt to failed: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP data command failed: %w", err)
	}

	_, err = w.Write(fullMsg)
	if err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close email data writer: %w", err)
	}

	return client.Quit()
}

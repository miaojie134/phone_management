package email

import (
	"fmt"
	"net/smtp"
	"os"
	"strconv"
	"strings"
)

// SMTPConfig holds the SMTP server configuration
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Sender   string
}

// LoadSMTPConfigFromEnv loads SMTP configuration from environment variables
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

	return &SMTPConfig{
		Host:     host,
		Port:     port,
		Username: username, // Username can be empty for some SMTP servers
		Password: password, // Password can be empty for some SMTP servers
		Sender:   sender,
	}, nil
}

// SendVerificationEmail sends a verification email to the user.
func SendVerificationEmail(toEmail string, employeeName string, verificationLink string) error {
	config, err := LoadSMTPConfigFromEnv()
	if err != nil {
		return fmt.Errorf("failed to load SMTP config: %w", err)
	}

	subject := "【重要】企业手机号码使用情况确认"
	body := fmt.Sprintf(`
<html>
<body>
    <p>%s老师,</p>
    <p>您好！为了确保公司手机号码资源得到有效管理和准确记录，我们需要您配合完成当前名下登记手机号码的使用情况确认。</p>
    <p>请点击以下专属链接，查看您名下登记的号码并进行确认：</p>
    <p><a href="%s">%s</a></p>
    <p>此链接有效期为7天，请尽快处理。如果您在操作过程中遇到任何问题，或对名下号码信息有疑问，请及时联系苗杰。</p>
    <p>感谢您的理解与配合！</p>
    <p><small>（这是一封自动发送的邮件，请勿直接回复。）</small></p>
</body>
</html>
`, employeeName, verificationLink, verificationLink)

	// Construct email message with CRLF line endings
	// and correct Content-Type header
	msg := []byte(strings.Join([]string{
		"To: " + toEmail,
		"From: " + config.Sender,
		"Subject: " + subject,
		"MIME-version: 1.0",
		"Content-Type: text/html; charset=\"UTF-8\"", // Corrected Content-Type
		"",
		body,
	}, "\r\n")) // Use CRLF line endings for email headers

	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	// For SMTP servers that don't require authentication, auth can be nil
	// However, smtp.SendMail requires a non-nil smtp.Auth if username/password are not empty
	// If username and password are provided, PlainAuth is typically used.
	// If the SMTP server allows sending without authentication and username/password are empty,
	// we might need a different approach or ensure the library handles it.
	// For now, we assume PlainAuth is fine if username/password are provided,
	// or the server handles empty credentials gracefully with PlainAuth.
	err = smtp.SendMail(addr, auth, config.Sender, []string{toEmail}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

package email

import (
	"os"
	"testing"
)

func TestSendVerificationEmail(t *testing.T) {
	// 从环境变量读取测试配置
	recipientEmail := os.Getenv("TEST_RECIPIENT_EMAIL")
	employeeName := os.Getenv("TEST_EMPLOYEE_NAME")
	verificationLink := os.Getenv("TEST_VERIFICATION_LINK")

	// 检查必要的 SMTP 配置环境变量是否已设置，这些由 SendVerificationEmail 内部检查，这里确保测试的接收者邮箱已设置
	if recipientEmail == "" {
		t.Skip("Skipping email sending test: TEST_RECIPIENT_EMAIL environment variable not set.")
	}

	// 如果测试用的员工名和链接未通过环境变量设置，则使用默认值
	if employeeName == "" {
		employeeName = "测试员工"
	}
	if verificationLink == "" {
		verificationLink = "http://localhost:3000/verify?token=testtokenfromgounittest"
	}

	t.Logf("Attempting to send verification email to %s using SMTP server %s:%s...",
		recipientEmail, os.Getenv("SMTP_HOST"), os.Getenv("SMTP_PORT"))
	t.Log("Ensure SMTP environment variables are set: SMTP_HOST, SMTP_PORT, SMTP_SENDER_EMAIL, SMTP_USERNAME, SMTP_PASSWORD")

	err := SendVerificationEmail(recipientEmail, employeeName, verificationLink)
	if err != nil {
		// 如果邮件发送失败，记录错误，并提示检查 SMTP 配置
		t.Errorf("SendVerificationEmail failed: %v", err)
		t.Log("Please ensure all SMTP related environment variables (SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD, SMTP_SENDER_EMAIL) are correctly set and the SMTP server is reachable.")
	} else {
		t.Logf("Email sent request processed for %s. Please check the inbox to confirm reception.", recipientEmail)
	}
}

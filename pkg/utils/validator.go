package utils

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var (
	ErrInvalidPhoneNumberFormat = errors.New("无效的手机号码格式，必须是11位数字")
	ErrInvalidPhoneNumberPrefix = errors.New("无效的手机号码前缀，必须以1开头")
	ErrInvalidEmailFormat       = errors.New("无效的邮箱格式")
	ErrInvalidDateFormat        = errors.New("日期格式无效，请使用 YYYY-MM-DD 或类似格式") // 保持通用错误信息
)

// IsNumeric 检查字符串是否只包含数字
func IsNumeric(s string) bool {
	if s == "" {
		return false // 空字符串不视为数字
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// ValidatePhoneNumber 校验手机号码格式。
// 如果有效，返回 nil；否则返回具体的错误。
func ValidatePhoneNumber(phone string) error {
	trimmedPhone := strings.TrimSpace(phone)
	if len(trimmedPhone) != 11 {
		return ErrInvalidPhoneNumberFormat
	}
	if !IsNumeric(trimmedPhone) {
		return ErrInvalidPhoneNumberFormat
	}
	if !strings.HasPrefix(trimmedPhone, "1") {
		return ErrInvalidPhoneNumberPrefix
	}
	return nil
}

// ValidateEmailFormat 校验邮箱格式。
func ValidateEmailFormat(email string) bool {
	trimmedEmail := strings.TrimSpace(email)
	if trimmedEmail == "" {
		return true // 空字符串不进行格式校验，业务逻辑决定是否允许为空
	}
	// 一个常用且相对简单的邮箱正则
	match, _ := regexp.MatchString(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, trimmedEmail)
	return match
}

// ParseDate 解析日期字符串，支持多种常见格式。
// 支持 YYYY-MM-DD, YYYY/MM/DD, YYYY-M-D, YYYY/M/D 等及其变体。
func ParseDate(dateStr string) (time.Time, error) {
	trimmedDateStr := strings.TrimSpace(dateStr)
	if trimmedDateStr == "" {
		return time.Time{}, ErrInvalidDateFormat // 空日期字符串视为无效
	}

	normalizedDateStr := strings.ReplaceAll(trimmedDateStr, "/", "-")

	// 包含补零和不补零的情况
	dateLayouts := []string{
		"2006-01-02", // YYYY-MM-DD
		"2006-1-2",   // YYYY-M-D
		"2006-01-2",  // YYYY-MM-D
		"2006-1-02",  // YYYY-M-DD
	}

	var parsedDate time.Time
	var err error

	for _, layout := range dateLayouts {
		parsedDate, err = time.Parse(layout, normalizedDateStr)
		if err == nil {
			return parsedDate, nil // 解析成功，立即返回
		}
	}
	// 所有格式尝试完毕后仍失败
	return time.Time{}, ErrInvalidDateFormat
}

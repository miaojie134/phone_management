package utils

// CompareStringSlices 比较两个字符串切片是否在长度和内容上都完全相同。
// 如果两个切片都为 nil，则认为它们是相同的。
func CompareStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	// 如果两者都为 nil 或都为空（长度已判断为0），则它们是相等的
	if (a == nil && b == nil) || (len(a) == 0 && len(b) == 0) {
		return true
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

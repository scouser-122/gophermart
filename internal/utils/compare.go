package utils

// EqualFloat32Ptr checks that 2 float32 pointers contains same value or both nil
func EqualFloat32Ptr(a, b *float32, epsilon float32) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	diff := *a - *b
	if diff < 0 {
		diff = -diff
	}
	return diff <= epsilon
}

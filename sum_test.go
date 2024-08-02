package llm_adaptor

import "testing"

// TestSum 测试 Sum 函数。
func TestSum(t *testing.T) {
	result := Sum(5, 3)
	if result != 8 {
		t.Errorf("Sum(5, 3) = %d; want 8", result)
	}
}

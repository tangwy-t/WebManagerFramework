package util

// Coalesce 解引用可能为 nil 的指针：nil 时返回 T 的零值，否则返回 *p。
// 常用于把可空指针扁平化为值语义字段（如监控指标缺值时回退 0）。
func Coalesce[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

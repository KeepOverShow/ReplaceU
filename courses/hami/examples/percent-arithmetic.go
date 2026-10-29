// HAMi v2.7.0 的表达式算术演示：仅验证 Go 运算，不调用 GPU。
package main

import "fmt"

func main() {
	var totalMemory, percentage int32 = 7680, 50
	scoreMemory := totalMemory * (percentage / 100.0)
	fitMemory := totalMemory * percentage / 100
	fmt.Printf("评分表达式: %d MiB\nFit 表达式: %d MiB\n", scoreMemory, fitMemory)
}

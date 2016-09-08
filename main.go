package main

import (
	"zlog/log"
)

func main() {
	// 用法示例
	for i := 0; i < 10; i++ {
		log.Debug(i, "哈哈哈哈哈")
		log.Info(i, "iiiiiii")
		log.Error(i, "fdsjaklfdjsaklfd")
	}
}

package main

import (
	"zlog/log"
	"time"
)

func main() {


	log.Log.Config(true, true, true, true, "logs/abc")

	now := time.Now()

	// 用法示例
	for i := 0; i < 10; i++ {
		log.Debug(i, "哈哈哈哈哈")
		log.Info(i, "iiiiiii")
		log.Error(i, "fdsjaklfdjsaklfd")

	}

	log.Info(time.Since(now))

}

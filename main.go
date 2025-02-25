package main

import (
	"course_seckill/internal"
)

func main() {
	internal.InitDB()
	internal.InitRedis()
	internal.InitKafka()
	internal.RunServer()
	defer closeAll()
}

func closeAll() {
	internal.CloseKafka()
	internal.CloseRedis()
}

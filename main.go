package main

import (
	"ddns/utils"
	"log"

	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("加载 .env 文件失败:", err)
	}
}

func main() {
	log.Println("DDNS 工具启动...")

	if err := utils.UpdateDNS(); err != nil {
		log.Fatal("更新DNS失败:", err)
	}

	log.Println("DNS更新完成")
}

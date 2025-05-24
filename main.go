package main

import (
	"ddns/utils"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func init() {
	// 尝试加载.env文件，如果失败则使用环境变量
	if err := godotenv.Load(); err != nil {
		log.Println("未找到.env文件，使用环境变量")
	}

	// 检查必要的环境变量
	if os.Getenv("CF_API_TOKEN") == "" {
		log.Fatal("CF_API_TOKEN 环境变量未设置")
	}
	if os.Getenv("DOMAIN") == "" {
		log.Fatal("DOMAIN 环境变量未设置")
	}
}

func main() {
	log.Println("DDNS 服务启动...")

	// 启动定时任务
	startDDNSScheduler()
}

// startDDNSScheduler 启动DDNS定时检查任务
func startDDNSScheduler() {
	log.Println("启动DDNS定时检查任务，每分钟检查一次...")

	// 立即执行一次
	if err := utils.UpdateDNS(); err != nil {
		log.Printf("初始DNS检查失败: %v", err)
	}

	// 创建定时器，每分钟执行一次
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("执行定时DNS检查...")
			if err := utils.UpdateDNS(); err != nil {
				log.Printf("DNS检查失败: %v", err)
			}
		}
	}
}

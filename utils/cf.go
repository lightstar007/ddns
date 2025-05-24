package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// CloudflareRecord DNS记录结构
type CloudflareRecord struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

// CloudflareZone Zone结构
type CloudflareZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CloudflareResponse API响应结构
type CloudflareResponse struct {
	Success bool                     `json:"success"`
	Errors  []map[string]interface{} `json:"errors"`
	Result  []CloudflareRecord       `json:"result,omitempty"`
}

// CloudflareZoneResponse Zone API响应结构
type CloudflareZoneResponse struct {
	Success bool                     `json:"success"`
	Errors  []map[string]interface{} `json:"errors"`
	Result  []CloudflareZone         `json:"result"`
}

// UpdateDNS 更新DNS记录
func UpdateDNS() error {
	// 获取当前公网IP
	publicIP, err := getCurrentPublicIP()
	if err != nil {
		return fmt.Errorf("获取公网IP失败: %v", err)
	}

	fmt.Printf("当前公网IP: %s\n", publicIP)

	// 获取环境变量
	apiToken := os.Getenv("CF_API_TOKEN")
	domain := os.Getenv("DOMAIN")

	if apiToken == "" || domain == "" {
		return fmt.Errorf("请在.env文件中设置CF_API_TOKEN和DOMAIN")
	}

	// 从域名中提取根域名来获取Zone ID
	rootDomain := extractRootDomain(domain)
	fmt.Printf("根域名: %s\n", rootDomain)

	// 获取Zone ID
	zoneID, err := getZoneID(apiToken, rootDomain)
	if err != nil {
		return fmt.Errorf("获取Zone ID失败: %v", err)
	}

	fmt.Printf("Zone ID: %s\n", zoneID)

	// 获取现有DNS记录
	recordID, currentIP, err := getDNSRecordID(apiToken, zoneID, domain)
	if err != nil {
		return fmt.Errorf("获取DNS记录ID失败: %v", err)
	}

	// 检查IP是否需要更新
	if currentIP == publicIP {
		fmt.Printf("IP地址未变化，无需更新: %s\n", publicIP)
		return nil
	}

	// 更新DNS记录
	if err := updateDNSRecord(apiToken, zoneID, recordID, domain, publicIP); err != nil {
		return fmt.Errorf("更新DNS记录失败: %v", err)
	}

	fmt.Printf("DNS记录更新成功: %s %s -> %s\n", domain, currentIP, publicIP)
	return nil
}

// getCurrentPublicIP 获取当前机器的公网IP
func getCurrentPublicIP() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get("http://169.254.169.254/latest/meta-data/public-ipv4")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}

// extractRootDomain 从子域名中提取根域名
func extractRootDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], ".")
	}
	return domain
}

// getZoneID 获取域名的Zone ID
func getZoneID(apiToken, rootDomain string) (string, error) {
	url := "https://api.cloudflare.com/client/v4/zones"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var response CloudflareZoneResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if !response.Success {
		return "", fmt.Errorf("API调用失败: %v", response.Errors)
	}

	// 查找匹配的zone
	for _, zone := range response.Result {
		if zone.Name == rootDomain {
			return zone.ID, nil
		}
	}

	return "", fmt.Errorf("未找到域名 %s 的Zone", rootDomain)
}

// getDNSRecordID 获取DNS记录ID和当前IP
func getDNSRecordID(apiToken, zoneID, domain string) (string, string, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?name=%s&type=A", zoneID, domain)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var response CloudflareResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", "", err
	}

	if !response.Success {
		return "", "", fmt.Errorf("API调用失败: %v", response.Errors)
	}

	if len(response.Result) == 0 {
		return "", "", fmt.Errorf("未找到域名 %s 的A记录", domain)
	}

	record := response.Result[0]
	return record.ID, record.Content, nil
}

// updateDNSRecord 更新DNS记录
func updateDNSRecord(apiToken, zoneID, recordID, domain, newIP string) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", zoneID, recordID)

	record := CloudflareRecord{
		Type:    "A",
		Name:    domain,
		Content: newIP,
		TTL:     300,
	}

	jsonData, err := json.Marshal(record)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var response CloudflareResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	if !response.Success {
		return fmt.Errorf("更新失败: %v", response.Errors)
	}

	return nil
}

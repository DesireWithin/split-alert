package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"

	"gopkg.in/yaml.v2"
)

type Config struct {
	PrometheusAlertUrl string                       `yaml:"prometheusAlertUrl"`
	Config             map[string]map[string]string `yaml:"config"`
}

var (
	config     Config
	configLock sync.RWMutex
	configPath = "/opt/splitAlert/config/config.yml"
)

type Alert struct {
	Status string `json:"status"`
}

type AlertGroup struct {
	Status string  `json:"status"`
	Alerts []Alert `json:"alerts"`
}

// splitAlerts 将告警按照 status 分组
func splitAlerts(alerts map[string]interface{}) ([]map[string]interface{}, error) {
	firingData := make(map[string]interface{})
	resolvedData := make(map[string]interface{})

	// 复制顶层字段到两个分类对象中（除了 alerts）
	for key, value := range alerts {
		if key != "alerts" {
			firingData[key] = value
			resolvedData[key] = value
		}
	}

	// 设置分类对象的 status
	firingData["status"] = "firing"
	resolvedData["status"] = "resolved"

	// 初始化 alerts 列表
	firingAlerts := []interface{}{}
	resolvedAlerts := []interface{}{}

	alertsList, ok := alerts["alerts"].([]interface{})
	if !ok {
		return nil, errors.New("invalid alerts format")
	}

	// 根据 alerts 的 status 分类
	for _, alert := range alertsList {
		alertMap, ok := alert.(map[string]interface{})
		if !ok {
			continue
		}
		status, _ := alertMap["status"].(string)
		if status == "firing" {
			firingAlerts = append(firingAlerts, alertMap)
		} else if status == "resolved" {
			resolvedAlerts = append(resolvedAlerts, alertMap)
		}
	}

	// 将分组后的 alerts 添加到对应的分类对象中
	firingData["alerts"] = firingAlerts
	resolvedData["alerts"] = resolvedAlerts

	// 构建结果
	var result []map[string]interface{}
	if len(firingAlerts) > 0 {
		result = append(result, firingData)
	}
	if len(resolvedAlerts) > 0 {
		result = append(result, resolvedData)
	}

	return result, nil
}

// loadConfig 加载配置文件
func loadConfig() error {
	log.Println("📥 Loading config.yaml...")
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var tmp Config
	if err := yaml.Unmarshal(data, &tmp); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	if tmp.PrometheusAlertUrl == "" || tmp.Config == nil {
		return errors.New("missing required keys in config")
	}

	configLock.Lock()
	defer configLock.Unlock()
	config = tmp
	log.Printf("✅ config.yaml reloaded with %d entries\n", len(config.Config))
	return nil
}

// reloadHandler 手动重新加载配置
func reloadHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("🔁 Manual reload triggered via /reload")
	if err := loadConfig(); err != nil {
		log.Printf("❌ Manual reload failed: %v\n", err)
		http.Error(w, "Failed to reload config", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Config reloaded"))
}

// alertHandler 处理告警请求
func alertHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("📥 Incoming alert request")
	cfgName := r.URL.Query().Get("config")
	log.Printf("🔍 Config param: %s\n", cfgName)

	if cfgName == "" {
		log.Println("❗ Missing config param")
		http.Error(w, "Missing config param", http.StatusBadRequest)
		return
	}

	configLock.RLock()
	cfg, exists := config.Config[cfgName]
	configLock.RUnlock()

	if !exists {
		log.Printf("❗ Config %s not found\n", cfgName)
		http.Error(w, "Config not found", http.StatusBadRequest)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Read body error: %v\n", err)
		http.Error(w, "Read error", http.StatusBadRequest)
		return
	}

	var alerts map[string]interface{}
	if err := json.Unmarshal(body, &alerts); err != nil {
		log.Printf("❌ Split alert json error: %v\n", err)
		http.Error(w, "Split error", http.StatusInternalServerError)
		return
	}

	splitAlert, err := splitAlerts(alerts)
	if err != nil {
		log.Printf("❌ Split alert error: %v\n", err)
		http.Error(w, "Split error", http.StatusInternalServerError)
		return
	}

	for _, alertGroup := range splitAlert {
		alertGroupJSON, _ := json.Marshal(alertGroup)
		log.Printf("📦 Forwarding body: %s\n", string(alertGroupJSON))

		queryParams := url.Values{}
		for key, value := range cfg {
			queryParams.Add(key, value)
		}
		finalURL := fmt.Sprintf("%s?%s", config.PrometheusAlertUrl, queryParams.Encode())
		log.Printf("➡️ Forwarding alert to %s\n", finalURL)

		// 发送请求
		resp, err := http.Post(finalURL, "application/json", bytes.NewBuffer(alertGroupJSON))
		if err != nil {
			log.Printf("❌ Forward error: %v\n", err)
			http.Error(w, "Forward failed", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		log.Printf("✅ Successfully forwarded %s alert to %s\n", cfgName, finalURL)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	log.Println("🚀 Starting Alert Router...")

	if err := loadConfig(); err != nil {
		log.Fatalf("❌ Failed to load config: %v\n", err)
		os.Exit(1)
	}

	http.HandleFunc("/reload", reloadHandler)
	http.HandleFunc("/alert", alertHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

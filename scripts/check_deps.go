package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DependencyItem 定義單一依賴項目結構
type DependencyItem struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256,omitempty"`
}

// DependencyConfig 對應 dependencies.json 的結構
type DependencyConfig map[string]DependencyItem

func main() {
	checkMode := flag.Bool("check", false, "唯讀校驗模式：檢查連結並確認 SHA256 是否與 dependencies.json 一致")
	updateMode := flag.Bool("update", false, "自動更新模式：下載依賴並更新 dependencies.json 中的 SHA256 值")
	forceMode := flag.Bool("force", false, "強制更新模式：即使 dependencies.json 中已存在 SHA256，仍強制重新下載並更新")
	flag.Parse()

	if !*checkMode && !*updateMode {
		fmt.Println("❌ 請指定運行模式: --check 或 --update")
		os.Exit(1)
	}

	projectRoot, _ := filepath.Abs(".")
	jsonPath := filepath.Join(projectRoot, "conf", "dependencies.json")
	
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		fmt.Printf("❌ 讀取設定檔失敗: %v\n", err)
		os.Exit(1)
	}

	var cfg DependencyConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Printf("❌ 解析設定檔失敗: %v\n", err)
		os.Exit(1)
	}

	modeName := "校驗模式"
	if *updateMode {
		modeName = "更新模式"
	}
	fmt.Printf("🔍 開始以 [%s] 處理依賴，共 %d 項...\n", modeName, len(cfg))
	
	tempDir, err := os.MkdirTemp("", "wincmp-deps-*")
	if err != nil {
		fmt.Printf("❌ 建立臨時目錄失敗: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	var wg sync.WaitGroup
	var mu sync.Mutex
	hasError := false
	
	for name, item := range cfg {
		wg.Add(1)
		go func(depName string, depItem DependencyItem) {
			defer wg.Done()
			
			// update 模式下，若已存在 SHA-256 且未指定 force，則直接跳過，省去重複下載的時間與頻寬
			if *updateMode && !*forceMode && depItem.SHA256 != "" {
				mu.Lock()
				fmt.Printf("ℹ️ [%s] 已存在 SHA-256 校驗值，跳過下載計算。\n", depName)
				mu.Unlock()
				return
			}
			
			// 1. 檢查連結可用性 (HEAD 請求)
			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Head(depItem.URL)
			urlWorking := false
			if err == nil && (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusFound) {
				resp.Body.Close()
				urlWorking = true
			} else {
				if resp != nil {
					resp.Body.Close()
				}
				// HEAD 失敗時改用 GET 嘗試 (有些 CDN 會封鎖 HEAD)
				respGet, errGet := client.Get(depItem.URL)
				if errGet == nil && respGet.StatusCode == http.StatusOK {
					respGet.Body.Close()
					urlWorking = true
				} else if respGet != nil {
					respGet.Body.Close()
				}
			}

			if !urlWorking {
				mu.Lock()
				fmt.Printf("❌ [%s] 連結失效或伺服器無回應! URL: %s\n", depName, depItem.URL)
				hasError = true
				mu.Unlock()
				return
			}
			
			// 2. 下載檔案計算 SHA-256
			tempFile := filepath.Join(tempDir, depName)
			out, err := os.Create(tempFile)
			if err != nil {
				mu.Lock()
				fmt.Printf("❌ [%s] 無法建立臨時檔案: %v\n", depName, err)
				hasError = true
				mu.Unlock()
				return
			}
			defer out.Close()
			
			dlResp, err := client.Get(depItem.URL)
			if err != nil || dlResp.StatusCode != http.StatusOK {
				mu.Lock()
				fmt.Printf("❌ [%s] 下載失敗 (GET 請求返回異常)\n", depName)
				hasError = true
				mu.Unlock()
				return
			}
			defer dlResp.Body.Close()
			
			h := sha256.New()
			if _, err := io.Copy(out, io.TeeReader(dlResp.Body, h)); err != nil {
				mu.Lock()
				fmt.Printf("❌ [%s] 寫入或計算 Hash 失敗: %v\n", depName, err)
				hasError = true
				mu.Unlock()
				return
			}
			
			shaSum := fmt.Sprintf("%x", h.Sum(nil))
			
			mu.Lock()
			if *checkMode {
				if depItem.SHA256 == "" {
					fmt.Printf("⚠️ [%s] 未設定 SHA-256 值。下載檔案計算值為: %s\n", depName, shaSum)
					hasError = true
				} else if !strings.EqualFold(depItem.SHA256, shaSum) {
					fmt.Printf("❌ [%s] SHA-256 不匹配! 預期: %s, 實際: %s\n", depName, depItem.SHA256, shaSum)
					hasError = true
				} else {
					fmt.Printf("✅ [%s] 連結與 SHA-256 校驗通過。\n", depName)
				}
			} else if *updateMode {
				depItem.SHA256 = shaSum
				cfg[depName] = depItem
				fmt.Printf("📝 [%s] 已取得 SHA-256: %s\n", depName, shaSum)
			}
			mu.Unlock()
		}(name, item)
	}
	
	wg.Wait()
	
	if *updateMode {
		newData, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			fmt.Printf("❌ 序列化更新設定失敗: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(jsonPath, newData, 0644); err != nil {
			fmt.Printf("❌ 寫入設定檔失敗: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("🎉 conf/dependencies.json 更新成功！")
	}
	
	if hasError && *checkMode {
		fmt.Println("❌ 部分依賴連結失效或 SHA-256 校驗失敗。")
		os.Exit(1)
	}
}

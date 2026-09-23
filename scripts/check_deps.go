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
	Version   string `json:"version"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256,omitempty"`
	License   string `json:"license,omitempty"`
	Homepage  string `json:"homepage,omitempty"`
	SourceURL string `json:"source_url,omitempty"`
}

// DependencyConfig 對應 dependencies.json 的結構
type DependencyConfig map[string]DependencyItem

// isValidSHA256 檢查雜湊格式是否為合法的 64 位元十六進位字串
func isValidSHA256(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 64 {
		return false
	}
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

// probeURL 探測 URL 存活性，若遇 PHP 官方 releases 404 則自動嘗試 archives 備援
func probeURL(client *http.Client, rawURL string) (bool, string, error) {
	urls := []string{rawURL}
	if strings.Contains(rawURL, "windows.php.net/downloads/releases/") && !strings.Contains(rawURL, "/archives/") {
		urls = append(urls, strings.Replace(rawURL, "/downloads/releases/", "/downloads/releases/archives/", 1))
	}

	for i, u := range urls {
		req, err := http.NewRequest("HEAD", u, nil)
		if err == nil {
			req.Header.Set("User-Agent", "WinCMP-CI-Checker/2.0")
			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusFound {
					return true, u, nil
				}
			}
		}

		// HEAD 失敗或被擋時改用 GET 帶 Range 探測前 1KB
		reqGet, errGet := http.NewRequest("GET", u, nil)
		if errGet == nil {
			reqGet.Header.Set("User-Agent", "WinCMP-CI-Checker/2.0")
			reqGet.Header.Set("Range", "bytes=0-1023")
			respGet, errGet := client.Do(reqGet)
			if errGet == nil {
				respGet.Body.Close()
				if respGet.StatusCode == http.StatusOK || respGet.StatusCode == http.StatusPartialContent || respGet.StatusCode == http.StatusFound {
					return true, u, nil
				}
				if respGet.StatusCode == http.StatusNotFound && i < len(urls)-1 {
					continue // 嘗試 fallback archives
				}
			}
		}
	}

	return false, "", fmt.Errorf("連結無回應或回傳非成功狀態碼")
}

func main() {
	checkMode := flag.Bool("check", false, "校驗模式：檢查連結並確認 SHA-256")
	fastMode := flag.Bool("fast", false, "快速校驗模式：僅驗證下載連結存活性與 SHA-256 格式，不進行全量下載 (推薦用於 CI 發版)")
	updateMode := flag.Bool("update", false, "自動更新模式：下載依賴並更新 dependencies.json 中的 SHA-256 值")
	forceMode := flag.Bool("force", false, "強制更新模式：即使 dependencies.json 中已存在 SHA-256，仍強制重新下載並更新")
	concurrency := flag.Int("concurrency", 4, "最大下載/校驗併發數 (預設 4)")
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

	modeName := "全量校驗模式 (下載並計算 SHA-256)"
	if *fastMode && *checkMode {
		modeName = "快速校驗模式 (驗證存活與雜湊規格，免下載)"
	} else if *updateMode {
		modeName = "更新模式"
	}
	fmt.Printf("🔍 開始以 [%s] 處理依賴，共 %d 項 (最大併發數: %d)...\n", modeName, len(cfg), *concurrency)

	var tempDir string
	if !*fastMode || *updateMode {
		tempDir, err = os.MkdirTemp("", "wincmp-deps-*")
		if err != nil {
			fmt.Printf("❌ 建立臨時目錄失敗: %v\n", err)
			os.Exit(1)
		}
		defer os.RemoveAll(tempDir)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	hasError := false

	sem := make(chan struct{}, *concurrency)
	client := &http.Client{Timeout: 600 * time.Second}

	for name, item := range cfg {
		wg.Add(1)
		go func(depName string, depItem DependencyItem) {
			sem <- struct{}{}
			defer func() {
				<-sem
				wg.Done()
			}()

			// update 模式下，若已存在 SHA-256 且未指定 force，則直接跳過
			if *updateMode && !*forceMode && depItem.SHA256 != "" {
				mu.Lock()
				fmt.Printf("ℹ️ [%s] 已存在 SHA-256 校驗值，跳過下載計算。\n", depName)
				mu.Unlock()
				return
			}

			// 1. 探測連結存活性 (支援 PHP archives 自動 fallback)
			alive, resolvedURL, _ := probeURL(client, depItem.URL)
			if !alive {
				mu.Lock()
				fmt.Printf("❌ [%s] 連結失效或伺服器無回應! URL: %s\n", depName, depItem.URL)
				hasError = true
				mu.Unlock()
				return
			}

			if resolvedURL != depItem.URL {
				mu.Lock()
				fmt.Printf("⚠️ [%s] 原連結已歸檔或變更，可改用備援 URL: %s\n", depName, resolvedURL)
				mu.Unlock()
			}

			// 2. 快速模式直接完成校驗
			if *fastMode && *checkMode {
				mu.Lock()
				if !isValidSHA256(depItem.SHA256) {
					fmt.Printf("❌ [%s] SHA-256 格式無效或未設定 (長度必須為 64 碼十六進位)! 值: %s\n", depName, depItem.SHA256)
					hasError = true
				} else {
					fmt.Printf("✅ [%s] 連結與 SHA-256 規格校驗通過。\n", depName)
				}
				mu.Unlock()
				return
			}

			// 3. 下載檔案計算 SHA-256（最多重試 3 次）
			targetDownloadURL := resolvedURL
			var shaSum string
			downloadOK := false
			for attempt := 1; attempt <= 3; attempt++ {
				tempFile := filepath.Join(tempDir, fmt.Sprintf("%s-%d", depName, attempt))
				out, err := os.Create(tempFile)
				if err != nil {
					mu.Lock()
					fmt.Printf("❌ [%s] 無法建立臨時檔案: %v\n", depName, err)
					hasError = true
					mu.Unlock()
					return
				}

				req, err := http.NewRequest("GET", targetDownloadURL, nil)
				if err != nil {
					out.Close()
					continue
				}
				req.Header.Set("User-Agent", "WinCMP-CI-Checker/2.0")

				dlResp, err := client.Do(req)
				if err != nil || dlResp == nil || dlResp.StatusCode != http.StatusOK {
					if dlResp != nil {
						dlResp.Body.Close()
					}
					out.Close()
					mu.Lock()
					if err != nil {
						fmt.Printf("⚠️ [%s] 下載失敗 (第 %d/3 次): %v\n", depName, attempt, err)
					} else {
						fmt.Printf("⚠️ [%s] 下載失敗 (第 %d/3 次), status=%d\n", depName, attempt, dlResp.StatusCode)
					}
					mu.Unlock()
					time.Sleep(time.Duration(attempt) * 2 * time.Second)
					continue
				}

				h := sha256.New()
				_, copyErr := io.Copy(io.MultiWriter(out, h), dlResp.Body)
				dlResp.Body.Close()
				out.Close()
				if copyErr != nil {
					mu.Lock()
					fmt.Printf("⚠️ [%s] 寫入或計算 Hash 失敗 (第 %d/3 次): %v\n", depName, attempt, copyErr)
					mu.Unlock()
					time.Sleep(time.Duration(attempt) * 2 * time.Second)
					continue
				}

				shaSum = fmt.Sprintf("%x", h.Sum(nil))
				downloadOK = true
				break
			}

			if !downloadOK {
				mu.Lock()
				fmt.Printf("❌ [%s] 下載失敗，已重試 3 次。URL: %s\n", depName, targetDownloadURL)
				hasError = true
				mu.Unlock()
				return
			}

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
				if resolvedURL != depItem.URL {
					depItem.URL = resolvedURL
				}
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

package composer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHeadTailBufferNoTruncation(t *testing.T) {
	buf := NewHeadTailBuffer(1024, 2048)
	buf.Write([]byte("line 1\nline 2\nline 3"))

	out, truncated, origBytes, origLines := buf.Result()
	if truncated {
		t.Fatalf("預期不應截斷")
	}
	if out != "line 1\nline 2\nline 3" {
		t.Errorf("輸出不符: %q", out)
	}
	if origBytes != 20 || origLines != 3 {
		t.Errorf("統計不符: bytes=%d lines=%d", origBytes, origLines)
	}
}

func TestHeadTailBufferTruncationPreservesHeadTailAndUTF8(t *testing.T) {
	buf := NewHeadTailBuffer(256, 512)
	var total int
	for i := 0; i < 80; i++ {
		line := fmt.Sprintf("第%03d行測試輸出資料內容abcdefghij\n", i)
		total += len(line)
		buf.Write([]byte(line))
	}

	out, truncated, origBytes, origLines := buf.Result()
	if !truncated {
		t.Fatalf("預期應觸發截斷")
	}
	if origBytes != total || origLines != 80 {
		t.Errorf("原始統計不符: got (%d, %d), want (%d, 80)", origBytes, origLines, total)
	}
	if !strings.Contains(out, "第000行測試輸出資料內容") {
		t.Errorf("頭部內容應保留第 000 行")
	}
	if !strings.Contains(out, "第079行測試輸出資料內容") {
		t.Errorf("尾部內容應保留最後一行第 079 行")
	}
	if !strings.Contains(out, "已省略") {
		t.Errorf("應包含省略提示標記，實際: %s", out)
	}
}

func TestHistoryStoreEvictionPaginationAndPathValidation(t *testing.T) {
	base := t.TempDir()
	store := NewHistoryStore(base)

	// 路徑遍歷防禦測試
	for _, bad := range []string{"", "../evil", `a\b`, "a/b", "my.proj"} {
		if err := ValidateProjectName(bad); err == nil {
			t.Errorf("預期拒絕非法專案名稱 %q", bad)
		}
	}

	// 寫入 55 筆，應淘汰最舊 5 筆，只保留最新 50 筆 (05..54)
	for i := 0; i < 55; i++ {
		err := store.Append(HistoryEntry{
			ID:      fmt.Sprintf("id-%02d", i),
			Project: "demo_app",
			Command: fmt.Sprintf("cmd-%02d", i),
			Status:  "success",
		})
		if err != nil {
			t.Fatalf("Append 第 %d 筆失敗: %v", i, err)
		}
	}

	// 第 1 頁：最新 5 筆 (50..54)
	p1, err := store.List("demo_app", 0, 5)
	if err != nil {
		t.Fatalf("List 第 1 頁失敗: %v", err)
	}
	if p1.Total != 50 || !p1.HasMore || len(p1.Items) != 5 {
		t.Fatalf("第 1 頁分頁資訊不符: %+v", p1)
	}
	if p1.Items[0].Command != "cmd-50" || p1.Items[4].Command != "cmd-54" {
		t.Errorf("第 1 頁順序不符: %+v", p1.Items)
	}

	// 最後一頁：offset=45, limit=10 -> 僅剩 5 筆 (05..09), HasMore=false
	pLast, err := store.List("demo_app", 45, 10)
	if err != nil {
		t.Fatalf("List 最後一頁失敗: %v", err)
	}
	if pLast.HasMore || len(pLast.Items) != 5 || pLast.Items[0].Command != "cmd-05" {
		t.Errorf("最後一頁資訊不符: %+v", pLast)
	}

	// 清除歷史
	if err := store.Clear("demo_app"); err != nil {
		t.Fatalf("Clear 失敗: %v", err)
	}
	pAfter, err := store.List("demo_app", 0, 5)
	if err != nil {
		t.Fatalf("Clear 後 List 失敗: %v", err)
	}
	if pAfter.Total != 0 || len(pAfter.Items) != 0 {
		t.Errorf("清除後應為空，實際: %+v", pAfter)
	}
}

// TestHelperProcess 供 Runner 單元測試模擬子行程輸出、失敗、停止與超時
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	mode := os.Getenv("HELPER_MODE")
	switch mode {
	case "ok":
		fmt.Println("Composer version 2.11.2")
		os.Exit(0)
	case "fail":
		fmt.Fprintln(os.Stderr, "Package not found")
		os.Exit(2)
	case "sleep":
		fmt.Println("started...")
		time.Sleep(5 * time.Second)
		os.Exit(0)
	}
	os.Exit(0)
}

func TestRunnerLifecycleAndTimeouts(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	exePath, err := os.Executable()
	if err != nil {
		t.Fatalf("取得測試執行檔失敗: %v", err)
	}

	base := t.TempDir()
	workDir := filepath.Join(base, "www", "demo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("建立工作目錄失敗: %v", err)
	}

	runCase := func(mode string, idle, hard time.Duration, stopEarly bool) HistoryEntry {
		t.Setenv("HELPER_MODE", mode)
		mgr := NewManager(base)
		mgr.IdleTimeout = idle
		mgr.HardTimeout = hard

		doneCh := make(chan HistoryEntry, 1)
		runID, err := mgr.Start(RunOptions{
			ProjectName:     "demo",
			WorkDir:         workDir,
			PHPExe:          exePath,
			PHPVersion:      "8.2.36",
			ComposerVersion: "2.11.2",
			Args:            []string{"show"},
			ExtraArgsPrefix: []string{"-test.run=TestHelperProcess", "--"},
		}, nil, func(_ string, entry HistoryEntry) {
			doneCh <- entry
		})
		if err != nil {
			t.Fatalf("Start(%s) 失敗: %v", mode, err)
		}
		if stopEarly {
			time.Sleep(120 * time.Millisecond)
			_ = mgr.Stop(runID)
		}
		select {
		case entry := <-doneCh:
			return entry
		case <-time.After(4 * time.Second):
			t.Fatalf("等待執行結果逾時 (%s)", mode)
			return HistoryEntry{}
		}
	}

	okEntry := runCase("ok", 5*time.Second, 5*time.Second, false)
	if okEntry.Status != "success" || okEntry.ExitCode != 0 {
		t.Errorf("成功案例狀態異常: %+v", okEntry)
	}

	errEntry := runCase("fail", 5*time.Second, 5*time.Second, false)
	if errEntry.Status != "error" || errEntry.ExitCode != 2 {
		t.Errorf("失敗案例狀態異常: %+v", errEntry)
	}

	stopEntry := runCase("sleep", 5*time.Second, 5*time.Second, true)
	if stopEntry.Status != "stopped" || stopEntry.ErrorReason != "stopped_by_user" {
		t.Errorf("手動停止案例異常: %+v", stopEntry)
	}

	idleEntry := runCase("sleep", 200*time.Millisecond, 5*time.Second, false)
	if idleEntry.Status != "timeout" || !strings.HasPrefix(idleEntry.ErrorReason, "idle_timeout_") {
		t.Errorf("閒置超時案例異常: %+v", idleEntry)
	}
}

func TestBuildPHPComposerArgsInjectsIniAndExtDir(t *testing.T) {
	base := t.TempDir()
	phpDir := filepath.Join(base, "bin", "php", "php-8.2.36")
	extDir := filepath.Join(phpDir, "ext")
	confPhpDir := filepath.Join(base, "conf", "php")
	sslDir := filepath.Join(base, "conf", "ssl")
	for _, d := range []string{extDir, confPhpDir, sslDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("建立目錄失敗: %v", err)
		}
	}
	iniFile := filepath.Join(confPhpDir, "php.ini")
	caFile := filepath.Join(sslDir, "cacert.pem")
	if err := os.WriteFile(iniFile, []byte("extension=openssl\n"), 0644); err != nil {
		t.Fatalf("建立 php.ini 失敗: %v", err)
	}
	if err := os.WriteFile(caFile, []byte("CA"), 0644); err != nil {
		t.Fatalf("建立 cacert.pem 失敗: %v", err)
	}

	args := buildPHPComposerArgs(base, RunOptions{
		PHPExe:       filepath.Join(phpDir, "php.exe"),
		ComposerPhar: filepath.Join(base, "bin", "composer", "composer-2.11.2", "composer.phar"),
		Args:         []string{"show"},
	})
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"-c " + iniFile,
		"-d extension_dir=" + extDir,
		"-d openssl.cafile=" + filepath.ToSlash(caFile),
		"-d curl.cainfo=" + filepath.ToSlash(caFile),
		"-d memory_limit=-1",
		"--ansi show",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("預期參數包含 %q，實際為 %q", want, joined)
		}
	}
}

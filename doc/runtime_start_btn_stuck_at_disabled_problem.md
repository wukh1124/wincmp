# Runtime Start 按鈕卡在 Disabled 狀態的分析報告

**分析日期**: 2026-04-13  
**適用版本**: WinCMP v1.2.0+  
**分析檔案**: `ui_runtime.go` (Line 357 ~ 493)

---

## 摘要

本文分析 WinCMP Runtime Tab 中，Action 區塊的 Start 按鈕會出現黑色、不可點擊 (Disabled) 狀態的所有情境。

---

## 🔴 永久性 Disabled（按鈕一直卡住，需人為改設定才能恢復）

### 1. 專案未啟用 (`proj.Enabled == false`)

**程式碼位置**: `ui_runtime.go:385-388`

```go
if !proj.Enabled {
    startStopBtn.SetText("Start")
    startStopBtn.SetIcon(theme.MediaPlayIcon())
    startStopBtn.Disable()   // ← 按鈕 disabled
    modeSelect.Disable()
}
```

**說明**: 在專案編輯器中將 `Enabled` 取消勾選後，Start 按鈕會永久變黑，Mode 下拉選單也會同時被禁用。

**恢復方式**: 進入專案設定，重新勾選 `Enabled`。

---

### 2. Node / Bun 類型但掃描不到任何版本

**程式碼位置**: `ui_runtime.go:390-394`

```go
else if len(versions) == 0 && (resolvedRT == "node" || resolvedRT == "bun") {
    startStopBtn.SetText("Start")
    startStopBtn.SetIcon(theme.MediaPlayIcon())
    startStopBtn.Disable()   // ← 按鈕 disabled
    modeSelect.Disable()
}
```

**說明**: 當 `RuntimeType` 為 `"node"` 或 `"bun"`（包含 `"auto"` 解析後為 node 的情況）時，如果 `bin/node/` 或 `bin/bun/` 目錄下掃描不到任何可執行檔，按鈕會永久變黑。

**版本掃描邏輯** (`ui_runtime.go:304-341`):
```go
resolvedRT := proj.RuntimeType
if resolvedRT == "auto" {
    if hasBun { resolvedRT = "bun" }
    else      { resolvedRT = "node" }
}
// node → 遍歷 scanRes.NodeList
// bun  → 遍歷 scanRes.BunList
// python/go_air/go_run/custom → 使用固定版本字串
```

**影響 `versions` 為空 (`len(versions) == 0`) 的情境**:
- `RuntimeType == "node"`: `scanRes.NodeList` 為空（`bin/node/` 沒有執行檔）
- `RuntimeType == "bun"`: `scanRes.BunList` 為空（`bin/bun/` 沒有執行檔）
- `RuntimeType == "auto"`: `scanRes.BunList` 和 `scanRes.NodeList` 都為空

**恢復方式**: 在 `bin/node/` 或 `bin/bun/` 目錄放入對應的執行檔，然後重啟或切換 Tab 觸發重新掃描。

---

## 🟡 暫時性 Disabled（正常操作流程會自動恢復）

### 3. 正在啟動中 (`Starting...`)

**程式碼位置**: `ui_runtime.go:402-404`

```go
startStopBtn.SetText("Starting...")
startStopBtn.SetIcon(theme.ViewRefreshIcon())
startStopBtn.Disable()   // ← 按鈕暫時 disabled
filterBtn.Disable()
```

**說明**: 使用者按下 Start 後，按鈕短暫變灰（顯示 "Starting..." 並旋轉圖示），等 `procMgr.StartRuntime()` 在 goroutine 中執行完畢後，會自動恢復。

**goroutine 回調邏輯** (`ui_runtime.go:480-491`):
```go
go func() {
    err := procMgr.StartRuntime(proj, modeSelect.Selected, exePath)
    fyne.Do(func() {
        if err != nil {
            startStopBtn.SetText("Start")
            startStopBtn.SetIcon(theme.MediaPlayIcon())
            startStopBtn.Enable()
            filterBtn.Enable()
        }
        list.RefreshItem(i)
    })
}()
```

---

### 4. 正在停止中 (`Stopping...`)

**程式碼位置**: `ui_runtime.go:373-376`

```go
startStopBtn.SetText("Stopping...")
startStopBtn.SetIcon(theme.ViewRefreshIcon())
startStopBtn.Disable()   // ← 按鈕暫時 disabled
filterBtn.Disable()
```

**說明**: 使用者按下 Stop 後，按鈕暫時變灰（顯示 "Stopping..." 並旋轉圖示），等 `procMgr.StopRuntime()` 完成後恢復。

---

### 5. 啟動失敗但 UI 未正確刷新

**程式碼位置**: `ui_runtime.go:480-491`

**說明**: `StartRuntime()` 在 goroutine 中執行，失敗時雖然會呼叫 `startStopBtn.Enable()` 和 `list.RefreshItem(i)`，但如果 `RefreshItem` 因 List 狀態問題未正確觸發，或 `fyne.Do` 排程被延遲，按鈕可能看起來仍然卡住。

**恢復方式**: 手動切換 Tab 或重啟應用程式強制刷新。

---

## 🟠 啟動流程中會中斷（按鈕會恢復，但服務未啟動）

以下情境會在啟動流程中透過 `return` 中斷，但按鈕會恢復為可用狀態，所以**不是永久黑掉**，但可能造成「按了沒反應」的困惑。

### 6. Python / Go 環境檢查失敗

**程式碼位置**: `ui_runtime.go:408-424`

```go
if process.IsRuntimeTypeNeedEnvCheck(proj.RuntimeType) {
    ver, err := process.CheckRuntimeEnv(proj.RuntimeType)
    if err != nil {
        addErrorLog("runtime", fmt.Sprintf("[%s] %v", proj.Name, err), nil)
        fyne.Do(func() {
            dialog.ShowError(err, win)
            startStopBtn.SetText("Start")
            startStopBtn.SetIcon(theme.MediaPlayIcon())
            startStopBtn.Enable()   // ← 按鈕恢復
            filterBtn.Enable()
        })
        return  // ← 啟動中斷
    }
    if ver != "" {
        addLog("runtime", fmt.Sprintf("ℹ️ [%s] 偵測到 %s", proj.Name, ver))
    }
}
```

**適用類型** (`internal/process/runtime.go:87-93`):
- `"python"`
- `"go_air"`
- `"go_run"`
- `"go"` (舊版相容)

**失敗原因**: 系統 PATH 中找不到 `python` 或 `go` 執行檔。

---

### 7. 版本路徑為空

**程式碼位置**: `ui_runtime.go:427-438`

```go
if proj.RuntimeType != "custom" && proj.RuntimeType != "python"
   && proj.RuntimeType != "go_air" && proj.RuntimeType != "go_run" {
    if proj.RuntimeVersion == "" || len(versionPathMap) == 0 {
        runtimeLabel := GetRuntimeTypeLabel(proj.RuntimeType)
        addErrorLog("runtime", fmt.Sprintf("[%s] 沒有可用的 %s 版本，請至 bin/ 檢查", proj.Name, runtimeLabel), nil)
        fyne.Do(func() {
            startStopBtn.SetText("Start")
            startStopBtn.SetIcon(theme.MediaPlayIcon())
            startStopBtn.Enable()   // ← 按鈕恢復
            filterBtn.Enable()
        })
        return  // ← 啟動中斷
    }
}
```

**說明**: Node / Bun 類型在 `UseWinCMPBin == true` 時，需要 `RuntimeVersion` 有值且 `versionPathMap` 不為空。如果 `RuntimeVersion` 為空字串（從未選過版本），就會阻擋啟動。

---

### 8. Port 被佔用

**程式碼位置**: `ui_runtime.go:442-452`

```go
port := proj.RuntimePort
if port > 0 && !process.IsPortAvailable(port) {
    addErrorLog("runtime", fmt.Sprintf("[%s] 啟動失敗當前端口 %d 不可用", proj.Name, port), nil)
    fyne.Do(func() {
        dialog.ShowInformation("啟動失敗", fmt.Sprintf("當前端口 %d 不可用", port), win)
        startStopBtn.SetText("Start")
        startStopBtn.SetIcon(theme.MediaPlayIcon())
        startStopBtn.Enable()   // ← 按鈕恢復
        filterBtn.Enable()
    })
    return  // ← 啟動中斷
}
```

**說明**: 如果 `RuntimePort > 0` 且該 port 已被其他進程佔用，啟動會中斷並顯示訊息。

---

## 📊 總結表格

| # | 情境 | 按鈕狀態 | 恢復方式 | Mode 下拉 | 對應程式碼 |
|---|------|---------|---------|-----------|-----------|
| 1 | `Enabled == false` | **永久黑色** | 專案設定勾選 Enabled | 也被禁用 | `ui_runtime.go:385` |
| 2 | Node/Bun 無掃描版本 | **永久黑色** | 放入執行檔到 bin/ | 也被禁用 | `ui_runtime.go:390` |
| 3 | Starting... 啟動中 | 暫時黑色 | goroutine 完成後恢復 | 也被禁用 | `ui_runtime.go:402` |
| 4 | Stopping... 停止中 | 暫時黑色 | goroutine 完成後恢復 | 也被禁用 | `ui_runtime.go:373` |
| 5 | 啟動失敗 + UI 未刷新 | 疑似卡住 | 切換 Tab 或重啟 | - | `ui_runtime.go:480` |
| 6 | Python/Go 不在 PATH | 啟動被阻擋 | 安裝工具到 PATH | - | `ui_runtime.go:408` |
| 7 | 版本路徑為空 | 啟動被阻擋 | 重新掃描 bin/ | - | `ui_runtime.go:427` |
| 8 | Port 被佔用 | 啟動被阻擋 | 釋放或改 port | - | `ui_runtime.go:442` |

---

## 🔍 快速診斷流程圖

```
Start 按鈕是黑的？
  │
  ├─ 是否為 Stop 狀態（非 Running）？
  │    │
  │    ├─ YES → proj.Enabled 是否為 true？
  │    │         ├─ NO  → 原因 1：Enabled = false（去專案設定勾回來）
  │    │         └─ YES → RuntimeType 是 node/bun/auto？
  │    │                   ├─ YES → bin/node/ 或 bin/bun/ 有執行檔嗎？
  │    │                   │         ├─ NO  → 原因 2：無掃描版本（放入執行檔）
  │    │                   │         └─ YES → 其他原因（回報 Issue）
  │    │                   └─ NO  → 確認系統有對應工具（Python/Go）
  │    │
  │    └─ NO  → 是否顯示 "Starting..." 或 "Stopping..."？
  │              ├─ YES → 原因 3/4：正常等待流程
  │              └─ NO  → 原因 5：UI 未刷新（切換 Tab 或重啟）
  │
  └─ 是否為 Running 狀態？（按鈕顯示 Stop）
       └─ YES → 這是正常的，Stop 按鈕本來就是紅色的
```

---

## 🛠 常用 Debug 指令

### 檢查 bin 目錄是否有 Node/Bun 執行檔
```cmd
dir /b bin\node\*
dir /b bin\bun\*
```

### 檢查系統 PATH 是否有 Python / Go
```cmd
python -V
go version
```

### 檢查 Port 是否被佔用
```cmd
netstat -ano | findstr :<port>
```

---

## 📝 備註

- 按鈕的紅色（Stop 狀態）和綠色（Start 狀態）由 `coloredButtonTheme` 控制（`main.go:1257`），是裝飾性主題，不影響邏輯 disabled 狀態。
- 按鈕的「黑色 disabled」狀態是 Fyne Widget 預設的視覺效果，`Disable()` 呼叫後自動套用。
- Node/Bun 的版本掃描由 `scanner.ScanBinDir()` 執行（`internal/scanner/`），結果快取在 `scanRes` 全域變數中。

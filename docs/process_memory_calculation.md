# 行程記憶體計算方式

## 概述

WinCMP 使用 `github.com/shirou/gopsutil/v3/process` 讀取行程記憶體資訊。在 Windows 上，gopsutil 的 `MemoryInfo()` 回傳 `MemoryInfoStat` 結構體，底層呼叫 Windows API `GetProcessMemoryInfo()`（`PROCESS_MEMORY_COUNTERS`）。

本文件說明 WinCMP 資源監控使用的兩種記憶體計算方式。

---

## 記憶體欄位

### RSS — Resident Set Size（`memInfo.RSS`）

**實作（Windows）：**
```go
// gopsutil/process/process_windows.go
ret := &MemoryInfoStat{
    RSS: uint64(mem.WorkingSetSize),   // PROCESS_MEMORY_COUNTERS.WorkingSetSize
    VMS: uint64(mem.PagefileUsage),    // PROCESS_MEMORY_COUNTERS.PagefileUsage
}
```

| 欄位 | Windows API 計數器 | 說明 |
|------|------------------|------|
| `RSS` | `WorkingSetSize` | 目前實際駐留在實體 RAM 中的記憶體分頁。**含所有已載入 DLL 及共享區段**的分頁。 |
| `VMS` | `PagefileUsage` | 此行程已預留的分頁檔虛擬記憶體。**僅含私有已認可的分頁**。 |

---

## 兩個數值為何不同

```
RSS (WorkingSetSize)  = 424 MB   ← 目前在實體記憶體中的所有分頁（含共享 DLL）
VMS (PagefileUsage)   = 339 MB   ← 私有已認可的虛擬記憶體
```

以一般的 Fyne 桌面應用程式為例：

| 記憶體來源 | 計入 RSS | 計入 VMS |
|-----------|:-------:|:-------:|
| Go runtime heap（私有） | ✅ | ✅ |
| Fyne UI / OpenGL 紋理 | ✅ | ✅ |
| 共享 DLL（kernel32、gdi32、user32、opengl32 等） | ✅ | ❌ |
| 記憶體映射共享區段 | ✅ | ❌ |
| Fyne/GPU 驅動程式載入的 DLL | ✅ | ❌ |

**RSS > VMS** 是正常的，所有載入大型共享系統 DLL 的 Windows 桌面應用程式都是如此。

---

## 與工作管理員的相容性

Windows 工作管理員不同位置的計算方式有顯著差異：

*   **「詳細資料」 (Details) 頁面**：預設顯示的是 **Working Set (Memory)**，這與 WinCMP 目前使用的 `RSS` 數值**完全一致**。
*   **「處理程序」 (Processes) 頁面**：預設顯示的是 **Private Working Set**。它會從 Working Set 中扣除「與其他行程共享的記憶體」（例如 `kernel32.dll`、`opengl32.dll` 等系統共享庫）。

### 為何仍有細微差距？

若 WinCMP 顯示 323 MB 而工作管理員顯示 240 MB，這是正常的：
- **WinCMP (RSS)** = 323 MB (Private + Shared)
- **工作管理員 (Private)** = 240 MB (僅 Private)
- **差距 (83 MB)** = 程式載入的共享系統元件、GPU 驅動 DLL 等。

WinCMP 選擇使用 `RSS` 是為了反映程式實際佔用的總物理記憶體，且這也是跨平台工具（如 `gopsutil`）最通用的指標。

---

## 程式碼參照

### WinCMP 資源監控（`internal/resource/monitor.go`）

```go
// WinCMP 主程式記憶體（RSS = WorkingSetSize，實體記憶體）
if memInfo, err := m.proc.MemoryInfo(); err == nil {
    ramMB := memInfo.RSS / 1024 / 1024  // WorkingSetSize
}

// Stack Total 記憶體（所有子行程加總）
for _, pid := range pm.GetAllPIDs() {
    if p, err := process.NewProcess(int32(pid)); err == nil {
        if memInfo, err := p.MemoryInfo(); err == nil {
            totalRAM += memInfo.RSS  // WorkingSetSize
        }
    }
}
```

---

## 計算方式比較

目前 WinCMP 顯示值與工作管理員相近，主要是因為使用了 `RSS`：

| 指標 | gopsutil 欄位 | 計算公式 | 意義 |
|------|-------------|---------|------|
| RSS（WorkingSetSize） | `memInfo.RSS` | `memInfo.RSS / 1024 / 1024` | 實際在物理RAM中的大小（含共享分頁） |
| VMS（PagefileUsage） | `memInfo.VMS` | `memInfo.VMS / 1024 / 1024` | 已認可的虛擬記憶體大小 |

修改位置：`internal/resource/monitor.go` 中的 `fetchResourceData()` 函式。

---

## 延伸討論：CPU 百分比正規化

若需要 **CPU 百分比正規化**，目前做法是：

```go
normalizedCPU := cpuPercent / float64(runtime.NumCPU())
```

這會除以機器的邏輯 CPU 核心總數。例如在 8 核心機器上，一個只使用單核心的行程即使該核心滿載，正規化後也只會顯示約 12.5%。

這個做法有助於在不同核心數的機器間比較資源使用，但可能讓單核心高負載看起來比預期小。

---

## 特殊情況：動態 PID 追蹤 (Dynamic PID Tracking)

對於某些服務（如 **Node.js / Next.js**），啟動時的 PID 可能是暫時的 shell (cmd.exe)，或者會動態產生多層子進程。

WinCMP 針對這類服務實作了動態追蹤機制：
1. **Port 偵測**：對於 Node.js 項目，WinCMP 會定期透過 `netstat` 找到真正監聽該 Port 的進程 PID。
2. **進程樹遞迴**：找到主進程後，會自動遞迴抓取其所有子進程 (Children PIDs)，並將整個進程樹納入 `Stack Total` 計算。
3. **動態更新**：這類服務的 PID 列表會在運行期間每隔數秒自動更新一次。

---

## 備註

- **這裡顯示的是 WinCMP 主程式本體資源，不是整個開發環境的總資源。**
- 若要顯示整個 WinCMP Stack Total Usage（主程式 + Caddy + MariaDB + PHP-CGI + Node.js 的 PID 資源總和），可將 `EnableStackTotal` 常數設為 `true`，並透過 `process.Manager.GetAllPIDs()` 取得所有子行程 PID 後逐一加總。

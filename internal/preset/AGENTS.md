# internal/preset

## OVERVIEW
專案類型 Preset 系統：定義 13 種框架配置與 8 種 Runtime，負責偵測、指令生成、向後相容映射。

## STRUCTURE
```
preset.go  (1105行)
├── 常數定義：13 Type + 8 Runtime
├── Preset 結構體與預設配置表
├── 查詢函式：GetPreset / GetAllPresets / GetPresetLabels
├── Runtime 解析：ResolveRuntime / ResolveRuntimeFromProject
├── 指令構建：BuildStartCommand / BuildStartCommandWithExePath
├── 專案偵測：DetectProjectPreset + 各類型偵測器
└── 向後相容：NormalizeProjectType / NormalizeRuntimeType
```

## WHERE TO LOOK
| 任務 | 函式 |
|------|------|
| 取得 Preset 配置 | `GetPreset(typeID)` |
| 列出所有 Preset | `GetAllPresets()` |
| 解析 Auto Runtime | `ResolveRuntime(typeID, hasBun)` |
| 生成啟動指令 | `BuildStartCommand(typeID, runtime, port, customCmd)` |
| 帶 exePath 的指令 | `BuildStartCommandWithExePath(...)` |
| 偵測專案類型 | `DetectProjectPreset(rootDir)` |
| 判斷是否 Python | `IsPythonType(typeID)` |
| 標準化舊值 | `NormalizeProjectType()` / `NormalizeRuntimeType()` |
| Label ↔ ID 轉換 | `GetPresetIDByLabel()` / `GetRuntimeIDByLabel()` |

## KEY CONSTANTS
```go
// 13 種專案類型
TypeStatic, TypeLaravel, TypeNext, TypeNuxt, TypeAstro,
TypeVite, TypePython, TypePythonDjango, TypePythonFastAPI,
TypePythonFlask, TypeGoAPI, TypePocketBase, TypeCustom

// 8 種 Runtime
RuntimeAuto, RuntimeNone, RuntimeNode, RuntimeBun,
RuntimePython, RuntimeGoAir, RuntimeGoRun, RuntimeCustom
```

## COMMAND TEMPLATES
佔位符會被替換：
- `%PORT%` → 實際 Port 號
- `%HOST%` → 綁定主機 (通常 0.0.0.0)
- `%PROJECT_DIR%` → 專案根目錄
- `%BIN_DIR%` → bin 目錄路徑

## CONVENTIONS
- **DetectPriority**：數字越小優先度越高 (Laravel=1, Custom=999)
- **Auto Runtime**：檢查 bin/bun 存在 → Bun，否則 Node
- **Runtime 專案**：`IsRuntimeProject=true` 會顯示在 Runtime Tab
- **CommandDirtyProtected**：設為 true 時保護使用者手動修改的命令
- **向後相容**：舊值 "go" → TypeGoAPI，"node"/"bun" → TypeVite

## ANTI-PATTERNS
- 不要直接修改 `presets` map，使用查詢函式
- 不要假設 Runtime 已解析，呼叫端負責處理 `RuntimeAuto`
- 不要把 Python 子類型混用，用 `IsPythonType()` 判斷
- 不要跳過 `Normalize*()` 處理使用者輸入的舊資料

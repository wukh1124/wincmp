package main

import (
	"fmt"
	"image/color"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"wincmp/internal/config"
	"wincmp/internal/i18n"
	"wincmp/internal/preset"
	"wincmp/internal/process"
	"wincmp/internal/scanner"

	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
)

// runtimeRowLayout 自定義佈局，確保各欄位權重與寬度一致
type runtimeRowLayout struct{}

func (n *runtimeRowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 7 {
		return fyne.NewSize(0, 0)
	}
	h := float32(0)
	for _, o := range objects {
		ms := o.MinSize()
		if ms.Height > h {
			h = ms.Height
		}
	}
	return fyne.NewSize(630, h)
}

func (n *runtimeRowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 7 {
		return
	}
	// 寬度定義
	nameW := float32(150)
	statusW := float32(80)
	portW := float32(50)
	typeW := float32(130) // 剛好顯示最長的 "Python FastAPI"
	modeW := float32(150)
	actionW := float32(180) // Start + Filter 按鈕
	padding := theme.Padding()

	// 計算彈性寬度 (Domain)
	fixedW := nameW + statusW + portW + typeW + modeW + actionW + (padding * 6)
	domainW := size.Width - fixedW
	if domainW < 150 {
		domainW = 150
	}

	x := float32(0)
	// 0: Project Name (150)
	objects[0].Resize(fyne.NewSize(nameW, size.Height))
	objects[0].Move(fyne.NewPos(x, 0))
	x += nameW + padding

	// 1: Status (80)
	objects[1].Resize(fyne.NewSize(statusW, size.Height))
	objects[1].Move(fyne.NewPos(x, 0))
	x += statusW + padding

	// 2: Port (50)
	objects[2].Resize(fyne.NewSize(portW, size.Height))
	objects[2].Move(fyne.NewPos(x, 0))
	x += portW + padding

	// 3: Runtime Type (100)
	objects[3].Resize(fyne.NewSize(typeW, size.Height))
	objects[3].Move(fyne.NewPos(x, 0))
	x += typeW + padding

	// 4: Domain (Expanding)
	objects[4].Resize(fyne.NewSize(domainW, size.Height))
	objects[4].Move(fyne.NewPos(x, 0))
	x += domainW + padding

	// 5: Run Mode
	objects[5].Resize(fyne.NewSize(modeW, size.Height))
	objects[5].Move(fyne.NewPos(x, 0))
	x += modeW + padding

	// 6: Action
	objects[6].Resize(fyne.NewSize(actionW, size.Height))
	objects[6].Move(fyne.NewPos(x, 0))
}

// IsRuntimeProject 判斷專案是否為 Runtime 類型 (需要顯示在 Runtime Tab)
func IsRuntimeProject(p config.ProjectConfig) bool {
	return preset.IsRuntimeProject(p.Type)
}

// GetRuntimeTypeLabel 取得 Runtime 類型的顯示名稱
func GetRuntimeTypeLabel(t string) string {
	return preset.GetRuntimeLabel(t)
}

// GetProjectFullTypeLabel 取得 框架 (Runtime) 格式標籤
func GetProjectFullTypeLabel(p config.ProjectConfig) string {
	hasBun := len(scanRes.BunList) > 0
	return preset.GetFullTypeLabel(p.Type, p.RuntimeType, hasBun)
}

func createRuntimeTab(win fyne.Window) (fyne.CanvasObject, func()) {
	title := widget.NewLabelWithStyle("Runtime Projects", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// 進入分頁時重新掃描 bin/ 取得最新版本（異步執行避免阻塞 UI）
	go func() {
		tmpRes, err := scanner.ScanBinDir(baseDir)
		if err == nil {
			fyne.Do(func() {
				scanRes.NodeList = tmpRes.NodeList
				scanRes.BunList = tmpRes.BunList
			})
		}
	}()

	// Header layout
	createHeaderRect := func(w float32, label string) fyne.CanvasObject {
		r := canvas.NewRectangle(color.Transparent)
		r.SetMinSize(fyne.NewSize(w, 0))
		return container.NewStack(widget.NewLabelWithStyle(label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), r)
	}

	header := container.New(&runtimeRowLayout{},
		createHeaderRect(150, "Project Name"),
		createHeaderRect(80, "Status"),
		createHeaderRect(50, "Port"),
		createHeaderRect(130, "Type"),
		createHeaderRect(150, "Domain"),
		createHeaderRect(150, "Run Mode"),
		createHeaderRect(180, "Action"),
	)
	headerContainer := container.NewVBox(header)

	// List items tracking
	var runtimeProjects []int
	refreshList := func() {
		runtimeProjects = nil
		for i, p := range appCfg.Projects {
			if IsRuntimeProject(p) {
				runtimeProjects = append(runtimeProjects, i)
			}
		}
	}
	refreshList()

	var list *widget.List
	list = widget.NewList(
		func() int { return len(runtimeProjects) },
		func() fyne.CanvasObject {
			projectBox := container.NewStack()
			pathBox := container.NewStack()
			statusBox := container.NewStack()
			portBox := container.NewStack()
			typeBox := container.NewStack()

			startStopBtn := widget.NewButton("Start", nil)
			startStopBtnWrapped := container.NewThemeOverride(startStopBtn, &coloredButtonTheme{
				isStop: func() bool { return startStopBtn.Text == "Stop" || startStopBtn.Text == "Stopping..." },
			})

			modeSelect := widget.NewSelect([]string{"Background", "Terminal"}, nil)
			modeSelect.PlaceHolder = "Mode"
			modeSelectWrapped := container.NewStack(modeSelect)

			// Filter 按鈕：切換 Terminal Logs 至此項目的 Runtime 分頁
			filterBtn := ttwidget.NewButtonWithIcon("", theme.DocumentIcon(), nil)
			filterBtn.Importance = widget.LowImportance

			actionGroup := container.NewBorder(nil, nil, nil, filterBtn, startStopBtnWrapped)

			return container.New(&runtimeRowLayout{},
				projectBox,
				statusBox,
				portBox,
				typeBox,
				pathBox,
				modeSelectWrapped,
				actionGroup,
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if int(i) >= len(runtimeProjects) {
				return
			}
			idx := runtimeProjects[i]
			if idx >= len(appCfg.Projects) {
				return
			}
			proj := appCfg.Projects[idx]

			row := o.(*fyne.Container)
			projectBox := row.Objects[0].(*fyne.Container)
			statusBox := row.Objects[1].(*fyne.Container)
			portBox := row.Objects[2].(*fyne.Container)
			typeBox := row.Objects[3].(*fyne.Container)
			pathBox := row.Objects[4].(*fyne.Container)

			modeSelectWrapped := row.Objects[5].(*fyne.Container)
			modeSelect := modeSelectWrapped.Objects[0].(*widget.Select)

			actionGroup := row.Objects[6].(*fyne.Container)
			startStopBtnWrapped := actionGroup.Objects[0].(*container.ThemeOverride)
			startStopBtn := startStopBtnWrapped.Content.(*widget.Button)
			filterBtn := actionGroup.Objects[1].(*ttwidget.Button)

			// Filter 按鈕：切換到此項目的 log 並啟動 Runtime 分頁
			filterBtn.OnTapped = func() {
				ensureRuntimeLogBinding(proj.Name)
				switchRuntimeLog(proj.Name)
				if logTabs != nil {
					logTabs.SelectIndex(5) // Runtime 是第 6 個 tab（index 5）
				}
			}
			filterBtn.SetToolTip(i18n.Tfmt("切換 Terminal Logs 至 Runtime (%s)", proj.Name))

			// 1. Project
			projectNameHover := ttwidget.NewLabel(proj.Name)
			projectNameHover.SetToolTip(proj.Name)
			projectNameHover.TextStyle = fyne.TextStyle{Bold: true}
			projectNameHover.Truncation = fyne.TextTruncateEllipsis
			projectBox.Objects = []fyne.CanvasObject{projectNameHover}
			projectBox.Refresh()

			// 2. Status
			serviceKey := process.RuntimeServiceKey(proj.Name)
			isRunningByManager := procMgr.IsRunning(serviceKey)
			port := proj.RuntimePort
			if port == 0 {
				port = 3000
			}
			isRunningByPort := process.CheckRuntimeRunning(port)
			isRunning := isRunningByManager || isRunningByPort

			statusText := "Stopped"
			var statusColor color.Color = color.NRGBA{R: 158, G: 158, B: 158, A: 255}
			if isRunning {
				statusText = "Running"
				statusColor = color.NRGBA{R: 76, G: 175, B: 80, A: 255}
			}

			statusLabel := canvas.NewText(statusText, statusColor)
			statusLabel.TextStyle = fyne.TextStyle{Bold: true}
			statusBox.Objects = []fyne.CanvasObject{statusLabel}
			statusBox.Refresh()

			// 3. Port
			portStr := fmt.Sprintf("%d", proj.RuntimePort)
			if proj.RuntimePort == 0 {
				portStr = "-"
			}
			portLabel := canvas.NewText(portStr, theme.ForegroundColor())
			portBox.Objects = []fyne.CanvasObject{portLabel}
			portBox.Refresh()

			// 4. Runtime Type
			fullTypeLabel := GetProjectFullTypeLabel(proj)
			typeText := canvas.NewText(fullTypeLabel, theme.ForegroundColor())
			typeText.TextStyle = fyne.TextStyle{Bold: true}
			typeBox.Objects = []fyne.CanvasObject{typeText}
			typeBox.Refresh()

			// 5. Domain + Copy Button
			domainStr := "-"
			if len(proj.Domains) > 0 {
				domainStr = proj.Domains[0]
			}
			domainHover := ttwidget.NewLabel(domainStr)
			domainHover.SetToolTip(domainStr)
			domainHover.Truncation = fyne.TextTruncateEllipsis

			copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
				if domainStr == "" || domainStr == "-" {
					dialog.ShowInformation(i18n.T("複製失敗"), i18n.T("無效的 Domain，無法複製連結"), win)
					addLog("system", i18n.Tfmt("❌ 複製連結失敗 [%s]: 無效的 Domain", proj.Name))
					return
				}
				urlPrefix := "http://"
				if proj.UseSSL {
					urlPrefix = "https://"
				}
				win.Clipboard().SetContent(urlPrefix + domainStr)
				addLog("system", i18n.Tfmt("✅ 已複製連結 [%s]: %s%s", proj.Name, urlPrefix, domainStr))
			})
			copyBtn.Importance = widget.LowImportance

			pathBox.Objects = []fyne.CanvasObject{container.NewBorder(nil, nil, nil, copyBtn, domainHover)}
			pathBox.Refresh()

			// 6. Version selector - 根據 Runtime 類型顯示不同版本清單
			var versions []string
			var versionPathMap map[string]string
			resolvedRT := proj.RuntimeType
			hasBun := len(scanRes.BunList) > 0
			if resolvedRT == "auto" {
				if hasBun {
					resolvedRT = "bun"
				} else {
					resolvedRT = "node"
				}
			}
			switch resolvedRT {
			case "node":
				versions = []string{}
				versionPathMap = map[string]string{}
				for _, n := range scanRes.NodeList {
					versions = append(versions, n.Version)
					versionPathMap[n.Version] = n.ExePath
				}
			case "bun":
				versions = []string{}
				versionPathMap = map[string]string{}
				for _, b := range scanRes.BunList {
					versions = append(versions, b.Version)
					versionPathMap[b.Version] = b.ExePath
				}
			case "python":
				versions = []string{"System Python"}
				versionPathMap = map[string]string{"System Python": "python"}
			case "go_air":
				versions = []string{"System Go + Air"}
				versionPathMap = map[string]string{"System Go + Air": "air"}
			case "go_run":
				versions = []string{"System Go"}
				versionPathMap = map[string]string{"System Go": "go"}
			case "custom":
				versions = []string{"Custom Command"}
				versionPathMap = map[string]string{}
			default:
				versions = []string{}
				versionPathMap = map[string]string{}
			}

			if proj.RuntimeMode == "" {
				proj.RuntimeMode = "Background"
			}
			modeSelect.Selected = proj.RuntimeMode

			if proj.RuntimeVersion == "" && len(versions) > 0 {
				proj.RuntimeVersion = versions[0]
			}

			modeSelect.OnChanged = func(s string) {
				appCfg.Projects[idx].RuntimeMode = s
				appCfg.Save(filepath.Join(baseDir, "conf", "wincmp.json"))
			}

			if isRunning {
				if !proj.Enabled {
					go func() {
						procMgr.StopRuntime(proj)
						fyne.Do(func() {
							list.RefreshItem(i)
						})
					}()
				}

				startStopBtn.SetText("Stop")
				startStopBtn.SetIcon(theme.CancelIcon())
				startStopBtn.Enable()
				modeSelect.Disable()

				startStopBtn.OnTapped = func() {
					startStopBtn.SetText("Stopping...")
					startStopBtn.SetIcon(theme.ViewRefreshIcon())
					startStopBtn.Disable()
					filterBtn.Disable()
					go func() {
						procMgr.StopRuntime(proj)
						fyne.Do(func() {
							list.RefreshItem(i)
						})
					}()
				}
			} else {
				// 預設重設按鈕狀態
				startStopBtn.SetText("Start")
				startStopBtn.SetIcon(theme.MediaPlayIcon())
				startStopBtn.OnTapped = nil

				canStart := false
				btnText := "Start"

				if !proj.Enabled {
					canStart = false
					startStopBtn.Disable()
					modeSelect.Disable()
				} else if len(versions) == 0 && (resolvedRT == "node" || resolvedRT == "bun") {
					// 沒有內建版本，檢查是否可以使用系統 PATH
					if !proj.UseWinCMPBin {
						_, hasSystemRuntime := process.CheckSystemRuntimeAvailable(resolvedRT)
						if hasSystemRuntime {
							canStart = true
						} else {
							btnText = "No " + GetRuntimeTypeLabel(resolvedRT) + " (Check PATH)"
						}
					} else {
						btnText = "No " + GetRuntimeTypeLabel(resolvedRT) + " (Add to bin/)"
					}
				} else {
					canStart = true
				}

				if canStart {
					startStopBtn.SetText("Start")
					startStopBtn.Enable()
					modeSelect.Enable()

					startStopBtn.OnTapped = func() {
						startStopBtn.SetText("Starting...")
						startStopBtn.SetIcon(theme.ViewRefreshIcon())
						startStopBtn.Disable()
						filterBtn.Disable()

						// Python/Go 版本檢查
						if process.IsRuntimeTypeNeedEnvCheck(proj.RuntimeType) {
							ver, err := process.CheckRuntimeEnv(proj.RuntimeType)
							if err != nil {
								addErrorLog("runtime", fmt.Sprintf("[%s] %v", proj.Name, err), nil)
								fyne.Do(func() {
									dialog.ShowError(err, win)
									startStopBtn.SetText("Start")
									startStopBtn.SetIcon(theme.MediaPlayIcon())
									startStopBtn.Enable()
									filterBtn.Enable()
								})
								return
							}
							if ver != "" {
								addLog("runtime", i18n.Tfmt("ℹ️ [%s] 偵測到 %s", proj.Name, ver))
							}
						}

						// 版本檢查（只有在使用內建 bundled 且是 node/bun 類型時才需要）
						if proj.UseWinCMPBin && proj.RuntimeType != "custom" && proj.RuntimeType != "python" && proj.RuntimeType != "go_air" && proj.RuntimeType != "go_run" {
							if proj.RuntimeVersion == "" || len(versionPathMap) == 0 {
								runtimeLabel := GetRuntimeTypeLabel(proj.RuntimeType)
								addErrorLog("runtime", i18n.Tfmt("[%s] 沒有可用的 %s 版本，請至 bin/ 檢查", proj.Name, runtimeLabel), nil)
								fyne.Do(func() {
									startStopBtn.SetText("Start")
									startStopBtn.SetIcon(theme.MediaPlayIcon())
									startStopBtn.Enable()
									filterBtn.Enable()
								})
								return
							}
						}

						port := proj.RuntimePort
						if port > 0 && !process.IsPortAvailable(port) {
							addErrorLog("runtime", i18n.Tfmt("[%s] 啟動失敗當前端口 %d 不可用", proj.Name, port), nil)
							fyne.Do(func() {
								dialog.ShowInformation(i18n.T("啟動失敗"), i18n.Tfmt("當前端口 %d 不可用", port), win)
								startStopBtn.SetText("Start")
								startStopBtn.SetIcon(theme.MediaPlayIcon())
								startStopBtn.Enable()
								filterBtn.Enable()
							})
							return
						}

						exePath := ""
						if proj.UseWinCMPBin {
							// 使用 WinCMP 內建 bundled 執行檔路徑
							if proj.RuntimeVersion != "" {
								exePath = versionPathMap[proj.RuntimeVersion]
							}
						} else {
							// 使用系統 PATH 的執行檔名稱
							switch resolvedRT {
							case "node":
								exePath = "npm"
							case "bun":
								exePath = "bun"
							case "auto":
								if hasBun {
									exePath = "bun"
								} else {
									exePath = "npm"
								}
							}
						}

						// 啟動前綁定此項目的 Runtime log
						ensureRuntimeLogBinding(proj.Name)
						switchRuntimeLog(proj.Name)

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
					}
				} else {
					startStopBtn.SetText(btnText)
					startStopBtn.Disable()
					modeSelect.Disable()
				}
			}

			startStopBtn.Refresh()
			filterBtn.Enable()
			filterBtn.Refresh()
			modeSelect.Refresh()
		},
	)

	content := container.NewBorder(
		container.NewVBox(title, widget.NewSeparator(), headerContainer, widget.NewSeparator()),
		nil, nil, nil, list,
	)

	refreshFunc := func() {
		if isMainTabLoading.Load() {
			return
		}
		isMainTabLoading.Store(true)
		go func() {
			tmpRes, err := scanner.ScanBinDir(baseDir)
			if err == nil {
				scanRes.NodeList = tmpRes.NodeList
				scanRes.BunList = tmpRes.BunList
			}
			fyne.Do(func() {
				refreshList()
				list.Refresh()
				isMainTabLoading.Store(false)
			})
		}()
	}

	return container.NewPadded(content), refreshFunc
}

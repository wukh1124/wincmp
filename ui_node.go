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

	"wincmp/internal/process"
	"wincmp/internal/scanner"

	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
)

// nodeRowLayout 自定義佈局，確保各欄位權重與寬度一致
type nodeRowLayout struct{}

func (n *nodeRowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
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
	// 返回一個最小寬度預期
	return fyne.NewSize(600, h)
}

func (n *nodeRowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 7 {
		return
	}
	// 寬度定義
	nameW := float32(150)
	statusW := float32(100)
	portW := float32(100)
	verW := float32(150)
	modeW := float32(150)
	actionW := float32(150)
	padding := theme.Padding()

	// 計算彈性寬度 (Domain)
	fixedW := nameW + statusW + portW + verW + modeW + actionW + (padding * 6)
	domainW := size.Width - fixedW
	if domainW < 150 {
		domainW = 150
	}

	x := float32(0)
	// 0: Project Name
	objects[0].Resize(fyne.NewSize(nameW, size.Height))
	objects[0].Move(fyne.NewPos(x, 0))
	x += nameW + padding

	// 1: Project Domain (Expanding)
	objects[1].Resize(fyne.NewSize(domainW, size.Height))
	objects[1].Move(fyne.NewPos(x, 0))
	x += domainW + padding

	// 2: Status
	objects[2].Resize(fyne.NewSize(statusW, size.Height))
	objects[2].Move(fyne.NewPos(x, 0))
	x += statusW + padding

	// 3: Port
	objects[3].Resize(fyne.NewSize(portW, size.Height))
	objects[3].Move(fyne.NewPos(x, 0))
	x += portW + padding

	// 4: Node Version
	objects[4].Resize(fyne.NewSize(verW, size.Height))
	objects[4].Move(fyne.NewPos(x, 0))
	x += verW + padding

	// 5: Run Mode
	objects[5].Resize(fyne.NewSize(modeW, size.Height))
	objects[5].Move(fyne.NewPos(x, 0))
	x += modeW + padding

	// 6: Action
	objects[6].Resize(fyne.NewSize(actionW, size.Height))
	objects[6].Move(fyne.NewPos(x, 0))
}

func createNodeTab(win fyne.Window) (fyne.CanvasObject, func()) {
	title := widget.NewLabelWithStyle("Node.js Projects", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// 進入分頁時重新掃描 bin/node/ 取得最新版本
	tmpRes, err := scanner.ScanBinDir(baseDir)
	if err == nil {
		scanRes.NodeList = tmpRes.NodeList
	}

	// Header layout - 也使用自定義佈局以確保一致
	createHeaderRect := func(w float32, label string) fyne.CanvasObject {
		r := canvas.NewRectangle(color.Transparent)
		r.SetMinSize(fyne.NewSize(w, 0))
		return container.NewStack(widget.NewLabelWithStyle(label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), r)
	}

	header := container.New(&nodeRowLayout{},
		createHeaderRect(150, "Project Name"),
		createHeaderRect(150, "Project Domain"),
		createHeaderRect(100, "Status"),
		createHeaderRect(100, "Port"),
		createHeaderRect(150, "Node Version"),
		createHeaderRect(150, "Run Mode"),
		createHeaderRect(150, "Action"),
	)
	headerContainer := container.NewVBox(header)

	// List items tracking
	var nodeProjects []int
	refreshList := func() {
		nodeProjects = nil
		for i, p := range appCfg.Projects {
			if p.Type == "node" || p.NodePort > 0 {
				nodeProjects = append(nodeProjects, i)
			}
		}
	}
	refreshList()

	var list *widget.List
	list = widget.NewList(
		func() int { return len(nodeProjects) },
		func() fyne.CanvasObject {
			// Template - 扁平化結構，避免深度嵌套導致的索引問題
			projectBox := container.NewStack()
			pathBox := container.NewStack()
			statusBox := container.NewStack()
			portBox := container.NewStack()

			startStopBtn := widget.NewButton("Start", nil)
			startStopBtnWrapped := container.NewThemeOverride(startStopBtn, &coloredButtonTheme{
				Theme:  theme.DefaultTheme(),
				isStop: func() bool { return startStopBtn.Text == "Stop" },
			})

			modeSelect := widget.NewSelect([]string{"Background", "Terminal"}, nil)
			modeSelect.PlaceHolder = "Mode"
			modeSelectWrapped := container.NewStack(modeSelect)

			versionSelect := widget.NewSelect([]string{}, nil)
			versionSelect.PlaceHolder = "Node Ver"
			versionSelectWrapped := container.NewStack(versionSelect)

			startStopBtnWrappedFixed := container.NewStack(startStopBtnWrapped)

			return container.New(&nodeRowLayout{},
				projectBox,
				pathBox,
				statusBox,
				portBox,
				versionSelectWrapped,
				modeSelectWrapped,
				startStopBtnWrappedFixed,
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if int(i) >= len(nodeProjects) {
				return
			}
			idx := nodeProjects[i]
			if idx >= len(appCfg.Projects) {
				return
			}
			proj := appCfg.Projects[idx]

			row := o.(*fyne.Container)
			// 直接按索引訪問，因為我們使用了扁平化的 nodeRowLayout
			projectBox := row.Objects[0].(*fyne.Container)
			pathBox := row.Objects[1].(*fyne.Container)
			statusBox := row.Objects[2].(*fyne.Container)
			portBox := row.Objects[3].(*fyne.Container)

			versionSelectWrapped := row.Objects[4].(*fyne.Container)
			versionSelect := versionSelectWrapped.Objects[0].(*widget.Select)

			modeSelectWrapped := row.Objects[5].(*fyne.Container)
			modeSelect := modeSelectWrapped.Objects[0].(*widget.Select)

			startStopBtnWrappedFixed := row.Objects[6].(*fyne.Container)
			startStopBtnWrapped := startStopBtnWrappedFixed.Objects[0].(*container.ThemeOverride)
			startStopBtn := startStopBtnWrapped.Content.(*widget.Button)

			// 1. Project
			projectNameHover := ttwidget.NewLabel(proj.Name)
			projectNameHover.SetToolTip(proj.Name)
			projectNameHover.TextStyle = fyne.TextStyle{Bold: true}
			projectNameHover.Truncation = fyne.TextTruncateEllipsis
			projectBox.Objects = []fyne.CanvasObject{projectNameHover}
			projectBox.Refresh()

			// 2. Domain + Copy Button
			domainStr := "-"
			if len(proj.Domains) > 0 {
				domainStr = proj.Domains[0]
			}
			domainHover := ttwidget.NewLabel(domainStr)
			domainHover.SetToolTip(proj.RootPath)
			domainHover.Truncation = fyne.TextTruncateEllipsis

			copyBtn := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
				if domainStr == "" || domainStr == "-" {
					dialog.ShowInformation("複製失敗", "無效的 Domain，無法複製連結", win)
					addLog("system", fmt.Sprintf("❌ 複製連結失敗 [%s]: 無效的 Domain", proj.Name))
					return
				}
				urlPrefix := "http://"
				if proj.UseSSL {
					urlPrefix = "https://"
				}
				win.Clipboard().SetContent(urlPrefix + domainStr)
				addLog("system", fmt.Sprintf("✅ 已複製連結 [%s]: %s%s", proj.Name, urlPrefix, domainStr))
			})
			copyBtn.Importance = widget.LowImportance

			pathBox.Objects = []fyne.CanvasObject{container.NewBorder(nil, nil, nil, copyBtn, domainHover)}
			pathBox.Refresh()

			// 3. Status
			serviceKey := process.NodeServiceKey(proj.ID)
			isRunningByManager := procMgr.IsRunning(serviceKey)
			isRunningByPort := process.CheckNodeRunning(proj.NodePort)
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

			// 4. Port
			portStr := fmt.Sprintf("%d", proj.NodePort)
			if proj.NodePort == 0 {
				portStr = "-"
			}
			portLabel := canvas.NewText(portStr, theme.ForegroundColor())
			portBox.Objects = []fyne.CanvasObject{portLabel}
			portBox.Refresh()

			// 載入 Node 版本清單
			nodeVersions := []string{}
			nodePathMap := map[string]string{}
			for _, n := range scanRes.NodeList {
				nodeVersions = append(nodeVersions, n.Version)
				nodePathMap[n.Version] = n.ExePath
			}
			versionSelect.Options = nodeVersions

			if proj.NodeMode == "" {
				proj.NodeMode = "Background"
			}
			modeSelect.Selected = proj.NodeMode

			if proj.NodeVersion == "" && len(nodeVersions) > 0 {
				proj.NodeVersion = nodeVersions[0]
			}
			versionSelect.Selected = proj.NodeVersion

			modeSelect.OnChanged = func(s string) {
				appCfg.Projects[idx].NodeMode = s
				appCfg.Save(filepath.Join(baseDir, "conf", "wincmp.json"))
			}

			versionSelect.OnChanged = func(s string) {
				appCfg.Projects[idx].NodeVersion = s
				appCfg.Save(filepath.Join(baseDir, "conf", "wincmp.json"))
			}

			if isRunning {
				// 如果專案被禁用，自動停止
				if !proj.Enabled {
					go func() {
						procMgr.StopNode(proj)
						fyne.Do(func() {
							list.RefreshItem(i)
						})
					}()
				}

				startStopBtn.SetText("Stop")
				startStopBtn.SetIcon(theme.CancelIcon())
				startStopBtn.Enable()
				modeSelect.Disable()
				versionSelect.Disable()

				startStopBtn.OnTapped = func() {
					startStopBtn.Disable()
					go func() {
						procMgr.StopNode(proj)
						fyne.Do(func() {
							list.RefreshItem(i)
						})
					}()
				}
			} else {
				// 如果專案被禁用，禁用 Start 按鈕
				if !proj.Enabled {
					startStopBtn.SetText("Start")
					startStopBtn.SetIcon(theme.MediaPlayIcon())
					startStopBtn.Disable()
					modeSelect.Disable()
					versionSelect.Disable()
				} else if len(nodeVersions) == 0 {
					// 沒有可用的 Node 版本，禁用 Start 按鈕
					startStopBtn.SetText("Start")
					startStopBtn.SetIcon(theme.MediaPlayIcon())
					startStopBtn.Disable()
					modeSelect.Disable()
					versionSelect.Disable()
				} else {
					startStopBtn.SetText("Start")
					startStopBtn.SetIcon(theme.MediaPlayIcon())
					startStopBtn.Enable()
					modeSelect.Enable()
					versionSelect.Enable()

					startStopBtn.OnTapped = func() {
						if proj.NodeVersion == "" || len(nodePathMap) == 0 {
							addErrorLog("node", fmt.Sprintf("[%s] 沒有可用的 Node.js 版本，請至 bin/node 檢查", proj.Name), nil)
							return
						}
						if proj.NodePort > 0 && !process.IsPortAvailable(proj.NodePort) {
							addErrorLog("node", fmt.Sprintf("[%s] 啟動失敗當前端口 %d 不可用", proj.Name, proj.NodePort), nil)
							fyne.Do(func() {
								dialog.ShowInformation("啟動失敗", fmt.Sprintf("當前端口 %d 不可用", proj.NodePort), win)
								startStopBtn.Enable()
							})
							return
						}
						exePath := nodePathMap[proj.NodeVersion]
						startStopBtn.Disable()
						go func() {
							err := procMgr.StartNode(proj, modeSelect.Selected, exePath)
							fyne.Do(func() {
								if err != nil {
									startStopBtn.Enable()
								}
								list.RefreshItem(i)
							})
						}()
					}
				}
			}

			startStopBtn.Refresh()
			modeSelect.Refresh()
			versionSelect.Refresh()
		},
	)

	content := container.NewBorder(
		container.NewVBox(title, widget.NewSeparator(), headerContainer, widget.NewSeparator()),
		nil, nil, nil, list,
	)

	refreshFunc := func() {
		tmpRes, err := scanner.ScanBinDir(baseDir)
		if err == nil {
			scanRes.NodeList = tmpRes.NodeList
		}
		refreshList()
		list.Refresh()
	}

	return container.NewPadded(content), refreshFunc
}

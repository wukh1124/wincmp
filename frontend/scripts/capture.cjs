const { chromium } = require('playwright');
const path = require('path');
const fs = require('fs');

// 擷圖固定偏好：字體小 S、英文介面（不受開發機 conf / localStorage 影響）
const CAPTURE_FONT_SIZE = 'small';
const CAPTURE_LANGUAGE = 'en-US';

// 定義每個要截圖的頁面、切換操作與檔案名稱
const screenshotTasks = [
    { name: 'dashboard', selector: '#nav-btn-dashboard', action: async (page) => { } },
    {
        name: 'add_new_project', selector: '#nav-btn-projects', action: async (page) => {
            await page.waitForTimeout(300);
            // 點擊新增專案按鈕
            await page.click('#btn-add-project');
            await page.waitForTimeout(500); // 等待 Drawer 展開動畫
        }
    },
    {
        name: 'projects', selector: null, action: async (page) => {
            // 直接點擊取消關閉 Drawer，不用點擊 Sidebar，以免被遮罩攔截點擊
            const cancelBtn = page.locator('#btn-cancel-add');
            if (await cancelBtn.isVisible()) {
                await cancelBtn.click();
                await page.waitForTimeout(500); // 等待 Drawer 關閉動畫
            }
        }
    },
    {
        name: 'project_terminal', selector: '#nav-btn-projects', action: async (page) => {
            // 確保 Drawer 已關閉
            const cancelBtn = page.locator('#btn-cancel-add');
            if (await cancelBtn.isVisible()) {
                await cancelBtn.click();
                await page.waitForTimeout(500);
            }
            // 點擊列表裡第一個專案的 Terminal 按鈕 (使用 btn-open-terminal class)
            const terminalBtn = page.locator('button.btn-open-terminal').first();
            if (await terminalBtn.isVisible()) {
                await terminalBtn.click();
                await page.waitForTimeout(1200); // 等待終端展開動畫與 PTY 啟動
            }
        }
    },
    {
        name: 'db_explorer', selector: null, action: async (page) => {
            // 在進入 db_explorer 前，關閉前面的專案終端，避免遮罩遮擋點擊
            const closeTerminalBtn = page.locator('#btn-close-terminal');
            if (await closeTerminalBtn.isVisible()) {
                await closeTerminalBtn.click();
                await page.waitForTimeout(500); // 等待關閉動畫
            }
            await page.click('#nav-btn-db_explorer');
            await page.waitForTimeout(600); // 等待頁面載入

            // 點擊 'information_schema' 資料庫並等待資料表渲染
            const schemaBtn = page.locator('button', { hasText: 'information_schema' }).first();
            if (await schemaBtn.isVisible()) {
                await schemaBtn.click();
                await page.waitForTimeout(800); // 等待資料表載入與渲染
            }
        }
    },
    {
        name: 'resource_monitor', selector: '#nav-btn-resources', action: async (page) => {
            // 系統資源數據載入與圖表繪製需要時間，多等待 2.5 秒以避免擷取到 Loading 畫面
            await page.waitForTimeout(2500);
        }
    },
    { name: 'settings', selector: '#nav-btn-settings', action: async (page) => { } },
    {
        name: 'wincmp_dependencies', selector: '#nav-btn-dashboard', action: async (page) => {
            await page.waitForTimeout(300);
            // 點擊儀表板上的「依賴庫管理」按鈕以打開 DependencyManager 彈窗
            await page.click('#btn-open-dep-manager');
            await page.waitForTimeout(600); // 等待彈窗動畫
        }
    },
];

// 支援的主題列表（Carbon 對應官網的暗色版，Sketch 對應手繪版）
// Cycle 順序 carbon → cream → sketch → carbon
const themes = [
    { id: 'carbon', folder: 'dark', htmlAttr: 'carbon' },
    { id: 'sketch', folder: 'sketch', htmlAttr: 'sketch' }
];

/**
 * 讀取目前頁面的字體 / 語系 / 主題 / 側邊欄狀態，供驗證與補償切換使用
 */
async function readUiState(page) {
    return page.evaluate(() => {
        const buttons = Array.from(document.querySelectorAll('button'));
        const findBySvg = (cls) =>
            buttons.find((b) => b.querySelector(`svg.${cls}`)) || null;

        const fontBtn = findBySvg('lucide-type');
        const langBtn = findBySvg('lucide-languages');
        const paletteBtn = findBySvg('lucide-palette');
        const sidebar = document.querySelector('aside');
        const sidebarWidth = sidebar ? sidebar.getBoundingClientRect().width : 0;

        const bodyText = document.body ? document.body.innerText : '';

        return {
            fontSizeAttr: document.documentElement.getAttribute('data-font-size') || '',
            themeAttr: document.documentElement.getAttribute('data-theme') || 'carbon',
            fontBtnLabel: (fontBtn && fontBtn.textContent ? fontBtn.textContent.trim() : ''),
            langBtnLabel: (langBtn && langBtn.textContent ? langBtn.textContent.trim() : ''),
            langBtnTitle: (langBtn && langBtn.getAttribute('title')) || '',
            paletteBtnLabel: (paletteBtn && paletteBtn.textContent ? paletteBtn.textContent.trim() : ''),
            sidebarCollapsed: sidebarWidth > 0 && sidebarWidth < 100,
            sidebarWidth,
            // 英文介面會出現 Dashboard / Projects；中文則是儀表板 / 專案管理
            looksEnglish: bodyText.includes('Dashboard') && bodyText.includes('Projects') && !bodyText.includes('儀表板'),
            looksChinese: bodyText.includes('儀表板') || bodyText.includes('專案管理'),
        };
    });
}

/**
 * 確保側邊欄為展開狀態（projects 頁預設會自動收合，官網截圖需一致展開）
 */
async function ensureSidebarExpanded(page) {
    const state = await readUiState(page);
    if (!state.sidebarCollapsed) {
        return true;
    }
    const toggleSidebarBtn = page.locator('#btn-toggle-sidebar');
    if (await toggleSidebarBtn.isVisible()) {
        await toggleSidebarBtn.click();
        await page.waitForTimeout(500);
        const after = await readUiState(page);
        return !after.sidebarCollapsed;
    }
    return false;
}

/**
 * 確保擷圖偏好（字體 S、英文）已套用；必要時點擊側欄快速切換鈕補償
 * @param {import('playwright').Page} page
 * @returns {Promise<{fontOk: boolean, langOk: boolean, state: object}>}
 */
async function ensureCapturePreferences(page) {
    await page.waitForTimeout(800); // 等待 React 掛載並套用 GetConfig

    // DOM 層強制字體屬性（即使 React state 尚未同步，CSS 也會跟上）
    await page.evaluate((fontSize) => {
        document.documentElement.setAttribute('data-font-size', fontSize);
    }, CAPTURE_FONT_SIZE);
    await page.waitForTimeout(200);

    // 側邊欄展開（避免 projects 頁自動收合導致截圖不一致）
    await ensureSidebarExpanded(page);

    let state = await readUiState(page);
    // 字體驗證以 data-font-size 為主；收合時按鈕不會顯示 S 標籤
    let fontOk = state.fontSizeAttr === CAPTURE_FONT_SIZE;
    let langOk = state.looksEnglish || state.langBtnTitle.includes('English');

    // 字體未生效時，點擊字體快速切換鈕直到 data-font-size=small
    if (!fontOk) {
        const fontBtn = page.locator('button:has(svg.lucide-type)');
        for (let i = 0; i < 3 && !fontOk; i++) {
            if (!(await fontBtn.isVisible())) break;
            await fontBtn.click();
            await page.waitForTimeout(400);
            state = await readUiState(page);
            fontOk = state.fontSizeAttr === CAPTURE_FONT_SIZE;
        }
    }

    // 語系未生效時，點擊語言快速切換鈕（zh-TW ↔ en-US）
    if (!langOk) {
        const langBtn = page.locator('button:has(svg.lucide-languages)');
        for (let i = 0; i < 2 && !langOk; i++) {
            if (!(await langBtn.isVisible())) break;
            await langBtn.click();
            await page.waitForTimeout(400);
            state = await readUiState(page);
            langOk = state.looksEnglish || state.langBtnTitle.includes('English');
        }
    }

    state = await readUiState(page);
    return {
        fontOk: state.fontSizeAttr === CAPTURE_FONT_SIZE,
        langOk: state.looksEnglish || state.langBtnTitle.includes('English'),
        state,
    };
}

/**
 * 將主題切換到指定 id，最多嘗試 maxAttempts 次
 * @returns {Promise<boolean>} 是否成功
 */
async function switchTheme(page, theme, maxAttempts = 6) {
    let attempts = 0;
    while (attempts < maxAttempts) {
        const currentThemeAttr = await page.evaluate(() => {
            return document.documentElement.getAttribute('data-theme') || 'carbon';
        });

        if (currentThemeAttr === theme.htmlAttr) {
            return true;
        }

        const paletteBtn = page.locator('button:has(svg.lucide-palette)');
        if (!(await paletteBtn.isVisible())) {
            console.error('找不到主題切換按鈕 (lucide-palette)！');
            return false;
        }
        await paletteBtn.click();
        // 等待 React 狀態更新與 data-theme 重繪
        await page.waitForTimeout(700);
        attempts++;
    }
    return false;
}

/**
 * 擷圖前後驗證：字體 S、英文、主題 id 是否正確
 */
async function logCaptureState(page, theme, stage) {
    const pref = await ensureCapturePreferences(page);
    const themeOk = pref.state.themeAttr === theme.htmlAttr;
    const fontLabel = pref.state.fontBtnLabel || '(unknown)';
    const langLabel = pref.state.langBtnTitle || pref.state.langBtnLabel || '(unknown)';

    if (pref.fontOk && pref.langOk && themeOk) {
        console.log(`   ✓ [${stage}] theme=${pref.state.themeAttr}, font=${fontLabel}, lang=${langLabel}`);
    } else {
        console.warn(`   ⚠ [${stage}] theme=${pref.state.themeAttr} (expect ${theme.htmlAttr}), font=${fontLabel}, lang=${langLabel}, looksEnglish=${pref.state.looksEnglish}`);
    }
    return { ...pref, themeOk };
}

(async () => {
    // 啟動瀏覽器
    const browser = await chromium.launch({ headless: true });
    // 設定 1264x729 (指定的理想擷圖解析度)，deviceScaleFactor: 2 輸出雙倍清晰截圖
    // locale: en-US 讓 navigator.language 在 conf 為空時也回退到英文
    const context = await browser.newContext({
        viewport: { width: 1264, height: 729 },
        deviceScaleFactor: 2,
        locale: 'en-US',
    });

    // 透過 init script 在 HTML/JS 載入前：
    // 1) 寫入 localStorage 字體偏好
    // 2) 代理 GetConfig：關閉 onboarding、強制 font_size=small、language=en-US
    // 3) 代理 SaveQuickSettings / SaveConfig：避免截圖過程把偏好寫回使用者 conf
    await context.addInitScript((opts) => {
        try {
            localStorage.setItem('wincmp-font-size', opts.fontSize);
            // 鎖定側邊欄，避免切到 projects 時自動收合，確保官網截圖側邊欄一致展開
            localStorage.setItem('wincmp-sidebar-locked', 'true');
            localStorage.setItem('wincmp_sidebar_locked', 'true');
        } catch (e) { /* ignore */ }

        let goVal;

        const forceGlobal = (cfg) => {
            if (cfg && cfg.global) {
                cfg.global.wincmp_onboarding_shown = true;
                cfg.global.wincmp_dep_onboarding_shown = true;
                cfg.global.wincmp_sidebar_guide_shown = true;
                cfg.global.font_size = opts.fontSize;
                cfg.global.language = opts.language;
            }
            return cfg;
        };

        Object.defineProperty(window, 'go', {
            get() {
                return goVal;
            },
            set(val) {
                goVal = val;
                if (goVal && goVal.main && goVal.main.App) {
                    const App = goVal.main.App;

                    if (App.GetConfig) {
                        const originalGetConfig = App.GetConfig;
                        App.GetConfig = async function () {
                            const cfg = await originalGetConfig();
                            return forceGlobal(cfg);
                        };
                    }

                    // 截圖期間不要把主題/語系/字體寫回開發機設定檔
                    if (App.SaveQuickSettings) {
                        App.SaveQuickSettings = async function () {
                            return;
                        };
                    }
                    if (App.SaveConfig) {
                        App.SaveConfig = async function () {
                            return;
                        };
                    }
                }
            },
            configurable: true,
        });
    }, { fontSize: CAPTURE_FONT_SIZE, language: CAPTURE_LANGUAGE });

    const page = await context.newPage();

    /**
     * 於 runtime 再包一層 go.main.App 儲存方法，避免擷圖把偏好寫回 conf
     * （Wails 有時會在 init script 之後重綁方法，需要重複套用）
     */
    const reapplySaveGuards = async () => {
        await page.evaluate((opts) => {
            const App = window.go && window.go.main && window.go.main.App;
            if (!App) return;

            if (App.GetConfig) {
                // 已在 init script 包過；此處僅在被覆寫時重新包
                const original = App.GetConfig;
                if (!original.__wincmpCaptureGuard) {
                    App.GetConfig = async function () {
                        const cfg = await original.call(this);
                        if (cfg && cfg.global) {
                            cfg.global.wincmp_onboarding_shown = true;
                            cfg.global.wincmp_dep_onboarding_shown = true;
                            cfg.global.wincmp_sidebar_guide_shown = true;
                            cfg.global.font_size = opts.fontSize;
                            cfg.global.language = opts.language;
                        }
                        return cfg;
                    };
                    App.GetConfig.__wincmpCaptureGuard = true;
                }
            }

            App.SaveQuickSettings = async function () { return; };
            App.SaveConfig = async function () { return; };
        }, { fontSize: CAPTURE_FONT_SIZE, language: CAPTURE_LANGUAGE });
    };

    const targetUrl = 'http://localhost:34115'; // Wails Dev 預設網址
    console.log(`[WinCMP] 連線至 Wails 開發伺服器: ${targetUrl}...`);
    console.log(`[WinCMP] 擷圖偏好: font=${CAPTURE_FONT_SIZE}, language=${CAPTURE_LANGUAGE}`);

    try {
        await page.goto(targetUrl);
        await page.waitForLoadState('networkidle');
    } catch (e) {
        console.error('無法連線到 Wails 開發伺服器。請確保您已在 `wincmp` 目錄執行 `wails dev`！');
        process.exit(1);
    }

    await reapplySaveGuards();

    // 首次載入後立刻套用並驗證擷圖偏好
    const bootPref = await ensureCapturePreferences(page);
    if (!bootPref.fontOk) {
        console.warn(`[Warn] 字體可能未切換到 S：fontSizeAttr=${bootPref.state.fontSizeAttr}, fontBtn=${bootPref.state.fontBtnLabel}`);
    }
    if (!bootPref.langOk) {
        console.warn(`[Warn] 語系可能未切換到英文：lang=${bootPref.state.langBtnTitle || bootPref.state.langBtnLabel}, looksEnglish=${bootPref.state.looksEnglish}`);
    }
    if (bootPref.fontOk && bootPref.langOk) {
        console.log(`[WinCMP] 已確認擷圖偏好: font=S, language=English`);
    }

    const captureReport = [];

    for (const theme of themes) {
        console.log(`\n[主題] 切換至: [${theme.id}] -> screenshot/${theme.folder}/`);

        // 主題切換前再套用儲存攔截，避免 palette click 觸發 SaveQuickSettings 寫回 conf
        await reapplySaveGuards();

        const themeOk = await switchTheme(page, theme);
        if (themeOk) {
            console.log(`   ✓ 成功載入主題: [${theme.id}]`);
        } else {
            console.warn(`   主題切換可能未完全成功，將繼續執行...`);
        }

        await reapplySaveGuards();

        // 主題切換後等待重繪，再驗證偏好是否仍在
        await page.waitForTimeout(500);
        const prefAfterTheme = await logCaptureState(page, theme, 'theme-ready');
        if (!prefAfterTheme.fontOk || !prefAfterTheme.langOk) {
            console.warn(`   主題切換後偏好偏移，已嘗試補償`);
            await ensureCapturePreferences(page);
        }

        // 確保根目錄下的 screenshot/ 目錄存在
        const outputDir = path.join(__dirname, '..', '..', 'screenshot', theme.folder);
        if (!fs.existsSync(outputDir)) {
            fs.mkdirSync(outputDir, { recursive: true });
        }

        // 每張擷圖前確保側邊欄展開（projects 頁若鎖定失效仍會收合）
        await ensureSidebarExpanded(page);

        // 開始逐項擷圖
        for (const task of screenshotTasks) {
            console.log(`[擷圖] [${theme.id}] ${task.name}`);

            // 點擊導覽按鈕
            if (task.selector) {
                const navBtn = page.locator(task.selector);
                await navBtn.click();
                await page.waitForTimeout(400); // 等待切換動畫
            }

            // 執行專屬動作
            await task.action(page);
            await page.waitForTimeout(400); // 等待狀態穩定

            // 擷圖前再確認側邊欄展開、字體 / 語系仍正確
            await ensureSidebarExpanded(page);
            const preShot = await readUiState(page);
            const fontStillOk = preShot.fontSizeAttr === CAPTURE_FONT_SIZE;
            const langStillOk = preShot.looksEnglish || preShot.langBtnTitle.includes('English');
            const themeStillOk = preShot.themeAttr === theme.htmlAttr;
            if (!fontStillOk || !langStillOk || !themeStillOk) {
                console.warn(`   [pre-shot] 偏好偏移 font=${preShot.fontSizeAttr} langEnglish=${preShot.looksEnglish} theme=${preShot.themeAttr}`);
                await ensureCapturePreferences(page);
            }

            // 存檔路徑
            const outputPath = path.join(outputDir, `${task.name}.png`);
            await page.screenshot({ path: outputPath });
            console.log(`   ✓ 已儲存 -> screenshot/${theme.folder}/${task.name}.png`);
            captureReport.push({
                theme: theme.id,
                folder: theme.folder,
                name: task.name,
                file: `screenshot/${theme.folder}/${task.name}.png`,
                themeAttr: theme.htmlAttr,
                fontOk: fontStillOk,
                langOk: langStillOk,
                themeOk: themeStillOk,
            });
        }

        // 當前主題所有擷圖完成後，如果依賴管理器彈窗開著，點擊將它關閉，防止遮擋下一輪主題切換
        const closeDepBtn = page.locator('#btn-close-dep-manager');
        if (await closeDepBtn.isVisible()) {
            await closeDepBtn.click();
            await page.waitForTimeout(500);
        }
    }

    await browser.close();

    // 匯總驗證結果
    const total = captureReport.length;
    const fontIssues = captureReport.filter((r) => !r.fontOk).length;
    const langIssues = captureReport.filter((r) => !r.langOk).length;
    const themeIssues = captureReport.filter((r) => !r.themeOk).length;

    console.log('\n===================================================');
    console.log('[完成] 自動化截圖工作結束');
    console.log(`  張數: ${total}（${themes.length} 主題 x ${screenshotTasks.length} 頁面）`);
    console.log(`  字體 S 異常: ${fontIssues}`);
    console.log(`  英文語系 異常: ${langIssues}`);
    console.log(`  主題 異常: ${themeIssues}`);
    if (fontIssues === 0 && langIssues === 0 && themeIssues === 0) {
        console.log('  驗證: 全部通過');
    } else {
        console.log('  驗證: 有項目未通過，請檢查上方 warning 與產出截圖');
        process.exitCode = 2;
    }
    console.log('===================================================');
})();

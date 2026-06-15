const { chromium } = require('playwright');
const path = require('path');
const fs = require('fs');

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
            // 💡 在進入 db_explorer 前，關閉前面的專案終端，避免遮罩遮擋點擊
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
            // 💡 系統資源數據載入與圖表繪製需要時間，多等待 2.5 秒以避免擷取到 Loading 畫面
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
const themes = [
    { id: 'carbon', folder: 'dark', htmlAttr: 'carbon' },
    { id: 'sketch', folder: 'sketch', htmlAttr: 'sketch' }
];

(async () => {
    // 啟動瀏覽器
    const browser = await chromium.launch({ headless: true });
    // 設定 1264x729 (指定的理想擷圖解析度)，deviceScaleFactor: 2 輸出雙倍清晰截圖
    const context = await browser.newContext({
        viewport: { width: 1264, height: 729 },
        deviceScaleFactor: 2
    });

    // 💡 透過 init script 在 HTML/JS 載入前預先寫入 localStorage，徹底關閉 onboarding 導引氣泡
    await context.addInitScript(() => {
        localStorage.setItem('wincmp_onboarding_shown', 'true');
        localStorage.setItem('wincmp_dep_onboarding_shown', 'true');
    });

    const page = await context.newPage();

    const targetUrl = 'http://localhost:34115'; // Wails Dev 預設網址
    console.log(`🚀 [WinCMP] 開始嘗試連線至 Wails 開發伺服器: ${targetUrl}...`);

    try {
        await page.goto(targetUrl);
        await page.waitForLoadState('networkidle');
        // 💡 預先將 onboarding 教學標記為已看過，避免影響擷圖
        await page.evaluate(() => {
            localStorage.setItem('wincmp_dep_onboarding_shown', 'true');
            localStorage.setItem('wincmp_onboarding_shown', 'true');
        });
    } catch (e) {
        console.error('❌ 無法連線到 Wails 開發伺服器。請確保您已在 `wincmp` 目錄執行 `wails dev`！');
        process.exit(1);
    }

    for (const theme of themes) {
        console.log(`\n🎨 [主題設定] 正在將系統主題切換至: [${theme.id}]...`);

        let attempts = 0;
        let success = false;

        while (attempts < 6) {
            // 讀取當前的 HTML data-theme 屬性 (如果為空，代表 Carbon 預設)
            const currentThemeAttr = await page.evaluate(() => {
                return document.documentElement.getAttribute('data-theme') || 'carbon';
            });

            if (currentThemeAttr === theme.htmlAttr) {
                success = true;
                break;
            }

            // 點擊主題快速切換按鈕
            const paletteBtn = page.locator('button:has(svg.lucide-palette)');
            if (await paletteBtn.isVisible()) {
                await paletteBtn.click();
                await page.waitForTimeout(600); // 等待 React 設定儲存並重繪
            } else {
                console.error('❌ 找不到主題切換按鈕 (lucide-palette)！');
                break;
            }
            attempts++;
        }

        if (success) {
            console.log(`   ✓ 成功載入主題: [${theme.id}]`);
        } else {
            console.warn(`   ⚠️ 主題切換可能未完全成功，將繼續執行...`);
        }

        await page.waitForTimeout(500);

        // 💡 再次確保 onboarding 教學被點掉，防止重載時氣泡彈出
        await page.evaluate(() => {
            localStorage.setItem('wincmp_dep_onboarding_shown', 'true');
            localStorage.setItem('wincmp_onboarding_shown', 'true');
        });

        await page.waitForTimeout(500);

        // 確保根目錄下的 screenshot/ 目錄存在
        const outputDir = path.join(__dirname, '..', '..', 'screenshot', theme.folder);
        if (!fs.existsSync(outputDir)) {
            fs.mkdirSync(outputDir, { recursive: true });
        }

        // 開始逐項擷圖
        for (const task of screenshotTasks) {
            console.log(`📸 [擷圖] 正在擷取 [${theme.id}] 的 ${task.name}...`);

            // 點擊導覽按鈕
            if (task.selector) {
                const navBtn = page.locator(task.selector);
                await navBtn.click();
                await page.waitForTimeout(400); // 等待切換動畫
            }

            // 執行專屬動作
            await task.action(page);
            await page.waitForTimeout(400); // 等待狀態穩定

            // 存檔路徑
            const outputPath = path.join(outputDir, `${task.name}.png`);
            await page.screenshot({ path: outputPath });
            console.log(`   ✓ 已儲存 -> screenshot/${theme.folder}/${task.name}.png`);
        }

        // 💡 當前主題所有擷圖完成後，如果依賴管理器彈窗開著，點擊將它關閉，防止遮擋下一輪主題切換
        const closeDepBtn = page.locator('#btn-close-dep-manager');
        if (await closeDepBtn.isVisible()) {
            await closeDepBtn.click();
            await page.waitForTimeout(500);
        }
    }

    await browser.close();
    console.log('\n🎉 [完成] 自動化截圖工作已順利結束！');
})();

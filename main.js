document.addEventListener('DOMContentLoaded', () => {
    let cachedReleaseData = null;

    // ==========================================================================
    // 0. 圖片載入優化與主題切換邏輯 (Image Loading & Theme Switching)
    // ==========================================================================
    const themeBtn = document.getElementById('theme-switch-btn');

    // 漸變切換圖片函數，防白屏與閃爍
    function changeImage(imgEl, newSrc, onSrcChangeCallback, useFade = true) {
        if (!imgEl) return;
        if (imgEl.getAttribute('src') === newSrc) {
            if (onSrcChangeCallback) onSrcChangeCallback();
            return;
        }

        if (!useFade) {
            imgEl.src = newSrc;
            if (onSrcChangeCallback) onSrcChangeCallback();
            return;
        }

        // 先讓圖片透明度降低 (淡出)
        imgEl.style.opacity = '0.3';

        // 建立臨時 Image 物件來載入圖片，完成後才切換
        const tempImg = new Image();
        tempImg.onload = () => {
            imgEl.src = newSrc;
            if (onSrcChangeCallback) onSrcChangeCallback();
            // 切換完 src 後，讓它淡入
            setTimeout(() => {
                imgEl.style.opacity = '1';
            }, 50);
        };
        tempImg.onerror = () => {
            imgEl.src = newSrc;
            if (onSrcChangeCallback) onSrcChangeCallback();
            imgEl.style.opacity = '1';
        };
        tempImg.src = newSrc;
    }

    function updateScreenshotThemes(theme, useFade = true) {
        const isSketch = (theme === 'sketch');
        const tabs = document.querySelectorAll('.gallery-tab');
        tabs.forEach(tab => {
            const darkPath = tab.getAttribute('data-img-dark');
            const sketchPath = tab.getAttribute('data-img-sketch');
            tab.setAttribute('data-img', isSketch ? sketchPath : darkPath);
        });

        const displayImg = document.getElementById('gallery-display-img');
        const activeTab = document.querySelector('.gallery-tab.active');
        if (displayImg && activeTab) {
            const currentImgSrc = activeTab.getAttribute('data-img');
            changeImage(displayImg, currentImgSrc, null, useFade);
        }

        const heroImg = document.getElementById('hero-main-img');
        if (heroImg) {
            const darkPath = heroImg.getAttribute('data-img-dark');
            const sketchPath = heroImg.getAttribute('data-img-sketch');
            const currentHeroSrc = isSketch ? sketchPath : darkPath;
            changeImage(heroImg, currentHeroSrc, null, useFade);
        }
    }

    function applyTheme(theme, useFade = true) {
        if (theme === 'sketch') {
            document.documentElement.setAttribute('data-theme', 'sketch');
        } else {
            document.documentElement.removeAttribute('data-theme');
        }
        localStorage.setItem('wincmp_theme', theme);
        updateScreenshotThemes(theme, useFade);
    }

    // 載入時恢復已儲存的主題偏好，預設為 'sketch' (亮色手繪風)
    const savedTheme = localStorage.getItem('wincmp_theme') || 'sketch';
    applyTheme(savedTheme, false); // 初始載入時不使用淡出淡入

    // 監聽主題切換按鈕
    if (themeBtn) {
        themeBtn.addEventListener('click', () => {
            const currentTheme = document.documentElement.getAttribute('data-theme');
            const nextTheme = currentTheme === 'sketch' ? 'dark' : 'sketch';
            applyTheme(nextTheme, true); // 手動切換主題時使用淡出淡入
        });
    }

    // ==========================================================================
    // 1. 多國語言字典與核心切換邏輯 (i18n)
    // ==========================================================================
    const translations = {
        en: {
            doc_title: "WinCMP - Portable Local Development Control Panel for Windows (Caddy + MariaDB + PHP + Mailpit + Redis)",
            nav_features: "Features",
            nav_gallery: "Screenshots",
            nav_comparison: "Tech Specs",
            nav_architecture: "Architecture",
            nav_changelog: "Changelog",
            lang_btn_text: "繁體中文",
            theme_btn_dark: "Dark",
            theme_btn_sketch: "Sketch",
            hero_badge_prefix: "Latest Version:",
            hero_title: "Lightweight & Portable<br><span class=\"gradient-text\">Local Dev Control Panel</span> for Windows",
            hero_subtitle: "Integrated with <strong>Caddy</strong> + <strong>MariaDB</strong> + <strong>PHP</strong> + <strong>Mailpit</strong> + <strong>Redis</strong> and an interactive project terminal. Core services run without admin privileges — lightweight, portable, and practical.",
            hero_download_btn: "Download for Windows",
            hero_github_btn: "View GitHub Repo",
            features_title: "Why Choose WinCMP?",
            features_subtitle: "A portable control panel for local Web development on Windows. Core binaries stay out of system PATH; dependencies can be fetched from the built-in downloader.",
            feat_1_title: "Lightweight",
            feat_1_desc: "Statically compiled in Go + Wails, rendering via native Windows WebView2 instead of Electron. Fast startup; idle memory is about 150-250MB (includes WebView2).",
            feat_2_title: "Admin-Privilege-Free Core",
            feat_2_desc: "Caddy, PHP-CGI, MariaDB, Mailpit, and Redis run under restricted user permissions. No registry writes or global PATH pollution. Note: Hosts sync and some dependency downloads may require UAC.",
            feat_3_title: "Interactive Terminal Drawer",
            feat_3_desc: "Built-in Windows ConPTY and xterm.js at the project root. Slide out a terminal drawer for PowerShell/CMD/Git Bash/WSL with interactive commands and TAB completion.",
            feat_4_title: "PHP Multi-Process Balancing",
            feat_4_desc: "Uses Caddy upstream load balancing to run multiple independent FastCGI processes per PHP version (default 3, adjustable) for more reliable local development.",
            feat_5_title: "Runtime Multi-Environment",
            feat_5_desc: "Supports Node.js, Bun, Python, Go and more. Auto-detects presets such as Laravel, Next.js, Nuxt, Astro, Vite, Django/FastAPI/Flask, PocketBase, and Go API; start in background or a separate terminal.",
            feat_6_title: "Hosts Sync & Backup (Needs Admin)",
            feat_6_desc: "Sync custom local domains to the Windows hosts file. Because Windows protects this file, write and backup require UAC elevation.",
            feat_7_title: "Portable Single-EXE",
            feat_7_desc: "Runs via a single executable without installation. Migrate by taking conf/wincmp.json and the /bin directory to any new location.",
            feat_8_title: "One-Click Dependency Downloader",
            feat_8_desc: "Built-in downloader for Caddy, PHP, MariaDB, Mailpit, Redis, Node.js, Bun, Composer, HeidiSQL and more — place binaries under /bin and WinCMP scans them automatically.",
            feat_9_title: "Database Explorer",
            feat_9_desc: "Browse databases and tables from the panel, then open the same connection in HeidiSQL with one click when you need a full GUI client.",
            gallery_title: "Intuitive & Modern User Interface",
            gallery_subtitle: "Every screen is carefully polished, with smooth micro-interactions and real-time service status monitoring.",
            gallery_tab_dashboard: "Dashboard",
            gallery_tab_projects: "Projects",
            gallery_tab_terminal: "Built-in Terminal",
            gallery_tab_db: "Database Explorer",
            gallery_tab_add_project: "Add Project",
            gallery_tab_resource: "Resource Monitor",
            gallery_tab_settings: "Preferences",
            gallery_tab_dependencies: "Dependency Downloader",
            compare_title: "WinCMP Architecture & Configuration",
            compare_subtitle: "Technical stack, managed services, and default configuration of the current WinCMP release (not a historical version comparison):",
            compare_head_item: "Item",
            compare_head_desc: "Current WinCMP",
            compare_row_1_name: "App Architecture",
            compare_row_1_val: "Go 1.26 + Wails v2 + React 18 (rendered via Windows WebView2, not Electron)",
            compare_row_2_name: "Platform",
            compare_row_2_val: "Windows; portable single-executable launch, no installer required",
            compare_row_3_name: "Core Services",
            compare_row_3_val: "Caddy (reverse proxy / HTTPS), MariaDB, PHP-CGI, Mailpit, Redis",
            compare_row_4_name: "Privilege Model",
            compare_row_4_val: "Core services run in user space; Hosts sync and some dependency downloads may prompt for UAC",
            compare_row_5_name: "PHP Runtime",
            compare_row_5_val: "Multi-version parallel execution; default 3 FastCGI processes per version (configurable); port rule 3&lt;major&gt;&lt;minor&gt;&lt;index&gt;",
            compare_row_6_name: "App Runtimes",
            compare_row_6_val: "Node.js / Bun / Python / Go / Custom; Background or Terminal launch modes",
            compare_row_7_name: "Built-in Terminal",
            compare_row_7_val: "ConPTY + xterm.js drawer; PowerShell / CMD / Git Bash / WSL with TAB completion",
            compare_row_8_name: "Config Apply",
            compare_row_8_val: "Project settings written under conf/ and applied via Caddy hot-reload",
            compare_row_9_name: "Idle Footprint",
            compare_row_9_val: "Control panel UI itself is about 150-250MB idle (includes WebView2; varies by environment)",
            compare_row_10_name: "Dependency Manager",
            compare_row_10_val: "Built-in downloader: Caddy, PHP, MariaDB, Mailpit, Redis, Node.js, Bun, Composer, HeidiSQL",
            compare_row_11_name: "Database Tools",
            compare_row_11_val: "Built-in DB Explorer; one-click open in HeidiSQL",
            compare_row_12_name: "Portability",
            compare_row_12_val: "Migrate with conf/wincmp.json + /bin directory only",
            compare_row_13_name: "Framework Presets",
            compare_row_13_val: "Auto-detect Laravel, Next.js, Nuxt, Astro, Vite, Django/FastAPI/Flask, PocketBase, Go API and more",
            compare_table_note: "* Note: Idle RAM refers to the control panel UI itself. Actual usage varies with the OS environment and which services are running.",
            arch_title: "How WinCMP Works",
            arch_subtitle: "Design principles behind isolation and service management:",
            arch_card_1_title: "Dynamic Env Isolation",
            arch_card_1_desc: "Instead of modifying Windows system global variables, WinCMP injects a temporary PATH only for the child process it starts. Different projects can therefore run different PHP/Node versions at the same time without polluting the system.",
            arch_card_2_title: "Smart Ports & Seamless Reload",
            arch_card_2_desc: "Ports are allocated by a deterministic rule (for example PHP 8.2 uses 38200-382xx) so versions do not collide. When site settings change, WinCMP rewrites conf/sites and reloads Caddy in the background.",
            changelog_title: "Latest Updates",
            changelog_subtitle: "See what new features and fixes have been added recently",
            changelog_loading: "Fetching latest changelog from GitHub...",
            changelog_view_more: "View all releases on GitHub",
            footer_desc: "WinCMP gratefully integrates and acknowledges open-source projects including Caddy, MariaDB, PHP, Mailpit, Redis, Node.js, Composer, HeidiSQL, Wails, and React."
        },
        zh: {
            doc_title: "WinCMP - 專為 Windows 打造的輕量、免安裝本地開發控制面板 (整合 Caddy + MariaDB + PHP + Mailpit + Redis)",
            nav_features: "特色功能",
            nav_gallery: "介面展示",
            nav_comparison: "技術規格",
            nav_architecture: "架構原理",
            nav_changelog: "更新日誌",
            lang_btn_text: "English",
            theme_btn_dark: "Dark",
            theme_btn_sketch: "Sketch",
            hero_badge_prefix: "最新版本:",
            hero_title: "專為 Windows 打造的<br><span class=\"gradient-text\">輕量快速免安裝</span>開發面板",
            hero_subtitle: "整合 <strong>Caddy</strong> + <strong>MariaDB</strong> + <strong>PHP</strong> + <strong>Mailpit</strong> + <strong>Redis</strong> 與專案互動終端。核心開發服務啟動免管理員權限，輕量、可攜、好上手！",
            hero_download_btn: "立即下載 Windows 版",
            hero_github_btn: "瀏覽 GitHub 倉庫",
            features_title: "為什麼選擇 WinCMP？",
            features_subtitle: "可攜式 Windows 本地 Web 開發控制面板。核心執行檔不寫入系統 PATH，所需依賴可透過內建下載器取得。",
            feat_1_title: "輕量 (Statically Compiled)",
            feat_1_desc: "基於 Go + Wails 靜態編譯，透過 Windows 原生 WebView2 渲染，無需 Electron。啟動快速，日常閒置記憶體約 150-250MB（含 WebView2 引擎）。",
            feat_2_title: "核心服務免 Admin 權限",
            feat_2_desc: "Caddy、PHP-CGI、MariaDB、Mailpit、Redis 等核心服務皆在受限用戶權限下執行，免寫系統登錄檔、免污染全域環境變數。註：Hosts 同步與部分依賴下載可能需要 UAC。",
            feat_3_title: "整合式互動終端 Drawer",
            feat_3_desc: "集成 Windows ConPTY 與 xterm.js，在專案根目錄一鍵滑出互動式終端（支援 PowerShell / CMD / Git Bash / WSL），可執行動態指令並支援 TAB 補齊。",
            feat_4_title: "PHP 多進程負載均衡",
            feat_4_desc: "利用 Caddy 的 upstream 負載均衡，為每個 PHP 版本啟動多個獨立 FastCGI 進程（預設 3 個，可依需求調整），提升本地開發穩定度。",
            feat_5_title: "運行時多環境支援",
            feat_5_desc: "支援 Node.js、Bun、Python、Go 等 Runtime，並自動偵測 Laravel、Next.js、Nuxt、Astro、Vite、Django/FastAPI/Flask、PocketBase、Go API 等 Preset；可選背景執行或獨立終端啟動。",
            feat_6_title: "Hosts 備份與更新 (需要 Admin)",
            feat_6_desc: "支援一鍵同步自訂網域到 Windows Hosts 檔案。因系統安全防護，此寫入與自動備份功能需要 UAC 提權確認。",
            feat_7_title: "單檔運作與綠色便攜",
            feat_7_desc: "只需單個 executable 即可獨立運作，啟動時自動生成設定檔。遷移時將 conf/wincmp.json 與整個 /bin 目錄帶走即可。",
            feat_8_title: "依賴一鍵下載器",
            feat_8_desc: "內建下載 Caddy、PHP、MariaDB、Mailpit、Redis、Node.js、Bun、Composer、HeidiSQL 等元件；放入 /bin 後由 WinCMP 自動掃描版本。",
            feat_9_title: "資料庫管理器",
            feat_9_desc: "可在面板內瀏覽資料庫與資料表，需要完整 GUI 時一鍵以 HeidiSQL 開啟同一連線。",
            gallery_title: "直觀、美觀的現代介面",
            gallery_subtitle: "每一處 UI 都經過細心打磨，具備流暢的微動畫與即時服務狀態監控。",
            gallery_tab_dashboard: "主控台",
            gallery_tab_projects: "專案管理",
            gallery_tab_terminal: "內建終端",
            gallery_tab_db: "資料庫管理器",
            gallery_tab_add_project: "新增專案",
            gallery_tab_resource: "資源佔用監控",
            gallery_tab_settings: "偏好設定",
            gallery_tab_dependencies: "依賴下載器",
            compare_title: "WinCMP 架構與配置",
            compare_subtitle: "以下為目前版本的技術棧、核心服務與預設配置說明（非歷史版本對比）：",
            compare_head_item: "項目",
            compare_head_desc: "目前 WinCMP",
            compare_row_1_name: "應用架構",
            compare_row_1_val: "Go 1.26 + Wails v2 + React 18（透過 Windows WebView2 渲染，非 Electron）",
            compare_row_2_name: "執行平台",
            compare_row_2_val: "Windows；單一執行檔啟動，免安裝",
            compare_row_3_name: "核心服務",
            compare_row_3_val: "Caddy（反向代理 / HTTPS）、MariaDB、PHP-CGI、Mailpit、Redis",
            compare_row_4_name: "權限模型",
            compare_row_4_val: "核心服務於用戶空間執行；Hosts 同步與部分依賴下載可能需要 UAC",
            compare_row_5_name: "PHP 執行",
            compare_row_5_val: "多版本並行；每版預設 3 個 FastCGI 進程（可調整）；Port 規則為 3&lt;主版本&gt;&lt;次版本&gt;&lt;序號&gt;",
            compare_row_6_name: "應用 Runtime",
            compare_row_6_val: "Node.js / Bun / Python / Go / Custom；Background 或 Terminal 雙模式",
            compare_row_7_name: "內建終端",
            compare_row_7_val: "ConPTY + xterm.js 抽屜終端；支援 PowerShell / CMD / Git Bash / WSL，含 TAB 補齊",
            compare_row_8_name: "配置生效",
            compare_row_8_val: "專案設定寫入 conf/ 後，由 Caddy 熱重載套用",
            compare_row_9_name: "閒置資源",
            compare_row_9_val: "控制面板本身約 150-250MB（含 WebView2；依環境略有差異）",
            compare_row_10_name: "依賴管理",
            compare_row_10_val: "內建下載器：Caddy、PHP、MariaDB、Mailpit、Redis、Node.js、Bun、Composer、HeidiSQL",
            compare_row_11_name: "資料庫工具",
            compare_row_11_val: "內建 DB Explorer，可一鍵以 HeidiSQL 開啟",
            compare_row_12_name: "可攜性",
            compare_row_12_val: "遷移只需 conf/wincmp.json 與 /bin 目錄",
            compare_row_13_name: "框架 Preset",
            compare_row_13_val: "自動偵測 Laravel、Next.js、Nuxt、Astro、Vite、Django/FastAPI/Flask、PocketBase、Go API 等",
            compare_table_note: "* 註：閒置記憶體指控制面板主程式本身；實際數值會因系統環境與已啟動服務而異。",
            arch_title: "WinCMP 運作原理",
            arch_subtitle: "環境隔離與服務管理背後的設計原則：",
            arch_card_1_title: "動態環境變數隔離",
            arch_card_1_desc: "不修改 Windows 系統全域設定，WinCMP 只在啟動子進程時為其注入暫時的 PATH。不同專案可同時使用不同 PHP / Node 版本，系統依然保持乾淨。",
            arch_card_2_title: "智慧連接埠與無縫重載",
            arch_card_2_desc: "連接埠依固定規則分配（例如 PHP 8.2 使用 38200-382xx），避免版本互相衝突。站台設定變更時會重寫 conf/sites 並在背景熱重載 Caddy。",
            changelog_title: "最新版本更新日誌",
            changelog_subtitle: "查看最近 WinCMP 的更新和修正項目",
            changelog_loading: "正在從 GitHub 獲取最新發布日誌...",
            changelog_view_more: "前往 GitHub 查看所有歷史發布與日誌",
            footer_desc: "WinCMP 整合並致敬優秀的開源生態組件，包括 Caddy、MariaDB、PHP、Mailpit、Redis、Node.js、Composer、HeidiSQL、Wails 與 React。"
        }
    };

    function switchLanguage(lang) {
        document.querySelectorAll('[data-i18n]').forEach(element => {
            const key = element.getAttribute('data-i18n');
            if (translations[lang] && translations[lang][key]) {
                if (element.tagName === 'TITLE') {
                    document.title = translations[lang][key];
                } else {
                    element.innerHTML = translations[lang][key];
                }
            }
        });

        // 儲存偏好
        localStorage.setItem('wincmp_lang', lang);

        // 重新初始化 Lucide 圖示 (避免動態重寫後圖示消失)
        if (window.lucide) {
            window.lucide.createIcons();
        }

        // 修改網頁根元素的 lang 屬性
        document.documentElement.lang = lang === 'zh' ? 'zh-TW' : 'en';

        // 💡 如果有當前已選的 gallery tab，同步更新視窗標題名稱
        const activeTab = document.querySelector('.gallery-tab.active span');
        const windowTitle = document.getElementById('gallery-window-title');
        if (activeTab && windowTitle) {
            windowTitle.innerText = `WinCMP - ${activeTab.innerText}`;
        }

        // 同步更新更新日誌語系
        if (cachedReleaseData) {
            renderReleaseInfo(cachedReleaseData);
        }
    }

    // 監聽語言切換按鈕
    const langBtn = document.getElementById('lang-switch-btn');
    if (langBtn) {
        langBtn.addEventListener('click', () => {
            const currentLang = localStorage.getItem('wincmp_lang') || 'en';
            const nextLang = currentLang === 'en' ? 'zh' : 'en';
            switchLanguage(nextLang);
        });
    }

    // ==========================================================================
    // 2. 實機截圖切換 (Gallery Tabs)
    // ==========================================================================
    const tabs = document.querySelectorAll('.gallery-tab');
    const displayImg = document.getElementById('gallery-display-img');
    const windowTitle = document.getElementById('gallery-window-title');

    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            // 移除所有活耀狀態
            tabs.forEach(t => t.classList.remove('active'));

            // 設定當前按鈕為活耀
            tab.classList.add('active');

            // 取得圖片路徑與標題
            const imgSrc = tab.getAttribute('data-img');
            const tabName = tab.querySelector('span').innerText;

            // 使用 changeImage 進行漸變切換，等待圖片載入完成後才淡入，防閃爍
            changeImage(displayImg, imgSrc, () => {
                displayImg.alt = `WinCMP ${tabName}`;
                windowTitle.innerText = `WinCMP - ${tabName}`;
            }, true);
        });
    });

    // ==========================================================================
    // 3. 獲取最新 Release 版本與更新日誌
    // ==========================================================================
    const owner = 'wukh1124';
    const repo = 'wincmp';

    const latestTagEl = document.getElementById('latest-tag');
    const downloadBtn = document.getElementById('download-btn');
    const changelogLoading = document.getElementById('changelog-loading');
    const changelogContent = document.getElementById('changelog-content');

    async function getLatestRelease() {
        try {
            const response = await fetch('./release.json');
            if (!response.ok) {
                throw new Error(`Failed to load release.json: ${response.status}`);
            }
            const data = await response.json();
            cachedReleaseData = data;
            renderReleaseInfo(data);
        } catch (error) {
            console.error('Failed to fetch release info:', error);
            renderFallback();
        }
    }

    function renderReleaseInfo(data) {
        const tagName = data.tag_name || 'v2.0.0';
        latestTagEl.innerText = tagName;
        downloadBtn.href = data.exe_url || `https://github.com/${owner}/${repo}/releases/latest`;

        const currentLang = localStorage.getItem('wincmp_lang') || 'en';
        const bodyContent = currentLang === 'zh' ? data.changelog_zh : data.changelog_en;

        if (window.marked && bodyContent) {
            changelogLoading.style.display = 'none';
            changelogContent.style.display = 'block';
            changelogContent.innerHTML = window.marked.parse(bodyContent);
        } else {
            renderFallback();
        }
    }

    function renderFallback() {
        latestTagEl.innerText = 'v2.0.0';
        downloadBtn.href = `https://github.com/${owner}/${repo}/releases/latest`;
        changelogLoading.style.display = 'none';
        changelogContent.style.display = 'block';

        const currentLang = localStorage.getItem('wincmp_lang') || 'en';
        if (currentLang === 'zh') {
            changelogContent.innerHTML = `
                <h3>無法動態載入更新日誌</h3>
                <p>由於 GitHub API 請求頻率限制或網路連線問題，暫時無法抓取最新更新詳情。</p>
                <p>請直接點擊下方連結前往 GitHub Releases 頁面查看：</p>
                <p><a href="https://github.com/${owner}/${repo}/releases" target="_blank" style="color: var(--primary-hover); text-decoration: underline;">前往 GitHub 查看所有歷史發布與日誌</a></p>
            `;
        } else {
            changelogContent.innerHTML = `
                <h3>Unable to load changelog dynamically</h3>
                <p>Due to GitHub API rate limiting or networking issues, we couldn't fetch the latest release details.</p>
                <p>Please click the link below to view it directly on GitHub:</p>
                <p><a href="https://github.com/${owner}/${repo}/releases" target="_blank" style="color: var(--primary-hover); text-decoration: underline;">Go to GitHub Releases to view history</a></p>
            `;
        }
    }

    function detectBrowserLanguage() {
        const lang = navigator.language || navigator.userLanguage || 'en';
        if (lang.toLowerCase().startsWith('zh')) {
            return 'zh';
        }
        return 'en';
    }

    // 預加載圖片函數，防止切換主題或 Tab 時白屏閃爍
    function preloadImages() {
        const urls = [];
        // 收集 tabs 中的圖片
        document.querySelectorAll('.gallery-tab').forEach(tab => {
            const dark = tab.getAttribute('data-img-dark');
            const sketch = tab.getAttribute('data-img-sketch');
            if (dark) urls.push(dark);
            if (sketch) urls.push(sketch);
        });
        // 收集 hero 中的圖片
        const hero = document.getElementById('hero-main-img');
        if (hero) {
            const dark = hero.getAttribute('data-img-dark');
            const sketch = hero.getAttribute('data-img-sketch');
            if (dark) urls.push(dark);
            if (sketch) urls.push(sketch);
        }

        // 去重並在背景載入
        const uniqueUrls = [...new Set(urls)];
        uniqueUrls.forEach(url => {
            const img = new Image();
            img.src = url;
        });
    }

    const savedLang = localStorage.getItem('wincmp_lang') || detectBrowserLanguage();
    switchLanguage(savedLang);
    getLatestRelease();
    preloadImages(); // 啟動預加載

    // ==========================================================================
    // 5. 手機版選單切換 (Hamburger Menu)
    // ==========================================================================
    const navbar = document.getElementById('navbar');
    const navToggle = document.getElementById('nav-toggle');
    const navLinks = document.querySelectorAll('.nav-link, #lang-switch-btn, #theme-switch-btn, #github-nav-btn');

    if (navToggle && navbar) {
        navToggle.addEventListener('click', () => {
            const isOpen = navbar.classList.toggle('nav-open');
            document.body.classList.toggle('menu-open', isOpen);

            // 切換 Lucide 圖示 (menu <-> x)
            const toggleIcon = navToggle.querySelector('i');
            if (toggleIcon) {
                if (isOpen) {
                    toggleIcon.setAttribute('data-lucide', 'x');
                } else {
                    toggleIcon.setAttribute('data-lucide', 'menu');
                }
                if (window.lucide) {
                    window.lucide.createIcons();
                }
            }
        });

        // 點擊任何導覽連結或按鈕時自動關閉選單
        navLinks.forEach(link => {
            link.addEventListener('click', () => {
                if (navbar.classList.contains('nav-open')) {
                    navbar.classList.remove('nav-open');
                    document.body.classList.remove('menu-open');
                    const toggleIcon = navToggle.querySelector('i');
                    if (toggleIcon) {
                        toggleIcon.setAttribute('data-lucide', 'menu');
                        if (window.lucide) {
                            window.lucide.createIcons();
                        }
                    }
                }
            });
        });
    }
});

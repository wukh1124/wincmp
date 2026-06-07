document.addEventListener('DOMContentLoaded', () => {
    // ==========================================================================
    // 1. 多國語言字典與核心切換邏輯 (i18n)
    // ==========================================================================
    const translations = {
        en: {
            doc_title: "WinCMP - Portable Local Development Control Panel for Windows (Caddy + MariaDB + PHP + Mailpit)",
            nav_features: "Features",
            nav_gallery: "Screenshots",
            nav_comparison: "Comparison",
            nav_architecture: "Architecture",
            nav_changelog: "Changelog",
            lang_btn_text: "繁體中文",
            hero_badge_prefix: "Latest Version:",
            hero_title: "Extreme Lightweight & Portable<br><span class=\"gradient-text\">Local Dev Control Panel</span> for Windows",
            hero_subtitle: "Integrated with <strong>Caddy</strong> + <strong>MariaDB</strong> + <strong>PHP</strong> + <strong>Mailpit</strong> and interactive terminal. Core development services run entirely without admin privileges. Fast, lightweight, and hassle-free!",
            hero_download_btn: "Download Now (WinCMP.exe)",
            hero_github_btn: "View GitHub Repo",
            features_title: "Why Choose WinCMP?",
            features_subtitle: "Experience the ultimate local Web development environment on Windows with zero setup and maximum performance!",
            feat_1_title: "Extreme Lightweight",
            feat_1_desc: "Statically compiled in Go + Wails, leveraging native OS WebView2 without Electron. Starts in milliseconds, idling memory usage is only about 30-50MB.",
            feat_2_title: "Admin-Privilege-Free Core",
            feat_2_desc: "Caddy, PHP-CGI, MariaDB, and Mailpit run entirely under restricted user permissions. No system registry writes or path pollution, fully portable.",
            feat_3_title: "Interactive Terminal Drawer",
            feat_3_desc: "Built-in Windows ConPTY and xterm.js at project root. Slide out a fully functional terminal drawer supporting PowerShell/CMD/Git Bash/WSL with autocomplete.",
            feat_4_title: "PHP Multi-Process Balancing",
            feat_4_desc: "Utilizes Caddy's upstream load balancing to spawn 3 independent FastCGI processes for each PHP version, boosting local development reliability.",
            feat_5_title: "Runtime Multi-Environment",
            feat_5_desc: "Supports Node.js, Bun, Python, and Go frameworks. Easily toggle between running in the background or spawning inside a separate terminal window.",
            feat_6_title: "Hosts Sync & Backup (Needs Admin)",
            feat_6_desc: "Easily sync custom domains to the Windows hosts file. Due to Windows security protection, this writing and backup feature requires UAC elevation.",
            gallery_title: "Intuitive & Modern User Interface",
            gallery_subtitle: "Every screen is carefully polished with smooth transitions and real-time status monitoring. Beautiful and powerful!",
            gallery_tab_dashboard: "Dashboard",
            gallery_tab_projects: "Projects",
            gallery_tab_terminal: "Built-in Terminal",
            gallery_tab_add_project: "Add Project",
            gallery_tab_resource: "Resource Monitor",
            gallery_tab_settings: "Preferences",
            gallery_tab_dependencies: "Dependency Downloader",
            compare_title: "WinCMP Specifications",
            compare_subtitle: "WinCMP core specifications and hardcore metrics on Windows:",
            compare_head_metric: "Metric",
            compare_row_1_name: "Architecture",
            compare_row_1_val_1: "Go 1.26 + Wails v2 + React 18",
            compare_row_2_name: "Startup Speed",
            compare_row_2_val_1: "Tends to be faster (Go Core)",
            compare_row_3_name: "Idle RAM",
            compare_row_3_val_1: "Lower",
            compare_row_4_name: "Privilege",
            compare_row_4_val_1: "User Space (No Admin)",
            compare_row_5_name: "Hosts Control",
            compare_row_5_val_1: "Yes (UAC check with auto-backup)",
            compare_row_6_name: "Terminal",
            compare_row_6_val_1: "Built-in xterm.js drawer",
            compare_row_7_name: "Email Sandbox",
            compare_row_7_val_1: "Built-in Mailpit (Zero configuration)",
            compare_row_8_name: "Config Apply",
            compare_row_8_val_1: "Auto Hot Reload (Zero Downtime)",
            compare_table_note: "* Note: RAM usage refers to the Control Panel UI itself in idle state.",
            arch_title: "WinCMP Architecture",
            arch_subtitle: "Here is a glimpse of how the backend is designed for high performance and isolation:",
            arch_card_1_title: "Dynamic Env Isolation",
            arch_card_1_desc: "Instead of modifying Windows system global variables, WinCMP creates a temporary execution path only for the active project. This allows multiple projects to run different PHP/Node versions simultaneously without any system pollution.",
            arch_card_2_title: "Smart Ports & Seamless Reload",
            arch_card_2_desc: "WinCMP automatically manages ports to prevent conflicts when running multiple PHP/Node versions. Any configuration changes are applied instantly in the background via Caddy's hot-reload, ensuring zero downtime.",
            changelog_title: "Latest Updates",
            changelog_subtitle: "See what new features and fixes have been added recently!",
            changelog_loading: "Fetching latest changelog from GitHub...",
            footer_desc: "WinCMP gratefully integrates and acknowledges open-source projects including Caddy, MariaDB, PHP, Mailpit, Node.js, Composer, HeidiSQL, Wails, and React."
        },
        zh: {
            doc_title: "WinCMP - 專為 Windows 打造的極致輕量、免安裝本地開發控制面板 (整合 Caddy + MariaDB + PHP + Mailpit)",
            nav_features: "特色功能",
            nav_gallery: "介面展示",
            nav_comparison: "性能對比",
            nav_architecture: "架構原理",
            nav_changelog: "更新日誌",
            lang_btn_text: "English",
            hero_badge_prefix: "最新版本:",
            hero_title: "專為 Windows 打造的<br><span class=\"gradient-text\">極致輕量免安裝</span>開發面板",
            hero_subtitle: "整合 <strong>Caddy</strong> + <strong>MariaDB</strong> + <strong>PHP</strong> + <strong>Mailpit</strong> 與專案互動終端。核心開發服務啟動免管理員權限，極速、輕量、無負擔！",
            hero_download_btn: "立即下載 (WinCMP.exe)",
            hero_github_btn: "瀏覽 GitHub 倉庫",
            features_title: "為什麼選擇 WinCMP？",
            features_subtitle: "為開發者提供極致的 Windows 本地 Web 開發體驗，零配置、高效能，讓開發更輕鬆流暢！",
            feat_1_title: "極致輕量 (Statically Compiled)",
            feat_1_desc: "基於 Go + Wails 靜態編譯，直接調用 Windows 原生 WebView2 引擎，無需 Electron 運作。啟動僅需毫秒級，日常閒置記憶體佔用僅約 30-50MB。",
            feat_2_title: "核心服務免 Admin 權限",
            feat_2_desc: "Caddy、PHP-CGI、MariaDB、Mailpit 啟動完全執行在受限的用戶權限下，免寫系統登錄檔、免污染全域環境變數，完全綠色可攜。",
            feat_3_title: "整合式互動終端 Drawer",
            feat_3_desc: "集成 Windows ConPTY 與 xterm.js，在專案根目錄一鍵滑出美觀的互動式終端（支援 PowerShell/CMD/Git Bash/WSL），內建自動補全與自訂樣式。",
            feat_4_title: "PHP 多進程負載均衡",
            feat_4_desc: "利用 Caddy 的 upstream 負載均衡機制，為每個 PHP 版本啟動 3 個獨立的 FastCGI 進程進行輪詢分發，大幅提升本地開發高併發穩定度。",
            feat_5_title: "運行時多環境支援",
            feat_5_desc: "支援 Node.js、Bun、Python、Go 等多種 Framework (Next.js/Nuxt/Astro/FastAPI/Go API)，並貼心提供「背景默默執行」與「獨立終端執行」雙模式。",
            feat_6_title: "Hosts 備份與更新 (需要 Admin)",
            feat_6_desc: "支援點擊一鍵同步自訂域名到 Windows Hosts 檔案。由於系統安全防護，此寫入與自動備份功能需要彈出管理員 (UAC) 提權確認。",
            gallery_title: "直觀、美觀的現代介面",
            gallery_subtitle: "每一處 UI 都經過的細心打磨，具備流暢的微動畫與實時狀態監控，好看又好用！",
            gallery_tab_dashboard: "主控台 Dashboard",
            gallery_tab_projects: "專案管理",
            gallery_tab_terminal: "內建終端",
            gallery_tab_add_project: "新增專案",
            gallery_tab_resource: "資源佔用監控",
            gallery_tab_settings: "偏好設定",
            gallery_tab_dependencies: "依賴下載器",
            compare_title: "WinCMP 技術規格",
            compare_subtitle: "整理 WinCMP 核心規格與硬派技術指標：",
            compare_head_metric: "項目",
            compare_head_wincmp: "WinCMP",
            compare_row_1_name: "底層架構",
            compare_row_1_val_1: "Go 1.26 + Wails v2 + React 18",
            compare_row_2_name: "啟動速度",
            compare_row_2_val_1: "傾向更快 (Go 核心)",
            compare_row_3_name: "記憶體佔用",
            compare_row_3_val_1: "較低",
            compare_row_4_name: "啟動權限",
            compare_row_4_val_1: "用戶空間 (免管理員權限)",
            compare_row_5_name: "Hosts 管理",
            compare_row_5_val_1: "支援 (需 UAC 提權，自動備份防鎖死)",
            compare_row_6_name: "內建終端",
            compare_row_6_val_1: "內建 ConPTY + xterm.js 抽屜終端",
            compare_row_7_name: "郵件測試",
            compare_row_7_val_1: "內建 Mailpit (免設定開箱即用)",
            compare_row_8_name: "配置生效",
            compare_row_8_val_1: "自動熱重載 (零停機)",
            compare_table_note: "* 註：記憶體佔用指控制面板主程式本身的閒置狀態。",
            arch_title: "WinCMP 底層運作原理",
            arch_subtitle: "後端寫了許多巧思，確保極致的效能與乾淨的環境隔離：",
            arch_card_1_title: "動態環境變數隔離",
            arch_card_1_desc: "不修改 Windows 系統全域設定，WinCMP 只在啟動專案時為其建立暫時的執行路徑。這能讓不同的專案同時執行不同版本的 PHP 或 Node，彼此互不干擾，系統也依然保持綠色乾淨！",
            arch_card_2_title: "智慧連接埠與無縫重載",
            arch_card_2_desc: "WinCMP 會自動管理並分配服務連接埠，避免多個 PHP 或 Node 版本同時執行時發生衝突。調整設定時，系統會在背景即時套用變更，過程完全不需重啟伺服器，保證網頁連線不中斷！",
            changelog_title: "最新版本更新日誌",
            changelog_subtitle: "看看最近幫 WinCMP 增加了什麼厲害的新魔法吧！",
            changelog_loading: "正在從 GitHub 獲取最新發布日誌...",
            footer_desc: "WinCMP 整合並致敬優秀的開源生態組件，包括 Caddy, MariaDB, PHP, Mailpit, Node.js, Composer, HeidiSQL, Wails 與 React。"
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

        // 更新語言切換按鈕文字
        const langBtnText = document.getElementById('lang-btn-text');
        if (langBtnText) {
            langBtnText.innerText = translations[lang]['lang_btn_text'];
        }

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

            // 漸變淡出淡入效果
            displayImg.style.opacity = '0.3';
            setTimeout(() => {
                displayImg.src = imgSrc;
                displayImg.alt = `WinCMP ${tabName}`;
                windowTitle.innerText = `WinCMP - ${tabName}`;
                displayImg.style.opacity = '1';
            }, 150);
        });
    });

    // ==========================================================================
    // 3. 獲取最新 Release 版本與更新日誌
    // ==========================================================================
    const owner = 'wukh1124';
    const repo = 'wincmp';
    const apiURL = `https://api.github.com/repos/${owner}/${repo}/releases/latest`;

    const latestTagEl = document.getElementById('latest-tag');
    const downloadBtn = document.getElementById('download-btn');
    const changelogLoading = document.getElementById('changelog-loading');
    const changelogContent = document.getElementById('changelog-content');

    async function getLatestRelease() {
        const cacheDataKey = 'wincmp_release_data';
        const cacheExpiryKey = 'wincmp_release_expiry';
        const cacheDuration = 15 * 60 * 1000;
        const now = Date.now();

        const cachedData = localStorage.getItem(cacheDataKey);
        const cachedExpiry = localStorage.getItem(cacheExpiryKey);

        if (cachedData && cachedExpiry && now < parseInt(cachedExpiry)) {
            renderReleaseInfo(JSON.parse(cachedData));
            return;
        }

        try {
            const response = await fetch(apiURL, {
                headers: {
                    'Accept': 'application.vnd.github.v3+json'
                }
            });

            if (!response.ok) {
                throw new Error(`GitHub API Error: ${response.status}`);
            }

            const data = await response.json();

            localStorage.setItem(cacheDataKey, JSON.stringify(data));
            localStorage.setItem(cacheExpiryKey, (now + cacheDuration).toString());

            renderReleaseInfo(data);
        } catch (error) {
            console.error('Failed to fetch latest release from GitHub:', error);
            if (cachedData) {
                renderReleaseInfo(JSON.parse(cachedData));
            } else {
                renderFallback();
            }
        }
    }

    function renderReleaseInfo(data) {
        const tagName = data.tag_name || 'v2.0.0';
        latestTagEl.innerText = tagName;

        let exeUrl = `https://github.com/${owner}/${repo}/releases/latest`;
        if (data.assets && data.assets.length > 0) {
            const exeAsset = data.assets.find(asset => asset.name.endsWith('.exe'));
            if (exeAsset) {
                exeUrl = exeAsset.browser_download_url;
            } else {
                exeUrl = data.assets[0].browser_download_url;
            }
        }
        downloadBtn.href = exeUrl;

        if (window.marked && data.body) {
            changelogLoading.style.display = 'none';
            changelogContent.style.display = 'block';
            changelogContent.innerHTML = window.marked.parse(data.body);
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

    // ==========================================================================
    // 4. 初始化語系與發布讀取
    // ==========================================================================
    const savedLang = localStorage.getItem('wincmp_lang') || 'en';
    switchLanguage(savedLang);
    getLatestRelease();

    // ==========================================================================
    // 5. 手機版選單切換 (Hamburger Menu)
    // ==========================================================================
    const navbar = document.getElementById('navbar');
    const navToggle = document.getElementById('nav-toggle');
    const navLinks = document.querySelectorAll('.nav-link, #lang-switch-btn, #github-nav-btn');

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

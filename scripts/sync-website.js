const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const PROJECT_ROOT = path.resolve(__dirname, '..');
const WEBSITE_DIR = path.join(PROJECT_ROOT, 'website');
const SCREENSHOT_SRC = path.join(PROJECT_ROOT, 'screenshot');
const SCREENSHOT_DST = path.join(WEBSITE_DIR, 'screenshot');
const ICON_SRC = path.join(PROJECT_ROOT, 'icon.svg');
const FAVICON_DST = path.join(WEBSITE_DIR, 'favicon.svg');

/**
 * 遞迴複製目錄
 */
function copyDirSync(src, dst) {
    if (!fs.existsSync(src)) return;
    if (!fs.existsSync(dst)) {
        fs.mkdirSync(dst, { recursive: true });
    }
    const entries = fs.readdirSync(src, { withFileTypes: true });
    for (const entry of entries) {
        const srcPath = path.join(src, entry.name);
        const dstPath = path.join(dst, entry.name);
        if (entry.isDirectory()) {
            copyDirSync(srcPath, dstPath);
        } else {
            fs.copyFileSync(srcPath, dstPath);
        }
    }
}

function syncWebsite() {
    console.log('===================================================');
    console.log('     WinCMP 官網資源同步工具 (Website Sync)        ');
    console.log('===================================================');

    if (!fs.existsSync(WEBSITE_DIR)) {
        fs.mkdirSync(WEBSITE_DIR, { recursive: true });
    }

    // 1. 同步圖標
    if (fs.existsSync(ICON_SRC)) {
        fs.copyFileSync(ICON_SRC, FAVICON_DST);
        console.log('[1/3] 已複製 icon.svg -> website/favicon.svg');
    } else {
        console.warn('[1/3] 警告: 未找到根目錄 icon.svg');
    }

    // 2. 同步截圖
    const themes = ['dark', 'sketch'];
    let copiedThemes = 0;
    for (const theme of themes) {
        const src = path.join(SCREENSHOT_SRC, theme);
        const dst = path.join(SCREENSHOT_DST, theme);
        if (fs.existsSync(src)) {
            copyDirSync(src, dst);
            copiedThemes++;
            console.log(`[2/3] 已同步截圖: screenshot/${theme}/ -> website/screenshot/${theme}/`);
        }
    }
    if (copiedThemes === 0) {
        console.warn('[2/3] 提示: 目前 screenshot/ 下尚無截圖。如需產出截圖請執行: node scripts/capture.js');
    }

    // 3. 生成 release.json
    console.log('[3/3] 產生 website/release.json ...');
    const genScript = path.join(__dirname, 'generate-release-json.js');
    try {
        execSync(`node "${genScript}"`, {
            cwd: PROJECT_ROOT,
            stdio: 'inherit',
        });
    } catch (e) {
        console.error('產生 release.json 失敗:', e.message);
    }

    console.log('===================================================');
    console.log('[成功] 官網資源已同步完成！');
    console.log('您可以直接在瀏覽器中預覽 website/index.html 進行本地測試。');
    console.log('===================================================');
}

if (require.main === module) {
    syncWebsite();
}

module.exports = { syncWebsite };

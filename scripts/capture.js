const fs = require('fs');
const path = require('path');
const http = require('http');
const { execSync, spawn } = require('child_process');

const PROJECT_ROOT = path.resolve(__dirname, '..');
const FRONTEND_DIR = path.join(PROJECT_ROOT, 'frontend');
const SCREENSHOT_DIR = path.join(PROJECT_ROOT, 'screenshot');
const BACKUP_DIR = path.join(SCREENSHOT_DIR, 'backup');
const TARGET_DEV_URL = 'http://localhost:34115';

/**
 * 讀取根目錄 VERSION
 */
function getTargetVersion() {
    const versionPath = path.join(PROJECT_ROOT, 'VERSION');
    if (!fs.existsSync(versionPath)) {
        throw new Error(`找不到 VERSION 檔案: ${versionPath}`);
    }
    const raw = fs.readFileSync(versionPath, 'utf-8').trim();
    return raw.replace(/^v/, '');
}

/**
 * 從 git tag 取得發布前的上一版本號
 */
function getPreviousVersion(newVersion) {
    try {
        const out = execSync('git tag -l --sort=-v:refname', {
            cwd: PROJECT_ROOT,
            encoding: 'utf-8',
        });
        const tags = out.split(/\r?\n/).map((t) => t.trim().replace(/^v/, '')).filter(Boolean);
        for (const t of tags) {
            if (t && t !== newVersion) {
                return t;
            }
        }
    } catch (e) {
        console.warn('   [Warn] 讀取 git tags 失敗');
    }
    return 'unknown';
}

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

/**
 * 備份當前截圖到 screenshot/backup/v{prev}/
 */
function backupExistingScreenshots(prevVersion) {
    const themes = ['dark', 'sketch'];
    const hasExisting = themes.some((td) => fs.existsSync(path.join(SCREENSHOT_DIR, td)));

    if (!hasExisting) {
        console.log('[1/4] 無現有截圖，跳過備份。');
        return;
    }

    const targetBackup = path.join(BACKUP_DIR, `v${prevVersion}`);
    console.log(`[1/4] 正在備份現有截圖 -> screenshot/backup/v${prevVersion}/ ...`);

    if (fs.existsSync(targetBackup)) {
        fs.rmSync(targetBackup, { recursive: true, force: true });
    }
    fs.mkdirSync(targetBackup, { recursive: true });

    for (const td of themes) {
        const src = path.join(SCREENSHOT_DIR, td);
        const dst = path.join(targetBackup, td);
        if (fs.existsSync(src)) {
            copyDirSync(src, dst);
            console.log(`   -> 已備份 ${td}`);
        }
    }
}

/**
 * 檢查 Wails 開發伺服器是否就緒
 */
function checkDevServer() {
    return new Promise((resolve) => {
        const req = http.get(TARGET_DEV_URL, (res) => {
            if (res.statusCode === 200 || res.statusCode === 304) {
                resolve(true);
            } else {
                resolve(false);
            }
        });
        req.on('error', () => resolve(false));
        req.setTimeout(3000, () => {
            req.destroy();
            resolve(false);
        });
    });
}

/**
 * 檢查 Playwright 模組
 */
function ensurePlaywrightInstalled() {
    const playwrightPkg = path.join(FRONTEND_DIR, 'node_modules', 'playwright');
    if (!fs.existsSync(playwrightPkg)) {
        console.log('   [Notice] 未偵測到 Playwright，正在安裝前端依賴...');
        execSync('npm install', { cwd: FRONTEND_DIR, stdio: 'inherit' });
        execSync('npx playwright install chromium', { cwd: FRONTEND_DIR, stdio: 'inherit' });
    }
}

/**
 * 執行截圖腳本
 */
function runPlaywrightCapture() {
    return new Promise((resolve, reject) => {
        const captureScript = path.join(FRONTEND_DIR, 'scripts', 'capture.cjs');
        console.log('[3/4] 啟動 Playwright 進行頁面與主題擷圖...');
        const proc = spawn('node', [captureScript], {
            cwd: FRONTEND_DIR,
            stdio: 'inherit',
            env: process.env,
        });

        proc.on('close', (code) => {
            if (code === 0) {
                resolve();
            } else {
                reject(new Error(`截圖腳本異常退出，代碼: ${code}`));
            }
        });
        proc.on('error', (err) => reject(err));
    });
}

async function main() {
    const args = process.argv.slice(2);
    const shouldSyncWebsite = args.includes('--sync-website');

    console.log('===================================================');
    console.log('     WinCMP 自動化截圖工具 (Node.js)               ');
    console.log('===================================================');

    const newVersion = getTargetVersion();
    const prevVersion = getPreviousVersion(newVersion);
    console.log(`[目標版本] v${newVersion}`);
    console.log(`[前一版本] v${prevVersion}`);

    // 1. 備份舊圖
    backupExistingScreenshots(prevVersion);

    // 2. 檢查 Wails dev server
    console.log(`[2/4] 檢查 Wails 開發伺服器 (${TARGET_DEV_URL})...`);
    const isDevReady = await checkDevServer();
    if (!isDevReady) {
        console.error('\n[錯誤] 無法連線至 Wails 開發伺服器。');
        console.error('請先在另一個終端機執行：');
        console.error('  wails dev');
        console.error('待伺服器啟動完成後，再重新執行本指令：');
        console.error('  node scripts/capture.js\n');
        process.exit(1);
    }
    console.log('   -> Wails 開發伺服器已就緒。');

    // 3. 確保 Playwright 就緒並截圖
    ensurePlaywrightInstalled();
    await runPlaywrightCapture();

    console.log('\n===================================================');
    console.log(`[成功] v${newVersion} 截圖已更新完畢！`);
    console.log(`歷史截圖已備份至: screenshot/backup/v${prevVersion}/`);
    console.log('===================================================');

    // 4. 若指定 --sync-website，自動同步至 website/
    if (shouldSyncWebsite) {
        console.log('\n[同步官網] 偵測到 --sync-website，正在同步至 website/...');
        const syncScript = path.join(__dirname, 'sync-website.js');
        if (fs.existsSync(syncScript)) {
            require(syncScript);
        }
    } else {
        console.log('\n提示: 如需同步截圖至官網目錄進行本地測試，可執行:');
        console.log('  node scripts/sync-website.js');
    }
}

main().catch((err) => {
    console.error('\n[執行失敗]', err.message);
    process.exit(1);
});

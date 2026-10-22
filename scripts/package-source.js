/**
 * 打包純源碼為 7z 壓縮檔。
 *
 * 沿用既有 bat/zip-exclude.txt 排除清單（排除 bin/、build/、node_modules、日誌與資料庫檔案），
 * 輸出到 bat/wincmp_<yyyyMMdd_HHmmss>.7z，與舊版 zip-wincmp.bat 的行為一致。
 *
 * 用法：node scripts/package-source.js
 */

const { execFileSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '..');
const PARENT = path.resolve(ROOT, '..');
const PROJECT_NAME = path.basename(ROOT);
const EXCLUDE_FILE = path.join(ROOT, 'bat', 'zip-exclude.txt');
const OUTPUT_DIR = path.join(ROOT, 'bat');

/** 依序尋找 7-Zip：常見安裝路徑優先，其次 PATH。 */
function find7z() {
  const candidates = [
    'C:\\Program Files\\7-Zip\\7z.exe',
    'C:\\Program Files (x86)\\7-Zip\\7z.exe',
  ];
  for (const candidate of candidates) {
    if (fs.existsSync(candidate)) return candidate;
  }
  try {
    execFileSync('7z', ['i'], { stdio: 'ignore' });
    return '7z';
  } catch {
    return null;
  }
}

/** yyyyMMdd_HHmmss（本地時間）。 */
function timestamp() {
  const d = new Date();
  const p = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}${p(d.getMonth() + 1)}${p(d.getDate())}_${p(d.getHours())}${p(d.getMinutes())}${p(d.getSeconds())}`;
}

function main() {
  const sevenZip = find7z();
  if (!sevenZip) {
    console.error('[WinCMP] 找不到 7-Zip，請先安裝後再執行 run-zip。');
    process.exit(1);
  }

  fs.mkdirSync(OUTPUT_DIR, { recursive: true });
  const output = path.join(OUTPUT_DIR, `wincmp_${timestamp()}.7z`);

  const hasExclude = fs.existsSync(EXCLUDE_FILE);
  if (!hasExclude) {
    console.warn('[WinCMP] 警告：找不到 bat/zip-exclude.txt，將打包整個專案目錄。');
  }

  // 於上層目錄執行，讓壓縮檔內保留專案資料夾層級。
  const args = ['a', '-t7z', output, PROJECT_NAME, '-r', '-mx5'];
  if (hasExclude) args.push(`-x@${EXCLUDE_FILE}`);

  console.log(`[WinCMP] 專案：${PROJECT_NAME}`);
  console.log(`[WinCMP] 排除清單：${hasExclude ? EXCLUDE_FILE : '（無）'}`);
  console.log(`[WinCMP] 輸出：${output}`);
  console.log('[WinCMP] 開始壓縮...');

  try {
    execFileSync(sevenZip, args, { cwd: PARENT, stdio: 'inherit' });
  } catch (err) {
    console.error(`[WinCMP] 壓縮失敗：${err.message}`);
    process.exit(1);
  }

  const sizeMB = fs.statSync(output).size / 1024 / 1024;
  console.log(`[WinCMP] 完成：${output}（${sizeMB.toFixed(2)} MB）`);
}

main();

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

// 自動向上尋找專案根目錄 (以包含 VERSION 或 wails.json 為基準)
let currentDir = __dirname;
let projectRoot = currentDir;
for (let i = 0; i < 6; i++) {
  if (fs.existsSync(path.join(currentDir, 'VERSION')) && fs.existsSync(path.join(currentDir, 'wails.json'))) {
    projectRoot = currentDir;
    break;
  }
  const parent = path.dirname(currentDir);
  if (parent === currentDir) break;
  currentDir = parent;
}

console.log('===================================================');
console.log('         WinCMP Release Pre-flight Validator       ');
console.log('===================================================');
console.log(`[專案目錄] ${projectRoot}`);

let hasError = false;

// 1. 檢查 VERSION 檔案
console.log('[1/5] 檢查 VERSION 檔案...');
const versionFile = path.join(projectRoot, 'VERSION');
if (!fs.existsSync(versionFile)) {
  console.error('  [錯誤] 找不到 VERSION 檔案！');
  process.exit(1);
}

const rawVersion = fs.readFileSync(versionFile, 'utf8').trim();
const match = rawVersion.match(/^v?(\d+\.\d+\.\d+)$/);
if (!match) {
  console.error(`  [錯誤] VERSION 格式不符合語意化版本 (x.y.z): '${rawVersion}'`);
  hasError = true;
}
const version = match ? match[1] : rawVersion;
console.log(`  -> 目標版本號: v${version}`);

// 2. 7 大分類標籤白名單
const allowedCategories = ['Added', 'Changed', 'Deprecated', 'Removed', 'Fixed', 'Security', 'Dependencies'];

function checkChangelog(filePath, fileLabel, ver) {
  if (!fs.existsSync(filePath)) {
    console.error(`  [錯誤] 找不到 ${fileLabel} (${filePath})`);
    return false;
  }

  const content = fs.readFileSync(filePath, 'utf8');
  const lines = content.split(/\r?\n/);

  let foundVersion = false;
  let inTargetSection = false;
  const sectionCategories = [];
  const invalidCategories = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim();

    // 匹配版本標題，如 "## [2.0.6]" 或 "## 2.0.6"
    const verHeaderRegex = new RegExp(`^##\\s+\\[?${ver.replace(/\./g, '\\.')}\\]?`);
    if (verHeaderRegex.test(line)) {
      foundVersion = true;
      inTargetSection = true;
      continue;
    }

    // 遇到下一個版本區塊則結束當前檢查
    if (inTargetSection && /^##\s+/.test(line)) {
      break;
    }

    if (inTargetSection) {
      const catMatch = line.match(/^###\s+(.+)$/);
      if (catMatch) {
        const cat = catMatch[1].trim();
        sectionCategories.push(cat);
        if (!allowedCategories.includes(cat)) {
          invalidCategories.push(`第 ${i + 1} 行: '${cat}'`);
        }
      }
    }
  }

  if (!foundVersion) {
    console.error(`  [錯誤] ${fileLabel} 中未找到版本 [v${ver}] 之更新記錄區塊！`);
    return false;
  }

  if (invalidCategories.length > 0) {
    console.error(`  [錯誤] ${fileLabel} 含有非白名單分類標籤:`);
    invalidCategories.forEach((inv) => console.error(`         - ${inv}`));
    console.warn(`         (僅允許: ${allowedCategories.join(', ')})`);
    return false;
  }

  if (sectionCategories.length === 0) {
    console.warn(`  [警告] ${fileLabel} 的 [v${ver}] 區塊中未包含任何分類標籤 (### Added, Fixed 等)。`);
  } else {
    console.log(`  -> ${fileLabel} 驗證通過 (包含分類: ${sectionCategories.join(', ')})`);
  }

  return true;
}

// 3. 檢查雙語 Changelog
console.log('[2/5] 檢查雙語 Changelog 格式...');
const zhPassed = checkChangelog(path.join(projectRoot, 'CHANGELOG_zh.md'), '繁體中文更新日誌 (CHANGELOG_zh.md)', version);
const enPassed = checkChangelog(path.join(projectRoot, 'CHANGELOG.md'), '英文更新日誌 (CHANGELOG.md)', version);

if (!zhPassed || !enPassed) {
  hasError = true;
}

// 4. 檢查內部稽核檔 audit_commits.md
console.log('[3/5] 檢查內部稽核檔案 (audit_commits.md)...');
const auditFile = path.join(projectRoot, 'release_note', `v${version}`, 'audit_commits.md');
if (fs.existsSync(auditFile)) {
  console.log(`  -> 找到審核底稿: release_note\\v${version}\\audit_commits.md`);
} else {
  console.warn(`  [提醒] 尚未建立審核底稿: release_note\\v${version}\\audit_commits.md (發布前強烈建議完成以供歷史溯源)`);
}

// 5. 檢查 Git 狀態
console.log('[4/5] 檢查 Git 工作區狀態...');
try {
  const gitStatus = execSync('git status --porcelain', { cwd: projectRoot, encoding: 'utf8' }).trim();
  if (gitStatus) {
    console.log('  [資訊] 目前工作區存在變更 (請於確認後提交發布)。');
  } else {
    console.log('  -> Git 工作區為純淨狀態。');
  }
} catch (e) {
  console.warn('  [警告] 無法執行 git status 檢查。');
}

// 6. 總結
console.log('[5/5] 驗證結果彙總...');
if (hasError) {
  console.error('\n[失敗] 發行前驗證未通過，請修正上述錯誤項目後再行發布！');
  process.exit(1);
} else {
  console.log('\n[成功] 發行規範驗證完全通過！可繼續執行編譯打包或 release.bat。');
  process.exit(0);
}

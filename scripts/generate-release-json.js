const fs = require('fs');
const path = require('path');

function run() {
    try {
        const owner = 'wukh1124';
        const repo = 'wincmp';

        const projectRoot = path.join(__dirname, '..');
        const releaseNoteDir = path.join(projectRoot, 'release_note');
        const infoJsonPath = path.join(releaseNoteDir, 'release_info.json');

        if (!fs.existsSync(infoJsonPath)) {
            throw new Error(`release_info.json not found at ${infoJsonPath}`);
        }

        console.log('Reading release_info.json...');
        const infoData = JSON.parse(fs.readFileSync(infoJsonPath, 'utf-8'));
        const rawVersion = infoData['latest-version'] || '2.0.0';
        // 確保格式為 vX.Y.Z
        const version = rawVersion.startsWith('v') ? rawVersion : `v${rawVersion}`;

        console.log(`Detected latest version: ${version}`);

        // 自動拼接下載連結
        const exeUrl = `https://github.com/${owner}/${repo}/releases/download/${version}/wincmp-${version}-win-x64.zip`;

        // 讀取中英文更新日誌
        const versionDir = path.join(releaseNoteDir, `v${rawVersion.startsWith('v') ? rawVersion.substring(1) : rawVersion}`);
        const enNotesPath = path.join(versionDir, 'release_notes.md');
        const zhNotesPath = path.join(versionDir, 'release_notes_zh.md');

        let changelogEn = '';
        let changelogZh = '';

        if (fs.existsSync(enNotesPath)) {
            changelogEn = fs.readFileSync(enNotesPath, 'utf-8').trim();
        } else {
            console.log(`Warning: English release notes not found at ${enNotesPath}`);
            changelogEn = `# WinCMP ${version}\n- Maintenance updates and stability improvements.`;
        }

        if (fs.existsSync(zhNotesPath)) {
            changelogZh = fs.readFileSync(zhNotesPath, 'utf-8').trim();
        } else {
            console.log(`Warning: Chinese release notes not found at ${zhNotesPath}`);
            changelogZh = `# WinCMP ${version}\n- 維護更新與穩定性優化。`;
        }

        const releaseData = {
            tag_name: version,
            exe_url: exeUrl,
            changelog_zh: changelogZh,
            changelog_en: changelogEn
        };

        const targetDir = path.join(projectRoot, 'website');
        if (!fs.existsSync(targetDir)) {
            fs.mkdirSync(targetDir, { recursive: true });
        }

        fs.writeFileSync(
            path.join(targetDir, 'release.json'),
            JSON.stringify(releaseData, null, 2),
            'utf-8'
        );
        console.log('Successfully generated website/release.json!');
    } catch (error) {
        console.log('Error generating release.json:', error);
        process.exit(1);
    }
}

run();

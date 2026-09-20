const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

function normalizeVersion(raw) {
    const trimmed = String(raw || '').trim();
    if (!trimmed) return null;
    return trimmed.startsWith('v') ? trimmed : `v${trimmed}`;
}

/** 從 release notes 文字擷取發布日期（支援中英文日期行） */
function extractReleaseDate(text) {
    if (!text) return null;
    const patterns = [
        /^Release\s*date\s*[：:]\s*(\d{4}-\d{2}-\d{2})/im,
        /^發布日期\s*[：:]\s*(\d{4}-\d{2}-\d{2})/im,
        /^发布日期\s*[：:]\s*(\d{4}-\d{2}-\d{2})/im,
    ];
    for (const re of patterns) {
        const m = text.match(re);
        if (m) return m[1];
    }
    return null;
}

/** 以 git tag 建立日作為日期 fallback */
function gitTagDate(version) {
    try {
        const out = execSync(
            `git for-each-ref --format=%(creatordate:short) refs/tags/${version}`,
            { cwd: path.join(__dirname, '..'), encoding: 'utf8' }
        ).trim();
        return /^\d{4}-\d{2}-\d{2}$/.test(out) ? out : null;
    } catch (e) {
        return null;
    }
}

function run() {
    try {
        const owner = 'wukh1124';
        const repo = 'wincmp';

        const projectRoot = path.join(__dirname, '..');
        const releaseNoteDir = path.join(projectRoot, 'release_note');
        const versionPath = path.join(projectRoot, 'VERSION');

        if (!fs.existsSync(versionPath)) {
            throw new Error(`VERSION not found at ${versionPath}`);
        }

        console.log('Reading VERSION...');
        const rawVersion = fs.readFileSync(versionPath, 'utf-8').trim();
        const version = normalizeVersion(rawVersion);
        if (!version || !/^\d+\.\d+\.\d+$/.test(version.replace(/^v/, ''))) {
            throw new Error(`Invalid VERSION value: '${rawVersion}'`);
        }

        console.log(`Detected latest version: ${version}`);

        const rawNoV = version.replace(/^v/, '');
        const exeUrl = `https://github.com/${owner}/${repo}/releases/download/${version}/wincmp-${version}-win-x64.zip`;

        const versionDir = path.join(releaseNoteDir, `v${rawNoV}`);
        const enNotesPath = path.join(versionDir, 'release_notes.md');
        const zhNotesPath = path.join(versionDir, 'release_notes_zh.md');

        let changelogEn = '';
        let changelogZh = '';
        let releaseDate = null;

        if (fs.existsSync(enNotesPath)) {
            changelogEn = fs.readFileSync(enNotesPath, 'utf-8').trim();
            releaseDate = extractReleaseDate(changelogEn);
        } else {
            console.warn(`Warning: English release notes not found at ${enNotesPath}`);
            changelogEn = `# WinCMP ${version}\n\nRelease date: \n\nMaintenance updates and stability improvements.`;
        }

        if (fs.existsSync(zhNotesPath)) {
            changelogZh = fs.readFileSync(zhNotesPath, 'utf-8').trim();
            if (!releaseDate) {
                releaseDate = extractReleaseDate(changelogZh);
            }
        } else {
            console.warn(`Warning: Chinese release notes not found at ${zhNotesPath}`);
            changelogZh = `# WinCMP ${version}\n\n發布日期：\n\n維護更新與穩定性優化。`;
        }

        if (!releaseDate) {
            releaseDate = gitTagDate(version);
            if (releaseDate) {
                console.log(`Release date from git tag: ${releaseDate}`);
            } else {
                console.warn('Warning: release date not found in notes and git tag is missing');
            }
        } else {
            console.log(`Release date from notes: ${releaseDate}`);
        }

        const releaseData = {
            tag_name: version,
            exe_url: exeUrl,
            release_date: releaseDate || '',
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

import React, { useState, useEffect } from 'react';
import { ArrowUpCircle, Info, ShieldAlert, Check, RefreshCw } from 'lucide-react';
import { CheckNewVersion, StartAutoUpdate, GetConfig, GetAppVersion } from '../../wailsjs/go/main/App';
import { EventsOn, BrowserOpenURL } from '../../wailsjs/runtime/runtime';
import { t, useLanguage } from '../i18n';

interface ReleaseInfo {
  has_update: boolean;
  latest_version: string;
  release_notes: string;
  release_notes_zh: string;
  release_notes_en: string;
  published_at: string;
  download_url: string;
  asset_type: string;
}

// 輕量級 Regex Markdown 渲染器
function renderMarkdown(md: string): string {
  if (!md) return '';

  // 1. 將文字按行拆分，並過濾掉空白的行
  const lines = md.split(/\r?\n/).filter(line => line.trim() !== '');

  // 2. 逐行解析並轉譯
  const htmlLines = lines.map(line => {
    // 轉義 HTML 特殊字元防止 XSS
    let processed = line
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;');

    // 處理粗體: **text**
    processed = processed.replace(/\*\*([^\*]+)\*\*/g, '<strong style="color:var(--fg)" class="font-bold">$1</strong>');

    // 處理行內 code: `code`
    processed = processed.replace(/`([^`\n]+)`/g, '<code style="background:var(--surface);color:var(--status-warn);font-family:var(--font-mono)" class="px-1 py-0.5 rounded text-[11px]">$1</code>');

    // 標題 1: # Header
    if (/^#\s+(.+)$/.test(processed)) {
      return processed.replace(/^#\s+(.+)$/, '<h1 style="color:var(--fg);border-bottom:1px solid var(--border-soft);padding-bottom:6px;margin-top:14px;margin-bottom:10px" class="text-base font-bold">$1</h1>');
    }
    // 標題 2: ## Header
    if (/^##\s+(.+)$/.test(processed)) {
      return processed.replace(/^##\s+(.+)$/, '<h2 style="color:var(--fg-2);margin-top:20px;margin-bottom:8px" class="text-sm font-bold flex items-center gap-1">$1</h2>');
    }
    // 標題 3: ### Header
    if (/^###\s+(.+)$/.test(processed)) {
      return processed.replace(/^###\s+(.+)$/, '<h3 style="color:var(--muted);margin-top:14px;margin-bottom:6px" class="text-xs font-bold">$1</h3>');
    }
    // 無序清單: - list
    if (/^[-\*]\s+(.+)$/.test(processed)) {
      return processed.replace(/^[-\*]\s+(.+)$/, '<li style="color:var(--muted);margin-bottom:6px" class="ml-4 list-disc text-xs leading-normal">$1</li>');
    }

    // 一般文字段落，包在 div 中並加入微調的 margin 避免粘連
    return `<div style="margin-bottom:10px;color:var(--muted)" class="text-xs leading-normal">${processed}</div>`;
  });

  return htmlLines.join('');
}

export default function VersionUpdate() {
  const lang = useLanguage(); // 訂閱語系變更並取得當前語言

  const [currentVersion, setCurrentVersion] = useState('v2.0.0');
  const [releaseInfo, setReleaseInfo] = useState<ReleaseInfo | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [config, setConfig] = useState<any>(null);

  // 更新下載狀態
  const [updateStatus, setUpdateStatus] = useState<'idle' | 'downloading' | 'completed' | 'error'>('idle');
  const [downloadProgress, setDownloadProgress] = useState({ percent: 0, currentMB: 0, totalMB: 0 });
  const [errorMessage, setErrorMessage] = useState('');

  // 1. 初始化讀取設定與版本
  useEffect(() => {
    async function init() {
      try {
        const ver = await GetAppVersion();
        setCurrentVersion(ver);

        const cfg = await GetConfig();
        setConfig(cfg);

        // 初始化時靜默獲取最新版本 (後端已加載快取會瞬間返回)
        const info = await CheckNewVersion();
        setReleaseInfo(info as ReleaseInfo);
      } catch (err) {
        console.error("初始化版本更新頁面失敗:", err);
      } finally {
        setIsLoading(false);
      }
    }
    init();
  }, []);

  // 2. 監聽後端推送的下載進度事件
  useEffect(() => {
    const unsubscribe = EventsOn('update_progress', (data: any) => {
      if (!data) return;
      if (data.status === 'downloading') {
        setUpdateStatus('downloading');
        setDownloadProgress({
          percent: data.percent || 0,
          currentMB: data.currentMB || 0,
          totalMB: data.totalMB || 0
        });
      } else if (data.status === 'completed') {
        setUpdateStatus('completed');
      } else if (data.status === 'error') {
        setUpdateStatus('error');
        setErrorMessage(data.error || 'Unknown error');
      }
    });

    return () => {
      unsubscribe(); // 僅註銷此監聽器，不使用 EventsOff
    };
  }, []);

  // 3. 一鍵啟動自動更新
  const handleStartUpdate = async () => {
    if (!releaseInfo || !releaseInfo.download_url) return;

    setUpdateStatus('downloading');
    setDownloadProgress({ percent: 0, currentMB: 0, totalMB: 0 });
    setErrorMessage('');

    try {
      await StartAutoUpdate(releaseInfo.download_url, releaseInfo.asset_type);
    } catch (err) {
      setUpdateStatus('error');
      setErrorMessage(String(err));
    }
  };

  // 取得對應語系的更新日誌內容
  const getNotesContent = () => {
    if (!releaseInfo) return '';
    const isZh = lang === 'zh-TW';
    const notes = isZh
      ? (releaseInfo.release_notes_zh || releaseInfo.release_notes)
      : (releaseInfo.release_notes_en || releaseInfo.release_notes);
    return renderMarkdown(notes);
  };

  if (!config) {
    return <div className="p-8 text-center select-none text-xs font-semibold" style={{ color: 'var(--muted)' }}>{t("載入設定中...")}</div>;
  }

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* 標頭 */}
      <div className="p-6 pb-4 flex justify-between items-center select-none shrink-0" style={{ borderBottom: '1px solid var(--border-soft)' }}>
        <div>
          <h1 className="text-xl font-bold tracking-tight" style={{ color: 'var(--fg)' }}>{t("版本更新")}</h1>
          <p className="text-xs mt-1" style={{ color: 'var(--muted)' }}>{t("檢查 WinCMP 最新發布版本，並進行一鍵自動替換升級")}</p>
        </div>
      </div>

      {/* 內容區：單一卡片置中優雅版面 */}
      <div className="flex-1 overflow-y-auto p-6 text-xs flex justify-center" style={{ color: 'var(--muted)' }}>
        <div className="w-full space-y-6">

          {/* 狀態提示 Alert 區 (移至卡片上方) */}
          {!isLoading && releaseInfo && (
            <>
              {!releaseInfo.has_update ? (
                // 已是最新版本狀態
                <div className="rounded-xl p-5 flex items-center gap-3.5" style={{ background: 'color-mix(in srgb, var(--status-ok) 5%, var(--card))', border: '1px solid color-mix(in srgb, var(--status-ok) 20%, transparent)' }}>
                  <div className="p-2.5 rounded-lg shrink-0" style={{ background: 'color-mix(in srgb, var(--status-ok) 20%, transparent)', color: 'var(--status-ok)' }}>
                    <Check size={20} />
                  </div>
                  <div>
                    <h4 className="font-bold text-sm" style={{ color: 'var(--status-ok)' }}>{t("恭喜！您目前使用的是最新版本")}</h4>
                    <p className="text-[10px] mt-1" style={{ color: 'var(--meta)' }}>
                      {t("最新發布版本為 %s，發布時間為 %s", releaseInfo.latest_version, releaseInfo.published_at.substring(0, 10))}
                    </p>
                  </div>
                </div>
              ) : (
                // 偵測到新版本狀態
                <div className="rounded-xl p-5 flex items-center justify-between gap-4" style={{ background: 'color-mix(in srgb, var(--status-info) 5%, var(--card))', border: '1px solid color-mix(in srgb, var(--status-info) 20%, transparent)' }}>
                  <div className="flex items-center gap-3.5">
                    <div className="p-2.5 rounded-lg shrink-0" style={{ background: 'color-mix(in srgb, var(--status-info) 20%, transparent)', color: 'var(--status-info)' }}>
                      <ArrowUpCircle size={20} />
                    </div>
                    <div>
                      <h4 className="font-bold text-sm" style={{ color: 'var(--status-info)' }}>{t("偵測到新版本 %s", releaseInfo.latest_version)}</h4>
                      <p className="text-[10px] mt-1" style={{ color: 'var(--meta)' }}>{t("發布時間")}: {releaseInfo.published_at.substring(0, 10)}</p>
                    </div>
                  </div>

                  {/* 立即更新按鈕 (非下載中) */}
                  {updateStatus === 'idle' && (
                    <button
                      onClick={handleStartUpdate}
                      className="px-5 py-2 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition duration-200 active:scale-[0.98] shrink-0"
                      style={{ background: 'var(--accent)', color: 'var(--accent-on)', boxShadow: 'var(--shadow-md)' }}
                    >
                      <span>{t("立即更新")}</span>
                    </button>
                  )}
                </div>
              )}
            </>
          )}

          {/* 大卡片區塊 */}
          <div className="rounded-xl p-6 space-y-6" style={{ background: 'var(--card)', border: '1px solid var(--border)', boxShadow: 'var(--shadow-lg)' }}>

            {/* 標題與版本對比區 */}
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-5 select-none" style={{ borderBottom: '1px solid var(--border-soft)' }}>
              <div className="flex items-center gap-3">
                <div className="p-2 rounded-lg" style={{ background: 'color-mix(in srgb, var(--status-info) 10%, transparent)', color: 'var(--status-info)' }}>
                  <ArrowUpCircle size={20} />
                </div>
                <div>
                  <h3 className="font-bold text-sm" style={{ color: 'var(--fg)' }}>{t("軟體版本資訊")}</h3>
                  <p className="text-[10px] mt-0.5" style={{ color: 'var(--meta)' }}>{t("管理與安裝最新軟體版本")}</p>
                </div>
              </div>

              <div className="flex items-center gap-3">
                <div className="px-4 py-2 rounded-lg flex flex-col items-center min-w-[100px]" style={{ background: 'var(--bg-deep)', border: '1px solid var(--border)' }}>
                  <span className="text-[9px] font-bold uppercase tracking-wider" style={{ color: 'var(--meta)' }}>{t("當前版本")}</span>
                  <span className="text-sm font-bold mt-0.5" style={{ color: 'var(--fg-2)', fontFamily: 'var(--font-mono)' }}>{currentVersion}</span>
                </div>
                <div className="px-4 py-2 rounded-lg flex flex-col items-center min-w-[100px]" style={{ background: 'var(--bg-deep)', border: '1px solid var(--border)' }}>
                  <span className="text-[9px] font-bold uppercase tracking-wider" style={{ color: 'var(--meta)' }}>{t("最新版本")}</span>
                  <span className="text-sm font-bold mt-0.5" style={{ color: 'var(--fg-2)', fontFamily: 'var(--font-mono)' }}>
                    {isLoading ? '...' : (releaseInfo ? releaseInfo.latest_version : '---')}
                  </span>
                </div>
              </div>
            </div>

            {/* 狀態與更新說明內容 */}
            {isLoading ? (
              // 載入狀態
              <div className="py-12 flex flex-col items-center justify-center text-center space-y-3 select-none">
                <RefreshCw size={24} className="animate-spin" style={{ color: 'var(--status-info)' }} />
                <span className="font-semibold text-xs" style={{ color: 'var(--muted)' }}>{t("正在獲取最新版本資訊...")}</span>
              </div>
            ) : !releaseInfo ? (
              // 獲取失敗狀態
              <div className="rounded-xl p-6 flex flex-col items-center justify-center text-center select-none" style={{ background: 'color-mix(in srgb, var(--status-error) 5%, transparent)', border: '1px solid color-mix(in srgb, var(--status-error) 20%, transparent)' }}>
                <Info size={24} className="mb-2" style={{ color: 'var(--status-error)' }} />
                <span className="font-semibold text-xs" style={{ color: 'var(--status-error)' }}>{t("無法獲取最新版本資訊")}</span>
                <span className="text-[10px] mt-1" style={{ color: 'var(--meta)' }}>{t("請檢查網路連線或稍後再試")}</span>
              </div>
            ) : (
              // 載入完成，顯示對應內容
              <div className="space-y-5">
                {/* 下載中狀態 */}
                {releaseInfo.has_update && updateStatus === 'downloading' && (
                  <div className="rounded-xl p-5 space-y-3" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
                    <div className="flex justify-between items-center text-[10px] font-bold uppercase select-none" style={{ color: 'var(--muted)' }}>
                      <span className="flex items-center gap-1.5">
                        <RefreshCw size={11} className="animate-spin" style={{ color: 'var(--status-info)' }} />
                        {t("更新準備中...")}
                      </span>
                      <span>
                        {downloadProgress.currentMB.toFixed(2)} MB / {downloadProgress.totalMB.toFixed(2)} MB ({Math.round(downloadProgress.percent * 100)}%)
                      </span>
                    </div>
                    <div className="w-full h-2 rounded-full overflow-hidden" style={{ background: 'var(--bg-deep)', border: '1px solid var(--border)' }}>
                      <div
                        className="h-full rounded-full transition-all duration-150 ease-out animate-pulse"
                        style={{ background: 'var(--status-info)', width: `${downloadProgress.percent * 100}%` }}
                      />
                    </div>
                  </div>
                )}

                {/* 更新成功 */}
                {releaseInfo.has_update && updateStatus === 'completed' && (
                  <div className="p-4 rounded-xl font-semibold flex items-center gap-2" style={{ background: 'color-mix(in srgb, var(--status-ok) 10%, transparent)', border: '1px solid color-mix(in srgb, var(--status-ok) 20%, transparent)', color: 'var(--status-ok)' }}>
                    <Check size={16} />
                    <span>{t("更新成功，即將重啟...")}</span>
                  </div>
                )}

                {/* 更新失敗 */}
                {releaseInfo.has_update && updateStatus === 'error' && (
                  <div className="rounded-xl p-5 space-y-4" style={{ background: 'color-mix(in srgb, var(--status-error) 5%, transparent)', border: '1px solid color-mix(in srgb, var(--status-error) 20%, transparent)' }}>
                    <div className="flex flex-col gap-1" style={{ color: 'var(--status-error)' }}>
                      <span className="font-bold text-xs">{t("更新失敗")}</span>
                      <span className="text-[10px] opacity-80" style={{ fontFamily: 'var(--font-mono)' }}>{errorMessage}</span>
                    </div>

                    <div className="text-[11px] leading-relaxed" style={{ color: 'var(--muted)' }}>
                      {t("💡 提示：如果因防毒軟體攔截或系統權限不足導致自動更新失敗，建議您前往 GitHub 手動下載最新版本的 ZIP 壓縮包，解壓覆蓋即可。")}
                    </div>

                    <div className="flex items-center gap-3">
                      <button
                        onClick={handleStartUpdate}
                        className="px-4 py-2 rounded-lg font-semibold transition duration-200 text-xs active:scale-[0.98]"
                        style={{ background: 'var(--card)', border: '1px solid var(--border)', color: 'var(--fg-2)' }}
                      >
                        <span>{t("重試更新")}</span>
                      </button>
                      <button
                        onClick={() => BrowserOpenURL('https://github.com/wukh1124/wincmp/releases/latest')}
                        className="px-4 py-2 rounded-lg font-semibold transition duration-200 text-xs active:scale-[0.98]"
                        style={{ background: 'color-mix(in srgb, var(--status-info) 10%, transparent)', border: '1px solid color-mix(in srgb, var(--status-info) 20%, transparent)', color: 'var(--status-info)' }}
                      >
                        <span>{t("手動下載 ZIP 更新")}</span>
                      </button>
                    </div>
                  </div>
                )}

                {/* SmartScreen 提示 */}
                {releaseInfo.has_update && (
                  <div className="p-3.5 rounded-lg flex gap-3 text-[11px]" style={{ background: 'color-mix(in srgb, var(--status-warn) 5%, transparent)', border: '1px solid color-mix(in srgb, var(--status-warn) 10%, transparent)', color: 'var(--status-warn)' }}>
                    <ShieldAlert size={16} className="shrink-0 mt-0.5 opacity-80" />
                    <div className="leading-normal">
                      {t("更新重啟後若遇到 Windows SmartScreen 警告，請點選「其他資訊」並選擇「仍要執行」。")}
                    </div>
                  </div>
                )}

                {/* 更新說明 */}
                <div className="space-y-2">
                  <div className="text-[10px] font-bold uppercase tracking-wider select-none" style={{ color: 'var(--meta)' }}>
                    {releaseInfo.has_update ? t("新版本更新說明") : t("版本更新說明")}
                  </div>
                  <div
                    className="rounded-lg p-5 leading-normal text-xs"
                    style={{ background: 'var(--bg-deep)', border: '1px solid var(--border)' }}
                    dangerouslySetInnerHTML={{ __html: getNotesContent() || `<span style="color:var(--meta)">${t("無詳細更新說明")}</span>` }}
                  />
                </div>
              </div>
            )}

          </div>

        </div>
      </div>
    </div>
  );
}

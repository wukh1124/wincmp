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
  // 轉義 HTML 特殊字元防止 XSS
  let html = md
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');

  // 標題 1: # Header
  html = html.replace(/^#\s+(.+)$/gm, '<h1 class="text-sm font-bold text-gray-100 mt-3 mb-1.5 border-b border-darkBorder/30 pb-1">$1</h1>');
  // 標題 2: ## Header
  html = html.replace(/^##\s+(.+)$/gm, '<h2 class="text-xs font-bold text-gray-200 mt-2.5 mb-1 flex items-center gap-1">$1</h2>');
  // 標題 3: ### Header
  html = html.replace(/^###\s+(.+)$/gm, '<h3 class="text-[11px] font-bold text-gray-300 mt-2 mb-0.5">$1</h3>');

  // 粗體: **text**
  html = html.replace(/\*\*([^\*]+)\*\*/g, '<strong class="font-bold text-gray-100">$1</strong>');

  // 行內 code `code`
  html = html.replace(/`([^`\n]+)`/g, '<code class="bg-[#18181c] px-1 py-0.5 rounded text-amber-400 font-mono text-[10px]">$1</code>');

  // 無序清單: - list
  html = html.replace(/^[-\*]\s+(.+)$/gm, '<li class="ml-4 list-disc mb-1 text-gray-300 text-[11px]">$1</li>');

  // 段落換行
  html = html.replace(/\n/g, '<br />');

  return html;
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
    return <div className="p-8 text-center text-gray-400 select-none text-xs font-semibold">{t("載入設定中...")}</div>;
  }

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* 標頭 */}
      <div className="p-6 pb-4 flex justify-between items-center select-none border-b border-darkBorder/40 shrink-0">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-white">{t("版本更新")}</h1>
          <p className="text-xs text-gray-400 mt-1">{t("檢查 WinCMP 最新發布版本，並進行一鍵自動替換升級")}</p>
        </div>
      </div>

      {/* 內容區：單一卡片置中優雅版面 */}
      <div className="flex-1 overflow-y-auto p-6 text-xs text-gray-300 flex justify-center">
        <div className="w-full space-y-6">

          {/* 大卡片區塊 */}
          <div className="bg-darkCard border border-darkBorder rounded-xl shadow-lg shadow-black/10 p-6 space-y-6">

            {/* 標題與版本對比區 */}
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-darkBorder/40 pb-5 select-none">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-blue-500/10 rounded-lg text-blue-400">
                  <ArrowUpCircle size={20} />
                </div>
                <div>
                  <h3 className="font-bold text-sm text-gray-200">{t("軟體版本資訊")}</h3>
                  <p className="text-[10px] text-gray-500 mt-0.5">{t("管理與安裝最新軟體版本")}</p>
                </div>
              </div>

              <div className="flex items-center gap-3">
                <div className="bg-[#0c0c0e]/50 px-4 py-2 border border-darkBorder rounded-lg flex flex-col items-center min-w-[100px]">
                  <span className="text-[9px] text-gray-500 font-bold uppercase tracking-wider">{t("當前版本")}</span>
                  <span className="text-sm font-bold text-gray-200 mt-0.5 font-mono">{currentVersion}</span>
                </div>
                <div className="bg-[#0c0c0e]/50 px-4 py-2 border border-darkBorder rounded-lg flex flex-col items-center min-w-[100px]">
                  <span className="text-[9px] text-gray-500 font-bold uppercase tracking-wider">{t("最新版本")}</span>
                  <span className="text-sm font-bold text-gray-200 mt-0.5 font-mono">
                    {isLoading ? '...' : (releaseInfo ? releaseInfo.latest_version : '---')}
                  </span>
                </div>
              </div>
            </div>

            {/* 狀態渲染內容 */}
            {isLoading ? (
              // 載入狀態
              <div className="py-12 flex flex-col items-center justify-center text-center space-y-3 select-none">
                <RefreshCw size={24} className="text-blue-500 animate-spin" />
                <span className="text-gray-400 font-semibold text-xs">{t("正在獲取最新版本資訊...")}</span>
              </div>
            ) : !releaseInfo ? (
              // 獲取失敗狀態
              <div className="bg-red-500/5 border border-red-500/20 rounded-xl p-6 flex flex-col items-center justify-center text-center select-none">
                <Info size={24} className="text-red-400 mb-2" />
                <span className="text-red-400 font-semibold text-xs">{t("無法獲取最新版本資訊")}</span>
                <span className="text-gray-500 text-[10px] mt-1">{t("請檢查網路連線或稍後再試")}</span>
              </div>
            ) : !releaseInfo.has_update ? (
              // 已是最新版本狀態
              <div className="space-y-4">
                <div className="bg-emerald-500/5 border border-emerald-500/20 rounded-xl p-5 flex items-center gap-3.5">
                  <div className="p-2.5 bg-emerald-500/20 rounded-lg text-emerald-400 shrink-0">
                    <Check size={20} />
                  </div>
                  <div>
                    <h4 className="font-bold text-emerald-400 text-sm">{t("恭喜！您目前使用的是最新版本")}</h4>
                    <p className="text-[10px] text-gray-500 mt-1">
                      {t("最新發布版本為 %s，發布時間為 %s", releaseInfo.latest_version, releaseInfo.published_at.substring(0, 10))}
                    </p>
                  </div>
                </div>

                {/* 顯示當前版本的更新說明 */}
                <div className="space-y-2">
                  <div className="text-[10px] text-gray-500 font-bold uppercase tracking-wider select-none">{t("版本更新說明")}</div>
                  <div
                    className="bg-[#0c0c0e]/60 border border-darkBorder rounded-lg p-5 max-h-96 overflow-y-auto leading-relaxed text-[11px]"
                    dangerouslySetInnerHTML={{ __html: getNotesContent() || `<span class="text-gray-500">${t("無詳細更新說明")}</span>` }}
                  />
                </div>
              </div>
            ) : (
              // 偵測到新版本狀態
              <div className="space-y-5">
                <div className="bg-blue-500/5 border border-blue-500/20 rounded-xl p-5 flex items-center justify-between gap-4">
                  <div className="flex items-center gap-3.5">
                    <div className="p-2.5 bg-blue-500/20 rounded-lg text-blue-400 shrink-0">
                      <ArrowUpCircle size={20} />
                    </div>
                    <div>
                      <h4 className="font-bold text-blue-400 text-sm">{t("偵測到新版本 %s", releaseInfo.latest_version)}</h4>
                      <p className="text-[10px] text-gray-500 mt-1">{t("發布時間")}: {releaseInfo.published_at.substring(0, 10)}</p>
                    </div>
                  </div>

                  {/* 立即更新按鈕 (非下載中) */}
                  {updateStatus === 'idle' && (
                    <button
                      onClick={handleStartUpdate}
                      className="px-5 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 transition duration-200 shadow-md shadow-blue-600/10 active:scale-[0.98] shrink-0"
                    >
                      <span>{t("立即更新")}</span>
                    </button>
                  )}
                </div>

                {/* 下載中狀態 */}
                {updateStatus === 'downloading' && (
                  <div className="bg-[#0c0c0e]/30 border border-darkBorder rounded-xl p-5 space-y-3">
                    <div className="flex justify-between items-center text-[10px] font-bold uppercase text-gray-400 select-none">
                      <span className="flex items-center gap-1.5">
                        <RefreshCw size={11} className="animate-spin text-blue-400" />
                        {t("更新準備中...")}
                      </span>
                      <span>
                        {downloadProgress.currentMB.toFixed(2)} MB / {downloadProgress.totalMB.toFixed(2)} MB ({Math.round(downloadProgress.percent * 100)}%)
                      </span>
                    </div>
                    <div className="w-full h-2 bg-[#0c0c0e] rounded-full overflow-hidden border border-darkBorder">
                      <div
                        className="h-full bg-blue-500 rounded-full transition-all duration-150 ease-out animate-pulse"
                        style={{ width: `${downloadProgress.percent * 100}%` }}
                      />
                    </div>
                  </div>
                )}

                {/* 更新成功 */}
                {updateStatus === 'completed' && (
                  <div className="p-4 bg-emerald-500/10 border border-emerald-500/20 rounded-xl text-emerald-400 font-semibold flex items-center gap-2">
                    <Check size={16} />
                    <span>{t("更新成功，即將重啟...")}</span>
                  </div>
                )}

                {/* 更新失敗 */}
                {updateStatus === 'error' && (
                  <div className="bg-red-500/5 border border-red-500/20 rounded-xl p-5 space-y-4">
                    <div className="flex flex-col gap-1 text-red-400">
                      <span className="font-bold text-xs">{t("更新失敗")}</span>
                      <span className="font-mono text-[10px] opacity-80">{errorMessage}</span>
                    </div>
                    
                    <div className="text-[11px] text-gray-400 leading-relaxed">
                      {t("💡 提示：如果因防毒軟體攔截或系統權限不足導致自動更新失敗，建議您前往 GitHub 手動下載最新版本的 ZIP 壓縮包，解壓覆蓋即可。")}
                    </div>

                    <div className="flex items-center gap-3">
                      <button
                        onClick={handleStartUpdate}
                        className="px-4 py-2 bg-darkCard border border-darkBorder hover:border-gray-500 text-gray-200 rounded-lg font-semibold transition duration-200 text-xs active:scale-[0.98]"
                      >
                        <span>{t("重試更新")}</span>
                      </button>
                      <button
                        onClick={() => BrowserOpenURL('https://github.com/wukh1124/wincmp/releases/latest')}
                        className="px-4 py-2 bg-blue-600/10 border border-blue-500/20 hover:border-blue-500 text-blue-400 rounded-lg font-semibold transition duration-200 text-xs active:scale-[0.98]"
                      >
                        <span>{t("手動下載 ZIP 更新")}</span>
                      </button>
                    </div>
                  </div>
                )}

                {/* SmartScreen 提示 */}
                <div className="p-3.5 bg-amber-500/5 border border-amber-500/10 rounded-lg flex gap-3 text-[11px] text-amber-500">
                  <ShieldAlert size={16} className="shrink-0 mt-0.5 opacity-80" />
                  <div className="leading-normal">
                    {t("更新重啟後若遇到 Windows SmartScreen 警告，請點選「其他資訊」並選擇「仍要執行」。")}
                  </div>
                </div>

                {/* Release Notes */}
                <div className="space-y-2">
                  <div className="text-[10px] text-gray-500 font-bold uppercase tracking-wider select-none">{t("新版本更新說明")}</div>
                  <div
                    className="bg-[#0c0c0e]/60 border border-darkBorder rounded-lg p-5 max-h-96 overflow-y-auto leading-relaxed text-[11px]"
                    dangerouslySetInnerHTML={{ __html: getNotesContent() || `<span class="text-gray-500">${t("無詳細更新說明")}</span>` }}
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

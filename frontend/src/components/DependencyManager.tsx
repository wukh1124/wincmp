import React, { useState, useEffect } from 'react';
import {
  X, RefreshCw, Download, ArrowUpCircle, CheckCircle2,
  Loader2, AlertTriangle, Cpu, Database, Settings as SettingsIcon,
  HelpCircle, Server, Terminal, HardDrive
} from 'lucide-react';
import {
  GetDependencyConfig,
  FetchRemoteDependencies,
  DownloadDependency,
  ScanServices,
  GetScanResult
} from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { t, useLanguage } from '../i18n';

interface DependencyItem {
  version: string;
  url: string;
}

type DependencyConfig = Record<string, DependencyItem>;

interface ProgressData {
  status: 'downloading' | 'extracting' | 'completed' | 'error' | 'preparing';
  percent: number;
  currentMB: number;
  totalMB: number;
  error: string;
}

interface DependencyManagerProps {
  isOpen: boolean;
  onClose: () => void;
  onInstalled?: () => void; // 當依賴安裝完成後的通知回呼
}

export default function DependencyManager({ isOpen, onClose, onInstalled }: DependencyManagerProps) {
  useLanguage(); // 訂閱語系變更
  const [depConfig, setDepConfig] = useState<DependencyConfig | null>(null);
  const [scanResult, setScanResult] = useState<any>(null);
  const [isLoadingConfig, setIsLoadingConfig] = useState(false);
  const [isFetchingRemote, setIsFetchingRemote] = useState(false);
  const [progressMap, setProgressMap] = useState<Record<string, ProgressData>>({});

  // 載入依賴配置與本地掃描
  useEffect(() => {
    if (isOpen) {
      loadData();
    }
  }, [isOpen]);

  // 訂閱 Go 端推送的下載進度
  useEffect(() => {
    const handleProgress = (data: any) => {
      if (data && data.key) {
        setProgressMap(prev => ({
          ...prev,
          [data.key]: {
            status: data.status,
            percent: data.percent,
            currentMB: data.currentMB,
            totalMB: data.totalMB,
            error: data.error
          }
        }));

        if (data.status === 'completed') {
          // 下載完成，重新掃描本地環境並觸發回呼
          refreshLocalScan();
          if (onInstalled) {
            onInstalled();
          }
        }
      }
    };

    const unsubscribe = EventsOn('dependency_progress', handleProgress);

    return () => {
      unsubscribe();
    };
  }, [onInstalled]);

  const loadData = async () => {
    setIsLoadingConfig(true);
    try {
      const cfg = await GetDependencyConfig();
      setDepConfig(cfg);
      const scan = await GetScanResult();
      setScanResult(scan);
    } catch (err) {
      console.error("載入依賴資訊失敗:", err);
    } finally {
      setIsLoadingConfig(false);
    }
  };

  const refreshLocalScan = async () => {
    try {
      const scan = await ScanServices();
      setScanResult(scan);
    } catch (err) {
      console.error("刷新服務掃描失敗:", err);
    }
  };

  const handleFetchRemote = async () => {
    setIsFetchingRemote(true);
    try {
      const cfg = await FetchRemoteDependencies();
      setDepConfig(cfg);
      (window as any).customAlert(t("成功從遠端獲取最新的建議依賴配置！"));
    } catch (err) {
      (window as any).customAlert(`${t("獲取遠端依賴配置失敗")}: ${err}`);
    } finally {
      setIsFetchingRemote(false);
    }
  };

  const handleDownload = async (key: string) => {
    // 設為準備狀態
    setProgressMap(prev => ({
      ...prev,
      [key]: {
        status: 'preparing',
        percent: 0,
        currentMB: 0,
        totalMB: 0,
        error: ''
      }
    }));

    try {
      await DownloadDependency(key);
    } catch (err: any) {
      setProgressMap(prev => ({
        ...prev,
        [key]: {
          status: 'error',
          percent: 0,
          currentMB: 0,
          totalMB: 0,
          error: err.toString()
        }
      }));
    }
  };

  // 版本比較邏輯
  const compareVersions = (v1: string, v2: string) => {
    if (!v1) return -1;
    if (!v2) return 1;
    const clean = (v: string) => v.replace(/^v/, '').split('-')[0];
    const p1 = clean(v1).split('.');
    const p2 = clean(v2).split('.');
    for (let i = 0; i < Math.max(p1.length, p2.length); i++) {
      const n1 = parseInt(p1[i] || '0', 10);
      const n2 = parseInt(p2[i] || '0', 10);
      if (n1 < n2) return -1;
      if (n1 > n2) return 1;
    }
    return 0;
  };

  // 獲取本機已安裝的版本
  const getInstalledVersion = (key: string): string => {
    if (!scanResult) return '';
    if (key === 'caddy') return scanResult.CaddyList?.[0]?.Version || '';
    if (key === 'mariadb') return scanResult.MariaDBList?.[0]?.Version || '';
    if (key === 'composer') return scanResult.ComposerList?.[0]?.Version || '';
    if (key === 'heidisql') return scanResult.HeidiSQLList?.[0]?.Version || '';
    if (key === 'node') return scanResult.NodeList?.[0]?.Version || '';
    if (key === 'mailpit') return scanResult.MailpitList?.[0]?.Version || '';

    if (key.startsWith('php')) {
      // php82 -> 8.2
      const majorMin = key.replace('php', '').replace(/(\d)(\d)/, '$1.$2');
      const phpItem = scanResult.PHPList?.find((p: any) => p.MajorMin === majorMin);
      return phpItem?.Version || '';
    }

    return '';
  };

  if (!isOpen) return null;

  // 過濾並排序 PHP 版本
  const phpKeys = depConfig
    ? Object.keys(depConfig)
      .filter(k => k.startsWith('php') && k !== 'php')
      .sort((a, b) => {
        const vA = depConfig[a].version;
        const vB = depConfig[b].version;
        return compareVersions(vB, vA); // 降序
      })
    : [];

  const renderDependencyRow = (key: string, label: string, icon: React.ReactNode) => {
    if (!depConfig) return null;
    const spec = depConfig[key];
    if (!spec) return null;

    const localVer = getInstalledVersion(key);
    const recVer = spec.version;
    const progress = progressMap[key];

    let statusText = '';
    let statusColor = 'text-gray-400';
    let showBtn = true;
    let btnText = t('下載安裝');
    let btnTheme = 'bg-blue-600 hover:bg-blue-700 text-white';
    let btnIcon = <Download size={13} />;

    if (localVer === '') {
      statusText = `${t("未安裝")} (${t("建議")}: v${recVer})`;
      statusColor = 'text-red-400';
      btnText = t('下載');
      btnTheme = 'bg-blue-600 hover:bg-blue-700 text-white shadow-md shadow-blue-500/10';
    } else {
      const cmp = compareVersions(localVer, recVer);
      if (cmp < 0) {
        statusText = `${t("已安裝")}: v${localVer} (${t("有新版")}: v${recVer})`;
        statusColor = 'text-amber-400';
        btnText = t('更新');
        btnTheme = 'bg-amber-600 hover:bg-amber-700 text-white';
        btnIcon = <ArrowUpCircle size={13} />;
      } else {
        statusText = `${t("已安裝")}: v${localVer} (${t("最新")})`;
        statusColor = 'text-green-400';
        btnText = t('重裝');
        btnTheme = 'bg-darkBorder hover:bg-opacity-80 text-gray-300';
      }
    }

    // 處理下載/解壓中狀態
    if (progress) {
      const { status, percent, currentMB, totalMB, error } = progress;
      if (status === 'preparing') {
        return (
          <div className="flex flex-col gap-2 p-3 bg-darkBg bg-opacity-30 rounded-lg border border-darkBorder">
            <div className="flex justify-between items-center text-xs">
              <span className="font-semibold text-gray-200 flex items-center gap-2">
                {icon} {t(label)}
              </span>
              <span className="text-blue-400 flex items-center gap-1.5 animate-pulse">
                <Loader2 size={12} className="animate-spin" /> {t("準備下載環境...")}
              </span>
            </div>
          </div>
        );
      }
      if (status === 'downloading') {
        const pct = Math.round(percent * 100);
        return (
          <div className="flex flex-col gap-2 p-3 bg-darkBg bg-opacity-30 rounded-lg border border-blue-500/20">
            <div className="flex justify-between items-center text-xs">
              <span className="font-semibold text-gray-200 flex items-center gap-2">
                {icon} {t(label)}
              </span>
              <span className="text-blue-400 flex items-center gap-1">
                {t("下載中")}... {currentMB.toFixed(1)}MB / {totalMB > 0 ? `${totalMB.toFixed(1)}MB` : '--'} ({pct}%)
              </span>
            </div>
            <div className="w-full h-1.5 bg-darkInput rounded-full overflow-hidden">
              <div style={{ width: `${pct}%` }} className="h-full bg-blue-500 rounded-full transition-all duration-300" />
            </div>
          </div>
        );
      }
      if (status === 'extracting') {
        return (
          <div className="flex flex-col gap-2 p-3 bg-darkBg bg-opacity-30 rounded-lg border border-teal-500/20">
            <div className="flex justify-between items-center text-xs">
              <span className="font-semibold text-gray-200 flex items-center gap-2">
                {icon} {t(label)}
              </span>
              <span className="text-teal-400 flex items-center gap-1.5 animate-pulse">
                <Loader2 size={12} className="animate-spin" /> {t("正在解壓縮並配置...")}
              </span>
            </div>
            <div className="w-full h-1.5 bg-darkInput rounded-full overflow-hidden">
              <div className="h-full bg-teal-400 rounded-full animate-pulse" style={{ width: '70%' }} />
            </div>
          </div>
        );
      }
      if (status === 'completed') {
        statusText = `${t("安裝成功")}: v${recVer}`;
        statusColor = 'text-green-400';
        showBtn = false;
      }
      if (status === 'error') {
        return (
          <div className="flex flex-col gap-2 p-3 bg-red-950/20 border border-red-500/30 rounded-lg">
            <div className="flex justify-between items-center text-xs">
              <span className="font-semibold text-red-300 flex items-center gap-2">
                {icon} {t(label)}
              </span>
              <span className="text-red-400 flex items-center gap-1">
                <AlertTriangle size={12} /> {t("安裝失敗")}
              </span>
            </div>
            <p className="text-[11px] text-red-300/80 font-mono mt-1 break-all bg-red-950/40 p-1.5 rounded">{error}</p>
            <button
              onClick={() => handleDownload(key)}
              className="mt-1 text-center py-1 bg-red-900/40 hover:bg-red-900/60 text-red-300 rounded text-xs transition"
            >
              {t("重試安裝")}
            </button>
          </div>
        );
      }
    }

    return (
      <div className="flex items-center justify-between py-2.5 hover:bg-darkBg hover:bg-opacity-10 px-2 rounded-lg transition duration-150">
        <div className="flex items-center gap-3">
          <span className="text-gray-400">{icon}</span>
          <div>
            <span className="text-sm font-semibold text-gray-200 block">{t(label)}</span>
            <span className={`text-xs ${statusColor} mt-0.5 block font-medium`}>{statusText}</span>
          </div>
        </div>
        {showBtn && (
          <button
            onClick={() => handleDownload(key)}
            className={`px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition ${btnTheme}`}
          >
            {btnIcon}
            <span>{btnText}</span>
          </button>
        )}
      </div>
    );
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* 遮罩背景 */}
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />

      {/* 彈出視窗主體 */}
      <div className="relative w-full max-w-2xl max-h-[90vh] bg-[#1a1a26] border border-darkBorder rounded-2xl flex flex-col shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-200 text-gray-200 select-none">

        {/* 標頭 */}
        <div className="px-6 py-4 border-b border-darkBorder flex items-center justify-between bg-darkCard select-none">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-blue-500/10 flex items-center justify-center text-blue-400">
              <HardDrive size={18} />
            </div>
            <div>
              <h2 className="text-lg font-bold tracking-wide">📦 {t("WinCMP 依賴庫管理")}</h2>
              <p className="text-xs text-gray-400 mt-0.5">{t("下載或升級本機 Web 開發依賴")}</p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={handleFetchRemote}
              disabled={isFetchingRemote}
              className="p-2 hover:bg-darkBorder rounded-lg transition text-gray-400 hover:text-gray-200 flex items-center gap-1 text-xs font-semibold"
              title={t("從遠端獲取最新版本")}
            >
              <RefreshCw size={14} className={isFetchingRemote ? 'animate-spin' : ''} />
              <span>{isFetchingRemote ? t('獲取中...') : t('獲取最新')}</span>
            </button>
            <button
              onClick={onClose}
              className="p-1.5 hover:bg-darkBorder rounded-lg transition text-gray-400 hover:text-gray-200"
            >
              <X size={18} />
            </button>
          </div>
        </div>

        {/* 內容區 */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          {isLoadingConfig ? (
            <div className="py-20 flex flex-col items-center justify-center text-gray-400 gap-3">
              <Loader2 size={32} className="animate-spin text-blue-500" />
              <span className="text-sm font-semibold">{t("正在讀取依賴設定...")}</span>
            </div>
          ) : (
            <>
              {/* 1. 核心依賴 */}
              <div className="bg-darkCard bg-opacity-40 border border-darkBorder rounded-xl p-4 space-y-3">
                <h3 className="text-xs text-blue-400 font-bold uppercase tracking-wider flex items-center gap-1.5 select-none border-b border-darkBorder border-opacity-60 pb-2">
                  <Cpu size={13} /> {t("核心執行環境")}
                </h3>
                <div className="divide-y divide-darkBorder divide-opacity-30">
                  {renderDependencyRow('caddy', 'Caddy Web 伺服器', <Server size={16} />)}
                  {renderDependencyRow('mariadb', 'MariaDB 資料庫', <Database size={16} />)}
                </div>
              </div>

              {/* 2. PHP 多版本環境 */}
              <div className="bg-darkCard bg-opacity-40 border border-darkBorder rounded-xl p-4 space-y-3">
                <h3 className="text-xs text-green-400 font-bold uppercase tracking-wider flex items-center gap-1.5 select-none border-b border-darkBorder border-opacity-60 pb-2">
                  <Server size={13} /> {t("PHP FastCGI 環境")}
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {phpKeys.map(key => {
                    const majorMin = key.replace('php', '').replace(/(\d)(\d)/, '$1.$2');
                    return (
                      <div key={key} className="border border-darkBorder border-opacity-40 rounded-lg p-2.5 bg-darkBg bg-opacity-10">
                        {renderDependencyRow(key, `PHP ${majorMin} NTS`, <Terminal size={14} />)}
                      </div>
                    );
                  })}
                </div>
              </div>

              {/* 3. 其他常用工具 */}
              <div className="bg-darkCard bg-opacity-40 border border-darkBorder rounded-xl p-4 space-y-3">
                <h3 className="text-xs text-purple-400 font-bold uppercase tracking-wider flex items-center gap-1.5 select-none border-b border-darkBorder border-opacity-60 pb-2">
                  <SettingsIcon size={13} /> {t("開發輔助工具與實用工具")}
                </h3>
                <div className="divide-y divide-darkBorder divide-opacity-30">
                  {renderDependencyRow('composer', 'Composer (PHP 包管理器)', <Terminal size={16} />)}
                  {renderDependencyRow('node', 'Node.js LTS 運行環境', <Terminal size={16} />)}
                  {renderDependencyRow('mailpit', 'Mailpit 郵件測試伺服器', <Server size={16} />)}
                  {renderDependencyRow('heidisql', 'HeidiSQL 資料庫 GUI 工具', <Database size={16} />)}
                </div>
              </div>
            </>
          )}
        </div>

        {/* 底部說明 */}
        <div className="px-6 py-4 border-t border-darkBorder bg-darkCard text-[11px] text-gray-500 flex justify-between items-center select-none">
          <span>{t("提示：安裝完成後系統會自動重新掃描環境。")}</span>
          <span className="font-mono">Downloader Pipeline</span>
        </div>

      </div>
    </div>
  );
}

import React, { useState, useEffect, useRef } from 'react';
import {
  X, RefreshCw, Download, ArrowUpCircle, CheckCircle2,
  Loader2, AlertTriangle, Cpu, Database, Settings as SettingsIcon,
  HelpCircle, Server, Terminal, HardDrive, Zap, ChevronDown,
  Trash2, RotateCw, FolderOpen, Copy, Check, ExternalLink, Scale
} from 'lucide-react';
import {
  GetDependencyConfig, FetchRemoteDependencies, DownloadDependency,
  ScanServices, GetScanResult, UninstallDependency, OpenDependencyFolder
} from '../../wailsjs/go/main/App';
import { EventsOn, BrowserOpenURL } from '../../wailsjs/runtime/runtime';
import { t, useLanguage } from '../i18n';

interface DependencyItem {
  version: string;
  url: string;
  sha256?: string;
  license?: string;
  homepage?: string;
  source_url?: string;
}
type DependencyConfig = Record<string, DependencyItem>;
interface ProgressData { status: 'downloading' | 'extracting' | 'completed' | 'error' | 'preparing'; percent: number; currentMB: number; totalMB: number; error: string; }
interface DependencyManagerProps { isOpen: boolean; onClose: () => void; onInstalled?: () => void; }

// 模組級背景檢查更新冷卻時間（60 秒防抖，避免頻繁開關彈窗重複向遠端請求）
let lastRemoteCheckTime = 0;
const REMOTE_CHECK_COOLDOWN_MS = 60 * 1000;

export default function DependencyManager({ isOpen, onClose, onInstalled }: DependencyManagerProps) {
  useLanguage();
  const [depConfig, setDepConfig] = useState<DependencyConfig | null>(null);
  const [scanResult, setScanResult] = useState<any>(null);
  const [isLoadingConfig, setIsLoadingConfig] = useState(false);
  const [isManualChecking, setIsManualChecking] = useState(false);
  const [showNoticeModal, setShowNoticeModal] = useState(false);
  const [isContentUpdating, setIsContentUpdating] = useState(false);
  const [progressMap, setProgressMap] = useState<Record<string, ProgressData>>({});
  const [activeDropdown, setActiveDropdown] = useState<string | null>(null);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const depConfigRef = useRef<DependencyConfig | null>(null);

  useEffect(() => {
    depConfigRef.current = depConfig;
  }, [depConfig]);

  const loadData = async () => {
    setIsLoadingConfig(true);
    try { setDepConfig(await GetDependencyConfig()); setScanResult(await GetScanResult()); }
    catch (err) { console.error("載入依賴資訊失敗:", err); }
    finally { setIsLoadingConfig(false); }
  };

  const refreshLocalScan = async () => {
    try { setScanResult(await ScanServices()); } catch (err) { console.error("刷新服務掃描失敗:", err); }
  };

  const isConfigDifferent = (oldCfg: DependencyConfig | null, newCfg: DependencyConfig | null): boolean => {
    if (!oldCfg || !newCfg) return false;
    const oldKeys = Object.keys(oldCfg).sort();
    const newKeys = Object.keys(newCfg).sort();
    if (oldKeys.length !== newKeys.length) return true;
    for (let i = 0; i < oldKeys.length; i++) {
      if (oldKeys[i] !== newKeys[i]) return true;
      if (oldCfg[oldKeys[i]]?.version !== newCfg[newKeys[i]]?.version) return true;
      if (oldCfg[oldKeys[i]]?.url !== newCfg[newKeys[i]]?.url) return true;
    }
    return false;
  };

  const fetchRemoteConfig = async (isManual = false) => {
    if (isManual) {
      setIsManualChecking(true);
    }
    const startTime = Date.now();
    try {
      const newConfig = await FetchRemoteDependencies();
      const oldConfig = depConfigRef.current;
      const hasChanges = isConfigDifferent(oldConfig, newConfig);

      if (hasChanges && oldConfig) {
        setIsContentUpdating(true);
        await new Promise(r => setTimeout(r, 150));
        setDepConfig(newConfig);
        await new Promise(r => setTimeout(r, 50));
        setIsContentUpdating(false);
      } else {
        setDepConfig(newConfig);
      }

      await refreshLocalScan();
      if (onInstalled) onInstalled();
    } catch (err) {
      console.error("獲取遠端依賴配置失敗:", err);
      if (isManual) {
        (window as any).customAlert(`${t("獲取遠端依賴配置失敗")}: ${err}`);
      }
    } finally {
      if (isManual) {
        const elapsed = Date.now() - startTime;
        if (elapsed < 500) {
          await new Promise(resolve => setTimeout(resolve, 500 - elapsed));
        }
        setIsManualChecking(false);
      }
    }
  };

  const handleFetchRemote = () => {
    lastRemoteCheckTime = Date.now();
    fetchRemoteConfig(true);
  };

  useEffect(() => {
    if (isOpen) {
      loadData();
      // 清除先前的錯誤與非進行中狀態，避免重開視窗時卡在舊的失敗畫面
      setProgressMap(prev => {
        const next: Record<string, ProgressData> = {};
        for (const [k, v] of Object.entries(prev)) {
          if (v.status === 'downloading' || v.status === 'extracting' || v.status === 'preparing') {
            next[k] = v;
          }
        }
        return next;
      });
      // 開啟依賴庫管理時，若距上次檢查超過 60 秒冷卻時間，才在背景檢查更新
      const now = Date.now();
      if (now - lastRemoteCheckTime >= REMOTE_CHECK_COOLDOWN_MS) {
        lastRemoteCheckTime = now;
        fetchRemoteConfig(false);
      }
    }
  }, [isOpen]);

  useEffect(() => {
    const handleGlobalClick = (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      if (!target.closest('.split-btn-dropdown')) {
        setActiveDropdown(null);
      }
    };
    window.addEventListener('click', handleGlobalClick);
    return () => window.removeEventListener('click', handleGlobalClick);
  }, []);

  useEffect(() => {
    const handleProgress = (data: any) => {
      if (data && data.key) {
        setProgressMap(prev => ({ ...prev, [data.key]: { status: data.status, percent: data.percent, currentMB: data.currentMB, totalMB: data.totalMB, error: data.error } }));
        if (data.status === 'completed') {
          refreshLocalScan();
          if (onInstalled) onInstalled();
          // 4 秒後自動清除進度狀態，自然回歸至已安裝常態展示
          setTimeout(() => {
            setProgressMap(prev => {
              if (prev[data.key]?.status === 'completed') {
                const copy = { ...prev };
                delete copy[data.key];
                return copy;
              }
              return prev;
            });
          }, 4000);
        }
      }
    };
    const unsubscribe = EventsOn('dependency_progress', handleProgress);
    return () => { unsubscribe(); };
  }, [onInstalled]);

  const handleDownload = async (key: string) => {
    setActiveDropdown(null);
    setProgressMap(prev => ({ ...prev, [key]: { status: 'preparing', percent: 0, currentMB: 0, totalMB: 0, error: '' } }));
    try { await DownloadDependency(key); }
    catch (err: any) { setProgressMap(prev => ({ ...prev, [key]: { status: 'error', percent: 0, currentMB: 0, totalMB: 0, error: err.toString() } })); }
  };

  const handleUninstall = async (key: string, label: string) => {
    setActiveDropdown(null);
    const confirmed = await (window as any).customConfirm(
      t("確定要移除 %s 嗎？移除後將刪除其二進位檔案。", label)
    );
    if (!confirmed) return;

    setProgressMap(prev => ({ ...prev, [key]: { status: 'preparing', percent: 0, currentMB: 0, totalMB: 0, error: '' } }));
    try {
      await UninstallDependency(key);
      await refreshLocalScan();
      if (onInstalled) onInstalled();
      (window as any).customAlert(t("移除成功"));
    } catch (err: any) {
      console.error("移除依賴失敗:", err);
      (window as any).customAlert(`${t("移除失敗")}: ${err}`);
    } finally {
      setProgressMap(prev => {
        const copy = { ...prev };
        delete copy[key];
        return copy;
      });
    }
  };

  const handleInstallRedisExt = async (phpKey: string) => {
    setActiveDropdown(null);
    const verSuffix = phpKey.replace(/^php_?/, '');
    const redisKey = 'php_redis_' + verSuffix;
    await handleDownload(redisKey);
  };

  const handleOpenFolder = async (key: string) => {
    setActiveDropdown(null);
    try {
      await OpenDependencyFolder(key);
    } catch (err: any) {
      console.error("開啟安裝目錄失敗:", err);
      (window as any).customAlert(`${t("開啟安裝目錄失敗")}: ${err}`);
    }
  };

  const handleCopyUrl = async (url: string, key: string) => {
    try {
      if (navigator?.clipboard?.writeText) {
        await navigator.clipboard.writeText(url);
      } else if ((window as any).runtime?.ClipboardSetText) {
        await (window as any).runtime.ClipboardSetText(url);
      }
      setCopiedKey(key);
      setTimeout(() => {
        setCopiedKey(null);
      }, 1800);
    } catch (err) {
      console.error("複製下載網址失敗:", err);
    }
  };

  const formatErrorMessage = (errStr: string): string => {
    if (!errStr) return '';
    const lines = errStr.split('\n');
    const prefixRules: Array<[string, string]> = [
      ['建議排查指引：', t('建議排查指引：')],
      ['1. 依賴目錄來源：', t('1. 依賴目錄來源：')],
      ['2. 可能原因：官方目錄尚未發布該版本，或本地設定檔缺少安全雜湊值。', t('2. 可能原因：官方目錄尚未發布該版本，或本地設定檔缺少安全雜湊值。')],
      ['3. 建議操作：請嘗試在右上角點擊「檢查更新」同步最新設定；若為開發測試環境，請確認 conf/dependencies.json 是否已填入正確的 sha256。', t('3. 建議操作：請嘗試在右上角點擊「檢查更新」同步最新設定；若為開發測試環境，請確認 conf/dependencies.json 是否已填入正確的 sha256。')],
      ['1. 遠端配置來源 (DependencyURL)：', t('1. 遠端配置來源 (DependencyURL)：')],
      ['1. 遠端配置來源：', t('1. 遠端配置來源：')],
      ['2. 可能原因：遠端建議設定檔尚未發布該項目版本、SHA-256 遺漏或本地配置缺少雜湊值。', t('2. 可能原因：遠端建議設定檔尚未發布該項目版本、SHA-256 遺漏或本地配置缺少雜湊值。')],
      ['3. 請嘗試在依賴管理面板點擊「獲取最新」同步設定；若為自訂/測試環境，請確認倉庫分支或本地 dependencies.json 是否已填入正確的 sha256。', t('3. 請嘗試在依賴管理面板點擊「獲取最新」同步設定；若為自訂/測試環境，請確認倉庫分支或本地 dependencies.json 是否已填入正確的 sha256。')],
      ['2. 請檢查您的網路連線或代理設定後重試。', t('2. 請檢查您的網路連線或代理設定後重試。')],
      ['3. 若持續失敗，可手動下載：', t('3. 若持續失敗，可手動下載：')],
      ['4. 並解壓放置於以下 bin 目錄位置：', t('4. 並解壓放置於以下 bin 目錄位置：')],
      ['2. 請先嘗試在右上角點擊「檢查更新」取得最新校驗資訊，然後重試下載。', t('2. 請先嘗試在右上角點擊「檢查更新」取得最新校驗資訊，然後重試下載。')],
      ['2. 請先嘗試在依賴管理面板點擊「獲取最新」，然後重試下載。', t('2. 請先嘗試在依賴管理面板點擊「獲取最新」，然後重試下載。')],
      ['3. 若問題持續，請手動下載：', t('3. 若問題持續，請手動下載：')],
      ['建議指引與診斷資訊：', t('建議指引與診斷資訊：')],
      ['建議指引：', t('建議指引：')],
      ['檔案可能損毀。請手動下載：', t('檔案可能損毀。請手動下載：')],
      ['並解壓放置於以下目錄：', t('並解壓放置於以下目錄：')],
      ['依賴項目未提供 SHA-256 校驗碼，基於安全考量拒絕下載', t('依賴項目未提供 SHA-256 校驗碼，基於安全考量拒絕下載')],
      ['SHA-256 完整性校驗失敗！下載的檔案可能損毀、不完整或遭受中間人篡改。', t('SHA-256 完整性校驗失敗！下載的檔案可能損毀、不完整或遭受中間人篡改。')],
    ];

    return lines.map(line => {
      const trimmed = line.trim();
      if (!trimmed) return line;
      const translated = t(trimmed);
      if (translated !== trimmed) {
        return line.replace(trimmed, translated);
      }
      for (const [zhPrefix, targetPrefix] of prefixRules) {
        if (line.includes(zhPrefix)) {
          return line.replace(zhPrefix, targetPrefix);
        }
      }
      return line;
    }).join('\n');
  };

  const compareVersions = (v1: string, v2: string) => {
    if (!v1) return -1; if (!v2) return 1;
    const clean = (v: string) => v.replace(/^v/, '').split('-')[0];
    const p1 = clean(v1).split('.'); const p2 = clean(v2).split('.');
    for (let i = 0; i < Math.max(p1.length, p2.length); i++) {
      const n1 = parseInt(p1[i] || '0', 10); const n2 = parseInt(p2[i] || '0', 10);
      if (n1 < n2) return -1; if (n1 > n2) return 1;
    }
    return 0;
  };

  const getInstalledVersion = (key: string): string => {
    if (!scanResult) return '';
    if (key === 'caddy') return scanResult.CaddyList?.[0]?.Version || '';
    if (key === 'mariadb') return scanResult.MariaDBList?.[0]?.Version || '';
    if (key === 'redis') return scanResult.RedisList?.[0]?.Version || '';
    if (key === 'composer') return scanResult.ComposerList?.[0]?.Version || '';
    if (key === 'heidisql') return scanResult.HeidiSQLList?.[0]?.Version || '';
    if (key === 'node') return scanResult.NodeList?.[0]?.Version || '';
    if (key === 'mailpit') return scanResult.MailpitList?.[0]?.Version || '';
    if (key.startsWith('php_redis_')) {
      const verSuffix = key.replace('php_redis_', '');
      const majorMin = verSuffix.replace(/(\d)(\d)/, '$1.$2');
      const phpInfo = scanResult.PHPList?.find((p: any) => p.MajorMin === majorMin);
      if (!phpInfo) return 'PHP_NOT_INSTALLED';
      const hasRedis = phpInfo.Extensions?.some((ext: string) => ext.toLowerCase() === 'php_redis.dll');
      return hasRedis ? (depConfig?.[key]?.version || 'installed') : '';
    }
    if (key.startsWith('php_')) {
      const verSuffix = key.replace('php_', '');
      const majorMin = verSuffix.replace(/(\d)(\d)/, '$1.$2');
      return scanResult.PHPList?.find((p: any) => p.MajorMin === majorMin)?.Version || '';
    }
    if (key.startsWith('php') && !key.startsWith('php_redis')) {
      const majorMin = key.replace('php', '').replace(/(\d)(\d)/, '$1.$2');
      return scanResult.PHPList?.find((p: any) => p.MajorMin === majorMin)?.Version || '';
    }
    return '';
  };

  const getPhpRedisStatus = (phpKey: string): { supported: boolean; installed: boolean; iniEnabled: boolean; version: string } => {
    if (!scanResult) return { supported: false, installed: false, iniEnabled: false, version: '' };
    const verSuffix = phpKey.replace(/^php_?/, '');
    const majorMin = verSuffix.replace(/(\d)(\d)/, '$1.$2');
    const phpInfo = scanResult.PHPList?.find((p: any) => p.MajorMin === majorMin);
    if (!phpInfo) return { supported: false, installed: false, iniEnabled: false, version: '' };
    const hasRedis = phpInfo.Extensions?.some((ext: string) => ext.toLowerCase() === 'php_redis.dll');
    const iniEnabled = !!(phpInfo as any).IniRedisEnabled;
    const redisKey = 'php_redis_' + verSuffix;
    const isSupported = !!depConfig?.[redisKey];
    const recVer = depConfig?.[redisKey]?.version || '';
    return { supported: isSupported, installed: !!hasRedis, iniEnabled, version: recVer };
  };

  const handleEnablePHPRedis = async () => {
    try {
      if ((window as any).go?.main?.App?.EnablePHPRedisExtension) {
        await (window as any).go.main.App.EnablePHPRedisExtension();
      }
      await (window as any).customAlert(t("已成功平滑啟用 php.ini 中的 Redis 擴充！若 PHP 已在運行，請重啟 PHP 服務以生效。"));
      setScanResult(await ScanServices());
    } catch (err: any) {
      (window as any).customAlert(`${t("啟用失敗")}: ${err}`);
    }
  };

  if (!isOpen) return null;

  const phpKeys = depConfig
    ? Object.keys(depConfig)
      .filter(k => (k.startsWith('php_') || k.startsWith('php')) && !k.startsWith('php_redis') && k !== 'php')
      .sort((a, b) => compareVersions(depConfig[b].version, depConfig[a].version))
    : [];

  // ─── Styles ─────────────────────────────────────────────
  const cardStyle: React.CSSProperties = { background: 'var(--card)', border: '1px solid var(--border)', borderRadius: 'var(--radius-lg)', padding: 16 };

  const renderDependencyRow = (key: string, label: string, icon: React.ReactNode) => {
    if (!depConfig) return null;
    const spec = depConfig[key];
    if (!spec) return null;
    const localVer = getInstalledVersion(key);
    const recVer = spec.version;
    const progress = progressMap[key];

    const isPhp = (key.startsWith('php_') || key.startsWith('php')) && !key.startsWith('php_redis');
    const phpRedisInfo = isPhp ? getPhpRedisStatus(key) : null;
    const isDropdownOpen = activeDropdown === key;

    let statusText = '';
    let statusColor: React.CSSProperties['color'] = 'var(--muted)';
    let showBtn = true;
    let btnDisabled = false;
    let btnText = t('下載安裝');
    let btnStyle: React.CSSProperties = { background: 'var(--status-info)', color: '#fff' };
    let btnIcon: React.ReactNode = <Download size={13} />;

    const cmp = localVer !== '' ? compareVersions(localVer, recVer) : 0;
    const isUpdate = localVer !== '' && cmp < 0;

    if (localVer === '') {
      statusText = `${t("未安裝")} (${t("建議")}: v${recVer})`;
      statusColor = 'var(--status-error)';
      btnText = t('下載');
    } else if (isUpdate) {
      statusText = `${t("已安裝")}: v${localVer} (${t("有新版")}: v${recVer})`;
      statusColor = 'var(--status-warn)';
      btnText = t('更新');
      btnStyle = { background: 'var(--status-warn)', color: '#fff' };
      btnIcon = <ArrowUpCircle size={13} />;
    } else {
      statusText = `${t("已安裝")}: v${localVer} (${t("最新")})`;
      statusColor = 'var(--status-ok)';
      btnText = t('重裝');
      btnStyle = { background: 'var(--card)', border: '1px solid var(--border)', color: 'var(--fg-2)' };
      btnIcon = <RotateCw size={12} style={{ color: 'var(--status-info)' }} />;
    }

    if (progress) {
      const { status, percent, currentMB, totalMB, error } = progress;
      if (status === 'preparing') {
        return (
          <div className="flex flex-col gap-2 p-3 rounded-lg" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
            <div className="flex justify-between items-center text-xs">
              <span className="font-semibold flex items-center gap-2" style={{ color: 'var(--fg)' }}>{icon} {t(label)}</span>
              <span className="flex items-center gap-1.5 animate-pulse" style={{ color: 'var(--status-info)' }}>
                <Loader2 size={12} className="animate-spin" /> {t("準備下載環境...")}
              </span>
            </div>
          </div>
        );
      }
      if (status === 'downloading') {
        const pct = Math.round(percent * 100);
        return (
          <div className="flex flex-col gap-2 p-3 rounded-lg" style={{ background: 'var(--surface)', border: '1px solid var(--status-info-bg)' }}>
            <div className="flex justify-between items-center text-xs">
              <span className="font-semibold flex items-center gap-2" style={{ color: 'var(--fg)' }}>{icon} {t(label)}</span>
              <span className="flex items-center gap-1" style={{ color: 'var(--status-info)' }}>
                {t("下載中")}... {currentMB.toFixed(1)}MB / {totalMB > 0 ? `${totalMB.toFixed(1)}MB` : '--'} ({pct}%)
              </span>
            </div>
            <div className="w-full h-1.5 rounded-full overflow-hidden" style={{ background: 'var(--input-bg)' }}>
              <div style={{ width: `${pct}%`, background: 'var(--status-info)' }} className="h-full rounded-full transition-all duration-300" />
            </div>
          </div>
        );
      }
      if (status === 'extracting') {
        return (
          <div className="flex flex-col gap-2 p-3 rounded-lg" style={{ background: 'var(--surface)', border: '1px solid var(--status-ok-bg)' }}>
            <div className="flex justify-between items-center text-xs">
              <span className="font-semibold flex items-center gap-2" style={{ color: 'var(--fg)' }}>{icon} {t(label)}</span>
              <span className="flex items-center gap-1.5 animate-pulse" style={{ color: 'var(--status-ok)' }}>
                <Loader2 size={12} className="animate-spin" /> {t("正在解壓縮並配置...")}
              </span>
            </div>
            <div className="w-full h-1.5 rounded-full overflow-hidden" style={{ background: 'var(--input-bg)' }}>
              <div className="h-full rounded-full animate-pulse" style={{ width: '70%', background: 'var(--status-ok)' }} />
            </div>
          </div>
        );
      }
      if (status === 'completed') {
        statusText = `${t("安裝成功")}: v${recVer}`;
        statusColor = 'var(--status-ok)';
      }
      if (status === 'error') {
        return (
          <div className="flex flex-col gap-2 p-3 rounded-lg" style={{ background: 'var(--status-error-bg)', border: '1px solid var(--status-error)' }}>
            <div className="flex justify-between items-center text-xs">
              <span className="font-semibold flex items-center gap-2" style={{ color: 'var(--status-error)' }}>{icon} {t(label)}</span>
              <div className="flex items-center gap-2">
                <span className="flex items-center gap-1" style={{ color: 'var(--status-error)' }}>
                  <AlertTriangle size={12} /> {t("安裝失敗")}
                </span>
                <button
                  onClick={() => {
                    setProgressMap(prev => {
                      const copy = { ...prev };
                      delete copy[key];
                      return copy;
                    });
                  }}
                  title={t("清除錯誤")}
                  className="p-1 rounded hover:bg-black/10 transition"
                  style={{ color: 'var(--status-error)' }}
                >
                  <X size={13} />
                </button>
              </div>
            </div>
            <p className="text-[11px] mt-1 break-all p-2 rounded select-text" style={{ color: 'var(--fg-2)', background: 'var(--surface)', fontFamily: 'var(--font-mono)', whiteSpace: 'pre-wrap', userSelect: 'text', WebkitUserSelect: 'text' }}>
              {formatErrorMessage(error)}
            </p>
            <div className="flex items-center gap-2 mt-1">
              <button
                onClick={() => handleDownload(key)}
                className="flex-1 text-center py-1.5 rounded text-xs font-semibold transition hover:brightness-110 active:brightness-95"
                style={{ background: 'var(--status-error)', color: '#fff' }}
              >
                {t("重試安裝")}
              </button>
              <button
                onClick={() => {
                  setProgressMap(prev => {
                    const copy = { ...prev };
                    delete copy[key];
                    return copy;
                  });
                }}
                className="px-3 py-1.5 rounded text-xs font-medium transition hover:bg-[var(--surface-hover)]"
                style={{ color: 'var(--fg-2)', border: '1px solid var(--border)' }}
              >
                {t("清除錯誤")}
              </button>
            </div>
          </div>
        );
      }
    }

    return (
      <div className="flex items-center justify-between py-2.5 px-2 rounded-lg transition duration-150 relative" style={{ color: 'var(--fg-2)' }}>
        <div className="flex items-center gap-3 min-w-0 flex-1 pr-2">
          <span className="shrink-0" style={{ color: 'var(--muted)' }}>{icon}</span>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2 min-w-0">
              <span className="text-sm font-semibold truncate" style={{ color: 'var(--fg)' }}>{t(label)}</span>
            </div>
            <span className="text-xs mt-0.5 block font-medium truncate" style={{ color: statusColor }}>{statusText}</span>
            {isPhp && localVer !== '' && (
              <div className="mt-1 flex items-center gap-1.5 text-[11px] truncate">
                <Zap size={11} className="shrink-0" style={{ color: (phpRedisInfo?.installed && phpRedisInfo?.iniEnabled) ? 'var(--status-ok)' : (phpRedisInfo?.installed ? 'var(--status-warn)' : 'var(--muted)') }} />
                <span className="truncate" style={{ color: (phpRedisInfo?.installed && phpRedisInfo?.iniEnabled) ? 'var(--status-ok)' : (phpRedisInfo?.installed ? 'var(--status-warn)' : 'var(--muted)') }}>
                  {phpRedisInfo?.installed
                    ? (phpRedisInfo?.iniEnabled
                        ? t("Redis 擴充: 已就緒 (v%s)", phpRedisInfo.version || '6.0.2')
                        : t("Redis 擴充: DLL 已安裝，未在 php.ini 啟用"))
                    : (phpRedisInfo?.supported ? t("Redis 擴充: 未配置") : t("Redis 擴充: 不支援"))}
                </span>
                {phpRedisInfo?.installed && !phpRedisInfo?.iniEnabled && (
                  <button
                    onClick={handleEnablePHPRedis}
                    className="ml-1 px-1.5 py-0.5 rounded text-[10px] font-semibold transition hover:opacity-80 active:scale-95"
                    style={{ background: 'var(--status-warn-bg)', color: 'var(--status-warn)', border: '1px solid var(--status-warn)' }}
                  >
                    {t("一鍵啟用")}
                  </button>
                )}
              </div>
            )}
          </div>
        </div>

        {showBtn && (
          <div className="relative shrink-0 split-btn-dropdown flex items-center">
            {/* 一體化精巧按鈕組 */}
            <div
              className="inline-flex items-stretch rounded-md shadow-sm overflow-hidden transition"
              style={{
                border: (localVer === '' || isUpdate) ? 'none' : '1px solid var(--border)',
                background: localVer === '' ? 'var(--status-info)' : (isUpdate ? 'var(--status-warn)' : 'var(--card)')
              }}
            >
              {/* 主按鈕 */}
              <button
                onClick={() => !btnDisabled && handleDownload(key)}
                disabled={btnDisabled}
                className={`px-2.5 py-1 text-xs font-semibold flex items-center justify-center gap-1.5 transition select-none whitespace-nowrap ${(localVer === '' || isUpdate)
                  ? 'text-white hover:brightness-110 active:brightness-95'
                  : 'text-[var(--fg-2)] hover:bg-[var(--surface-hover)] hover:text-[var(--fg)] active:bg-[var(--surface-active)]'
                  }`}
                title={localVer === '' ? t("下載安裝此依賴") : (isUpdate ? t("立即更新至最新版") : (isPhp ? (phpRedisInfo?.supported ? t("重裝 PHP (含 Redis)") : t("重裝 PHP")) : t("重裝此依賴")))}
              >
                {btnIcon}
                <span>{btnText}</span>
              </button>

              {/* 垂直細緻分隔線 */}
              <div
                className="w-[1px] self-stretch my-0.5"
                style={{ background: (localVer === '' || isUpdate) ? 'rgba(255,255,255,0.3)' : 'var(--border)' }}
              />

              {/* 下拉箭頭按鈕：精緻緊湊點擊區 */}
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  setActiveDropdown(prev => prev === key ? null : key);
                }}
                className={`w-6 shrink-0 flex items-center justify-center transition select-none ${(localVer === '' || isUpdate)
                  ? 'text-white hover:brightness-110 active:brightness-95'
                  : 'hover:bg-[var(--surface-hover)] hover:text-[var(--fg)] active:bg-[var(--surface-active)]'
                  }`}
                style={{ color: (localVer === '' || isUpdate) ? '#fff' : 'var(--fg-2)' }}
                title={t("更多操作與來源資訊")}
              >
                <ChevronDown size={13} className={`shrink-0 transition-transform duration-200 ${isDropdownOpen ? 'rotate-180' : ''}`} />
              </button>
            </div>

            {/* 下拉選單 Popover */}
            {isDropdownOpen && (
              <div
                className="absolute right-0 top-full mt-1.5 w-60 rounded-xl shadow-xl border p-1.5 z-50 animate-fade-in"
                style={{
                  backgroundColor: 'var(--menu-bg, var(--bg-deep))',
                  borderColor: 'var(--menu-border, var(--border))',
                  boxShadow: 'var(--menu-shadow, 0 12px 28px rgba(0,0,0,0.25))',
                  backdropFilter: 'blur(16px)',
                }}
              >
                <div className="px-2.5 py-1 text-[10px] font-bold uppercase tracking-wider text-[var(--muted)] select-none">
                  {localVer === '' ? t("依賴來源與操作") : t("服務維護與配置")}
                </div>

                {/* 複製下載連結（Hover 懸停提示完整 URL，內附微縮網址預覽） */}
                {spec?.url && (
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      handleCopyUrl(spec.url, key);
                    }}
                    title={spec.url}
                    className="dropdown-menu-item w-full text-left px-2.5 py-2 text-xs rounded-lg flex items-center gap-2 hover:bg-[var(--surface-hover)] text-[var(--fg)] transition group"
                  >
                    {copiedKey === key ? (
                      <Check size={13} className="shrink-0" style={{ color: 'var(--status-ok)' }} />
                    ) : (
                      <Copy size={13} className="shrink-0" style={{ color: 'var(--status-info)' }} />
                    )}
                    <div className="flex-1 min-w-0">
                      <div className="font-medium flex items-center justify-between">
                        <span>{copiedKey === key ? t("已複製下載連結") : t("複製下載連結")}</span>
                      </div>
                      <div className="text-[10px] text-[var(--muted)] truncate max-w-[195px]" style={{ fontFamily: 'var(--font-mono)' }}>
                        {spec.url}
                      </div>
                    </div>
                  </button>
                )}

                {/* 主動作（重裝 / 更新 / 下載） */}
                {localVer !== '' ? (
                  <button
                    onClick={() => { setActiveDropdown(null); handleDownload(key); }}
                    className="dropdown-menu-item w-full text-left px-2.5 py-2 text-xs rounded-lg flex items-center gap-2 hover:bg-[var(--surface-hover)] text-[var(--fg)] transition"
                  >
                    <RotateCw size={13} className="shrink-0" style={{ color: 'var(--status-info)' }} />
                    <span>
                      {isPhp
                        ? (phpRedisInfo?.supported ? t("重裝 PHP (含 Redis)") : t("重裝 PHP"))
                        : (isUpdate ? t("立即更新") : t("重裝此依賴"))}
                    </span>
                  </button>
                ) : (
                  <button
                    onClick={() => { setActiveDropdown(null); handleDownload(key); }}
                    className="dropdown-menu-item w-full text-left px-2.5 py-2 text-xs rounded-lg flex items-center gap-2 hover:bg-[var(--surface-hover)] text-[var(--fg)] transition"
                  >
                    <Download size={13} className="shrink-0" style={{ color: 'var(--status-info)' }} />
                    <span>{t("下載安裝此依賴")}</span>
                  </button>
                )}

                {/* PHP 相關額外動作 */}
                {isPhp && phpRedisInfo?.supported && (
                  <button
                    onClick={() => handleInstallRedisExt(key)}
                    className="dropdown-menu-item w-full text-left px-2.5 py-2 text-xs rounded-lg flex items-center gap-2 hover:bg-[var(--surface-hover)] text-[var(--fg)] transition"
                  >
                    <Zap size={13} className="shrink-0" style={{ color: phpRedisInfo?.installed ? 'var(--status-ok)' : 'var(--status-warn)' }} />
                    <span>{phpRedisInfo?.installed ? t("單獨重裝 Redis 擴充") : t("單獨安裝 Redis 擴充")}</span>
                  </button>
                )}

                {isPhp && phpRedisInfo?.installed && !phpRedisInfo?.iniEnabled && (
                  <button
                    onClick={handleEnablePHPRedis}
                    className="dropdown-menu-item w-full text-left px-2.5 py-2 text-xs rounded-lg flex items-center gap-2 hover:bg-[var(--surface-hover)] text-[var(--fg)] transition"
                  >
                    <Zap size={13} className="shrink-0" style={{ color: 'var(--status-warn)' }} />
                    <span>{t("一鍵啟用 php.ini Redis 擴充")}</span>
                  </button>
                )}

                {/* 開啟安裝目錄（僅已安裝項目顯示） */}
                {localVer !== '' && (
                  <button
                    onClick={() => handleOpenFolder(key)}
                    className="dropdown-menu-item w-full text-left px-2.5 py-2 text-xs rounded-lg flex items-center gap-2 hover:bg-[var(--surface-hover)] text-[var(--fg)] transition"
                  >
                    <FolderOpen size={13} className="shrink-0" style={{ color: 'var(--status-info)' }} />
                    <span>{t("開啟安裝目錄")}</span>
                  </button>
                )}

                {/* 開源授權與官方源碼管道 */}
                {spec?.license && (
                  <>
                    <div className="my-1 border-t border-[var(--border-soft)]" />
                    <div className="px-2.5 py-1.5 flex items-center justify-between text-[11px] select-none">
                      <span className="flex items-center gap-1.5 text-[var(--muted)]">
                        <Scale size={12} style={{ color: 'var(--muted)' }} />
                        <span>{t("授權協議")}</span>
                      </span>
                      <span className="font-mono font-medium px-1.5 py-0.5 rounded text-[10px]" style={{ background: 'var(--surface)', color: 'var(--fg)', border: '1px solid var(--border-soft)' }}>
                        {spec.license}
                      </span>
                    </div>
                    {(spec.source_url || spec.homepage) && (
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          setActiveDropdown(null);
                          BrowserOpenURL(spec.source_url || spec.homepage!);
                        }}
                        className="dropdown-menu-item w-full text-left px-2.5 py-1.5 text-xs rounded-lg flex items-center gap-2 hover:bg-[var(--surface-hover)] text-[var(--fg)] transition"
                      >
                        <ExternalLink size={13} className="shrink-0" style={{ color: 'var(--muted)' }} />
                        <span>{t("官方原始碼")} ↗</span>
                      </button>
                    )}
                  </>
                )}

                {/* 移除依賴（僅已安裝項目顯示） */}
                {localVer !== '' && (
                  <>
                    <div className="my-1 border-t border-[var(--border-soft)]" />
                    <button
                      onClick={() => handleUninstall(key, label)}
                      className="dropdown-menu-item dropdown-menu-item-danger w-full text-left px-2.5 py-2 text-xs rounded-lg flex items-center gap-2 hover:bg-[var(--status-error-bg)] text-[var(--status-error)] transition"
                    >
                      <Trash2 size={13} className="shrink-0" />
                      <span>{isPhp ? t("移除此版本") : t("移除依賴")}</span>
                    </button>
                  </>
                )}
              </div>
            )}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0" style={{ background: 'var(--overlay-bg)', backdropFilter: 'blur(4px)' }} onClick={onClose} />
      <div className="relative w-full max-w-2xl max-h-[90vh] rounded-2xl flex flex-col overflow-hidden animate-fade-in select-none dependency-manager-modal" style={{ background: 'var(--card)', border: '1px solid var(--border)', boxShadow: 'var(--shadow-lg)', color: 'var(--fg)' }}>
        {/* Header */}
        <div className="px-6 py-4 flex items-center justify-between shrink-0" style={{ borderBottom: '1px solid var(--border)', background: 'var(--bg-deep)' }}>
          <div className="flex items-center gap-2.5">
            <div>
              <h2 className="text-lg font-bold tracking-wide">{t("WinCMP 依賴庫管理")}</h2>
              <p className="text-xs mt-0.5" style={{ color: 'var(--muted)' }}>{t("下載或升級本機 Web 開發依賴")}</p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={handleFetchRemote}
              disabled={isManualChecking}
              className="p-2 rounded-lg transition flex items-center gap-1 text-xs font-semibold"
              style={{ color: 'var(--fg-2)', opacity: isManualChecking ? 0.7 : 1 }}
              title={t("從遠端檢查依賴版本")}
            >
              <RefreshCw size={14} className={isManualChecking ? 'animate-spin' : ''} />
              <span>{isManualChecking ? t('檢查中...') : t('檢查更新')}</span>
            </button>
            <button id="btn-close-dep-manager" onClick={onClose} className="p-1.5 rounded-lg transition" style={{ color: 'var(--muted)' }}><X size={18} /></button>
          </div>
        </div>

        {/* Content */}
        <div
          className={`flex-1 overflow-y-auto p-6 space-y-6 dependency-manager-content transition-opacity duration-300 ease-in-out ${
            isContentUpdating ? 'opacity-40 pointer-events-none' : 'opacity-100'
          }`}
        >
          {isLoadingConfig ? (
            <div className="py-20 flex flex-col items-center justify-center gap-3" style={{ color: 'var(--muted)' }}>
              <Loader2 size={32} className="animate-spin" style={{ color: 'var(--status-info)' }} />
              <span className="text-sm font-semibold">{t("正在讀取依賴設定...")}</span>
            </div>
          ) : (
            <>
              <div style={cardStyle}>
                <h3 className="text-xs font-bold uppercase tracking-wider flex items-center gap-1.5 select-none pb-2 mb-1" style={{ color: 'var(--status-info)', borderBottom: '1px solid var(--border-soft)' }}>
                  <Cpu size={13} /> {t("核心執行環境")}
                </h3>
                <div className="divide-y divide-[var(--border-soft)]">
                  {renderDependencyRow('caddy', 'Caddy Web 伺服器', <Server size={16} />)}
                  {renderDependencyRow('mariadb', 'MariaDB 資料庫', <Database size={16} />)}
                  {renderDependencyRow('redis', 'Redis 快取服務', <Zap size={16} />)}
                </div>
              </div>

              <div style={cardStyle}>
                <h3 className="text-xs font-bold uppercase tracking-wider flex items-center gap-1.5 select-none pb-2 mb-1" style={{ color: 'var(--status-ok)', borderBottom: '1px solid var(--border-soft)' }}>
                  <Server size={13} /> {t("PHP FastCGI 環境")}
                </h3>
                <div className="divide-y divide-[var(--border-soft)]">
                  {phpKeys.map(key => {
                    const cleanKey = key.replace(/^php_?/, '');
                    const majorMin = cleanKey.replace(/(\d)(\d)/, '$1.$2');
                    return renderDependencyRow(key, `PHP ${majorMin} NTS`, <Terminal size={16} />);
                  })}
                </div>
              </div>

              <div style={cardStyle}>
                <h3 className="text-xs font-bold uppercase tracking-wider flex items-center gap-1.5 select-none pb-2" style={{ color: 'var(--accent)', borderBottom: '1px solid var(--border-soft)' }}>
                  <SettingsIcon size={13} /> {t("開發輔助工具與實用工具")}
                </h3>
                <div className="divide-y divide-[var(--border-soft)]">
                  {renderDependencyRow('composer', 'Composer (PHP 包管理器)', <Terminal size={16} />)}
                  {renderDependencyRow('node', 'Node.js LTS 運行環境', <Terminal size={16} />)}
                  {renderDependencyRow('mailpit', 'Mailpit 郵件測試伺服器', <Server size={16} />)}
                  {renderDependencyRow('heidisql', 'HeidiSQL 資料庫 GUI 工具', <Database size={16} />)}
                </div>
              </div>
            </>
          )}
        </div>

        {/* Footer */}
        <div className="px-6 py-3.5 text-[11px] flex justify-between items-center select-none shrink-0" style={{ borderTop: '1px solid var(--border)', background: 'var(--bg-deep)', color: 'var(--meta)' }}>
          <div className="flex items-center gap-2">
            <span>{t("提示：安裝完成後系統會自動重新掃描環境。")}</span>
            <span style={{ color: 'var(--border-soft)' }}>·</span>
            <button
              onClick={() => setShowNoticeModal(true)}
              className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-[11px] font-medium transition cursor-pointer select-none hover:bg-[var(--surface-hover)]"
              style={{ color: 'var(--fg-2)' }}
              title={t("第三方開源授權與商標聲明")}
            >
              <Scale size={12} className="shrink-0" style={{ color: 'var(--accent)' }} />
              <span className="hover:text-[var(--fg)]">{t("開源授權與商標聲明")}</span>
            </button>
          </div>
          <span style={{ fontFamily: 'var(--font-mono)' }}>Downloader Pipeline</span>
        </div>
      </div>

      {/* 第三方開源授權與商標聲明 Modal */}
      {showNoticeModal && (
        <div className="fixed inset-0 z-[60] flex items-center justify-center p-4">
          <div
            className="absolute inset-0 bg-black/75 backdrop-blur-md"
            onClick={() => setShowNoticeModal(false)}
          />
          <div
            className="relative w-full max-w-2xl max-h-[85vh] flex flex-col rounded-2xl shadow-2xl border overflow-hidden animate-fade-in"
            style={{
              backgroundColor: 'var(--bg-deep)',
              borderColor: 'var(--border-strong, var(--border))',
              color: 'var(--fg)',
            }}
          >
            {/* Modal Header */}
            <div className="px-6 py-4 border-b flex justify-between items-center select-none shrink-0" style={{ borderColor: 'var(--border-soft)', background: 'var(--bg-deep)' }}>
              <div className="flex items-center gap-2.5">
                <Scale size={18} style={{ color: 'var(--accent)' }} />
                <div>
                  <h3 className="font-bold text-sm" style={{ color: 'var(--fg)' }}>{t("第三方開源授權與商標聲明")}</h3>
                  <p className="text-[10px] mt-0.5 text-[var(--muted)]">Third-Party Notices, Licenses &amp; Trademark Disclaimer</p>
                </div>
              </div>
              <button
                onClick={() => setShowNoticeModal(false)}
                className="p-1.5 rounded-lg hover:bg-[var(--surface-hover)] text-[var(--muted)] hover:text-[var(--fg)] transition cursor-pointer"
              >
                <X size={16} />
              </button>
            </div>

            {/* Modal Content */}
            <div className="p-6 overflow-y-auto space-y-5 text-xs leading-relaxed" style={{ color: 'var(--fg-2)', backgroundColor: 'var(--bg-deep)' }}>
              {/* WinCMP & Architecture */}
              <div className="p-3.5 rounded-xl border space-y-1.5" style={{ background: 'var(--surface-warm)', borderColor: 'var(--border-soft)' }}>
                <div className="font-bold text-xs" style={{ color: 'var(--fg)' }}>WinCMP (MIT License)</div>
                <p className="text-[11px] text-[var(--muted)]">
                  {t("WinCMP 核心主程式採用 MIT 授權發行。官方發行包未捆綁分發任何外部服務二進位檔；各項執行環境（如 MariaDB, Redis, PHP, Caddy, Node.js 等）由使用者透過依賴下載器自各官方伺服器直接取得。WinCMP 僅扮演調度器與進程管理器。")}
                </p>
              </div>

              {/* Trademark Disclaimer */}
              <div className="space-y-1.5">
                <h4 className="font-bold text-xs flex items-center gap-1.5" style={{ color: 'var(--fg)' }}>
                  <span>{t("商標與免責聲明")}</span>
                </h4>
                <p className="text-[11px] text-[var(--muted)]">
                  {t("WinCMP 是一個獨立的開源開發工具。軟體介面與文檔提及之 Caddy, MariaDB, Redis, PHP, Node.js, Composer, HeidiSQL, Mailpit, Bun 等產品名稱與商標均屬其各自商標權利人所有。WinCMP 與上述專案團隊無任何官方附屬、贊助或背書關係。")}
                </p>
              </div>

              {/* Managed Runtimes Table */}
              <div className="space-y-2">
                <h4 className="font-bold text-xs" style={{ color: 'var(--fg)' }}>{t("受控外部執行環境")}</h4>
                <div className="border rounded-xl overflow-hidden" style={{ borderColor: 'var(--border-soft)', backgroundColor: 'var(--bg-deep)' }}>
                  <table className="w-full text-left text-[11px]">
                    <thead style={{ background: 'var(--surface-warm)', borderBottom: '1px solid var(--border-soft)' }}>
                      <tr>
                        <th className="py-2 px-3 font-semibold text-[var(--muted)]">{t("組件")}</th>
                        <th className="py-2 px-3 font-semibold text-[var(--muted)]">{t("許可證")}</th>
                        <th className="py-2 px-3 font-semibold text-[var(--muted)]">{t("官方原始碼")}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-[var(--border-soft)]">
                      {[
                        { name: 'Caddy', license: 'Apache-2.0', url: 'https://github.com/caddyserver/caddy' },
                        { name: 'MariaDB', license: 'GPL-2.0-only', url: 'https://github.com/MariaDB/server' },
                        { name: 'Redis (Win Port)', license: 'BSD-3-Clause', url: 'https://github.com/tporadowski/redis' },
                        { name: 'PHP', license: 'PHP-3.01', url: 'https://github.com/php/php-src' },
                        { name: 'Node.js', license: 'MIT', url: 'https://github.com/nodejs/node' },
                        { name: 'Composer', license: 'MIT', url: 'https://github.com/composer/composer' },
                        { name: 'HeidiSQL', license: 'GPL-3.0-or-later', url: 'https://github.com/HeidiSQL/HeidiSQL' },
                        { name: 'Mailpit', license: 'MIT', url: 'https://github.com/axllent/mailpit' },
                      ].map((item) => (
                        <tr key={item.name} className="hover:bg-[var(--surface-hover)] transition">
                          <td className="py-2 px-3 font-medium text-[var(--fg)]">{item.name}</td>
                          <td className="py-2 px-3 font-mono text-[10px] text-[var(--accent)]">{item.license}</td>
                          <td className="py-2 px-3">
                            <button
                              onClick={() => BrowserOpenURL(item.url)}
                              className="inline-flex items-center gap-1 hover:underline text-[10px] cursor-pointer"
                              style={{ color: 'var(--status-info)' }}
                            >
                              <span>GitHub</span>
                              <ExternalLink size={10} />
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            </div>

            {/* Modal Footer */}
            <div className="px-6 py-3 border-t flex justify-end items-center gap-2 select-none shrink-0" style={{ borderColor: 'var(--border-soft)', background: 'var(--bg-deep)' }}>
              <button
                onClick={() => setShowNoticeModal(false)}
                className="px-4 py-1.5 rounded-lg text-xs font-semibold transition hover:opacity-90 active:scale-95 cursor-pointer shadow-sm"
                style={{
                  background: 'var(--accent)',
                  color: 'var(--accent-on)',
                }}
              >
                {t("關閉")}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

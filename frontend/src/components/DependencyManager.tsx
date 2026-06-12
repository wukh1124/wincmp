import React, { useState, useEffect } from 'react';
import {
  X, RefreshCw, Download, ArrowUpCircle, CheckCircle2,
  Loader2, AlertTriangle, Cpu, Database, Settings as SettingsIcon,
  HelpCircle, Server, Terminal, HardDrive
} from 'lucide-react';
import {
  GetDependencyConfig, FetchRemoteDependencies, DownloadDependency,
  ScanServices, GetScanResult
} from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { t, useLanguage } from '../i18n';

interface DependencyItem { version: string; url: string; }
type DependencyConfig = Record<string, DependencyItem>;
interface ProgressData { status: 'downloading' | 'extracting' | 'completed' | 'error' | 'preparing'; percent: number; currentMB: number; totalMB: number; error: string; }
interface DependencyManagerProps { isOpen: boolean; onClose: () => void; onInstalled?: () => void; }

export default function DependencyManager({ isOpen, onClose, onInstalled }: DependencyManagerProps) {
  useLanguage();
  const [depConfig, setDepConfig] = useState<DependencyConfig | null>(null);
  const [scanResult, setScanResult] = useState<any>(null);
  const [isLoadingConfig, setIsLoadingConfig] = useState(false);
  const [isFetchingRemote, setIsFetchingRemote] = useState(false);
  const [progressMap, setProgressMap] = useState<Record<string, ProgressData>>({});

  useEffect(() => { if (isOpen) loadData(); }, [isOpen]);

  useEffect(() => {
    const handleProgress = (data: any) => {
      if (data && data.key) {
        setProgressMap(prev => ({ ...prev, [data.key]: { status: data.status, percent: data.percent, currentMB: data.currentMB, totalMB: data.totalMB, error: data.error } }));
        if (data.status === 'completed') { refreshLocalScan(); if (onInstalled) onInstalled(); }
      }
    };
    const unsubscribe = EventsOn('dependency_progress', handleProgress);
    return () => { unsubscribe(); };
  }, [onInstalled]);

  const loadData = async () => {
    setIsLoadingConfig(true);
    try { setDepConfig(await GetDependencyConfig()); setScanResult(await GetScanResult()); }
    catch (err) { console.error("載入依賴資訊失敗:", err); }
    finally { setIsLoadingConfig(false); }
  };

  const refreshLocalScan = async () => {
    try { setScanResult(await ScanServices()); } catch (err) { console.error("刷新服務掃描失敗:", err); }
  };

  const handleFetchRemote = async () => {
    setIsFetchingRemote(true);
    try { setDepConfig(await FetchRemoteDependencies()); (window as any).customAlert(t("成功從遠端獲取最新的建議依賴配置！")); }
    catch (err) { (window as any).customAlert(`${t("獲取遠端依賴配置失敗")}: ${err}`); }
    finally { setIsFetchingRemote(false); }
  };

  const handleDownload = async (key: string) => {
    setProgressMap(prev => ({ ...prev, [key]: { status: 'preparing', percent: 0, currentMB: 0, totalMB: 0, error: '' } }));
    try { await DownloadDependency(key); }
    catch (err: any) { setProgressMap(prev => ({ ...prev, [key]: { status: 'error', percent: 0, currentMB: 0, totalMB: 0, error: err.toString() } })); }
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
    if (key === 'composer') return scanResult.ComposerList?.[0]?.Version || '';
    if (key === 'heidisql') return scanResult.HeidiSQLList?.[0]?.Version || '';
    if (key === 'node') return scanResult.NodeList?.[0]?.Version || '';
    if (key === 'mailpit') return scanResult.MailpitList?.[0]?.Version || '';
    if (key.startsWith('php')) {
      const majorMin = key.replace('php', '').replace(/(\d)(\d)/, '$1.$2');
      return scanResult.PHPList?.find((p: any) => p.MajorMin === majorMin)?.Version || '';
    }
    return '';
  };

  if (!isOpen) return null;

  const phpKeys = depConfig ? Object.keys(depConfig).filter(k => k.startsWith('php') && k !== 'php').sort((a, b) => compareVersions(depConfig[b].version, depConfig[a].version)) : [];

  // ─── Styles ─────────────────────────────────────────────
  const cardStyle: React.CSSProperties = { background: 'var(--card)', border: '1px solid var(--border)', borderRadius: 'var(--radius-lg)', padding: 16 };

  const renderDependencyRow = (key: string, label: string, icon: React.ReactNode) => {
    if (!depConfig) return null;
    const spec = depConfig[key];
    if (!spec) return null;
    const localVer = getInstalledVersion(key);
    const recVer = spec.version;
    const progress = progressMap[key];

    let statusText = '';
    let statusColor: React.CSSProperties['color'] = 'var(--muted)';
    let showBtn = true;
    let btnText = t('下載安裝');
    let btnStyle: React.CSSProperties = { background: 'var(--status-info)', color: '#fff' };
    let btnIcon = <Download size={13} />;

    if (localVer === '') {
      statusText = `${t("未安裝")} (${t("建議")}: v${recVer})`;
      statusColor = 'var(--status-error)';
      btnText = t('下載');
    } else {
      const cmp = compareVersions(localVer, recVer);
      if (cmp < 0) {
        statusText = `${t("已安裝")}: v${localVer} (${t("有新版")}: v${recVer})`;
        statusColor = 'var(--status-warn)';
        btnText = t('更新');
        btnStyle = { background: 'var(--status-warn)', color: '#fff' };
        btnIcon = <ArrowUpCircle size={13} />;
      } else {
        statusText = `${t("已安裝")}: v${localVer} (${t("最新")})`;
        statusColor = 'var(--status-ok)';
        btnText = t('重裝');
        btnStyle = { background: 'var(--input-bg)', border: '1px solid var(--border)', color: 'var(--fg-2)' };
      }
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
        showBtn = false;
      }
      if (status === 'error') {
        return (
          <div className="flex flex-col gap-2 p-3 rounded-lg" style={{ background: 'var(--status-error-bg)', border: '1px solid var(--status-error)' }}>
            <div className="flex justify-between items-center text-xs">
              <span className="font-semibold flex items-center gap-2" style={{ color: 'var(--status-error)' }}>{icon} {t(label)}</span>
              <span className="flex items-center gap-1" style={{ color: 'var(--status-error)' }}>
                <AlertTriangle size={12} /> {t("安裝失敗")}
              </span>
            </div>
            <p className="text-[11px] mt-1 break-all p-1.5 rounded" style={{ color: 'var(--fg-2)', background: 'var(--surface)', fontFamily: 'var(--font-mono)' }}>{error}</p>
            <button onClick={() => handleDownload(key)} className="mt-1 text-center py-1 rounded text-xs transition" style={{ background: 'var(--status-error-bg)', color: 'var(--status-error)' }}>
              {t("重試安裝")}
            </button>
          </div>
        );
      }
    }

    return (
      <div className="flex items-center justify-between py-2.5 px-2 rounded-lg transition duration-150" style={{ color: 'var(--fg-2)' }}>
        <div className="flex items-center gap-3">
          <span style={{ color: 'var(--muted)' }}>{icon}</span>
          <div>
            <span className="text-sm font-semibold block" style={{ color: 'var(--fg)' }}>{t(label)}</span>
            <span className="text-xs mt-0.5 block font-medium" style={{ color: statusColor }}>{statusText}</span>
          </div>
        </div>
        {showBtn && (
          <button onClick={() => handleDownload(key)} className="px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition" style={btnStyle}>
            {btnIcon}<span>{btnText}</span>
          </button>
        )}
      </div>
    );
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0" style={{ background: 'var(--overlay-bg)', backdropFilter: 'blur(4px)' }} onClick={onClose} />
      <div className="relative w-full max-w-2xl max-h-[90vh] rounded-2xl flex flex-col overflow-hidden animate-fade-in select-none" style={{ background: 'var(--card)', border: '1px solid var(--border)', boxShadow: 'var(--shadow-lg)', color: 'var(--fg)' }}>
        {/* Header */}
        <div className="px-6 py-4 flex items-center justify-between shrink-0" style={{ borderBottom: '1px solid var(--border)', background: 'var(--bg-deep)' }}>
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg flex items-center justify-center" style={{ background: 'var(--status-info-bg)', color: 'var(--status-info)' }}>
              <HardDrive size={18} />
            </div>
            <div>
              <h2 className="text-lg font-bold tracking-wide" style={{ fontFamily: 'var(--font-display)' }}>📦 {t("WinCMP 依賴庫管理")}</h2>
              <p className="text-xs mt-0.5" style={{ color: 'var(--muted)' }}>{t("下載或升級本機 Web 開發依賴")}</p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <button onClick={handleFetchRemote} disabled={isFetchingRemote} className="p-2 rounded-lg transition flex items-center gap-1 text-xs font-semibold" style={{ color: 'var(--fg-2)' }} title={t("從遠端獲取最新版本")}>
              <RefreshCw size={14} className={isFetchingRemote ? 'animate-spin' : ''} />
              <span>{isFetchingRemote ? t('獲取中...') : t('獲取最新')}</span>
            </button>
            <button onClick={onClose} className="p-1.5 rounded-lg transition" style={{ color: 'var(--muted)' }}><X size={18} /></button>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          {isLoadingConfig ? (
            <div className="py-20 flex flex-col items-center justify-center gap-3" style={{ color: 'var(--muted)' }}>
              <Loader2 size={32} className="animate-spin" style={{ color: 'var(--status-info)' }} />
              <span className="text-sm font-semibold">{t("正在讀取依賴設定...")}</span>
            </div>
          ) : (
            <>
              <div style={cardStyle}>
                <h3 className="text-xs font-bold uppercase tracking-wider flex items-center gap-1.5 select-none pb-2" style={{ color: 'var(--status-info)', borderBottom: '1px solid var(--border-soft)', fontFamily: 'var(--font-display)' }}>
                  <Cpu size={13} /> {t("核心執行環境")}
                </h3>
                <div className="divide-y divide-[var(--border-soft)]">
                  {renderDependencyRow('caddy', 'Caddy Web 伺服器', <Server size={16} />)}
                  {renderDependencyRow('mariadb', 'MariaDB 資料庫', <Database size={16} />)}
                </div>
              </div>

              <div style={cardStyle}>
                <h3 className="text-xs font-bold uppercase tracking-wider flex items-center gap-1.5 select-none pb-2" style={{ color: 'var(--status-ok)', borderBottom: '1px solid var(--border-soft)', fontFamily: 'var(--font-display)' }}>
                  <Server size={13} /> {t("PHP FastCGI 環境")}
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {phpKeys.map(key => {
                    const majorMin = key.replace('php', '').replace(/(\d)(\d)/, '$1.$2');
                    return (
                      <div key={key} className="rounded-lg p-2.5" style={{ border: '1px solid var(--border-soft)', background: 'var(--surface)' }}>
                        {renderDependencyRow(key, `PHP ${majorMin} NTS`, <Terminal size={14} />)}
                      </div>
                    );
                  })}
                </div>
              </div>

              <div style={cardStyle}>
                <h3 className="text-xs font-bold uppercase tracking-wider flex items-center gap-1.5 select-none pb-2" style={{ color: 'var(--accent)', borderBottom: '1px solid var(--border-soft)', fontFamily: 'var(--font-display)' }}>
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
        <div className="px-6 py-4 text-[11px] flex justify-between items-center select-none shrink-0" style={{ borderTop: '1px solid var(--border)', background: 'var(--bg-deep)', color: 'var(--meta)' }}>
          <span>{t("提示：安裝完成後系統會自動重新掃描環境。")}</span>
          <span style={{ fontFamily: 'var(--font-mono)' }}>Downloader Pipeline</span>
        </div>
      </div>
    </div>
  );
}

import React, { useState, useEffect } from 'react';
import { Play, Square, RefreshCw, Layers, Cpu, Database, Server, CheckCircle, XCircle, AlertTriangle, Package, Folder, LayoutGrid, Terminal, X } from 'lucide-react';
import {
  GetConfig, SaveConfig, ScanServices, GetScanResult, GetServicesStatus,
  StartCaddy, StopCaddy, ReloadCaddy, StartMariaDB, StopMariaDB,
  StartMailpit, StopMailpit, StartPHP, StopPHP,
  CheckMissingCoreDependencies, CheckPortConflicts
} from '../../wailsjs/go/main/App';
import { scanner } from '../../wailsjs/go/models';
import DependencyManager from './DependencyManager';
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime';
import { t, useLanguage } from '../i18n';

let logsAutoExpanded = false;

export default function Dashboard() {
  useLanguage();
  const [config, setConfig] = useState<any>(null);
  const [scanResult, setScanResult] = useState<scanner.ScanResult | null>(null);
  const [servicesStatus, setServicesStatus] = useState<Record<string, boolean>>({});
  const [loadingServices, setLoadingServices] = useState<Record<string, boolean>>({});
  const [isScanning, setIsScanning] = useState(false);
  const [showDepManager, setShowDepManager] = useState(false);
  const [missingCore, setMissingCore] = useState<{ caddy: boolean; php: boolean }>({ caddy: false, php: false });
  const [dismissBanner, setDismissBanner] = useState(false);
  const [portConflicts, setPortConflicts] = useState<Record<string, boolean>>({});

  useEffect(() => {
    async function initData() {
      try {
        const cfg = await GetConfig();
        setConfig(cfg);
        const scan = await GetScanResult();
        setScanResult(scan);
        await updateStatus();
        await updateConflicts();
        const missing = await CheckMissingCoreDependencies();
        setMissingCore({ caddy: !!missing?.caddy, php: !!missing?.php });
      } catch (err) { console.error("初始化資料失敗:", err); }
    }
    initData();
  }, []);

  useEffect(() => {
    const timer = setInterval(() => { updateStatus(); updateConflicts(); }, 2000);
    return () => clearInterval(timer);
  }, [scanResult]);

  const updateStatus = async () => {
    try { setServicesStatus(await GetServicesStatus()); } catch (err) { console.error("更新服務狀態失敗:", err); }
  };

  const updateConflicts = async () => {
    try { setPortConflicts((await CheckPortConflicts()) || {}); } catch (err) { console.error("更新埠口衝突失敗:", err); }
  };

  const handleScan = async () => {
    setIsScanning(true);
    try {
      const res = await ScanServices();
      setScanResult(res);
      await updateStatus();
      const missing = await CheckMissingCoreDependencies();
      setMissingCore({ caddy: !!missing?.caddy, php: !!missing?.php });
    } catch (err) { console.error("掃描二進位服務失敗:", err); }
    finally { setIsScanning(false); }
  };

  const triggerAutoExpandLogs = () => {
    if (!logsAutoExpanded) {
      logsAutoExpanded = true;
      window.dispatchEvent(new CustomEvent('wincmp_auto_expand_logs'));
    }
  };

  const handleServiceAction = async (serviceName: string, action: 'start' | 'stop' | 'reload', extraInfo?: any) => {
    const key = `${serviceName}-${action}`;
    setLoadingServices(prev => ({ ...prev, [key]: true }));
    try {
      if (serviceName === 'caddy') {
        if (action === 'start') { await StartCaddy(extraInfo.Version, extraInfo.ExePath); triggerAutoExpandLogs(); }
        else if (action === 'stop') await StopCaddy();
        else if (action === 'reload') await ReloadCaddy();
      } else if (serviceName.startsWith('mariadb')) {
        const version = extraInfo.Version;
        if (action === 'start') { await StartMariaDB(version); triggerAutoExpandLogs(); }
        else if (action === 'stop') await StopMariaDB(version);
      } else if (serviceName === 'mailpit') {
        if (action === 'start') {
          await StartMailpit(extraInfo.Version, extraInfo.ExePath, config?.global?.mailpit_smtp_port || 1025, config?.global?.mailpit_http_port || 8025, config?.global?.mailpit_use_db || false);
          triggerAutoExpandLogs();
        } else if (action === 'stop') await StopMailpit();
      } else if (serviceName.startsWith('php')) {
        const version = extraInfo.Version;
        if (action === 'start') { await StartPHP(version); triggerAutoExpandLogs(); }
        else if (action === 'stop') await StopPHP(version);
      }
      await updateStatus();
    } catch (err: any) { (window as any).customAlert(`${t("操作失敗")}: ${err}`); }
    finally { setLoadingServices(prev => ({ ...prev, [key]: false })); }
  };

  const handlePHPProcessChange = async (majorMin: string, count: number) => {
    if (!config) return;
    const newCfg = { ...config };
    if (!newCfg.global.php) newCfg.global.php = { processes: {} };
    newCfg.global.php.processes[majorMin] = count;
    try { await SaveConfig(newCfg); setConfig(newCfg); }
    catch (err) { (window as any).customAlert(`${t("保存設定失敗")}: ${err}`); }
  };

  const isRunning = (key: string) => !!servicesStatus[key];

  // ─── Reusable Styles ──────────────────────────────────────
  const cardStyle: React.CSSProperties = {
    background: 'var(--card)',
    border: '1px solid var(--border)',
    borderRadius: 'var(--radius-lg)',
    padding: 20,
    boxShadow: 'var(--shadow-sm)',
  };

  const sectionTitleStyle: React.CSSProperties = {
    fontFamily: 'var(--font-display)',
    fontSize: 13,
    fontWeight: 700,
    color: 'var(--fg)',
    letterSpacing: '0.05em',
    textTransform: 'uppercase' as const,
  };

  return (
    <div className="p-6 overflow-y-auto h-full space-y-6">

      {/* ─── Missing Dependencies Banner ────────────────────── */}
      {(missingCore.caddy || missingCore.php) && !dismissBanner && (
        <div className="relative rounded-xl p-4 pr-10 md:pr-14 flex flex-col md:flex-row justify-between items-start md:items-center gap-4" style={{ background: 'var(--status-error-bg)', border: '1px solid var(--status-error)', boxShadow: 'var(--shadow-md)' }}>
          <div className="flex items-center gap-3 flex-1 min-w-0">
            <div className="p-2.5 rounded-lg shrink-0" style={{ background: 'var(--status-error-bg)', color: 'var(--status-error)' }}>
              <AlertTriangle size={18} />
            </div>
            <div className="flex-1 min-w-0">
              <span className="font-bold block text-sm" style={{ color: 'var(--status-error)' }}>⚠️ {t("偵測到核心依賴元件缺失")}</span>
              <span className="text-xs mt-0.5 block" style={{ color: 'var(--fg-2)' }}>
                {t("本機未安裝：")}
                {[missingCore.caddy && t("Caddy Web 伺服器"), missingCore.php && t("PHP 執行環境")].filter(Boolean).join(t("、"))}{t("。請先完成依賴安裝以確保專案與服務正常運作。")}
              </span>
            </div>
          </div>
          <div className="flex items-center gap-3 w-full md:w-auto shrink-0">
            <button onClick={() => setShowDepManager(true)} className="flex-1 md:flex-none px-4 py-2 rounded-lg text-xs font-semibold flex items-center justify-center gap-1.5 transition whitespace-nowrap" style={{ background: 'var(--status-error)', color: '#fff' }}>
              <Package size={14} /> {t("立即下載")}
            </button>
          </div>
          <button onClick={() => setDismissBanner(true)} className="absolute top-3 right-3 p-1 rounded-lg transition" style={{ color: 'var(--muted)' }} title={t("關閉")}>
            <X size={16} />
          </button>
        </div>
      )}

      {/* ─── Header ─────────────────────────────────────────── */}
      <div className="flex justify-between items-center select-none">
        <div className="flex items-baseline gap-3">
          <h1 className="text-xl font-bold tracking-tight" style={{ color: 'var(--fg)', fontFamily: 'var(--font-display)' }}>{t("儀表板")}</h1>
          <p className="text-xs" style={{ color: 'var(--muted)' }}>{t("管理 Caddy, MariaDB, PHP-CGI 與背景開發服務")}</p>
        </div>
        <div className="flex gap-2.5">
          <button onClick={() => setShowDepManager(true)} className="px-3 py-2 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition duration-200" style={{ background: 'var(--card)', border: '1px solid var(--border)', color: 'var(--fg-2)' }}>
            <Package size={13} style={{ color: 'var(--status-info)' }} />
            <span>{t("依賴庫管理")}</span>
          </button>
          <button onClick={handleScan} disabled={isScanning} className="px-3 py-2 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition duration-200" style={{ background: 'var(--card)', border: '1px solid var(--border)', color: 'var(--fg-2)', opacity: isScanning ? 0.5 : 1 }}>
            <RefreshCw size={13} className={isScanning ? 'animate-spin' : ''} />
            {isScanning ? t("掃描中...") : t("重新掃描服務")}
          </button>
        </div>
      </div>



      {/* ─── Core System Services ───────────────────────────── */}
      <div className="space-y-4">
        <div className="flex items-center gap-2 select-none border-b pb-2" style={{ borderColor: 'var(--border-soft)' }}>
          <LayoutGrid size={15} style={{ color: 'var(--status-info)' }} />
          <h3 className="font-bold text-sm" style={{ color: 'var(--fg)', fontFamily: 'var(--font-display)' }}>{t("核心系統服務")}</h3>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {/* Caddy */}
          {(() => {
            const caddy = scanResult?.CaddyList?.[0];
            const running = isRunning('caddy');
            const loadingStart = loadingServices['caddy-start'];
            const loadingStop = loadingServices['caddy-stop'];
            const loadingReload = loadingServices['caddy-reload'];

            return (
              <div className="rounded-xl flex flex-col justify-between relative overflow-hidden transition-all duration-200" style={{ ...cardStyle, borderColor: running ? 'var(--border-strong)' : 'var(--border)' }}>
                <div className="flex justify-end items-center gap-1.5 select-none text-[11px] mb-1.5">
                  <span className="relative flex h-2 w-2">
                    {running && <span className="animate-ping absolute inline-flex h-full w-full rounded-full opacity-75" style={{ background: 'var(--status-ok)' }}></span>}
                    <span className="relative inline-flex rounded-full h-2 w-2" style={{ background: running ? 'var(--status-ok)' : 'var(--meta)' }}></span>
                  </span>
                  <span className="font-bold" style={{ color: running ? 'var(--status-ok)' : 'var(--muted)' }}>
                    {running ? t("運行中") : t("已停止")}
                  </span>
                </div>

                <div className="flex items-start gap-4">
                  <div className="p-2.5 rounded-lg" style={{ background: running ? 'var(--status-info-bg)' : 'var(--surface)', color: running ? 'var(--status-info)' : 'var(--muted)' }}>
                    <Server size={22} />
                  </div>
                  <div className="space-y-1">
                    <h4 className="font-bold text-sm" style={{ color: 'var(--fg)' }}>{t("Caddy 反向代理")}</h4>
                    <p className="text-[11px] font-medium" style={{ color: 'var(--meta)' }}>{t("版本: ")}{caddy ? caddy.Version : t("未安裝")}</p>
                    <p className="text-[11px]" style={{ color: 'var(--muted)', fontFamily: 'var(--font-mono)' }}>{t("埠口: ")}80, 443, 2019</p>
                  </div>
                </div>

                <div className="mt-5 pt-3.5 flex gap-2" style={{ borderTop: '1px solid var(--border-soft)' }}>
                  {!running ? (
                    <button onClick={() => handleServiceAction('caddy', 'start', caddy)} disabled={loadingStart || !caddy} className="w-full py-1.5 rounded-lg text-xs font-semibold flex items-center justify-center gap-1 transition select-none" style={{ background: 'var(--status-ok)', color: '#fff', opacity: loadingStart || !caddy ? 0.5 : 1 }}>
                      <Play size={12} /> {loadingStart ? t("啟動中...") : t("啟動服務")}
                    </button>
                  ) : (
                    <>
                      <button onClick={() => handleServiceAction('caddy', 'stop', caddy)} disabled={loadingStop} className="flex-1 py-1.5 rounded-lg text-xs font-semibold flex items-center justify-center gap-1 transition select-none" style={{ background: 'var(--status-error-bg)', border: '1px solid var(--status-error)', color: 'var(--status-error)' }}>
                        <Square size={12} /> {loadingStop ? t("停止中...") : t("停止")}
                      </button>
                      <button onClick={() => handleServiceAction('caddy', 'reload', caddy)} disabled={loadingReload} className="flex-1 py-1.5 rounded-lg text-xs font-semibold flex items-center justify-center gap-1 transition select-none" style={{ background: 'var(--input-bg)', border: '1px solid var(--border)', color: 'var(--fg-2)' }}>
                        <RefreshCw size={12} /> {loadingReload ? t("重載中...") : t("重載")}
                      </button>
                    </>
                  )}
                </div>
              </div>
            );
          })()}

          {/* MariaDB */}
          {(() => {
            const mariadb = scanResult?.MariaDBList?.[0];
            const serviceKey = mariadb ? `mariadb-${mariadb.Version}` : 'mariadb-none';
            const running = isRunning(serviceKey);
            const loadingStart = loadingServices[`${serviceKey}-start`];
            const loadingStop = loadingServices[`${serviceKey}-stop`];
            const dbPort = config?.global?.mariadb_port || 3306;

            return (
              <div className="rounded-xl flex flex-col justify-between relative overflow-hidden transition-all duration-200" style={{ ...cardStyle, borderColor: running ? 'var(--border-strong)' : 'var(--border)' }}>
                <div className="flex justify-end items-center gap-1.5 select-none text-[11px] mb-1.5">
                  <span className="relative flex h-2 w-2">
                    {running && <span className="animate-ping absolute inline-flex h-full w-full rounded-full opacity-75" style={{ background: 'var(--status-ok)' }}></span>}
                    <span className="relative inline-flex rounded-full h-2 w-2" style={{ background: running ? 'var(--status-ok)' : 'var(--meta)' }}></span>
                  </span>
                  <span className="font-bold" style={{ color: running ? 'var(--status-ok)' : 'var(--muted)' }}>
                    {running ? t("運行中") : t("已停止")}
                  </span>
                </div>

                <div className="flex items-start gap-4">
                  <div className="p-2.5 rounded-lg" style={{ background: running ? 'var(--status-ok-bg)' : 'var(--surface)', color: running ? 'var(--status-ok)' : 'var(--muted)' }}>
                    <Database size={22} />
                  </div>
                  <div className="space-y-1">
                    <h4 className="font-bold text-sm" style={{ color: 'var(--fg)' }}>{t("MariaDB 資料庫")}</h4>
                    <p className="text-[11px] font-medium" style={{ color: 'var(--meta)' }}>{t("版本: ")}{mariadb ? mariadb.Version : t("未安裝")}</p>
                    <p className="text-[11px]" style={{ color: 'var(--muted)', fontFamily: 'var(--font-mono)' }}>{t("埠口: ")}{dbPort}</p>
                  </div>
                </div>

                <div className="mt-5 pt-3.5 flex gap-2" style={{ borderTop: '1px solid var(--border-soft)' }}>
                  {!running ? (
                    <button onClick={() => handleServiceAction(serviceKey, 'start', mariadb)} disabled={loadingStart || !mariadb} className="w-full py-1.5 rounded-lg text-xs font-semibold flex items-center justify-center gap-1 transition select-none" style={{ background: 'var(--status-ok)', color: '#fff', opacity: loadingStart || !mariadb ? 0.5 : 1 }}>
                      <Play size={12} /> {loadingStart ? t("啟動中...") : t("啟動服務")}
                    </button>
                  ) : (
                    <button onClick={() => handleServiceAction(serviceKey, 'stop', mariadb)} disabled={loadingStop} className="w-full py-1.5 rounded-lg text-xs font-semibold flex items-center justify-center gap-1 transition select-none" style={{ background: 'var(--status-error-bg)', border: '1px solid var(--status-error)', color: 'var(--status-error)' }}>
                      <Square size={12} /> {loadingStop ? t("停止中...") : t("停止服務")}
                    </button>
                  )}
                </div>
              </div>
            );
          })()}

          {/* Mailpit */}
          {(() => {
            const mailpit = scanResult?.MailpitList?.[0];
            const running = isRunning('mailpit');
            const loadingStart = loadingServices['mailpit-start'];
            const loadingStop = loadingServices['mailpit-stop'];
            const httpPort = config?.global?.mailpit_http_port || 8025;
            const smtpPort = config?.global?.mailpit_smtp_port || 1025;

            return (
              <div className="rounded-xl flex flex-col justify-between relative overflow-hidden transition-all duration-200" style={{ ...cardStyle, borderColor: running ? 'var(--border-strong)' : 'var(--border)' }}>
                <div className="flex justify-end items-center gap-1.5 select-none text-[11px] mb-1.5">
                  <span className="relative flex h-2 w-2">
                    {running && <span className="animate-ping absolute inline-flex h-full w-full rounded-full opacity-75" style={{ background: 'var(--status-ok)' }}></span>}
                    <span className="relative inline-flex rounded-full h-2 w-2" style={{ background: running ? 'var(--status-ok)' : 'var(--meta)' }}></span>
                  </span>
                  <span className="font-bold" style={{ color: running ? 'var(--status-ok)' : 'var(--muted)' }}>
                    {running ? t("運行中") : t("已停止")}
                  </span>
                </div>

                <div className="flex items-start gap-4">
                  <div className="p-2.5 rounded-lg" style={{ background: running ? 'var(--accent-muted)' : 'var(--surface)', color: running ? 'var(--accent)' : 'var(--muted)' }}>
                    <Cpu size={22} />
                  </div>
                  <div className="space-y-1">
                    <h4 className="font-bold text-sm" style={{ color: 'var(--fg)' }}>{t("Mailpit 測試郵件")}</h4>
                    <p className="text-[11px] font-medium" style={{ color: 'var(--meta)' }}>{t("版本: ")}{mailpit ? mailpit.Version : t("未安裝")}</p>
                    <p className="text-[11px]" style={{ color: 'var(--muted)', fontFamily: 'var(--font-mono)' }}>SMTP: {smtpPort} | HTTP: {httpPort}</p>
                  </div>
                </div>

                <div className="mt-5 pt-3.5 flex gap-2" style={{ borderTop: '1px solid var(--border-soft)' }}>
                  {!running ? (
                    <button onClick={() => handleServiceAction('mailpit', 'start', mailpit)} disabled={loadingStart || !mailpit} className="w-full py-1.5 rounded-lg text-xs font-semibold flex items-center justify-center gap-1 transition select-none" style={{ background: 'var(--status-ok)', color: '#fff', opacity: loadingStart || !mailpit ? 0.5 : 1 }}>
                      <Play size={12} /> {loadingStart ? t("啟動中...") : t("啟動服務")}
                    </button>
                  ) : (
                    <button onClick={() => handleServiceAction('mailpit', 'stop', mailpit)} disabled={loadingStop} className="w-full py-1.5 rounded-lg text-xs font-semibold flex items-center justify-center gap-1 transition select-none" style={{ background: 'var(--status-error-bg)', border: '1px solid var(--status-error)', color: 'var(--status-error)' }}>
                      <Square size={12} /> {loadingStop ? t("停止中...") : t("停止服務")}
                    </button>
                  )}
                </div>
              </div>
            );
          })()}
        </div>
      </div>

      {/* ─── PHP FastCGI ────────────────────────────────────── */}
      <div className="space-y-4 pt-2">
        <div className="flex items-center gap-2 select-none border-b pb-2" style={{ borderColor: 'var(--border-soft)' }}>
          <Server size={15} style={{ color: 'var(--status-ok)' }} />
          <h3 className="font-bold text-sm" style={{ color: 'var(--fg)', fontFamily: 'var(--font-display)' }}>{t("PHP FastCGI 伺服器 (多端口負載平衡)")}</h3>
        </div>

        {scanResult?.PHPList && scanResult.PHPList.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {scanResult.PHPList.map((php, idx) => {
              const serviceKey = `php-${php.Version}`;
              const running = isRunning(serviceKey);
              const loadingStart = loadingServices[`${serviceKey}-start`];
              const loadingStop = loadingServices[`${serviceKey}-stop`];
              const configuredCount = config?.global?.php?.processes?.[php.MajorMin] || config?.global?.php?.processes_per_version || 3;
              const startPort = php.PortBase || (30000 + parseInt(php.MajorMin.replace('.', '')) * 10);
              const endPort = startPort + configuredCount - 1;
              const portDisplay = configuredCount > 1 ? `${startPort} ~ ${endPort}` : `${startPort}`;

              return (
                <div key={`php-${idx}`} className="rounded-xl flex flex-col justify-between relative overflow-hidden transition-all duration-200" style={{ ...cardStyle, borderColor: running ? 'var(--border-strong)' : 'var(--border)' }}>
                  <div className="flex justify-end items-center gap-1.5 select-none text-[11px] mb-1.5">
                    <span className="relative flex h-2 w-2">
                      {running && <span className="animate-ping absolute inline-flex h-full w-full rounded-full opacity-75" style={{ background: 'var(--status-ok)' }}></span>}
                      <span className="relative inline-flex rounded-full h-2 w-2" style={{ background: running ? 'var(--status-ok)' : 'var(--meta)' }}></span>
                    </span>
                    <span className="font-bold" style={{ color: running ? 'var(--status-ok)' : 'var(--muted)' }}>
                      {running ? t("運行中") : t("已停止")}
                    </span>
                  </div>

                  <div className="flex items-start gap-4">
                    <div className="p-2.5 rounded-lg" style={{ background: running ? 'var(--status-ok-bg)' : 'var(--surface)', color: running ? 'var(--status-ok)' : 'var(--muted)' }}>
                      <Server size={22} />
                    </div>
                    <div className="space-y-1">
                      <h4 className="font-bold text-sm" style={{ color: 'var(--fg)' }}>PHP {php.Version}</h4>
                      <p className="text-[11px]" style={{ color: 'var(--muted)', fontFamily: 'var(--font-mono)' }}>{t("埠口: ")}{portDisplay}</p>
                    </div>
                  </div>

                  <div className="mt-4 space-y-3 pt-3" style={{ borderTop: '1px solid var(--border-soft)' }}>
                    <div className="flex items-center justify-between text-xs select-none">
                      <span className="font-semibold" style={{ color: 'var(--muted)' }}>{t("進程數量 (Processes)")}</span>
                      <select
                        disabled={running}
                        value={configuredCount}
                        onChange={(e) => handlePHPProcessChange(php.MajorMin, parseInt(e.target.value))}
                        className="rounded-lg px-2 py-1 outline-none transition cursor-pointer text-xs font-semibold"
                        style={{ background: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--fg)' }}
                      >
                        {[1, 2, 3, 5, 10, 20, 50, 100].map(n => (
                          <option key={n} value={n}>{t("%s 個進程", n)}</option>
                        ))}
                      </select>
                    </div>

                    <div className="flex gap-2">
                      {!running ? (
                        <button onClick={() => handleServiceAction(serviceKey, 'start', php)} disabled={loadingStart} className="w-full py-1.5 rounded-lg text-xs font-semibold flex items-center justify-center gap-1 transition select-none" style={{ background: 'var(--status-ok)', color: '#fff' }}>
                          <Play size={12} /> {loadingStart ? t("啟動中...") : t("啟動 PHP")}
                        </button>
                      ) : (
                        <button onClick={() => handleServiceAction(serviceKey, 'stop', php)} disabled={loadingStop} className="w-full py-1.5 rounded-lg text-xs font-semibold flex items-center justify-center gap-1 transition select-none" style={{ background: 'var(--status-error-bg)', border: '1px solid var(--status-error)', color: 'var(--status-error)' }}>
                          <Square size={12} /> {loadingStop ? t("停止中...") : t("停止 PHP")}
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        ) : (
          <div className="rounded-xl p-8 text-center text-xs select-none" style={{ ...cardStyle, color: 'var(--muted)' }}>
            {t("未偵測到任何已安裝的 PHP 版本。請將 PHP 解壓縮後放入 ./bin/php/ 目錄下。")}
          </div>
        )}
      </div>

      {/* ─── System Status Overview ────────────────────────── */}
      <div className="space-y-4 pt-2">
        <div className="flex items-center gap-2 select-none border-b pb-2" style={{ borderColor: 'var(--border-soft)' }}>
          <Layers size={15} style={{ color: 'var(--status-info)' }} />
          <h3 className="font-bold text-sm" style={{ color: 'var(--fg)', fontFamily: 'var(--font-display)' }}>{t("系統狀態概覽")}</h3>
        </div>

        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 select-none">
          {/* Dependencies Status */}
          <div style={cardStyle}>
            <div className="flex items-center justify-between" style={{ color: 'var(--muted)' }}>
              <span style={sectionTitleStyle}>{t("依賴元件狀態")}</span>
              <Package size={16} style={{ color: 'var(--status-info)' }} />
            </div>
            <div className="mt-2.5">
              {(() => {
                const hasCaddy = !!scanResult?.CaddyList?.length;
                const hasMariaDB = !!scanResult?.MariaDBList?.length;
                const hasPHP = !!scanResult?.PHPList?.length;
                const hasMailpit = !!scanResult?.MailpitList?.length;
                let readyCount = 0;
                if (hasCaddy) readyCount++;
                if (hasMariaDB) readyCount++;
                if (hasPHP) readyCount++;
                if (hasMailpit) readyCount++;
                const missing = [];
                if (!hasCaddy) missing.push('Caddy');
                if (!hasPHP) missing.push('PHP');
                if (!hasMariaDB) missing.push('MariaDB');
                if (!hasMailpit) missing.push('Mailpit');
                return (
                  <>
                    <span className="text-xl font-black tracking-tight" style={{ color: readyCount === 4 ? 'var(--fg)' : 'var(--status-warn)', fontFamily: 'var(--font-mono)' }}>
                      {t("%s / 4 已就緒", readyCount)}
                    </span>
                    <p className="text-[10px] mt-2 font-medium" style={{ color: 'var(--meta)' }}>
                      {readyCount === 4 ? t("所有核心依賴配置正常") : `${t("缺: ")}${missing.join(', ')}`}
                    </p>
                  </>
                );
              })()}
            </div>
          </div>

          {/* Port Conflicts */}
          <div style={cardStyle}>
            <div className="flex items-center justify-between" style={{ color: 'var(--muted)' }}>
              <span style={sectionTitleStyle}>{t("埠口衝突檢測")}</span>
              <AlertTriangle size={16} style={{ color: Object.values(portConflicts).some(Boolean) ? 'var(--status-error)' : 'var(--status-ok)' }} />
            </div>
            <div className="mt-2.5">
              {(() => {
                const conflicts = Object.keys(portConflicts).filter(port => portConflicts[port]);
                const hasConflict = conflicts.length > 0;
                return (
                  <>
                    <span className="text-xl font-black tracking-tight" style={{ color: hasConflict ? 'var(--status-error)' : 'var(--status-ok)', fontFamily: 'var(--font-mono)' }}>
                      {hasConflict ? t("%s 個衝突", conflicts.length) : t("無埠口衝突")}
                    </span>
                    <p className="text-[10px] mt-2 font-medium truncate" style={{ color: 'var(--meta)' }}>
                      {hasConflict ? `Port: ${conflicts.join(', ')} ${t("被佔用")}` : t("本機埠口使用正常")}
                    </p>
                  </>
                );
              })()}
            </div>
          </div>

          {/* Hosts Domains */}
          <div style={cardStyle}>
            <div className="flex items-center justify-between" style={{ color: 'var(--muted)' }}>
              <span style={sectionTitleStyle}>{t("Hosts 本地網域")}</span>
              <Layers size={16} style={{ color: 'var(--status-ok)' }} />
            </div>
            <div className="mt-2.5">
              {(() => {
                let domainCount = 0;
                config?.projects?.forEach((p: any) => { if (p.enabled && p.domains) domainCount += p.domains.length; });
                const autoUpdate = config?.global?.auto_update_hosts;
                return (
                  <>
                    <span className="text-xl font-black tracking-tight" style={{ color: 'var(--fg)', fontFamily: 'var(--font-mono)' }}>{t("%s 個網域", domainCount)}</span>
                    <p className="text-[10px] mt-2 font-medium" style={{ color: 'var(--meta)' }}>
                      {t("Hosts 自動同步: ")}{autoUpdate ? t("開啟") : t("關閉")}
                    </p>
                  </>
                );
              })()}
            </div>
          </div>

          {/* Projects Overview */}
          <div style={cardStyle}>
            <div className="flex items-center justify-between" style={{ color: 'var(--muted)' }}>
              <span style={sectionTitleStyle}>{t("託管專案概覽")}</span>
              <Folder size={16} style={{ color: 'var(--accent)' }} />
            </div>
            <div className="mt-2.5">
              {(() => {
                const total = config?.projects?.length || 0;
                const enabled = config?.projects?.filter((p: any) => p.enabled).length || 0;
                const rate = total > 0 ? Math.round((enabled / total) * 100) : 0;
                return (
                  <>
                    <span className="text-xl font-black tracking-tight" style={{ color: 'var(--fg)', fontFamily: 'var(--font-mono)' }}>{t("%s / %s 啟用", enabled, total)}</span>
                    <p className="text-[10px] mt-2 font-medium" style={{ color: 'var(--meta)' }}>
                      {t("專案啟用率: ")}{rate}%
                    </p>
                  </>
                );
              })()}
            </div>
          </div>
        </div>
      </div>

      <DependencyManager isOpen={showDepManager} onClose={() => setShowDepManager(false)} onInstalled={handleScan} />
    </div>
  );
}

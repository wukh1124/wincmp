import React, { useState, useEffect } from 'react';
import { Cpu, HardDrive, Server, Layers, Activity, RefreshCw } from 'lucide-react';
import { GetDetailedResources } from '../../wailsjs/go/main/App';
import { t, useLanguage } from '../i18n';

interface ProcessResource {
  cpu: number;
  ram: number;
}

interface ServiceResource {
  name: string;
  cpu: number;
  ram: number;
  pids: number[];
}

interface DetailedResources {
  systemCpu: number;
  core: ProcessResource;
  web: ProcessResource;
  services: Record<string, ServiceResource>;
}

export default function ResourceMonitor() {
  useLanguage(); // 訂閱語系變更
  const [data, setData] = useState<DetailedResources | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isRefreshing, setIsRefreshing] = useState(false);

  const fetchResources = async (silent = false) => {
    if (!silent) setIsRefreshing(true);
    try {
      const res = await GetDetailedResources();
      if (res) {
        setData(res);
        setError(null);
      }
    } catch (err: any) {
      console.error('獲取資源監控明細失敗:', err);
      setError(String(err));
    } finally {
      setLoading(false);
      setIsRefreshing(false);
    }
  };

  useEffect(() => {
    // 首次載入
    fetchResources();

    // 定時輪詢 (每 2 秒)
    const timer = setInterval(() => {
      fetchResources(true);
    }, 2000);

    return () => {
      clearInterval(timer);
    };
  }, []);

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-3 select-none" style={{ color: 'var(--muted)' }}>
        <RefreshCw size={24} className="animate-spin" style={{ color: 'var(--accent)' }} />
        <span className="text-xs font-semibold">{t("正在載入系統與服務資源數據...")}</span>
      </div>
    );
  }

  if (error && !data) {
    return (
      <div className="p-6 text-center select-none">
        <div
          className="rounded-xl p-6 inline-block max-w-md"
          style={{ background: 'var(--status-error-bg)', border: '1px solid color-mix(in srgb, var(--status-error) 20%, transparent)' }}
        >
          <span className="font-bold block mb-2" style={{ color: 'var(--status-error)' }}>
            ⚠️ {t("載入資源監控失敗")}
          </span>
          <span className="text-xs block break-all font-mono mb-4" style={{ color: 'color-mix(in srgb, var(--status-error) 80%, transparent)' }}>
            {error}
          </span>
          <button
            onClick={() => {
              setLoading(true);
              fetchResources();
            }}
            className="px-4 py-2 rounded-lg text-xs font-semibold transition"
            style={{ background: 'var(--status-error)', color: 'var(--accent-on)' }}
          >
            {t("重新嘗試")}
          </button>
        </div>
      </div>
    );
  }

  const core = data?.core ?? { cpu: 0, ram: 0 };
  const web = data?.web ?? { cpu: 0, ram: 0 };
  const services = data?.services ?? {};
  const svcKeys = Object.keys(services);

  // 計算加總
  let totalCpu = core.cpu + web.cpu;
  let totalRam = core.ram + web.ram;
  svcKeys.forEach(key => {
    totalCpu += services[key].cpu;
    totalRam += services[key].ram;
  });

  return (
    <div className="p-6 overflow-y-auto h-full space-y-6 select-none">

      {/* 標頭 */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-xl font-bold tracking-tight flex items-center gap-2" style={{ color: 'var(--fg)' }}>
            {t("資源監控")}
          </h1>
          <p className="text-xs mt-1" style={{ color: 'var(--muted)' }}>
            {t("即時監控系統總體、主程式、Web 視窗介面及各背景服務的 CPU 與 RAM 使用狀況")}
          </p>
        </div>
        <button
          onClick={() => fetchResources()}
          disabled={isRefreshing}
          className={`px-3 py-2 rounded-lg text-xs font-semibold border flex items-center gap-1.5 transition duration-200 ${isRefreshing ? 'opacity-50' : ''}`}
          style={{
            borderColor: 'var(--border)',
            background: 'var(--card)',
            color: 'var(--fg-2)',
          }}
        >
          <RefreshCw size={13} className={isRefreshing ? 'animate-spin' : ''} />
          <span>{isRefreshing ? t('整理中...') : t('手動整理')}</span>
        </button>
      </div>

      {/* 1. WinCMP 環境總體佔用 */}
      <div
        className="rounded-xl p-5 grid grid-cols-1 md:grid-cols-2 gap-6 items-center"
        style={{ background: 'var(--card)', border: '1px solid var(--border)', boxShadow: 'var(--shadow-sm)' }}
      >
        {/* CPU */}
        <div className="space-y-2">
          <div className="flex justify-between items-center text-xs font-semibold uppercase tracking-wider" style={{ color: 'var(--muted)' }}>
            <span className="flex items-center gap-1.5">
              <Cpu size={14} style={{ color: 'var(--status-info)' }} /> {t("WinCMP CPU 總佔用")}
            </span>
            <span className="font-mono text-sm" style={{ color: 'var(--fg)' }}>{totalCpu.toFixed(1)}%</span>
          </div>
          <div className="w-full h-2.5 rounded-full overflow-hidden" style={{ background: 'var(--input-bg)' }}>
            <div
              style={{
                width: `${Math.min(totalCpu, 100)}%`,
                height: '100%',
                borderRadius: '9999px',
                transition: 'all 0.5s',
                background: totalCpu > 80 ? 'var(--status-error)' : totalCpu > 50 ? 'var(--status-warn)' : 'var(--status-info)',
              }}
            />
          </div>
        </div>

        {/* RAM */}
        <div className="space-y-2">
          <div className="flex justify-between items-center text-xs font-semibold uppercase tracking-wider" style={{ color: 'var(--muted)' }}>
            <span className="flex items-center gap-1.5">
              <HardDrive size={14} style={{ color: 'var(--accent-muted)' }} /> {t("WinCMP 記憶體 (RAM) 總佔用")}
            </span>
            <span className="font-mono text-sm" style={{ color: 'var(--fg)' }}>{totalRam} MB</span>
          </div>
          <div className="w-full h-2.5 rounded-full overflow-hidden" style={{ background: 'var(--input-bg)' }}>
            <div
              style={{
                width: `${Math.min(totalRam / 5, 100)}%`,
                height: '100%',
                borderRadius: '9999px',
                transition: 'all 0.5s',
                background: 'var(--accent)',
              }}
            />
          </div>
        </div>
      </div>

      {/* 2. 核心與介面資源 */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">

        {/* WinCMP 核心 (Go 後端) */}
        <div
          className="rounded-xl p-5 flex flex-col justify-between transition duration-200 relative overflow-hidden"
          style={{
            background: 'var(--card)',
            border: '1px solid var(--border)',
            boxShadow: 'var(--shadow-sm)',
          }}
          onMouseEnter={e => (e.currentTarget.style.borderColor = 'var(--border-strong)')}
          onMouseLeave={e => (e.currentTarget.style.borderColor = 'var(--border)')}
        >
          <div>
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2 rounded-lg" style={{ background: 'var(--status-info-bg)', color: 'var(--status-info)' }}>
                <Server size={18} />
              </div>
              <div>
                <h4 className="font-bold text-sm" style={{ color: 'var(--fg)' }}>{t("WinCMP 核心 (Go 後端)")}</h4>
                <p className="text-[10px] font-semibold uppercase tracking-wider" style={{ color: 'var(--meta)' }}>Core Process</p>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4 text-xs">
              <div className="space-y-1">
                <span className="font-semibold" style={{ color: 'var(--meta)' }}>{t("CPU 佔用率")}</span>
                <span className="block font-bold text-sm" style={{ color: 'var(--fg)' }}>{core.cpu.toFixed(1)}%</span>
              </div>
              <div className="space-y-1">
                <span className="font-semibold" style={{ color: 'var(--meta)' }}>{t("記憶體 (RAM)")}</span>
                <span className="block font-bold text-sm" style={{ color: 'var(--fg)' }}>{core.ram} MB</span>
              </div>
            </div>
          </div>
          <div className="mt-5 pt-3" style={{ borderTop: '1px solid var(--border-soft)' }}>
            <div className="w-full h-1.5 rounded-full overflow-hidden" style={{ background: 'var(--input-bg)' }}>
              <div
                style={{
                  width: `${Math.min(core.cpu * 5, 100)}%`,
                  height: '100%',
                  borderRadius: '9999px',
                  transition: 'all 0.5s',
                  background: 'var(--status-info)',
                }}
              />
            </div>
          </div>
        </div>

        {/* Web 視窗介面 (WebView2) */}
        <div
          className="rounded-xl p-5 flex flex-col justify-between transition duration-200 relative overflow-hidden"
          style={{
            background: 'var(--card)',
            border: '1px solid var(--border)',
            boxShadow: 'var(--shadow-sm)',
          }}
          onMouseEnter={e => (e.currentTarget.style.borderColor = 'var(--border-strong)')}
          onMouseLeave={e => (e.currentTarget.style.borderColor = 'var(--border)')}
        >
          <div>
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2 rounded-lg" style={{ background: 'var(--accent-muted)', color: 'var(--accent)' }}>
                <Layers size={18} />
              </div>
              <div>
                <h4 className="font-bold text-sm" style={{ color: 'var(--fg)' }}>{t("Web 視窗介面 (WebView2)")}</h4>
                <p className="text-[10px] font-semibold uppercase tracking-wider" style={{ color: 'var(--meta)' }}>UI Render Engine</p>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4 text-xs">
              <div className="space-y-1">
                <span className="font-semibold" style={{ color: 'var(--meta)' }}>{t("CPU 佔用率")}</span>
                <span className="block font-bold text-sm" style={{ color: 'var(--fg)' }}>{web.cpu.toFixed(1)}%</span>
              </div>
              <div className="space-y-1">
                <span className="font-semibold" style={{ color: 'var(--meta)' }}>{t("記憶體 (RAM)")}</span>
                <span className="block font-bold text-sm" style={{ color: 'var(--fg)' }}>{web.ram} MB</span>
              </div>
            </div>
          </div>
          <div className="mt-5 pt-3" style={{ borderTop: '1px solid var(--border-soft)' }}>
            <div className="w-full h-1.5 rounded-full overflow-hidden" style={{ background: 'var(--input-bg)' }}>
              <div
                style={{
                  width: `${Math.min(web.cpu * 5, 100)}%`,
                  height: '100%',
                  borderRadius: '9999px',
                  transition: 'all 0.5s',
                  background: 'var(--accent)',
                }}
              />
            </div>
          </div>
        </div>

      </div>

      {/* 3. 啟動中的依賴服務 */}
      <div className="space-y-4 pt-2">
        <div className="flex items-center gap-2 pb-2" style={{ borderBottom: '1px solid var(--border-soft)' }}>
          <Server size={15} style={{ color: 'var(--status-ok)' }} />
          <h3 className="font-bold text-sm" style={{ color: 'var(--fg-2)' }}>{t("啟動中的依賴服務")}</h3>
        </div>

        {svcKeys.length > 0 ? (
          <div className="rounded-xl overflow-hidden" style={{ background: 'var(--card)', border: '1px solid var(--border)', boxShadow: 'var(--shadow-sm)' }}>
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr
                    className="border-b text-[10px] font-bold uppercase tracking-wider"
                    style={{ borderColor: 'var(--border)', background: 'var(--bg-deep)', color: 'var(--meta)' }}
                  >
                    <th className="px-5 py-3">{t("服務名稱")}</th>
                    <th className="px-5 py-3">{t("狀態")}</th>
                    <th className="px-5 py-3">{t("CPU 佔用")}</th>
                    <th className="px-5 py-3">{t("記憶體 (RAM)")}</th>
                    <th className="px-5 py-3 font-mono">{t("進程 ID (PIDs)")}</th>
                  </tr>
                </thead>
                <tbody className="text-xs font-semibold" style={{ color: 'var(--fg-2)' }}>
                  {svcKeys.map((key) => {
                    const svc = services[key];
                    return (
                      <tr
                        key={key}
                        className="transition duration-150"
                        style={{ borderBottom: '1px solid var(--border-soft)' }}
                        onMouseEnter={e => (e.currentTarget.style.background = 'var(--card-hover)')}
                        onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                      >
                        <td className="px-5 py-4 flex items-center gap-2">
                          <span className="font-bold" style={{ color: 'var(--fg)' }}>{svc.name}</span>
                        </td>
                        <td className="px-5 py-4">
                          <span
                            className="inline-flex items-center gap-1 text-[11px] px-2 py-0.5 rounded-full"
                            style={{
                              color: 'var(--status-ok)',
                              background: 'var(--status-ok-bg)',
                              border: '1px solid color-mix(in srgb, var(--status-ok) 15%, transparent)',
                            }}
                          >
                            <span className="relative flex h-1.5 w-1.5">
                              <span className="animate-ping absolute inline-flex h-full w-full rounded-full opacity-75" style={{ background: 'var(--status-ok)' }}></span>
                              <span className="relative inline-flex rounded-full h-1.5 w-1.5" style={{ background: 'var(--status-ok)' }}></span>
                            </span>
                            {t("運行中")}
                          </span>
                        </td>
                        <td className="px-5 py-4 font-mono" style={{ color: 'var(--fg)' }}>{svc.cpu.toFixed(1)}%</td>
                        <td className="px-5 py-4 font-mono" style={{ color: 'var(--fg)' }}>{svc.ram} MB</td>
                        <td className="px-5 py-4 font-mono text-[10px]" style={{ color: 'var(--muted)' }}>
                          [{svc.pids.join(', ')}]
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        ) : (
          <div
            className="rounded-xl p-8 text-center select-none text-xs"
            style={{ background: 'var(--card)', border: '1px solid var(--border)', color: 'var(--meta)' }}
          >
            {t("目前沒有啟動中的子服務。請前往「儀表板」啟動 Caddy、PHP 或 MariaDB 服務。")}
          </div>
        )}
      </div>

    </div>
  );
}

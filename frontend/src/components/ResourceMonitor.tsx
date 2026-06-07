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
      <div className="flex flex-col items-center justify-center h-full text-gray-400 gap-3 select-none">
        <RefreshCw size={24} className="animate-spin text-blue-500" />
        <span className="text-xs font-semibold">{t("正在載入系統與服務資源數據...")}</span>
      </div>
    );
  }

  if (error && !data) {
    return (
      <div className="p-6 text-center select-none">
        <div className="bg-red-500/10 border border-red-500/20 rounded-xl p-6 inline-block max-w-md">
          <span className="text-red-400 font-bold block mb-2">⚠️ {t("載入資源監控失敗")}</span>
          <span className="text-xs text-red-300/80 block break-all font-mono mb-4">{error}</span>
          <button
            onClick={() => {
              setLoading(true);
              fetchResources();
            }}
            className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg text-xs font-semibold transition"
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
          <h1 className="text-xl font-bold tracking-tight text-white flex items-center gap-2">
            <Activity className="text-blue-500" size={20} /> {t("資源監控 (Resource Monitor)")}
          </h1>
          <p className="text-xs text-gray-400 mt-1">{t("即時監控系統總體、主程式、Web 視窗介面及各背景服務的 CPU 與 RAM 使用狀況")}</p>
        </div>
        <button
          onClick={() => fetchResources()}
          disabled={isRefreshing}
          className={`px-3 py-2 rounded-lg text-xs font-semibold border border-darkBorder flex items-center gap-1.5 bg-darkCard hover:bg-opacity-80 transition duration-200 ${isRefreshing ? 'opacity-50' : ''
            }`}
        >
          <RefreshCw size={13} className={isRefreshing ? 'animate-spin' : ''} />
          <span>{isRefreshing ? t('整理中...') : t('手動整理')}</span>
        </button>
      </div>

      {/* 1. WinCMP 環境總體佔用 */}
      <div className="bg-darkCard border border-darkBorder rounded-xl p-5 shadow-sm grid grid-cols-1 md:grid-cols-2 gap-6 items-center">
        {/* CPU */}
        <div className="space-y-2">
          <div className="flex justify-between items-center text-xs font-semibold text-gray-400 uppercase tracking-wider">
            <span className="flex items-center gap-1.5">
              <Cpu size={14} className="text-blue-400" /> {t("WinCMP CPU 總佔用")}
            </span>
            <span className="font-mono text-white text-sm">{totalCpu.toFixed(1)}%</span>
          </div>
          <div className="w-full h-2.5 bg-darkInput rounded-full overflow-hidden">
            <div
              style={{ width: `${Math.min(totalCpu, 100)}%` }}
              className={`h-full rounded-full transition-all duration-500 ${
                totalCpu > 80 ? 'bg-red-500' : totalCpu > 50 ? 'bg-yellow-500' : 'bg-blue-500'
              }`}
            />
          </div>
        </div>

        {/* RAM */}
        <div className="space-y-2">
          <div className="flex justify-between items-center text-xs font-semibold text-gray-400 uppercase tracking-wider">
            <span className="flex items-center gap-1.5">
              <HardDrive size={14} className="text-indigo-400" /> {t("WinCMP 記憶體 (RAM) 總佔用")}
            </span>
            <span className="font-mono text-white text-sm">{totalRam} MB</span>
          </div>
          <div className="w-full h-2.5 bg-darkInput rounded-full overflow-hidden">
            <div
              style={{ width: `${Math.min(totalRam / 5, 100)}%` }}
              className="h-full bg-indigo-500 rounded-full transition-all duration-500"
            />
          </div>
        </div>
      </div>

      {/* 2. 核心與介面資源 */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">

        {/* WinCMP 核心 (Go 後端) */}
        <div className="bg-darkCard border border-darkBorder rounded-xl p-5 flex flex-col justify-between hover:border-gray-700/80 transition duration-200 shadow-sm relative overflow-hidden">
          <div>
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2 bg-blue-500/10 text-blue-400 rounded-lg">
                <Server size={18} />
              </div>
              <div>
                <h4 className="font-bold text-sm text-gray-100">{t("WinCMP 核心 (Go 後端)")}</h4>
                <p className="text-[10px] text-gray-500 font-semibold uppercase tracking-wider">Core Process</p>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4 text-xs">
              <div className="space-y-1">
                <span className="text-gray-500 font-semibold">{t("CPU 佔用率")}</span>
                <span className="block font-bold text-gray-200 text-sm">{core.cpu.toFixed(1)}%</span>
              </div>
              <div className="space-y-1">
                <span className="text-gray-500 font-semibold">{t("記憶體 (RAM)")}</span>
                <span className="block font-bold text-gray-200 text-sm">{core.ram} MB</span>
              </div>
            </div>
          </div>
          <div className="mt-5 pt-3 border-t border-darkBorder/40">
            <div className="w-full h-1.5 bg-darkInput rounded-full overflow-hidden">
              <div
                style={{ width: `${Math.min(core.cpu * 5, 100)}%` }}
                className="h-full bg-blue-500 rounded-full transition-all duration-500"
              />
            </div>
          </div>
        </div>

        {/* Web 視窗介面 (WebView2) */}
        <div className="bg-darkCard border border-darkBorder rounded-xl p-5 flex flex-col justify-between hover:border-gray-700/80 transition duration-200 shadow-sm relative overflow-hidden">
          <div>
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2 bg-indigo-500/10 text-indigo-400 rounded-lg">
                <Layers size={18} />
              </div>
              <div>
                <h4 className="font-bold text-sm text-gray-100">{t("Web 視窗介面 (WebView2)")}</h4>
                <p className="text-[10px] text-gray-500 font-semibold uppercase tracking-wider">UI Render Engine</p>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4 text-xs">
              <div className="space-y-1">
                <span className="text-gray-500 font-semibold">{t("CPU 佔用率")}</span>
                <span className="block font-bold text-gray-200 text-sm">{web.cpu.toFixed(1)}%</span>
              </div>
              <div className="space-y-1">
                <span className="text-gray-500 font-semibold">{t("記憶體 (RAM)")}</span>
                <span className="block font-bold text-gray-200 text-sm">{web.ram} MB</span>
              </div>
            </div>
          </div>
          <div className="mt-5 pt-3 border-t border-darkBorder/40">
            <div className="w-full h-1.5 bg-darkInput rounded-full overflow-hidden">
              <div
                style={{ width: `${Math.min(web.cpu * 5, 100)}%` }}
                className="h-full bg-indigo-500 rounded-full transition-all duration-500"
              />
            </div>
          </div>
        </div>

      </div>

      {/* 3. 啟動中的依賴服務 */}
      <div className="space-y-4 pt-2">
        <div className="flex items-center gap-2 border-b border-darkBorder/40 pb-2">
          <Server size={15} className="text-green-500" />
          <h3 className="font-bold text-sm text-gray-300">{t("啟動中的依賴服務 (Services Stack)")}</h3>
        </div>

        {svcKeys.length > 0 ? (
          <div className="bg-darkCard border border-darkBorder rounded-xl overflow-hidden shadow-sm">
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="border-b border-darkBorder bg-black bg-opacity-20 text-gray-500 text-[10px] font-bold uppercase tracking-wider">
                    <th className="px-5 py-3">{t("服務名稱")}</th>
                    <th className="px-5 py-3">{t("狀態")}</th>
                    <th className="px-5 py-3">{t("CPU 佔用")}</th>
                    <th className="px-5 py-3">{t("記憶體 (RAM)")}</th>
                    <th className="px-5 py-3 font-mono">{t("進程 ID (PIDs)")}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-darkBorder/40 text-xs font-semibold text-gray-300">
                  {svcKeys.map((key) => {
                    const svc = services[key];
                    return (
                      <tr key={key} className="hover:bg-white/5 transition duration-150">
                        <td className="px-5 py-4 flex items-center gap-2">
                          <span className="font-bold text-gray-200">{svc.name}</span>
                        </td>
                        <td className="px-5 py-4">
                          <span className="inline-flex items-center gap-1 text-[11px] text-green-400 bg-green-500/10 px-2 py-0.5 rounded-full border border-green-500/15">
                            <span className="relative flex h-1.5 w-1.5">
                              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                              <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-green-500"></span>
                            </span>
                            {t("運行中")}
                          </span>
                        </td>
                        <td className="px-5 py-4 font-mono text-gray-100">{svc.cpu.toFixed(1)}%</td>
                        <td className="px-5 py-4 font-mono text-gray-100">{svc.ram} MB</td>
                        <td className="px-5 py-4 font-mono text-gray-400 text-[10px]">
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
          <div className="bg-darkCard border border-darkBorder rounded-xl p-8 text-center text-gray-500 select-none text-xs">
            {t("目前沒有啟動中的子服務。請前往「儀表板」啟動 Caddy、PHP 或 MariaDB 服務。")}
          </div>
        )}
      </div>

    </div>
  );
}

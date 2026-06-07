import React, { useState, useEffect, useRef } from 'react';
import { Home, Folder, Database, Settings as SettingsIcon, Terminal, Cpu, HardDrive, ChevronLeft, ChevronRight, ChevronUp, ChevronDown, Shield } from 'lucide-react';
import Dashboard from './components/Dashboard';
import Projects from './components/Projects';
import DBExplorer from './components/DBExplorer';
import Settings from './components/Settings';
import ResourceMonitor from './components/ResourceMonitor';
import TerminalLogs from './components/TerminalLogs';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { GetAppVersion, IsAdmin, GetConfig } from '../wailsjs/go/main/App';
import logo from './assets/images/icon.svg';
import { t, setLanguage, useLanguage } from './i18n';

export default function App() {
  useLanguage(); // 訂閱語系變更

  const [activeTab, setActiveTab] = useState<'dashboard' | 'projects' | 'db_explorer' | 'resources' | 'settings' | 'logs'>('dashboard');
  const [showLogs, setShowLogs] = useState(true);
  const [systemResources, setSystemResources] = useState({ cpu: 0, memory: 0 });
  const [isCollapsed, setIsCollapsed] = useState(() => localStorage.getItem('sidebar_collapsed') === 'true');
  const [customAlert, setCustomAlert] = useState<{ isOpen: boolean; message: string; resolve?: () => void }>({ isOpen: false, message: '' });
  const [customConfirm, setCustomConfirm] = useState<{ isOpen: boolean; message: string; resolve?: (value: boolean) => void }>({ isOpen: false, message: '' });

  // 新增版本號、管理員狀態與搜尋專案相關的 state
  const [isAdmin, setIsAdmin] = useState(false);
  const [appVersion, setAppVersion] = useState('v2.0.0');
  const [searchQuery, setSearchQuery] = useState('');
  const [isSearchFocused, setIsSearchFocused] = useState(false);
  const [config, setConfig] = useState<any>(null);
  const [highlightedProjectName, setHighlightedProjectName] = useState<string | null>(null);

  const searchInputRef = useRef<HTMLInputElement>(null);

  // 1. 初始化讀取版本與管理員權限及語系
  useEffect(() => {
    GetAppVersion().then(setAppVersion).catch((err: any) => console.error("獲取版本失敗:", err));
    IsAdmin().then(setIsAdmin).catch((err: any) => console.error("獲取管理員權限失敗:", err));
    GetConfig().then((cfg: any) => {
      if (cfg && cfg.global && cfg.global.language) {
        setLanguage(cfg.global.language);
      }
    }).catch((err: any) => console.error("獲取語系失敗:", err));
  }, []);

  // 2. 監聽 Ctrl+K 快速鍵聚焦搜尋框
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault();
        searchInputRef.current?.focus();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  // 3. 獲取專案清單 (搜尋使用)
  const refreshProjectsList = async () => {
    try {
      const cfg = await GetConfig();
      setConfig(cfg);
    } catch (err) {
      console.error("搜尋讀取設定失敗:", err);
    }
  };

  // 4. Tab 切換阻斷 (設定頁未儲存防禦機制)
  const handleTabChange = async (tabId: typeof activeTab) => {
    if (activeTab === 'settings' && tabId !== 'settings') {
      if ((window as any).isSettingsDirty) {
        const confirmLeave = await (window as any).customConfirm(t("您有尚未儲存的設定變更，確定要離開此頁面嗎？"));
        if (!confirmLeave) {
          return;
        }
      }
    }
    // 離開設定頁面時確保重置 dirty state
    (window as any).isSettingsDirty = false;
    setActiveTab(tabId);
  };

  // 覆寫與註冊漂亮的自訂彈窗，避免 wails.localhost 標題
  useEffect(() => {
    (window as any).customAlert = (message: any) => {
      return new Promise<void>((resolve) => {
        setCustomAlert({
          isOpen: true,
          message: String(message),
          resolve: () => {
            setCustomAlert({ isOpen: false, message: '', resolve: undefined });
            resolve();
          }
        });
      });
    };

    (window as any).customConfirm = (message: any) => {
      return new Promise<boolean>((resolve) => {
        setCustomConfirm({
          isOpen: true,
          message: String(message),
          resolve: (val: boolean) => {
            setCustomConfirm({ isOpen: false, message: '', resolve: undefined });
            resolve(val);
          }
        });
      });
    };

    window.alert = (message: any) => {
      (window as any).customAlert(message);
    };
  }, []);

  const toggleSidebar = () => {
    setIsCollapsed(prev => {
      const next = !prev;
      localStorage.setItem('sidebar_collapsed', String(next));
      return next;
    });
  };

  // 訂閱 Go 端推送的 CPU / RAM 資源佔用
  useEffect(() => {
    const handleResourceUpdate = (data: any) => {
      if (data) {
        setSystemResources({
          cpu: data.cpu || 0,
          memory: data.memory || 0
        });
      }
    };

    const unsubscribe = EventsOn('resource_usage', handleResourceUpdate);

    return () => {
      unsubscribe();
    };
  }, []);

  const renderActiveComponent = () => {
    switch (activeTab) {
      case 'dashboard':
        return <Dashboard />;
      case 'projects':
        return <Projects highlightedProjectName={highlightedProjectName} clearHighlight={() => setHighlightedProjectName(null)} />;
      case 'db_explorer':
        return <DBExplorer />;
      case 'resources':
        return <ResourceMonitor />;
      case 'settings':
        return <Settings />;
      case 'logs':
        return <TerminalLogs />;
      default:
        return <Dashboard />;
    }
  };

  const menuItems = [
    { id: 'dashboard', label: t('儀表板'), icon: <Home size={15} /> },
    { id: 'projects', label: t('專案管理'), icon: <Folder size={15} /> },
    { id: 'db_explorer', label: t('資料庫瀏覽'), icon: <Database size={15} /> },
    { id: 'resources', label: t('資源監控'), icon: <Cpu size={15} /> },
    { id: 'settings', label: t('系統設定'), icon: <SettingsIcon size={15} /> },
    { id: 'logs', label: t('終端日誌'), icon: <Terminal size={15} /> }
  ] as const;

  return (
    <div className="flex h-screen w-screen bg-darkBg text-gray-200 overflow-hidden font-sans select-none">

      {/* 1. 左側導航 Sidebar */}
      <aside className={`bg-[#0c0c0e] border-r border-darkBorder flex flex-col justify-between select-none transition-all duration-300 ease-in-out ${isCollapsed ? 'w-16' : 'w-64'}`}>
        <div>
          {/* Logo & 標題 */}
          <div className={`py-5 border-b border-darkBorder flex items-center transition-all duration-300 ${isCollapsed ? 'px-4 flex-col gap-3 justify-center' : 'px-6 gap-3 justify-between'}`}>
            <div className="flex items-center gap-3 overflow-hidden">
              <img src={logo} alt="WinCMP Logo" className="w-8 h-8 rounded-lg shadow-md shadow-blue-500/10 object-contain flex-shrink-0" />
              {!isCollapsed && (
                <div className="transition-opacity duration-300 opacity-100 whitespace-nowrap">
                  <span className="font-bold text-gray-100 tracking-wide block">WinCMP</span>
                  <span className="text-[10px] text-gray-500 block font-semibold uppercase tracking-wider">Local Dev Panel</span>
                </div>
              )}
            </div>
            <button
              onClick={toggleSidebar}
              className={`p-1.5 rounded-md text-gray-500 hover:text-gray-200 hover:bg-white/5 transition-colors ${isCollapsed ? 'w-8 h-8 flex items-center justify-center' : ''}`}
              title={isCollapsed ? t('展開側邊欄') : t('收起側邊欄')}
            >
              {isCollapsed ? <ChevronRight size={15} /> : <ChevronLeft size={15} />}
            </button>
          </div>

          {/* 選單列表 */}
          <nav className={`pt-4 space-y-1 transition-all duration-300 ${isCollapsed ? 'px-2' : 'p-3'}`}>
            {menuItems.map(item => (
              <button
                key={item.id}
                onClick={() => handleTabChange(item.id)}
                title={isCollapsed ? item.label.split(' ')[0] : undefined}
                className={`w-full text-left py-2.5 text-sm font-semibold flex items-center transition-all duration-150 ${isCollapsed ? 'justify-center px-0' : 'px-4 gap-3'
                  } ${activeTab === item.id
                    ? 'bg-blue-600/10 text-blue-400 border-l-[3px] border-blue-500 rounded-r-lg'
                    : 'text-gray-400 hover:bg-white/5 hover:text-gray-200 border-l-[3px] border-transparent rounded-r-lg'
                  }`}
              >
                <span className={`flex-shrink-0 ${activeTab === item.id ? 'text-blue-400' : 'text-gray-400'}`}>
                  {item.icon}
                </span>
                {!isCollapsed && <span className="whitespace-nowrap transition-opacity duration-300">{item.label}</span>}
              </button>
            ))}
          </nav>
        </div>

        {/* 底部系統監控狀態 */}
        <div className={`border-t border-darkBorder bg-black bg-opacity-20 transition-all duration-300 ${isCollapsed ? 'p-2 py-4 flex flex-col items-center gap-4' : 'p-4 space-y-3'}`}>
          {!isCollapsed ? (
            <>
              <div className="font-semibold text-[10px] text-gray-500 select-none uppercase tracking-wider">
                {t("系統監控 (WinCMP Core)")}
              </div>
              <div className="space-y-2.5 text-xs">
                {/* CPU */}
                <div className="space-y-1">
                  <div className="flex items-center justify-between">
                    <span className="text-gray-400 flex items-center gap-1.5">
                      <Cpu size={12} className="text-blue-400" /> {t("CPU 佔用")}
                    </span>
                    <span className="font-semibold text-gray-300">{systemResources.cpu.toFixed(1)}%</span>
                  </div>
                  <div className="w-full h-1 bg-darkInput rounded-full overflow-hidden">
                    <div
                      style={{ width: `${Math.min(systemResources.cpu * 5, 100)}%` }}
                      className="h-full bg-blue-500 rounded-full transition-all duration-500"
                    />
                  </div>
                </div>

                {/* RAM */}
                <div className="space-y-1 pt-0.5">
                  <div className="flex items-center justify-between">
                    <span className="text-gray-400 flex items-center gap-1.5">
                      <HardDrive size={12} className="text-indigo-400" /> {t("記憶體 (RAM)")}
                    </span>
                    <span className="font-semibold text-gray-300">{systemResources.memory} MB</span>
                  </div>
                  <div className="w-full h-1 bg-darkInput rounded-full overflow-hidden">
                    <div
                      style={{ width: `${Math.min(systemResources.memory / 2, 100)}%` }}
                      className="h-full bg-indigo-500 rounded-full transition-all duration-500"
                    />
                  </div>
                </div>
              </div>
            </>
          ) : (
            <div className="space-y-3 text-center w-full">
              {/* CPU collapsed */}
              <div className="flex flex-col items-center gap-1 cursor-help" title={`${t("CPU 佔用")}: ${systemResources.cpu.toFixed(1)}%`}>
                <Cpu size={14} className="text-blue-400 animate-pulse" style={{ animationDuration: '3s' }} />
                <span className="text-[9px] font-semibold text-gray-400">{systemResources.cpu.toFixed(0)}%</span>
              </div>

              {/* RAM collapsed */}
              <div className="flex flex-col items-center gap-1 cursor-help" title={`${t("記憶體 (RAM)")}: ${systemResources.memory} MB`}>
                <HardDrive size={14} className="text-indigo-400" />
                <span className="text-[9px] font-semibold text-gray-400">{systemResources.memory}M</span>
              </div>
            </div>
          )}
        </div>
      </aside>

      {/* 2. 右側主要內容區 */}
      <main className="flex-1 flex flex-col overflow-hidden">

        {/* Topbar 搜尋與連線狀態列 */}
        <header className="relative z-30 h-14 border-b border-darkBorder bg-darkCard/25 backdrop-blur-md px-6 flex items-center justify-between select-none">
          <div className="flex items-center gap-3 w-64">
            <div className="relative w-full">
              <input
                ref={searchInputRef}
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onFocus={() => {
                  setIsSearchFocused(true);
                  refreshProjectsList();
                }}
                onBlur={() => {
                  // 延遲關閉，好讓 onMouseDown 能被優先觸發
                  setTimeout(() => setIsSearchFocused(false), 200);
                }}
                placeholder={t("搜尋專案名稱、網域或路徑... (Ctrl+K)")}
                className="w-full bg-darkInput/40 border border-darkBorder rounded-lg pl-8 pr-3 py-1.5 text-xs text-gray-200 placeholder-gray-500 outline-none focus:border-blue-500 transition duration-150"
              />
              <div className="absolute left-2.5 top-1/2 -translate-y-1/2 text-gray-500">
                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
              </div>

              {/* 搜尋結果下拉選單 */}
              {isSearchFocused && searchQuery && (
                <div className="absolute z-[9999] left-0 right-0 mt-1 bg-[#0c0c0e] border border-darkBorder rounded-lg shadow-2xl max-h-60 overflow-y-auto divide-y divide-darkBorder/40">
                  {(() => {
                    const q = searchQuery.toLowerCase();
                    const filtered = config?.projects?.filter((proj: any) => {
                      return (
                        proj.name?.toLowerCase().includes(q) ||
                        proj.root_path?.toLowerCase().includes(q) ||
                        proj.domains?.some((d: string) => d.toLowerCase().includes(q))
                      );
                    }) || [];

                    if (filtered.length > 0) {
                      return filtered.map((proj: any) => (
                        <div
                          key={proj.name}
                          onMouseDown={() => {
                            handleTabChange('projects');
                            setHighlightedProjectName(proj.name);
                            setSearchQuery('');
                          }}
                          className="p-2.5 hover:bg-blue-600/10 cursor-pointer flex flex-col gap-0.5 text-left"
                        >
                          <div className="text-xs font-bold text-gray-200">{proj.name}</div>
                          <div className="text-[10px] text-gray-500 font-mono truncate">{proj.root_path}</div>
                          <div className="text-[10px] text-blue-400 font-mono truncate">{proj.domains?.join(', ')}</div>
                        </div>
                      ));
                    } else {
                      return (
                        <div className="p-3 text-[11px] text-gray-500 text-center">{t("找不到匹配的專案 😭")}</div>
                      );
                    }
                  })()}
                </div>
              )}
            </div>
          </div>
          <div className="flex items-center gap-4">
            {/* 管理員權限指示徽章 */}
            <div
              className={`flex items-center gap-1.5 text-xs font-semibold cursor-help select-none ${isAdmin ? 'text-blue-400' : 'text-amber-500'
                }`}
              title={
                isAdmin
                  ? t('已取得系統管理員權限，可自動配置 Hosts 網域別名')
                  : t('無管理員權限：可能無法自動修改 Hosts 檔，需手動管理網域別名')
              }
            >
              <Shield size={12} className={isAdmin ? 'text-blue-400' : 'text-amber-500'} />
              <span>{isAdmin ? t('管理員模式') : t('限制模式')}</span>
            </div>
            <div className="h-3 w-[1px] bg-darkBorder" />

            {/* 連線狀態指示燈 */}
            <div className="flex items-center gap-2 text-xs text-gray-400">
              <span className="relative flex h-1.5 w-1.5">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-green-500"></span>
              </span>
              <span>{t("Go 核心已連線")}</span>
            </div>
            <div className="h-3 w-[1px] bg-darkBorder" />
            <span className="text-[10px] text-gray-500 font-semibold tracking-wide">{appVersion}</span>
          </div>
        </header>

        {/* 上半部：當前分頁 */}
        <div className="flex-1 overflow-hidden relative">
          {renderActiveComponent()}
        </div>

        {/* 控制日誌的收放欄 */}
        {activeTab !== 'logs' && (
          <div className="h-9 border-t border-darkBorder bg-[#0e0e11] px-6 flex justify-between items-center select-none text-[11px]">
            <button
              onClick={() => setShowLogs(!showLogs)}
              className="flex items-center gap-1.5 font-semibold text-gray-400 hover:text-gray-200 transition"
            >
              <Terminal size={11} className="text-blue-400" />
              <span>{showLogs ? t('收起 Logs 控制台') : t('打開 Logs 控制台')}</span>
            </button>
            <button
              onClick={() => setShowLogs(!showLogs)}
              className="p-1 rounded-md text-gray-400 hover:text-gray-200 hover:bg-white/5 transition flex items-center justify-center"
              title={showLogs ? t('收起 Logs 控制台') : t('打開 Logs 控制台')}
            >
              {showLogs ? <ChevronDown size={14} /> : <ChevronUp size={14} />}
            </button>
          </div>
        )}

        {/* 下半部：即時日誌區 */}
        {activeTab !== 'logs' && showLogs && (
          <div className="h-[35%] min-h-[150px] border-t border-darkBorder bg-darkBg overflow-hidden">
            <TerminalLogs />
          </div>
        )}
      </main>

      {/* 全域自訂 Alert Modal */}
      {customAlert.isOpen && (
        <div className="fixed inset-0 z-[9999] flex items-center justify-center p-4 bg-black/60 backdrop-blur-[2px] animate-in fade-in duration-200 select-none">
          <div className="w-full max-w-sm bg-darkCard border border-darkBorder rounded-xl shadow-2xl overflow-hidden p-5 flex flex-col space-y-4 animate-in zoom-in-95 duration-200">
            <div className="flex items-center gap-2.5 text-blue-400 font-bold text-sm">
              <span className="text-base">🔔</span>
              <span>{t("系統提示")}</span>
            </div>
            <p className="text-xs text-gray-300 leading-relaxed break-all whitespace-pre-line">{customAlert.message}</p>
            <div className="flex justify-end pt-1">
              <button
                onClick={() => {
                  if (customAlert.resolve) customAlert.resolve();
                  else setCustomAlert({ isOpen: false, message: '' });
                }}
                className="px-4 py-1.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-xs font-semibold shadow-md shadow-blue-500/10 active:scale-[0.98] transition duration-150"
              >
                {t("確定")}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 全域自訂 Confirm Modal */}
      {customConfirm.isOpen && (
        <div className="fixed inset-0 z-[9999] flex items-center justify-center p-4 bg-black/60 backdrop-blur-[2px] animate-in fade-in duration-200 select-none">
          <div className="w-full max-w-sm bg-darkCard border border-darkBorder rounded-xl shadow-2xl overflow-hidden p-5 flex flex-col space-y-4 animate-in zoom-in-95 duration-200">
            <div className="flex items-center gap-2.5 text-blue-400 font-bold text-sm">
              <span className="text-base">❓</span>
              <span>{t("系統確認")}</span>
            </div>
            <p className="text-xs text-gray-300 leading-relaxed break-all whitespace-pre-line">{customConfirm.message}</p>
            <div className="flex justify-end gap-2.5 pt-1">
              <button
                onClick={() => {
                  if (customConfirm.resolve) customConfirm.resolve(false);
                  else setCustomConfirm({ isOpen: false, message: '' });
                }}
                className="px-4 py-1.5 bg-darkInput border border-darkBorder hover:bg-darkBorder text-gray-300 rounded-lg text-xs font-semibold active:scale-[0.98] transition duration-150"
              >
                {t("取消")}
              </button>
              <button
                onClick={() => {
                  if (customConfirm.resolve) customConfirm.resolve(true);
                  else setCustomConfirm({ isOpen: false, message: '' });
                }}
                className="px-4 py-1.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-xs font-semibold shadow-md shadow-blue-500/10 active:scale-[0.98] transition duration-150"
              >
                {t("確定")}
              </button>
            </div>
          </div>
        </div>
      )}

    </div>
  );
}

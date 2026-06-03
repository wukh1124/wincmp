import React, { useState, useEffect } from 'react';
import { Home, Folder, Database, Settings as SettingsIcon, Terminal, Cpu, HardDrive, ChevronLeft, ChevronRight } from 'lucide-react';
import Dashboard from './components/Dashboard';
import Projects from './components/Projects';
import DBExplorer from './components/DBExplorer';
import Settings from './components/Settings';
import ResourceMonitor from './components/ResourceMonitor';
import TerminalLogs from './components/TerminalLogs';
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime';
import logo from './assets/images/icon.svg';

export default function App() {
  const [activeTab, setActiveTab] = useState<'dashboard' | 'projects' | 'db_explorer' | 'resources' | 'settings'>('dashboard');
  const [showLogs, setShowLogs] = useState(true);
  const [systemResources, setSystemResources] = useState({ cpu: 0, memory: 0 });
  const [isCollapsed, setIsCollapsed] = useState(() => localStorage.getItem('sidebar_collapsed') === 'true');

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

    EventsOn('resource_usage', handleResourceUpdate);

    return () => {
      EventsOff('resource_usage');
    };
  }, []);

  const renderActiveComponent = () => {
    switch (activeTab) {
      case 'dashboard':
        return <Dashboard />;
      case 'projects':
        return <Projects />;
      case 'db_explorer':
        return <DBExplorer />;
      case 'resources':
        return <ResourceMonitor />;
      case 'settings':
        return <Settings />;
      default:
        return <Dashboard />;
    }
  };

  const menuItems = [
    { id: 'dashboard', label: '儀表板 (Dashboard)', icon: <Home size={15} /> },
    { id: 'projects', label: '專案管理 (Projects)', icon: <Folder size={15} /> },
    { id: 'db_explorer', label: '資料庫瀏覽 (Database)', icon: <Database size={15} /> },
    { id: 'resources', label: '資源監控 (Resources)', icon: <Cpu size={15} /> },
    { id: 'settings', label: '系統設定 (Settings)', icon: <SettingsIcon size={15} /> }
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
              title={isCollapsed ? '展開側邊欄' : '收起側邊欄'}
            >
              {isCollapsed ? <ChevronRight size={15} /> : <ChevronLeft size={15} />}
            </button>
          </div>

          {/* 選單列表 */}
          <nav className={`pt-4 space-y-1 transition-all duration-300 ${isCollapsed ? 'px-2' : 'p-3'}`}>
            {menuItems.map(item => (
              <button
                key={item.id}
                onClick={() => setActiveTab(item.id)}
                title={isCollapsed ? item.label.split(' ')[0] : undefined}
                className={`w-full text-left py-2.5 text-sm font-semibold flex items-center transition-all duration-150 ${
                  isCollapsed ? 'justify-center px-0' : 'px-4 gap-3'
                } ${
                  activeTab === item.id
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
                系統監控 (WinCMP Core)
              </div>
              <div className="space-y-2.5 text-xs">
                {/* CPU */}
                <div className="space-y-1">
                  <div className="flex items-center justify-between">
                    <span className="text-gray-400 flex items-center gap-1.5">
                      <Cpu size={12} className="text-blue-400" /> CPU 佔用
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
                      <HardDrive size={12} className="text-indigo-400" /> 記憶體 (RAM)
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
              <div className="flex flex-col items-center gap-1 cursor-help" title={`CPU 佔用: ${systemResources.cpu.toFixed(1)}%`}>
                <Cpu size={14} className="text-blue-400 animate-pulse" style={{ animationDuration: '3s' }} />
                <span className="text-[9px] font-semibold text-gray-400">{systemResources.cpu.toFixed(0)}%</span>
              </div>

              {/* RAM collapsed */}
              <div className="flex flex-col items-center gap-1 cursor-help" title={`記憶體 (RAM): ${systemResources.memory} MB`}>
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
        <header className="h-14 border-b border-darkBorder bg-darkCard/25 backdrop-blur-md px-6 flex items-center justify-between select-none">
          <div className="flex items-center gap-3 w-64">
            <div className="relative w-full">
              <input
                type="text"
                placeholder="搜尋專案或設定... (Ctrl+K)"
                disabled
                className="w-full bg-darkInput/40 border border-darkBorder rounded-lg pl-8 pr-3 py-1.5 text-xs text-gray-400 placeholder-gray-500 outline-none cursor-not-allowed"
              />
              <div className="absolute left-2.5 top-1/2 -translate-y-1/2 text-gray-500">
                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
              </div>
            </div>
          </div>
          <div className="flex items-center gap-4">
            {/* 連線狀態指示燈 */}
            <div className="flex items-center gap-2 text-xs text-gray-400">
              <span className="relative flex h-1.5 w-1.5">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                <span className="relative inline-flex rounded-full h-1.5 w-1.5 bg-green-500"></span>
              </span>
              <span>Go 核心已連線</span>
            </div>
            <div className="h-3 w-[1px] bg-darkBorder" />
            <span className="text-[10px] text-gray-500 font-semibold tracking-wide">v3.0.0</span>
          </div>
        </header>

        {/* 上半部：當前分頁 */}
        <div className="flex-1 overflow-hidden relative">
          {renderActiveComponent()}
        </div>

        {/* 控制日誌的收放欄 */}
        <div className="h-9 border-t border-darkBorder bg-[#0e0e11] px-6 flex justify-between items-center select-none text-[11px]">
          <button
            onClick={() => setShowLogs(!showLogs)}
            className="flex items-center gap-1.5 font-semibold text-gray-400 hover:text-gray-200 transition"
          >
            <Terminal size={11} className="text-blue-400" />
            <span>{showLogs ? '收起 Logs 控制台' : '打開 Logs 控制台'}</span>
          </button>
          <span className="text-[9px] text-gray-500 font-mono">Status: Connected to Go core</span>
        </div>

        {/* 下半部：即時日誌區 */}
        {showLogs && (
          <div className="h-[35%] min-h-[150px] border-t border-darkBorder bg-darkBg overflow-hidden">
            <TerminalLogs />
          </div>
        )}
      </main>

    </div>
  );
}

import React, { useState, useEffect, useRef } from 'react';
import { Home, Folder, Database, Settings as SettingsIcon, Terminal, Cpu, HardDrive, ChevronLeft, ChevronRight, ChevronUp, ChevronDown, Shield, Download, Palette, Languages, Type } from 'lucide-react';
import Dashboard from './components/Dashboard';
import Projects from './components/Projects';
import DBExplorer from './components/DBExplorer';
import Settings from './components/Settings';
import ResourceMonitor from './components/ResourceMonitor';
import TerminalLogs from './components/TerminalLogs';
import VersionUpdate from './components/VersionUpdate';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { GetAppVersion, IsAdmin, GetConfig, SaveQuickSettings } from '../wailsjs/go/main/App';
import logo from './assets/images/icon.svg';
import { t, setLanguage, useLanguage, getLanguage } from './i18n';
import { useTheme, THEMES } from './components/ThemeContext';

// 追蹤在本次 App 生命週期中是否已觸發過 Projects 自動收合 sidebar
let projectsCollapsedTriggered = false;

// 防抖儲存主題、語言與字體大小設定的定時器
let quickSaveTimer: any = null;

const saveQuickSettingsDebounced = (theme: string, lang: string, fontSize: string) => {
  if (quickSaveTimer) {
    clearTimeout(quickSaveTimer);
  }
  quickSaveTimer = setTimeout(async () => {
    try {
      await SaveQuickSettings(theme, lang, fontSize);
      // 發送事件通知 Settings 頁面同步最新後端設定
      window.dispatchEvent(new CustomEvent('wincmp_config_synced'));
    } catch (err) {
      console.error("快速儲存設定失敗:", err);
    }
  }, 1500);
};

export default function App() {
  useLanguage();
  const { theme, setTheme, fontSize, setFontSize, fontSizes } = useTheme();

  const [activeTab, setActiveTab] = useState<'dashboard' | 'projects' | 'db_explorer' | 'resources' | 'settings' | 'logs' | 'update'>('dashboard');
  const [showLogs, setShowLogs] = useState(false);
  const [systemResources, setSystemResources] = useState({ cpu: 0, memory: 0 });
  const [isCollapsed, setIsCollapsed] = useState(false);
  const [customAlert, setCustomAlert] = useState<{ isOpen: boolean; message: string; resolve?: () => void }>({ isOpen: false, message: '' });
  const [customConfirm, setCustomConfirm] = useState<{ isOpen: boolean; message: string; resolve?: (value: boolean) => void }>({ isOpen: false, message: '' });
  const [unsavedConfirm, setUnsavedConfirm] = useState<{
    isOpen: boolean;
    resolve?: (value: 'save' | 'discard' | 'cancel') => void;
  }>({ isOpen: false });

  const askUnsavedSettings = () => {
    return new Promise<'save' | 'discard' | 'cancel'>((resolve) => {
      setUnsavedConfirm({
        isOpen: true,
        resolve: (val) => {
          setUnsavedConfirm({ isOpen: false, resolve: undefined });
          resolve(val);
        }
      });
    });
  };

  const [isAdmin, setIsAdmin] = useState(false);
  const [appVersion, setAppVersion] = useState('v2.0.0');
  const [searchQuery, setSearchQuery] = useState('');
  const [isSearchFocused, setIsSearchFocused] = useState(false);
  const [config, setConfig] = useState<any>(null);
  const [highlightedProjectName, setHighlightedProjectName] = useState<string | null>(null);
  const [hasUpdate, setHasUpdate] = useState(false);

  const searchInputRef = useRef<HTMLInputElement>(null);

  // 1. 初始化讀取版本與管理員權限及語系、主題，並掛載全域防抖儲存函式
  useEffect(() => {
    (window as any).saveQuickSettingsDebounced = saveQuickSettingsDebounced;

    GetAppVersion().then(setAppVersion).catch((err: any) => console.error("獲取版本失敗:", err));
    IsAdmin().then(setIsAdmin).catch((err: any) => console.error("獲取管理員權限失敗:", err));
    GetConfig().then((cfg: any) => {
      if (cfg && cfg.global) {
        // 語言回退：若無此值，則預設為系統語言
        const lang = cfg.global.language;
        if (lang) {
          setLanguage(lang);
        } else {
          setLanguage(getLanguage());
        }

        // 主題回退：若無此值或格式有誤，則預設回退至 'xai'
        const validThemes = ['xai', 'claude', 'sketch'];
        const savedTheme = cfg.global.theme;
        if (savedTheme && validThemes.includes(savedTheme)) {
          setTheme(savedTheme);
        } else {
          setTheme('xai');
        }

        // 字型大小回退：若無此值或格式有誤，則預設為 'small'
        const validFontSizes = ['small', 'medium', 'large'];
        const savedFontSize = cfg.global.font_size;
        if (savedFontSize && validFontSizes.includes(savedFontSize)) {
          setFontSize(savedFontSize);
        } else {
          setFontSize('small');
        }
      }
    }).catch((err: any) => console.error("獲取語系、主題與字型大小失敗:", err));
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
        const choice = await askUnsavedSettings();
        if (choice === 'cancel') {
          return;
        }
        if (choice === 'save') {
          try {
            if ((window as any).saveSettings) {
              await (window as any).saveSettings(true);
            }
          } catch (err) {
            console.error("離開前自動儲存設定失敗:", err);
            return;
          }
        }
      }
    }
    (window as any).isSettingsDirty = false;
    setActiveTab(tabId);

    if (tabId === 'projects' && !projectsCollapsedTriggered) {
      projectsCollapsedTriggered = true;
      GetConfig().then((cfg: any) => {
        if (cfg && cfg.projects && cfg.projects.length > 0) {
          setIsCollapsed(true);
        }
      }).catch((err: any) => console.error("首次進入專案頁面獲取設定失敗:", err));
    }
  };

  // 覆寫與註冊漂亮的自訂彈窗
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

  const toggleSidebar = () => setIsCollapsed(prev => !prev);
  const handleToggleLogs = () => setShowLogs(prev => !prev);

  // 訂閱 Go 端推送的 CPU / RAM 資源佔用
  useEffect(() => {
    const handleResourceUpdate = (data: any) => {
      if (data) {
        setSystemResources({ cpu: data.cpu || 0, memory: data.memory || 0 });
      }
    };
    const unsubscribe = EventsOn('resource_usage', handleResourceUpdate);
    return () => { unsubscribe(); };
  }, []);

  // 訂閱版本更新通知
  useEffect(() => {
    const handleUpdateAvailable = () => setHasUpdate(true);
    const unsubscribe = EventsOn('update_available', handleUpdateAvailable);
    return () => { unsubscribe(); };
  }, []);

  // 監聽自動展開日誌事件
  useEffect(() => {
    const handleAutoExpand = () => setShowLogs(true);
    window.addEventListener('wincmp_auto_expand_logs', handleAutoExpand);
    return () => window.removeEventListener('wincmp_auto_expand_logs', handleAutoExpand);
  }, []);

  const renderActiveComponent = () => {
    switch (activeTab) {
      case 'dashboard': return <Dashboard />;
      case 'projects': return <Projects highlightedProjectName={highlightedProjectName} clearHighlight={() => setHighlightedProjectName(null)} />;
      case 'db_explorer': return <DBExplorer />;
      case 'resources': return <ResourceMonitor />;
      case 'settings': return <Settings />;
      case 'logs': return <TerminalLogs />;
      case 'update': return <VersionUpdate />;
      default: return <Dashboard />;
    }
  };

  const menuItems = [
    { id: 'dashboard', label: t('儀表板'), icon: <Home size={15} /> },
    { id: 'projects', label: t('專案管理'), icon: <Folder size={15} /> },
    { id: 'db_explorer', label: t('資料庫瀏覽'), icon: <Database size={15} /> },
    { id: 'resources', label: t('資源監控'), icon: <Cpu size={15} /> },
    { id: 'settings', label: t('系統設定'), icon: <SettingsIcon size={15} /> },
    { id: 'update', label: t('版本更新'), icon: <Download size={15} /> },
    { id: 'logs', label: t('終端日誌'), icon: <Terminal size={15} /> }
  ] as const;

  const cycleTheme = () => {
    const idx = THEMES.findIndex(th => th.id === theme);
    const next = THEMES[(idx + 1) % THEMES.length];
    setTheme(next.id);
    if ((window as any).saveQuickSettingsDebounced) {
      (window as any).saveQuickSettingsDebounced(next.id, getLanguage(), fontSize);
    }
  };

  const cycleFontSize = () => {
    const idx = fontSizes.findIndex(fs => fs.id === fontSize);
    const next = fontSizes[(idx + 1) % fontSizes.length];
    setFontSize(next.id);
    if ((window as any).saveQuickSettingsDebounced) {
      (window as any).saveQuickSettingsDebounced(theme, getLanguage(), next.id);
    }
  };

  const LANGS = ['zh-TW', 'en-US'] as const;
  const cycleLanguage = () => {
    const idx = LANGS.indexOf(getLanguage() as typeof LANGS[number]);
    const next = LANGS[(idx + 1) % LANGS.length];
    setLanguage(next);
    if ((window as any).saveQuickSettingsDebounced) {
      (window as any).saveQuickSettingsDebounced(theme, next, fontSize);
    }
  };

  return (
    <div className="flex h-screen w-screen overflow-hidden select-none" style={{ backgroundColor: 'var(--bg)', color: 'var(--fg)' }}>

      {/* ─── Sidebar ──────────────────────────────────────────── */}
      <aside
        className="flex flex-col justify-between select-none transition-all duration-300 ease-in-out border-r"
        style={{
          width: isCollapsed ? 64 : 256,
          background: 'var(--sidebar-bg)',
          borderColor: 'var(--border)',
        }}
      >
        <div>
          {/* Logo & Title */}
          <div
            className="py-5 border-b flex items-center transition-all duration-300"
            style={{
              borderColor: 'var(--border)',
              padding: isCollapsed ? '20px 12px' : '20px 20px',
              flexDirection: isCollapsed ? 'column' : 'row',
              gap: isCollapsed ? 12 : 12,
              justifyContent: isCollapsed ? 'center' : 'space-between',
            }}
          >
            <div className="flex items-center gap-3 overflow-hidden">
              <img src={logo} alt="WinCMP Logo" className="w-8 h-8 rounded-lg object-contain flex-shrink-0" style={{ boxShadow: 'var(--shadow-sm)' }} />
              {!isCollapsed && (
                <div className="transition-opacity duration-300 opacity-100 whitespace-nowrap">
                  <span className="font-bold tracking-wide block" style={{ color: 'var(--fg)', fontFamily: 'var(--font-display)' }}>WinCMP</span>
                  <span className="text-[10px] block font-semibold uppercase tracking-wider" style={{ color: 'var(--muted)' }}>Local Dev Panel</span>
                </div>
              )}
            </div>
            <button
              onClick={toggleSidebar}
              className="p-1.5 rounded-md transition-colors"
              style={{ color: 'var(--muted)' }}
              title={isCollapsed ? t('展開側邊欄') : t('收起側邊欄')}
            >
              {isCollapsed ? <ChevronRight size={15} /> : <ChevronLeft size={15} />}
            </button>
          </div>

          {/* Menu */}
          <nav className="pt-4 space-y-1 transition-all duration-300" style={{ padding: isCollapsed ? '16px 8px' : '16px 12px' }}>
            {menuItems.map(item => {
              const isActive = activeTab === item.id;
              return (
                <button
                  key={item.id}
                  onClick={() => handleTabChange(item.id)}
                  title={isCollapsed ? item.label : undefined}
                  className="w-full text-left py-2.5 text-sm font-semibold flex items-center transition-all duration-150 relative"
                  style={{
                    justifyContent: isCollapsed ? 'center' : 'flex-start',
                    padding: isCollapsed ? '10px 0' : '10px 16px',
                    gap: isCollapsed ? 0 : 12,
                    color: isActive ? 'var(--sidebar-active-fg)' : 'var(--fg-2)',
                    background: isActive ? 'var(--sidebar-active-bg)' : 'transparent',
                    borderLeft: isActive ? `3px solid var(--sidebar-active-border)` : '3px solid transparent',
                    borderRadius: '0 8px 8px 0',
                    fontFamily: 'var(--font-body)',
                  }}
                >
                  <span className="flex-shrink-0" style={{ color: isActive ? 'var(--sidebar-active-fg)' : 'var(--muted)' }}>
                    {item.icon}
                  </span>
                  {!isCollapsed && <span className="whitespace-nowrap transition-opacity duration-300">{item.label}</span>}
                  {item.id === 'update' && hasUpdate && (
                    <span
                      className="w-2 h-2 rounded-full absolute"
                      style={{
                        background: 'var(--status-error)',
                        top: isCollapsed ? 6 : '50%',
                        right: isCollapsed ? 6 : 16,
                        transform: isCollapsed ? 'none' : 'translateY(-50%)',
                      }}
                    />
                  )}
                </button>
              );
            })}
          </nav>
        </div>

        {/* Bottom: System Monitor + Theme Switcher */}
        <div
          className="border-t transition-all duration-300"
          style={{
            borderColor: 'var(--border)',
            background: 'var(--surface)',
            padding: isCollapsed ? '8px' : '16px',
          }}
        >
          {/* 快速設定按鈕組：展開時橫向排列以節省垂直空間，收合時垂直排列 */}
          <div className={isCollapsed ? "flex flex-col gap-2 mb-3" : "flex flex-row gap-1.5 mb-3"}>
            {/* Language Quick Switch */}
            <button
              onClick={cycleLanguage}
              className={`flex items-center gap-1.5 py-2 px-2.5 rounded-lg text-[10px] font-semibold transition-all duration-200 ${isCollapsed ? 'w-full justify-center' : 'flex-1 justify-center'}`}
              style={{
                color: 'var(--fg-2)',
                background: 'var(--surface-warm)',
                border: '1px solid var(--border-soft)',
              }}
              title={`Language: ${getLanguage() === 'zh-TW' ? '繁體中文' : 'English'}`}
            >
              <Languages size={13} style={{ color: 'var(--accent)' }} />
              {!isCollapsed && <span style={{ fontFamily: 'var(--font-mono)' }}>{getLanguage() === 'zh-TW' ? 'EN' : '繁'}</span>}
            </button>

            {/* Theme Quick Switch */}
            <button
              onClick={cycleTheme}
              className={`flex items-center gap-1.5 py-2 px-2 rounded-lg text-[10px] font-semibold transition-all duration-200 ${isCollapsed ? 'w-full justify-center' : 'flex-1 justify-center'}`}
              style={{
                color: 'var(--fg-2)',
                background: 'var(--surface-warm)',
                border: '1px solid var(--border-soft)',
              }}
              title={`Theme: ${theme}`}
            >
              <Palette size={13} style={{ color: 'var(--accent)' }} />
              {!isCollapsed && <span className="truncate" style={{ fontFamily: 'var(--font-mono)' }}>{theme.toUpperCase()}</span>}
            </button>

            {/* Font Size Quick Switch */}
            <button
              onClick={cycleFontSize}
              className={`flex items-center gap-1.5 py-2 px-2.5 rounded-lg text-[10px] font-semibold transition-all duration-200 ${isCollapsed ? 'w-full justify-center' : 'flex-1 justify-center'}`}
              style={{
                color: 'var(--fg-2)',
                background: 'var(--surface-warm)',
                border: '1px solid var(--border-soft)',
              }}
              title={`${t('字型大小')}: ${t(fontSize === 'small' ? '小' : fontSize === 'medium' ? '中' : '大')}`}
            >
              <Type size={13} style={{ color: 'var(--accent)' }} />
              {!isCollapsed && (
                <span style={{ fontFamily: 'var(--font-mono)' }}>
                  {fontSize === 'small' ? 'S' : fontSize === 'medium' ? 'M' : 'L'}
                </span>
              )}
            </button>
          </div>

          {!isCollapsed ? (
            <>
              <div className="font-semibold text-[10px] select-none uppercase tracking-wider" style={{ color: 'var(--meta)' }}>
                {t("系統監控 (WinCMP Core)")}
              </div>
              <div className="space-y-2.5 text-xs mt-2">
                {/* CPU */}
                <div className="space-y-1">
                  <div className="flex items-center justify-between">
                    <span className="flex items-center gap-1.5" style={{ color: 'var(--fg-2)' }}>
                      <Cpu size={12} style={{ color: 'var(--status-info)' }} /> {t("CPU 佔用")}
                    </span>
                    <span className="font-semibold" style={{ color: 'var(--fg)', fontFamily: 'var(--font-mono)' }}>{systemResources.cpu.toFixed(1)}%</span>
                  </div>
                  <div className="w-full h-1 rounded-full overflow-hidden" style={{ background: 'var(--input-bg)' }}>
                    <div
                      style={{ width: `${Math.min(systemResources.cpu * 5, 100)}%`, background: 'var(--status-info)' }}
                      className="h-full rounded-full transition-all duration-500"
                    />
                  </div>
                </div>

                {/* RAM */}
                <div className="space-y-1 pt-0.5">
                  <div className="flex items-center justify-between">
                    <span className="flex items-center gap-1.5" style={{ color: 'var(--fg-2)' }}>
                      <HardDrive size={12} style={{ color: 'var(--status-ok)' }} /> {t("記憶體 (RAM)")}
                    </span>
                    <span className="font-semibold" style={{ color: 'var(--fg)', fontFamily: 'var(--font-mono)' }}>{systemResources.memory} MB</span>
                  </div>
                  <div className="w-full h-1 rounded-full overflow-hidden" style={{ background: 'var(--input-bg)' }}>
                    <div
                      style={{ width: `${Math.min(systemResources.memory / 2, 100)}%`, background: 'var(--status-ok)' }}
                      className="h-full rounded-full transition-all duration-500"
                    />
                  </div>
                </div>
              </div>
            </>
          ) : (
            <div className="space-y-3 text-center w-full">
              <div className="flex flex-col items-center gap-1 cursor-help" title={`${t("CPU 佔用")}: ${systemResources.cpu.toFixed(1)}%`}>
                <Cpu size={14} className="animate-pulse" style={{ color: 'var(--status-info)', animationDuration: '3s' }} />
                <span className="text-[9px] font-semibold" style={{ color: 'var(--fg-2)', fontFamily: 'var(--font-mono)' }}>{systemResources.cpu.toFixed(0)}%</span>
              </div>
              <div className="flex flex-col items-center gap-1 cursor-help" title={`${t("記憶體 (RAM)")}: ${systemResources.memory} MB`}>
                <HardDrive size={14} style={{ color: 'var(--status-ok)' }} />
                <span className="text-[9px] font-semibold" style={{ color: 'var(--fg-2)', fontFamily: 'var(--font-mono)' }}>{systemResources.memory}M</span>
              </div>
            </div>
          )}
        </div>
      </aside>

      {/* ─── Main Content ─────────────────────────────────────── */}
      <main className="flex-1 flex flex-col overflow-hidden">

        {/* Topbar */}
        <header
          className="relative z-30 h-14 border-b px-6 flex items-center justify-between select-none"
          style={{
            borderColor: 'var(--border)',
            background: 'var(--card)',
            backdropFilter: 'blur(12px)',
          }}
        >
          <div className="flex items-center gap-3 w-64">
            <div className="relative w-full">
              <input
                ref={searchInputRef}
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onFocus={() => { setIsSearchFocused(true); refreshProjectsList(); }}
                onBlur={() => { setTimeout(() => setIsSearchFocused(false), 200); }}
                placeholder={t("搜尋專案名稱、網域或路徑... (Ctrl+K)")}
                className="w-full rounded-lg pl-8 pr-3 py-1.5 text-xs outline-none transition duration-150"
                style={{
                  background: 'var(--input-bg)',
                  border: '1px solid var(--input-border)',
                  color: 'var(--fg)',
                  fontFamily: 'var(--font-body)',
                }}
              />
              <div className="absolute left-2.5 top-1/2 -translate-y-1/2" style={{ color: 'var(--muted)' }}>
                <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
              </div>

              {/* Search Results Dropdown */}
              {isSearchFocused && searchQuery && (
                <div
                  className="absolute z-[9999] left-0 right-0 mt-1 rounded-lg max-h-60 overflow-y-auto divide-y divide-[var(--border-soft)]"
                  style={{
                    background: 'var(--sidebar-bg)',
                    border: '1px solid var(--border)',
                    boxShadow: 'var(--shadow-lg)',
                  }}
                >
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
                          className="p-2.5 cursor-pointer flex flex-col gap-0.5 text-left"
                          style={{ '--hover-bg': 'var(--status-info-bg)' } as any}
                        >
                          <div className="text-xs font-bold" style={{ color: 'var(--fg)' }}>{proj.name}</div>
                          <div className="text-[10px] truncate" style={{ color: 'var(--muted)', fontFamily: 'var(--font-mono)' }}>{proj.root_path}</div>
                          <div className="text-[10px] truncate" style={{ color: 'var(--status-info)', fontFamily: 'var(--font-mono)' }}>{proj.domains?.join(', ')}</div>
                        </div>
                      ));
                    } else {
                      return (
                        <div className="p-3 text-[11px] text-center" style={{ color: 'var(--muted)' }}>{t("找不到匹配的專案")}</div>
                      );
                    }
                  })()}
                </div>
              )}
            </div>
          </div>

          <div className="flex items-center gap-4">
            {/* Admin Badge */}
            <div
              className="flex items-center gap-1.5 text-xs font-semibold cursor-help select-none"
              style={{ color: isAdmin ? 'var(--status-info)' : 'var(--status-warn)' }}
              title={isAdmin ? t('已取得系統管理員權限，可自動配置 Hosts 網域別名') : t('無管理員權限：可能無法自動修改 Hosts 檔，需手動管理網域別名')}
            >
              <Shield size={12} />
              <span>{isAdmin ? t('管理員模式') : t('限制模式')}</span>
            </div>
            <div className="h-3 w-[1px]" style={{ background: 'var(--border)' }} />

            {/* Connection Status */}
            <div className="flex items-center gap-2 text-xs" style={{ color: 'var(--fg-2)' }}>
              <span className="relative flex h-1.5 w-1.5">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full opacity-75" style={{ background: 'var(--status-ok)' }}></span>
                <span className="relative inline-flex rounded-full h-1.5 w-1.5" style={{ background: 'var(--status-ok)' }}></span>
              </span>
              <span>{t("Go 核心已連線")}</span>
            </div>
            <div className="h-3 w-[1px]" style={{ background: 'var(--border)' }} />
            <span className="text-[10px] font-semibold tracking-wide" style={{ color: 'var(--muted)', fontFamily: 'var(--font-mono)' }}>{appVersion}</span>
          </div>
        </header>

        {/* Main Tab Content */}
        <div className="flex-1 overflow-hidden relative">
          {renderActiveComponent()}
        </div>

        {/* Log Toggle Bar */}
        {activeTab !== 'logs' && (
          <div
            className="h-9 border-t px-6 flex justify-between items-center select-none text-[11px]"
            style={{ borderColor: 'var(--border)', background: 'var(--bg-deep)' }}
          >
            <button
              onClick={handleToggleLogs}
              className="flex items-center gap-1.5 font-semibold transition"
              style={{ color: 'var(--fg-2)' }}
            >
              <Terminal size={11} style={{ color: 'var(--status-info)' }} />
              <span>{showLogs ? t('收起 Logs 控制台') : t('打開 Logs 控制台')}</span>
            </button>
            <button
              onClick={handleToggleLogs}
              className="p-1 rounded-md transition flex items-center justify-center"
              style={{ color: 'var(--fg-2)' }}
              title={showLogs ? t('收起 Logs 控制台') : t('打開 Logs 控制台')}
            >
              {showLogs ? <ChevronDown size={14} /> : <ChevronUp size={14} />}
            </button>
          </div>
        )}

        {/* Inline Logs Panel */}
        {activeTab !== 'logs' && showLogs && (
          <div className="h-[35%] min-h-[150px] border-t overflow-hidden" style={{ borderColor: 'var(--border)', background: 'var(--bg)' }}>
            <TerminalLogs />
          </div>
        )}
      </main>

      {/* ─── Alert Modal ──────────────────────────────────────── */}
      {customAlert.isOpen && (
        <div className="fixed inset-0 z-[9999] flex items-center justify-center p-4 animate-fade-in" style={{ background: 'var(--overlay-bg)', backdropFilter: 'blur(2px)' }}>
          <div className="w-full max-w-sm rounded-xl overflow-hidden p-5 flex flex-col space-y-4 animate-slide-in" style={{ background: 'var(--card)', border: '1px solid var(--border)', boxShadow: 'var(--shadow-lg)' }}>
            <div className="flex items-center gap-2.5 font-bold text-sm" style={{ color: 'var(--status-info)' }}>
              <span className="text-base">🔔</span>
              <span>{t("系統提示")}</span>
            </div>
            <p className="text-xs leading-relaxed break-all whitespace-pre-line" style={{ color: 'var(--fg-2)' }}>{customAlert.message}</p>
            <div className="flex justify-end pt-1">
              <button
                onClick={() => {
                  if (customAlert.resolve) customAlert.resolve();
                  else setCustomAlert({ isOpen: false, message: '' });
                }}
                className="px-4 py-1.5 rounded-lg text-xs font-semibold active:scale-[0.98] transition duration-150"
                style={{ background: 'var(--status-info)', color: '#fff' }}
              >
                {t("確定")}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ─── Confirm Modal ────────────────────────────────────── */}
      {customConfirm.isOpen && (
        <div className="fixed inset-0 z-[9999] flex items-center justify-center p-4 animate-fade-in" style={{ background: 'var(--overlay-bg)', backdropFilter: 'blur(2px)' }}>
          <div className="w-full max-w-sm rounded-xl overflow-hidden p-5 flex flex-col space-y-4 animate-slide-in" style={{ background: 'var(--card)', border: '1px solid var(--border)', boxShadow: 'var(--shadow-lg)' }}>
            <div className="flex items-center gap-2.5 font-bold text-sm" style={{ color: 'var(--status-info)' }}>
              <span className="text-base">❓</span>
              <span>{t("系統確認")}</span>
            </div>
            <p className="text-xs leading-relaxed break-all whitespace-pre-line" style={{ color: 'var(--fg-2)' }}>{customConfirm.message}</p>
            <div className="flex justify-end gap-2.5 pt-1">
              <button
                onClick={() => {
                  if (customConfirm.resolve) customConfirm.resolve(false);
                  else setCustomConfirm({ isOpen: false, message: '' });
                }}
                className="px-4 py-1.5 rounded-lg text-xs font-semibold active:scale-[0.98] transition duration-150"
                style={{ background: 'var(--input-bg)', border: '1px solid var(--border)', color: 'var(--fg-2)' }}
              >
                {t("取消")}
              </button>
              <button
                onClick={() => {
                  if (customConfirm.resolve) customConfirm.resolve(true);
                  else setCustomConfirm({ isOpen: false, message: '' });
                }}
                className="px-4 py-1.5 rounded-lg text-xs font-semibold active:scale-[0.98] transition duration-150"
                style={{ background: 'var(--status-info)', color: '#fff' }}
              >
                {t("確定")}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ─── Unsaved Settings Confirm Modal ────────────────────── */}
      {unsavedConfirm.isOpen && (
        <div className="fixed inset-0 z-[9999] flex items-center justify-center p-4 animate-fade-in" style={{ background: 'var(--overlay-bg)', backdropFilter: 'blur(2px)' }}>
          <div className="w-full max-w-md rounded-xl overflow-hidden p-5 flex flex-col space-y-4 animate-slide-in" style={{ background: 'var(--card)', border: '1px solid var(--border)', boxShadow: 'var(--shadow-lg)' }}>
            <div className="flex items-center gap-2.5 font-bold text-sm" style={{ color: 'var(--status-warn || #e6a23c)' }}>
              <span className="text-base">⚠️</span>
              <span>{t("系統提示")}</span>
            </div>
            <p className="text-xs leading-relaxed break-all whitespace-pre-line" style={{ color: 'var(--fg-2)' }}>
              {t("您有尚未儲存的設定變更，在離開前是否要先保存？")}
            </p>
            <div className="flex justify-end gap-2.5 pt-1">
              <button
                onClick={() => unsavedConfirm.resolve?.('cancel')}
                className="px-3.5 py-1.5 rounded-lg text-xs font-semibold active:scale-[0.98] transition duration-150"
                style={{ background: 'var(--input-bg)', border: '1px solid var(--border)', color: 'var(--fg-2)' }}
              >
                {t("取消")}
              </button>
              <button
                onClick={() => unsavedConfirm.resolve?.('discard')}
                className="px-3.5 py-1.5 rounded-lg text-xs font-semibold active:scale-[0.98] transition duration-150"
                style={{ background: 'var(--status-error-bg)', border: '1px solid var(--status-error)', color: 'var(--status-error)' }}
              >
                {t("否，不保存立即離開")}
              </button>
              <button
                onClick={() => unsavedConfirm.resolve?.('save')}
                className="px-3.5 py-1.5 rounded-lg text-xs font-semibold active:scale-[0.98] transition duration-150"
                style={{ background: 'var(--accent)', color: 'var(--accent-on)' }}
              >
                {t("是，保存後離開")}
              </button>
            </div>
          </div>
        </div>
      )}

    </div>
  );
}

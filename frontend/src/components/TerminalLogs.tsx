import React, { useState, useEffect, useRef } from 'react';
import { Trash2, ArrowDown, ChevronDown, Repeat, Copy, Check, FolderOpen, FolderSearch, ListChecks } from 'lucide-react';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { GetCategoryLogFilePath, OpenCategoryLogFile, OpenCategoryLogFolder } from '../../wailsjs/go/main/App';
import { logStore, LogData } from './logStore';
import { t, useLanguage } from '../i18n';

const CATEGORIES = [
  { id: 'system', label: '系統' },
  { id: 'caddy', label: 'Caddy' },
  { id: 'mariadb', label: 'MariaDB' },
  { id: 'mailpit', label: 'Mailpit' },
  { id: 'redis', label: 'Redis' },
  { id: 'php', label: 'PHP' },
  { id: 'runtime', label: 'Node / Custom' }
];

interface TerminalLogsProps {
  onCollapse?: () => void;
}

interface LogContextMenu {
  x: number;
  y: number;
  lineIndex: number;
  lineText: string;
}

interface LogFileInfoState {
  path: string;
  name: string;
  exists: boolean;
}

export default function TerminalLogs({ onCollapse }: TerminalLogsProps) {
  useLanguage(); // 訂閱語系變更
  const [activeTab, setActiveTab] = useState('system');
  const [activeRuntimeProject, setActiveRuntimeProject] = useState('');
  const [logs, setLogs] = useState<LogData>(logStore.getLogs());
  const [autoScroll, setAutoScroll] = useState(true);
  const [autoSwitchTab, setAutoSwitchTab] = useState<boolean>(() => {
    return localStorage.getItem('wincmp_auto_switch_tab') === 'true'; // 預設為 false，避免搶焦點
  });
  const [unreadTabs, setUnreadTabs] = useState<Record<string, boolean>>({});
  const [unreadCounts, setUnreadCounts] = useState<Record<string, number>>({});
  const [unreadProjects, setUnreadProjects] = useState<Record<string, boolean>>({});
  const [unreadProjectCounts, setUnreadProjectCounts] = useState<Record<string, number>>({});
  const [runtimeDropdownOpen, setRuntimeDropdownOpen] = useState(false);
  const [contextMenu, setContextMenu] = useState<LogContextMenu | null>(null);
  const [copiedFlag, setCopiedFlag] = useState<'selection' | 'line' | null>(null);
  const [logFileInfo, setLogFileInfo] = useState<LogFileInfoState | null>(null);

  const logEndRef = useRef<HTMLDivElement | null>(null);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const menuRef = useRef<HTMLDivElement | null>(null);
  const runtimeDropdownRef = useRef<HTMLDivElement | null>(null);

  // 用於追蹤當前啟動的分頁與狀態，避免閉包陳舊
  const activeTabRef = useRef(activeTab);
  const activeRuntimeProjectRef = useRef(activeRuntimeProject);
  const autoSwitchTabRef = useRef(autoSwitchTab);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    activeTabRef.current = activeTab;
  }, [activeTab]);

  useEffect(() => {
    activeRuntimeProjectRef.current = activeRuntimeProject;
  }, [activeRuntimeProject]);

  useEffect(() => {
    autoSwitchTabRef.current = autoSwitchTab;
  }, [autoSwitchTab]);

  // 訂閱全域 logStore 的日誌更新
  useEffect(() => {
    return logStore.subscribe((newLogs) => {
      setLogs(newLogs);
    });
  }, []);

  // 關閉右鍵選單與 runtime 下拉
  useEffect(() => {
    const closeMenu = () => setContextMenu(null);
    const closeDropdown = (e: MouseEvent) => {
      if (!runtimeDropdownRef.current?.contains(e.target as Node)) {
        setRuntimeDropdownOpen(false);
      }
    };
    window.addEventListener('click', closeMenu);
    window.addEventListener('click', closeDropdown);
    return () => {
      window.removeEventListener('click', closeMenu);
      window.removeEventListener('click', closeDropdown);
    };
  }, []);

  // 訂閱 Go 端的日誌 Event
  useEffect(() => {
    const handleAutoSwitch = (data: any) => {
      if (!data || !data.category) return;
      const category = data.category === 'node' ? 'runtime' : data.category;
      const projName = category === 'runtime' ? data.projectName : undefined;

      const isValidCategory = ['system', 'caddy', 'mariadb', 'mailpit', 'php', 'redis', 'runtime'].includes(category);
      if (!isValidCategory) return;

      // 若未開啟自動跳轉，只在非當前 tab / 非當前專案標註有未讀日誌，絕不強行奪取焦點
      if (!autoSwitchTabRef.current) {
        if (category !== activeTabRef.current) {
          setUnreadTabs(prev => ({ ...prev, [category]: true }));
          setUnreadCounts(prev => ({ ...prev, [category]: (prev[category] || 0) + 1 }));
        }
        if (
          category === 'runtime' &&
          projName &&
          !(activeTabRef.current === 'runtime' && projName === activeRuntimeProjectRef.current)
        ) {
          setUnreadProjects(prev => ({ ...prev, [projName]: true }));
          setUnreadProjectCounts(prev => ({ ...prev, [projName]: (prev[projName] || 0) + 1 }));
        }
        return;
      }

      // 檢查是否需要切換 tab 或切換專案
      const needsTabSwitch = category !== activeTabRef.current;
      const needsProjectSwitch = category === 'runtime' && projName !== activeRuntimeProjectRef.current;

      if (needsTabSwitch || needsProjectSwitch) {
        if (debounceTimerRef.current) {
          clearTimeout(debounceTimerRef.current);
        }
        debounceTimerRef.current = setTimeout(() => {
          if (needsTabSwitch) {
            setActiveTab(category);
            setUnreadTabs(prev => ({ ...prev, [category]: false }));
            setUnreadCounts(prev => ({ ...prev, [category]: 0 }));
          }
          if (category === 'runtime' && projName) {
            setActiveRuntimeProject(projName);
            setUnreadProjects(prev => ({ ...prev, [projName]: false }));
            setUnreadProjectCounts(prev => ({ ...prev, [projName]: 0 }));
          }
        }, 500);
      }
    };

    const unsubscribe = EventsOn('log', handleAutoSwitch);

    return () => {
      unsubscribe();
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, []);

  // 當 logs 或 activeTab 改變時，自動滾動到底部
  useEffect(() => {
    if (autoScroll && logEndRef.current) {
      logEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, activeTab, activeRuntimeProject, autoScroll]);

  // 取得當前所有已記錄的 Runtime 專案列表
  const runtimeProjects = Object.keys(logs.runtime);

  // 確保 activeRuntimeProject 有合理的值
  useEffect(() => {
    if (activeTab === 'runtime') {
      if (!activeRuntimeProject || !runtimeProjects.includes(activeRuntimeProject)) {
        if (runtimeProjects.length > 0) {
          setActiveRuntimeProject(runtimeProjects[0]);
          setUnreadProjects(prev => ({ ...prev, [runtimeProjects[0]]: false }));
          setUnreadProjectCounts(prev => ({ ...prev, [runtimeProjects[0]]: 0 }));
        } else {
          setActiveRuntimeProject('');
        }
      } else {
        // 已在檢視的專案清除未讀
        setUnreadProjects(prev => (prev[activeRuntimeProject] ? { ...prev, [activeRuntimeProject]: false } : prev));
        setUnreadProjectCounts(prev => (prev[activeRuntimeProject] ? { ...prev, [activeRuntimeProject]: 0 } : prev));
      }
    }
  }, [activeTab, runtimeProjects, activeRuntimeProject]);

  // 依目前分類／專案取得當天日誌檔資訊
  useEffect(() => {
    let cancelled = false;
    const loadLogFileInfo = async () => {
      if (!activeTab) {
        setLogFileInfo(null);
        return;
      }
      const sub = activeTab === 'runtime' ? activeRuntimeProject : '';
      try {
        if ((window as any).go?.main?.App?.GetCategoryLogFilePath) {
          const info = await GetCategoryLogFilePath(activeTab, sub);
          if (!cancelled) {
            setLogFileInfo({
              path: info?.path || '',
              name: info?.name || '',
              exists: !!info?.exists,
            });
          }
        } else {
          setLogFileInfo(null);
        }
      } catch (err) {
        console.error('取得日誌檔路徑失敗:', err);
        if (!cancelled) setLogFileInfo(null);
      }
    };
    loadLogFileInfo();
    return () => {
      cancelled = true;
    };
  }, [activeTab, activeRuntimeProject]);

  // 監聽使用者手動滾動，決定是否開啟自動滾動
  const handleScroll = () => {
    if (!containerRef.current) return;
    const { scrollTop, scrollHeight, clientHeight } = containerRef.current;
    const isAtBottom = scrollHeight - scrollTop - clientHeight < 50;
    setAutoScroll(isAtBottom);
  };

  const handleClearLogs = () => {
    logStore.clearLogs(activeTab, activeRuntimeProject);
  };

  const selectRuntimeProject = (proj: string) => {
    setActiveRuntimeProject(proj);
    setUnreadProjects(prev => ({ ...prev, [proj]: false }));
    setUnreadProjectCounts(prev => ({ ...prev, [proj]: 0 }));
    setRuntimeDropdownOpen(false);
  };

  const hasUnreadOtherProject = runtimeProjects.some(
    (proj) => proj !== activeRuntimeProject && unreadProjects[proj]
  );

  // Warp 風格日誌著色
  const getLineStyle = (text: string): React.CSSProperties => {
    const lower = text.toLowerCase();
    if (
      lower.includes('error') ||
      lower.includes('failed') ||
      lower.includes('🔴') ||
      lower.includes('❌') ||
      lower.includes('無法') ||
      lower.includes('失敗') ||
      lower.includes('missing') ||
      lower.includes('fatal')
    ) {
      return { color: 'var(--status-error)', fontWeight: 600 };
    }
    if (
      lower.includes('warn') ||
      lower.includes('warning') ||
      lower.includes('⚠️') ||
      lower.includes('警示') ||
      lower.includes('deprecated')
    ) {
      return { color: 'var(--status-warn)', fontWeight: 600 };
    }
    if (
      lower.includes('info') ||
      lower.includes('success') ||
      lower.includes('✅') ||
      lower.includes('運作中') ||
      lower.includes('運行中') ||
      lower.includes('已啟動') ||
      lower.includes('就緒') ||
      lower.includes('connected') ||
      lower.includes('started') ||
      lower.includes('listening')
    ) {
      return { color: 'var(--status-ok)', fontWeight: 500 };
    }
    return { color: 'var(--fg-2)' };
  };

  // 決定要渲染的日誌行數
  const currentTabLogs = activeTab === 'runtime'
    ? (logs.runtime[activeRuntimeProject] || [])
    : (logs[activeTab as keyof Omit<LogData, 'runtime'>] || []);

  const handleContextMenu = (e: React.MouseEvent, lineIndex: number, lineText: string) => {
    e.preventDefault();
    const menuWidth = 200;
    const menuHeight = 220;
    let x = e.clientX;
    if (x + menuWidth > window.innerWidth - 8) {
      x = Math.max(8, window.innerWidth - menuWidth - 8);
    }
    let y = e.clientY;
    // 若點擊處下方空間不足以容納選單，優先向上翻轉展開
    if (y + menuHeight > window.innerHeight - 8) {
      y = Math.max(8, e.clientY - menuHeight);
    }
    setContextMenu({ x, y, lineIndex, lineText });
  };

  // 當選單渲染完成後，根據實際 DOM 尺寸二次校正，確保完全不溢出視窗底部
  useEffect(() => {
    if (!contextMenu || !menuRef.current) return;
    const rect = menuRef.current.getBoundingClientRect();
    const pad = 8;
    let adjustedY = contextMenu.y;
    let adjustedX = contextMenu.x;

    if (rect.bottom > window.innerHeight - pad) {
      adjustedY = Math.max(pad, window.innerHeight - rect.height - pad);
    }
    if (rect.right > window.innerWidth - pad) {
      adjustedX = Math.max(pad, window.innerWidth - rect.width - pad);
    }

    if (adjustedY !== contextMenu.y || adjustedX !== contextMenu.x) {
      setContextMenu(prev => (prev ? { ...prev, x: adjustedX, y: adjustedY } : null));
    }
  }, [contextMenu?.x, contextMenu?.y]);

  const copyText = async (text: string, flag: 'selection' | 'line') => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedFlag(flag);
      setTimeout(() => setCopiedFlag(null), 1200);
    } catch (err) {
      console.error('複製日誌失敗:', err);
    }
  };

  const handleMenuCopySelection = () => {
    const sel = window.getSelection()?.toString() || '';
    if (!sel) return;
    copyText(sel, 'selection');
    setContextMenu(null);
  };

  const handleMenuCopyLine = () => {
    if (!contextMenu) return;
    copyText(contextMenu.lineText, 'line');
    setContextMenu(null);
  };

  const handleMenuSelectAll = () => {
    if (!containerRef.current) return;
    const range = document.createRange();
    range.selectNodeContents(containerRef.current);
    const sel = window.getSelection();
    sel?.removeAllRanges();
    sel?.addRange(range);
    setContextMenu(null);
  };

  const handleMenuOpenFile = async () => {
    const category = activeTab;
    const sub = activeTab === 'runtime' ? activeRuntimeProject : '';
    setContextMenu(null);
    if (!category) return;
    try {
      await OpenCategoryLogFile(category, sub);
    } catch (err) {
      console.error('開啟日誌檔失敗:', err);
      (window as any).customAlert(`${t('開啟檔案失敗')}: ${err}`);
    }
  };

  const handleMenuOpenFolder = async () => {
    const category = activeTab;
    const sub = activeTab === 'runtime' ? activeRuntimeProject : '';
    setContextMenu(null);
    if (!category) return;
    try {
      await OpenCategoryLogFolder(category, sub);
    } catch (err) {
      console.error('開啟所在資料夾失敗:', err);
      (window as any).customAlert(`${t('開啟所在資料夾失敗')}: ${err}`);
    }
  };

  const openFileDisabled = !logFileInfo?.exists || (activeTab === 'runtime' && !activeRuntimeProject);
  const openFileTitle = logFileInfo?.exists
    ? (logFileInfo.path || logFileInfo.name)
    : (logFileInfo?.path
      ? `${t('日誌檔案不存在')}: ${logFileInfo.path}`
      : (activeTab === 'runtime' && !activeRuntimeProject
        ? t('暫無運行專案')
        : t('日誌檔案不存在')));

  const getTabCount = (tabId: string): number => {
    if (tabId === 'runtime') {
      if (activeRuntimeProject && logs.runtime[activeRuntimeProject]) {
        return logs.runtime[activeRuntimeProject].length;
      }
      return Object.values(logs.runtime).reduce((acc, lines) => acc + (lines?.length || 0), 0);
    }
    const list = (logs as any)[tabId];
    return Array.isArray(list) ? list.length : 0;
  };

  return (
    <div
      className="flex flex-col h-full overflow-hidden select-none relative"
      style={{ backgroundColor: 'var(--bg-deep)' }}
    >
      {/* 分頁 Tab 與控制項 */}
      <div
        className="flex justify-between items-center border-b px-3 select-none"
        style={{ backgroundColor: 'var(--surface)', borderBottomColor: 'var(--border)' }}
      >
        <div className="flex overflow-x-auto scrollbar-none">
          {CATEGORIES.map(tab => {
            const isActive = activeTab === tab.id;
            const count = getTabCount(tab.id);
            const unread = unreadCounts[tab.id] || 0;
            const unreadSuffix = unread > 0 ? ` (${t('未讀 %d 行', unread)})` : '';
            const tooltip = tab.id === 'runtime' && activeRuntimeProject
              ? `${t(tab.label)} (${activeRuntimeProject}) - ${t('共 %d 行', count)}${unreadSuffix}`
              : `${t(tab.label)} - ${t('共 %d 行', count)}${unreadSuffix}`;

            return (
              <button
                key={tab.id}
                onClick={() => {
                  setActiveTab(tab.id);
                  setUnreadTabs(prev => ({ ...prev, [tab.id]: false }));
                  setUnreadCounts(prev => ({ ...prev, [tab.id]: 0 }));
                }}
                title={tooltip}
                className={`log-tab-btn font-bold shrink-0 flex items-center gap-1.5 ${
                  isActive ? 'active' : ''
                }`}
              >
                <span>{t(tab.label)}</span>
                {unreadTabs[tab.id] && !isActive && (
                  <span className="w-1.5 h-1.5 rounded-full inline-block" style={{ backgroundColor: 'var(--status-warn)' }} />
                )}
              </button>
            );
          })}
        </div>

        <div className="flex items-center gap-1 py-1 shrink-0">
          {/* 自動切換分頁開關 (幽靈按鈕風格) */}
          <button
            type="button"
            onClick={() => {
              const val = !autoSwitchTab;
              setAutoSwitchTab(val);
              localStorage.setItem('wincmp_auto_switch_tab', val ? 'true' : 'false');
            }}
            className={`log-action-btn cursor-pointer select-none transition ${
              autoSwitchTab
                ? 'text-[var(--accent)] bg-[var(--card-hover)]'
                : 'text-[var(--muted)] hover:text-[var(--fg-2)] hover:bg-[var(--card-hover)]'
            }`}
            title={t("有新日誌時自動切換到該分頁")}
          >
            <Repeat size={12} className={autoSwitchTab ? 'text-[var(--accent)]' : 'text-[var(--muted)]'} />
            <span>{t("自動切換")}</span>
          </button>

          {/* Runtime 專案自訂下拉選單 */}
          {activeTab === 'runtime' && (
            <div className="relative flex items-center" ref={runtimeDropdownRef}>
              <button
                type="button"
                onClick={(e) => {
                  e.stopPropagation();
                  setRuntimeDropdownOpen((v) => !v);
                }}
                className="log-action-btn font-medium cursor-pointer transition flex items-center gap-1.5"
                style={{
                  backgroundColor: 'var(--surface-warm)',
                  color: 'var(--accent)',
                  border: '1px solid transparent',
                }}
                title={t("選擇專案日誌")}
              >
                <span className="max-w-[140px] truncate">
                  {activeRuntimeProject || t("暫無運行專案")}
                </span>
                {hasUnreadOtherProject && (
                  <span className="w-1.5 h-1.5 rounded-full inline-block shrink-0" style={{ backgroundColor: 'var(--status-warn)' }} />
                )}
                <ChevronDown size={12} style={{ color: 'var(--accent)' }} />
              </button>
              {runtimeDropdownOpen && (
                <div
                  className="absolute right-0 top-full mt-1 z-50 min-w-[180px] max-h-56 overflow-y-auto rounded-lg border py-1 shadow-xl backdrop-blur-sm"
                  style={{
                    backgroundColor: 'var(--menu-bg, var(--bg-deep))',
                    borderColor: 'var(--menu-border, var(--border))',
                    boxShadow: 'var(--menu-shadow, var(--shadow-lg))',
                  }}
                >
                  {runtimeProjects.length > 0 ? (
                    runtimeProjects.map((proj) => {
                      const pUnread = unreadProjectCounts[proj] || 0;
                      const pTitle = pUnread > 0 ? `${proj} (${t('未讀 %d 行', pUnread)})` : proj;
                      return (
                        <button
                          key={proj}
                          type="button"
                          onClick={() => selectRuntimeProject(proj)}
                          title={pTitle}
                          className="w-full text-left px-3 py-2 text-xs flex items-center justify-between gap-2 transition hover:bg-[var(--card-hover)]"
                          style={{
                            color: proj === activeRuntimeProject ? 'var(--accent)' : 'var(--fg-2)',
                            backgroundColor: proj === activeRuntimeProject ? 'var(--surface-warm)' : 'transparent',
                          }}
                        >
                          <span className="truncate flex-1">{proj}</span>
                          {unreadProjects[proj] && proj !== activeRuntimeProject && (
                            <span className="w-1.5 h-1.5 rounded-full inline-block shrink-0" style={{ backgroundColor: 'var(--status-warn)' }} />
                          )}
                        </button>
                      );
                    })
                  ) : (
                    <div className="px-3 py-2 text-xs" style={{ color: 'var(--meta)' }}>{t("暫無運行專案")}</div>
                  )}
                </div>
              )}
            </div>
          )}

          {!autoScroll && (
            <button
              onClick={() => {
                setAutoScroll(true);
                logEndRef.current?.scrollIntoView({ behavior: 'smooth' });
              }}
              className="log-action-btn font-medium text-[var(--accent)] hover:bg-[var(--card-hover)] transition"
              title={t("滾動至最新日誌")}
            >
              <ArrowDown size={12} />
              <span>{t("置底")}</span>
            </button>
          )}
          <button
            onClick={handleClearLogs}
            className="log-action-btn font-medium text-[var(--muted)] hover:text-[var(--status-error)] hover:bg-[var(--card-hover)] transition"
            title={t("清空當前日誌")}
          >
            <Trash2 size={12} />
            <span>{t("清空")}</span>
          </button>

          {/* 收起日誌按鈕 (單行整合) */}
          {onCollapse && (
            <>
              <div className="h-3 w-[1px] mx-0.5" style={{ backgroundColor: 'var(--border)' }} />
              <button
                onClick={onCollapse}
                className="log-action-btn text-[var(--muted)] hover:text-[var(--fg-2)] hover:bg-[var(--card-hover)] transition !px-1.5"
                title={t("收起日誌")}
              >
                <ChevronDown size={14} />
              </button>
            </>
          )}
        </div>
      </div>

      {/* 日誌內容展示區 */}
      <div
        ref={containerRef}
        onScroll={handleScroll}
        className="flex-1 p-5 overflow-y-auto font-mono text-[11px] leading-relaxed select-text terminal-log-content"
        style={{ backgroundColor: 'var(--bg-deep)', color: 'var(--fg-2)' }}
      >
        {currentTabLogs.length > 0 ? (
          <div className="whitespace-pre-wrap break-all space-y-0.5">
            {currentTabLogs.map((line, idx) => (
              <div
                key={idx}
                className="hover:bg-[var(--card-hover)] px-1 py-0.5 rounded transition duration-75"
                onContextMenu={(e) => handleContextMenu(e, idx, line.text)}
              >
                <span
                  className="select-none mr-2 font-semibold"
                  style={{ color: 'var(--meta)' }}
                >
                  [{line.time}]
                </span>
                <span style={getLineStyle(line.text)}>{line.text}</span>
              </div>
            ))}
            <div ref={logEndRef} />
          </div>
        ) : (
          <div
            className="h-full flex items-center justify-center select-none italic text-xs font-semibold"
            style={{ color: 'var(--meta)' }}
          >
            {t("暫時沒有日誌輸出")}
          </div>
        )}
      </div>

      {/* 自訂右鍵選單 */}
      {contextMenu && (
        <div
          ref={menuRef}
          className="fixed z-[100] min-w-[160px] rounded-lg border py-1 shadow-xl backdrop-blur-sm"
          style={{
            left: contextMenu.x,
            top: contextMenu.y,
            backgroundColor: 'var(--menu-bg, var(--bg-deep))',
            borderColor: 'var(--menu-border, var(--border))',
            boxShadow: 'var(--menu-shadow, var(--shadow-lg))',
          }}
          onClick={(e) => e.stopPropagation()}
        >
          <button
            type="button"
            onClick={handleMenuCopySelection}
            disabled={!window.getSelection()?.toString()}
            className="w-full text-left px-3 py-2 text-xs flex items-center gap-2 transition hover:bg-[var(--card-hover)] disabled:opacity-40 disabled:cursor-not-allowed"
            style={{ color: 'var(--fg-2)' }}
          >
            {copiedFlag === 'selection' ? <Check size={12} /> : <Copy size={12} />}
            <span>{t("複製")}</span>
          </button>
          <button
            type="button"
            onClick={handleMenuCopyLine}
            className="w-full text-left px-3 py-2 text-xs flex items-center gap-2 transition hover:bg-[var(--card-hover)]"
            style={{ color: 'var(--fg-2)' }}
          >
            {copiedFlag === 'line' ? <Check size={12} /> : <Copy size={12} />}
            <span>{t("複製此行")}</span>
          </button>
          <button
            type="button"
            onClick={handleMenuSelectAll}
            className="w-full text-left px-3 py-2 text-xs flex items-center gap-2 transition hover:bg-[var(--card-hover)]"
            style={{ color: 'var(--fg-2)' }}
          >
            <ListChecks size={12} />
            <span>{t("全選")}</span>
          </button>
          <div className="my-1 border-t" style={{ borderColor: 'var(--border-soft)' }} />
          <button
            type="button"
            onClick={handleMenuOpenFile}
            disabled={openFileDisabled}
            title={openFileTitle}
            className="w-full text-left px-3 py-2 text-xs flex items-center gap-2 transition hover:bg-[var(--card-hover)] disabled:opacity-40 disabled:cursor-not-allowed"
            style={{ color: 'var(--fg-2)' }}
          >
            <FolderOpen size={12} />
            <span className="truncate">{t("開啟檔案")}</span>
            {logFileInfo?.name && (
              <span className="ml-auto text-[10px] truncate max-w-[90px]" style={{ color: 'var(--meta)' }}>
                {logFileInfo.name}
              </span>
            )}
          </button>
          <button
            type="button"
            onClick={handleMenuOpenFolder}
            disabled={activeTab === 'runtime' && !activeRuntimeProject}
            title={t("開啟檔案所在資料夾")}
            className="w-full text-left px-3 py-2 text-xs flex items-center gap-2 transition hover:bg-[var(--card-hover)] disabled:opacity-40 disabled:cursor-not-allowed"
            style={{ color: 'var(--fg-2)' }}
          >
            <FolderSearch size={12} />
            <span className="truncate">{t("開啟檔案所在資料夾")}</span>
          </button>
        </div>
      )}
    </div>
  );
}

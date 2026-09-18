import React, { useState, useEffect, useRef } from 'react';
import { Trash2, ArrowDown } from 'lucide-react';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { logStore, LogData, LogLine } from './logStore';
import { t, useLanguage } from '../i18n';

const CATEGORIES = [
  { id: 'system', label: '系統' },
  { id: 'caddy', label: 'Caddy' },
  { id: 'mariadb', label: 'MariaDB' },
  { id: 'mailpit', label: 'Mailpit' },
  { id: 'redis', label: 'Redis' },
  { id: 'php', label: 'PHP' },
  { id: 'runtime', label: 'Node / Bun' }
];

export default function TerminalLogs() {
  useLanguage(); // 訂閱語系變更
  const [activeTab, setActiveTab] = useState('system');
  const [activeRuntimeProject, setActiveRuntimeProject] = useState('');
  const [logs, setLogs] = useState<LogData>(logStore.getLogs());
  const [autoScroll, setAutoScroll] = useState(true);
  const [autoSwitchTab, setAutoSwitchTab] = useState<boolean>(() => {
    return localStorage.getItem('wincmp_auto_switch_tab') === 'true'; // 預設為 false，避免搶焦點
  });
  const [unreadTabs, setUnreadTabs] = useState<Record<string, boolean>>({});

  const logEndRef = useRef<HTMLDivElement | null>(null);
  const containerRef = useRef<HTMLDivElement | null>(null);

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

  // 訂閱 Go 端的日誌 Event
  useEffect(() => {
    const handleAutoSwitch = (data: any) => {
      if (!data || !data.category) return;
      const category = data.category === 'node' ? 'runtime' : data.category;
      const projName = category === 'runtime' ? data.projectName : undefined;

      const isValidCategory = ['system', 'caddy', 'mariadb', 'mailpit', 'php', 'redis', 'runtime'].includes(category);
      if (!isValidCategory) return;

      // 若未開啟自動跳轉，只在非當前 tab 標註有未讀日誌，絕不強行奪取焦點
      if (!autoSwitchTabRef.current) {
        if (category !== activeTabRef.current) {
          setUnreadTabs(prev => ({ ...prev, [category]: true }));
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
          }
          if (category === 'runtime' && projName) {
            setActiveRuntimeProject(projName);
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
        } else {
          setActiveRuntimeProject('');
        }
      }
    }
  }, [activeTab, runtimeProjects, activeRuntimeProject]);

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

  return (
    <div
      className="flex flex-col h-full overflow-hidden select-none"
      style={{ backgroundColor: 'var(--bg-deep)' }}
    >
      {/* 分頁 Tab 與控制項 */}
      <div
        className="flex justify-between items-center border-b px-3 select-none"
        style={{ backgroundColor: 'var(--surface)', borderBottomColor: 'var(--border)' }}
      >
        <div className="flex overflow-x-auto scrollbar-none">
          {CATEGORIES.map(tab => (
            <button
              key={tab.id}
              onClick={() => {
                setActiveTab(tab.id);
                setUnreadTabs(prev => ({ ...prev, [tab.id]: false }));
              }}
              className={`log-tab-btn font-bold border-b-2 transition duration-200 shrink-0 flex items-center gap-1.5 ${
                activeTab === tab.id
                  ? ''
                  : 'hover:text-[var(--fg)]'
              }`}
              style={activeTab === tab.id
                ? { borderBottomColor: 'var(--accent)', color: 'var(--accent)', backgroundColor: 'var(--card-hover)' }
                : { borderBottomColor: 'transparent', color: 'var(--muted)' }
              }
            >
              <span>{t(tab.label)}</span>
              {unreadTabs[tab.id] && activeTab !== tab.id && (
                <span className="w-1.5 h-1.5 rounded-full inline-block" style={{ backgroundColor: 'var(--status-warn)' }} />
              )}
            </button>
          ))}
        </div>

        <div className="flex items-center gap-2 py-1.5 shrink-0">
          {/* 自動切換分頁開關 */}
          <label
            className="log-control-btn cursor-pointer select-none border transition hover:border-[color:var(--accent)]"
            style={{ backgroundColor: 'var(--input-bg)', borderColor: 'var(--border)', color: autoSwitchTab ? 'var(--accent)' : 'var(--muted)' }}
            title={t("有新日誌時自動切換到該分頁")}
          >
            <input
              type="checkbox"
              checked={autoSwitchTab}
              onChange={(e) => {
                const val = e.target.checked;
                setAutoSwitchTab(val);
                localStorage.setItem('wincmp_auto_switch_tab', val ? 'true' : 'false');
              }}
              className="w-3.5 h-3.5 rounded cursor-pointer accent-blue-500"
            />
            <span>{t("自動切換")}</span>
          </label>

          {/* Runtime 專案下拉選單 */}
          {activeTab === 'runtime' && (
            <div className="flex items-center">
              <select
                value={activeRuntimeProject}
                onChange={(e) => setActiveRuntimeProject(e.target.value)}
                className="log-control-select border font-medium focus:outline-none focus:border-[color:var(--accent)] cursor-pointer"
                style={{ backgroundColor: 'var(--input-bg)', borderColor: 'var(--border)', color: 'var(--accent)' }}
                title={t("選擇專案日誌")}
              >
                {runtimeProjects.length > 0 ? (
                  runtimeProjects.map((proj) => (
                    <option key={proj} value={proj} style={{ backgroundColor: 'var(--bg)', color: 'var(--fg-2)' }}>
                      {proj}
                    </option>
                  ))
                ) : (
                  <option value="" style={{ backgroundColor: 'var(--bg)', color: 'var(--fg-2)' }}>{t("暫無運行專案")}</option>
                )}
              </select>
            </div>
          )}

          {!autoScroll && (
            <button
              onClick={() => {
                setAutoScroll(true);
                logEndRef.current?.scrollIntoView({ behavior: 'smooth' });
              }}
              className="log-control-btn btn-custom-hover border font-bold"
              style={{ borderColor: 'var(--border)', backgroundColor: 'var(--input-bg)', color: 'var(--accent)' }}
              title={t("滾動至最新日誌")}
            >
              <ArrowDown size={12} />
              <span>{t("置底")}</span>
            </button>
          )}
          <button
            onClick={handleClearLogs}
            className="log-control-btn btn-danger-hover border font-bold"
            style={{ borderColor: 'var(--border)', backgroundColor: 'var(--input-bg)', color: 'var(--status-error)' }}
            title={t("清空當前日誌")}
          >
            <Trash2 size={12} />
            <span>{t("清空")}</span>
          </button>
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
    </div>
  );
}

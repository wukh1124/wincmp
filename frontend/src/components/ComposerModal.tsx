import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  X, Package, Play, Square, Trash2,
  RotateCw, Copy, Check, Clock, AlertTriangle,
  ChevronDown, ChevronRight, ArrowDown, WrapText
} from 'lucide-react';
import { EventsOn, BrowserOpenURL } from '../../wailsjs/runtime/runtime';
import {
  RunComposerCommand,
  StopComposerCommand,
  GetComposerHistory,
  ClearComposerHistory,
} from '../../wailsjs/go/main/App';
import { composer } from '../../wailsjs/go/models';
import { t, useLanguage } from '../i18n';

interface ProjectInfo {
  name: string;
  php_version: string;
  root_path: string;
  type?: string;
}

interface ComposerModalProps {
  project: ProjectInfo | null;
  scanResult: any;
  isOpen: boolean;
  onClose: () => void;
}

interface CommandPreset {
  cmd: string;
  display?: string;
  descKey: string;
}

const LARAVEL_COMMAND_PRESETS: CommandPreset[] = [
  { cmd: 'install', descKey: '依 composer.lock 精準安裝依賴套件' },
  { cmd: 'install --no-dev --optimize-autoloader', descKey: '正式環境安裝（排除開發依賴並最佳化自動載入）' },
  { cmd: 'install --ignore-platform-reqs', descKey: '安裝依賴並忽略 PHP 版本與擴充模組檢查' },
  { cmd: 'update', descKey: '更新全部依賴套件並更新 composer.lock' },
  { cmd: 'update ', display: 'update <package> -W', descKey: '更新指定套件（建議搭配 -W 連帶更新相依）' },
  { cmd: 'update --dry-run', descKey: '模擬更新全部套件，不實際改動檔案' },
  { cmd: 'require ', display: 'require <package>', descKey: '新增一般依賴套件' },
  { cmd: 'require --dev ', display: 'require --dev <package>', descKey: '新增開發環境專用套件' },
  { cmd: 'remove ', display: 'remove <package>', descKey: '移除已安裝的套件' },
  { cmd: 'dump-autoload -o', descKey: '重建最佳化類別自動載入（修復 Laravel class not found）' },
  { cmd: 'dump-autoload', descKey: '重建類別自動載入檔' },
  { cmd: 'show', descKey: '顯示目前已安裝的套件列表' },
  { cmd: 'show ', display: 'show <package>', descKey: '檢視指定套件詳細版本與依賴資訊' },
  { cmd: 'show --tree', descKey: '以樹狀結構顯示套件相依關係' },
  { cmd: 'outdated', descKey: '列出有新版本可升級的過期套件' },
  { cmd: 'outdated --direct', descKey: '僅列出 composer.json 直接依賴的可升級套件' },
  { cmd: 'audit', descKey: '檢查已安裝套件是否有已知安全漏洞 (CVE)' },
  { cmd: 'status', descKey: '檢查本機 vendor 套件是否有未提交的修改' },
  { cmd: 'why ', display: 'why <package>', descKey: '查詢某個套件為何被安裝（相依鏈追蹤）' },
  { cmd: 'why-not ', display: 'why-not <package> <version>', descKey: '查詢套件為何無法升級至指定版本' },
  { cmd: 'search ', display: 'search <keyword>', descKey: '搜尋 Packagist 上的公開套件' },
  { cmd: 'validate', descKey: '驗證 composer.json 與鎖定檔格式是否正確' },
  { cmd: 'check-platform-reqs', descKey: '檢查目前 PHP 版本與擴充模組是否符合專案要求' },
  { cmd: 'run-script --list', descKey: '列出 composer.json 中定義的可用腳本' },
  { cmd: 'diagnose', descKey: '診斷網路、SSL 憑證與 Composer 本機環境問題' },
  { cmd: 'licenses', descKey: '列出所有已安裝套件的授權協議' },
  { cmd: 'clear-cache', descKey: '清除 Composer 本機下載快取' },
];

const MAX_LIVE_LINES = 8000;

// 解析支援引號的指令參數，例如：require "laravel/sanctum:^4.0"
function parseCommandArgs(input: string): string[] {
  const args: string[] = [];
  let current = '';
  let quote: '"' | "'" | null = null;

  for (let i = 0; i < input.length; i++) {
    const ch = input[i];
    if (quote) {
      if (ch === quote) {
        quote = null;
      } else {
        current += ch;
      }
      continue;
    }
    if (ch === '"' || ch === "'") {
      quote = ch;
    } else if (/\s/.test(ch)) {
      if (current.length > 0) {
        args.push(current);
        current = '';
      }
    } else {
      current += ch;
    }
  }
  if (current.length > 0) {
    args.push(current);
  }
  return args;
}

function formatDateTime(iso: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

// 輕量安全的 ANSI SGR 色彩解析器（不使用 dangerouslySetInnerHTML）
const ANSI_COLOR_MAP: Record<number, string> = {
  30: '#71717a',
  31: 'var(--status-error)',
  32: 'var(--status-ok)',
  33: 'var(--status-warn)',
  34: 'var(--status-info)',
  35: '#c084fc',
  36: '#22d3ee',
  37: 'var(--fg)',
  90: 'var(--meta)',
  91: '#f87171',
  92: '#4ade80',
  93: '#facc15',
  94: '#60a5fa',
  95: '#d8b4fe',
  96: '#67e8f9',
  97: 'var(--fg)',
};

function isSafeExternalUrl(url: string): boolean {
  return /^https?:\/\//i.test(url);
}

function stripAnsiAndOsc(raw: string): string {
  if (!raw) return '';
  return raw
    .replace(/\r\n/g, '\n')
    .replace(/\r/g, '\n')
    .replace(/\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)/g, '')
    .replace(/\x1b\[[0-9;?]*[A-Za-z]/g, '');
}

function renderAnsiText(raw: string): React.ReactNode {
  if (!raw) return null;
  const clean = raw.replace(/\r\n/g, '\n').replace(/\r/g, '\n');
  // 同時支援 OSC 8 終端超連結 (\x1b]8;;URL\x1b\\text\x1b]8;;\x1b\\)、其他 OSC 序列、SGR 色彩碼與其餘 CSI 控制碼
  const tokenRegex = /\x1b\]8;[^;\x07\x1b]*;([^\x07\x1b]*)(?:\x07|\x1b\\)|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b\[([0-9;]*)m|\x1b\[[0-9;?]*[A-Za-z]/g;
  const nodes: React.ReactNode[] = [];
  let lastIdx = 0;
  let color: string | undefined;
  let bold = false;
  let activeLink: string | undefined;
  let key = 0;
  let match: RegExpExecArray | null;

  const pushSegment = (seg: string) => {
    if (!seg) return;
    const style: React.CSSProperties = { color, fontWeight: bold ? 700 : undefined };
    if (activeLink && isSafeExternalUrl(activeLink)) {
      const targetUrl = activeLink;
      nodes.push(
        <span
          key={key++}
          onClick={() => BrowserOpenURL(targetUrl)}
          title={targetUrl}
          className="hover:underline cursor-pointer"
          style={style}
        >
          {seg}
        </span>
      );
      return;
    }
    nodes.push(
      <span key={key++} style={style}>
        {seg}
      </span>
    );
  };

  while ((match = tokenRegex.exec(clean)) !== null) {
    if (match.index > lastIdx) {
      pushSegment(clean.slice(lastIdx, match.index));
    }

    if (match[1] !== undefined) {
      // OSC 8 超連結開關：非空 URL 開啟連結，空字串關閉連結
      const url = match[1].trim();
      activeLink = url && isSafeExternalUrl(url) ? url : undefined;
    } else if (match[2] !== undefined) {
      // CSI SGR 色彩與粗體
      const codes = match[2] ? match[2].split(';').map(Number) : [0];
      for (const c of codes) {
        if (c === 0) {
          color = undefined;
          bold = false;
        } else if (c === 1) {
          bold = true;
        } else if (c === 22) {
          bold = false;
        } else if (c === 39) {
          color = undefined;
        } else if (ANSI_COLOR_MAP[c]) {
          color = ANSI_COLOR_MAP[c];
        }
      }
    }
    // 其餘 OSC / 非 SGR CSI 控制序列直接忽略過濾

    lastIdx = tokenRegex.lastIndex;
  }

  if (lastIdx < clean.length) {
    pushSegment(clean.slice(lastIdx));
  }
  return nodes;
}

export default function ComposerModal({ project, scanResult, isOpen, onClose }: ComposerModalProps) {
  useLanguage();
  const [selectedComposerVer, setSelectedComposerVer] = useState<string>('');
  const [commandInput, setCommandInput] = useState<string>('');
  const [showSuggestions, setShowSuggestions] = useState<boolean>(false);
  const [highlightedIndex, setHighlightedIndex] = useState<number>(-1);

  const [historyItems, setHistoryItems] = useState<composer.HistoryEntry[]>([]);
  const [historyTotal, setHistoryTotal] = useState<number>(0);
  const [hasMore, setHasMore] = useState<boolean>(false);
  const [loadingHistory, setLoadingHistory] = useState<boolean>(false);
  const [loadingMore, setLoadingMore] = useState<boolean>(false);

  const [activeRunId, setActiveRunId] = useState<string | null>(null);
  const [activeCommand, setActiveCommand] = useState<string>('');
  const [activeStartedAt, setActiveStartedAt] = useState<string>('');
  const [liveOutput, setLiveOutput] = useState<string>('');
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [expandedIds, setExpandedIds] = useState<Set<string>>(() => new Set());
  const [wrappedIds, setWrappedIds] = useState<Set<string>>(() => new Set());
  const [fullOutputIds, setFullOutputIds] = useState<Set<string>>(() => new Set());
  const [visibleLimit, setVisibleLimit] = useState<number>(5);
  const [showScrollToBottom, setShowScrollToBottom] = useState<boolean>(false);

  const outputContainerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const suggestionRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const activeRunIdRef = useRef<string | null>(null);
  const liveOutputRef = useRef<string>('');

  useEffect(() => {
    activeRunIdRef.current = activeRunId;
  }, [activeRunId]);

  useEffect(() => {
    liveOutputRef.current = liveOutput;
  }, [liveOutput]);

  const composerList: Array<{ Version: string; ExePath: string }> = scanResult?.ComposerList || [];
  const phpList: Array<{ Version: string; MajorMin: string }> = scanResult?.PHPList || [];

  const matchedPHP = useMemo(() => {
    if (!project?.php_version) return null;
    return phpList.find((p) => p.MajorMin === project.php_version) || null;
  }, [phpList, project?.php_version]);

  // 初始化 Composer 預設版本
  useEffect(() => {
    if (!isOpen) return;
    if (composerList.length > 0 && !composerList.some((c) => c.Version === selectedComposerVer)) {
      setSelectedComposerVer(composerList[0].Version);
    }
  }, [isOpen, composerList]);

  // 開啟面板時載入最新 5 筆歷史紀錄（不自動聚焦輸入框，避免直接彈出提示清單）
  useEffect(() => {
    if (!isOpen || !project) return;
    setCommandInput('');
    setShowSuggestions(false);
    setHighlightedIndex(-1);
    setLiveOutput('');
    liveOutputRef.current = '';
    setActiveRunId(null);
    setExpandedIds(new Set());
    setFullOutputIds(new Set());
    setVisibleLimit(5);
    setShowScrollToBottom(false);
    setLoadingHistory(true);

    GetComposerHistory(project.name, 0, 5)
      .then((page: composer.HistoryPage) => {
        const items = page?.items || [];
        setHistoryItems(items);
        setHistoryTotal(page?.total || 0);
        setHasMore(!!page?.has_more);
        // 預設歷史記錄都收起，僅展開最新一條
        if (items.length > 0) {
          setExpandedIds(new Set([items[items.length - 1].id]));
        }
        requestAnimationFrame(() => {
          if (outputContainerRef.current) {
            outputContainerRef.current.scrollTop = outputContainerRef.current.scrollHeight;
          }
        });
      })
      .catch((err: unknown) => {
        console.error('載入 Composer 歷史失敗:', err);
      })
      .finally(() => {
        setLoadingHistory(false);
      });

    return () => {
      if (activeRunIdRef.current) {
        StopComposerCommand(activeRunIdRef.current).catch(() => {});
      }
    };
  }, [isOpen, project?.name]);

  // 監聽即時串流輸出與結束事件
  useEffect(() => {
    if (!isOpen) return;

    const unsubOutput = EventsOn('composer_output', (res: { runId: string; data: string }) => {
      if (!res || res.runId !== activeRunIdRef.current) return;
      setLiveOutput((prev) => {
        const next = prev + res.data;
        const lines = next.split('\n');
        if (lines.length > MAX_LIVE_LINES) {
          return lines.slice(lines.length - MAX_LIVE_LINES).join('\n');
        }
        return next;
      });
      requestAnimationFrame(() => {
        if (outputContainerRef.current) {
          outputContainerRef.current.scrollTop = outputContainerRef.current.scrollHeight;
        }
      });
    });

    const unsubExit = EventsOn('composer_exit', (res: { runId: string; entry: composer.HistoryEntry }) => {
      if (!res || res.runId !== activeRunIdRef.current) return;
      const sessionFullOutput = liveOutputRef.current;
      setActiveRunId(null);
      setLiveOutput('');
      liveOutputRef.current = '';
      if (res.entry) {
        let displayEntry = res.entry;
        // 首次執行完成時保留記憶體中的完整即時輸出（歷史檔仍按頭尾截斷儲存）
        if (res.entry.truncated && sessionFullOutput.length > 0) {
          displayEntry = composer.HistoryEntry.createFrom({
            ...res.entry,
            output: sessionFullOutput,
          });
          setFullOutputIds((prev) => {
            const next = new Set(prev);
            next.add(res.entry.id);
            return next;
          });
        }
        setHistoryItems((prev) => [...prev, displayEntry]);
        setHistoryTotal((prev) => prev + 1);
        setExpandedIds(new Set([res.entry.id]));
      }
      requestAnimationFrame(() => {
        if (outputContainerRef.current) {
          outputContainerRef.current.scrollTop = outputContainerRef.current.scrollHeight;
        }
      });
    });

    return () => {
      unsubOutput();
      unsubExit();
    };
  }, [isOpen]);

  const visibleItems = useMemo(
    () => historyItems.slice(Math.max(0, historyItems.length - visibleLimit)),
    [historyItems, visibleLimit]
  );
  const hiddenCount = Math.max(0, Math.max(historyTotal, historyItems.length) - visibleItems.length);

  const updateScrollToBottomState = () => {
    const el = outputContainerRef.current;
    if (!el) return;
    setShowScrollToBottom(el.scrollHeight - el.scrollTop - el.clientHeight > 80);
  };

  useEffect(() => {
    if (!isOpen) return;
    const raf = requestAnimationFrame(updateScrollToBottomState);
    return () => cancelAnimationFrame(raf);
  }, [isOpen, visibleItems, expandedIds, liveOutput, activeRunId]);

  const handleLoadMore = async () => {
    if (!project || loadingMore || hiddenCount <= 0) return;
    const container = outputContainerRef.current;
    const prevScrollHeight = container ? container.scrollHeight : 0;
    const nextLimit = visibleLimit + 5;

    setLoadingMore(true);
    try {
      if (historyItems.length < nextLimit && hasMore) {
        const page = await GetComposerHistory(project.name, historyItems.length, 5);
        if (page?.items && page.items.length > 0) {
          setHistoryItems((prev) => [...page.items, ...prev]);
        }
        setHistoryTotal(page?.total ?? historyTotal);
        setHasMore(!!page?.has_more);
      }
      setVisibleLimit(nextLimit);

      requestAnimationFrame(() => {
        if (container) {
          container.scrollTop += container.scrollHeight - prevScrollHeight;
        }
        updateScrollToBottomState();
      });
    } catch (err) {
      console.error('載入更早 Composer 紀錄失敗:', err);
    } finally {
      setLoadingMore(false);
    }
  };

  const handleScroll = () => {
    updateScrollToBottomState();
  };

  const handleScrollToBottom = () => {
    const el = outputContainerRef.current;
    if (!el) return;
    el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' });
  };

  const executeCommand = async (rawCmd: string) => {
    if (!project || activeRunId) return;
    const trimmed = rawCmd.trim();
    if (!trimmed) return;

    const args = parseCommandArgs(trimmed);
    if (args.length === 0) return;

    // 若使用者誤輸入開頭的 composer，自動去掉方便相容
    if (args[0].toLowerCase() === 'composer' && args.length > 1) {
      args.shift();
    }

    setShowSuggestions(false);
    setHighlightedIndex(-1);
    setLiveOutput('');
    setActiveCommand(args.join(' '));
    setActiveStartedAt(new Date().toISOString());

    try {
      const runID = await RunComposerCommand(project.name, selectedComposerVer, args);
      setActiveRunId(runID);
      setCommandInput('');
      requestAnimationFrame(() => {
        if (outputContainerRef.current) {
          outputContainerRef.current.scrollTop = outputContainerRef.current.scrollHeight;
        }
      });
    } catch (err: any) {
      setActiveRunId(null);
      (window as any).customAlert(`${t("執行 Composer 失敗")}: ${err}`);
    }
  };

  const handleStop = async () => {
    if (!activeRunId) return;
    try {
      await StopComposerCommand(activeRunId);
    } catch (err) {
      console.error('停止 Composer 失敗:', err);
    }
  };

  const handleClearHistory = async () => {
    if (!project || activeRunId) return;
    const ok = await (window as any).customConfirm(t("確定要清除此專案的所有 Composer 指令歷史紀錄嗎？"));
    if (!ok) return;
    try {
      await ClearComposerHistory(project.name);
      setHistoryItems([]);
      setHistoryTotal(0);
      setHasMore(false);
    } catch (err: any) {
      (window as any).customAlert(`${t("清除歷史紀錄失敗")}: ${err}`);
    }
  };

  const handleCopyOutput = async (id: string, text: string) => {
    try {
      await navigator.clipboard.writeText(stripAnsiAndOsc(text));
      setCopiedId(id);
      setTimeout(() => setCopiedId(null), 1500);
    } catch (err) {
      console.error('複製失敗:', err);
    }
  };

  // 即時匹配指令建議（支援關鍵字過濾指令名稱與說明）
  const filteredSuggestions = useMemo(() => {
    const q = commandInput.trim().toLowerCase();
    if (!q) return LARAVEL_COMMAND_PRESETS;
    return LARAVEL_COMMAND_PRESETS.filter(
      (item) =>
        item.cmd.toLowerCase().includes(q) ||
        (item.display && item.display.toLowerCase().includes(q)) ||
        t(item.descKey).toLowerCase().includes(q)
    );
  }, [commandInput]);

  // 鍵盤上下選取時自動捲動可見範圍
  useEffect(() => {
    if (highlightedIndex >= 0 && suggestionRefs.current[highlightedIndex]) {
      suggestionRefs.current[highlightedIndex]?.scrollIntoView({ block: 'nearest' });
    }
  }, [highlightedIndex]);

  const applySuggestion = (preset: CommandPreset) => {
    setCommandInput(preset.cmd);
    setShowSuggestions(false);
    setHighlightedIndex(-1);
    inputRef.current?.focus();
  };

  const handleInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'ArrowDown') {
      if (filteredSuggestions.length === 0) return;
      e.preventDefault();
      if (!showSuggestions) {
        setShowSuggestions(true);
        setHighlightedIndex(0);
      } else {
        setHighlightedIndex((prev) => (prev + 1) % filteredSuggestions.length);
      }
      return;
    }
    if (e.key === 'ArrowUp') {
      if (filteredSuggestions.length === 0) return;
      e.preventDefault();
      if (!showSuggestions) {
        setShowSuggestions(true);
        setHighlightedIndex(filteredSuggestions.length - 1);
      } else {
        setHighlightedIndex((prev) => (prev <= 0 ? filteredSuggestions.length - 1 : prev - 1));
      }
      return;
    }
    if (e.key === 'Tab') {
      if (showSuggestions && filteredSuggestions.length > 0) {
        e.preventDefault();
        const target = filteredSuggestions[highlightedIndex >= 0 ? highlightedIndex : 0];
        if (target) applySuggestion(target);
      }
      return;
    }
    if (e.key === 'Escape') {
      setShowSuggestions(false);
      setHighlightedIndex(-1);
      return;
    }
    if (e.key === 'Enter') {
      e.preventDefault();
      if (showSuggestions && highlightedIndex >= 0 && filteredSuggestions[highlightedIndex]) {
        applySuggestion(filteredSuggestions[highlightedIndex]);
        return;
      }
      executeCommand(commandInput);
    }
  };

  if (!isOpen || !project) return null;

  const canRun = !!matchedPHP && composerList.length > 0;

  return (
    <div className="fixed inset-0 z-50 overflow-hidden select-none">
      <div
        className="absolute inset-0 transition-opacity duration-300"
        style={{ background: 'var(--overlay-bg)', backdropFilter: 'blur(1px)' }}
        onClick={onClose}
      />
      <div className="absolute inset-y-0 right-0 pl-10 max-w-full flex">
        <div
          className="w-screen max-w-4xl flex flex-col h-full overflow-hidden animate-slide-in"
          style={{ background: 'var(--card)', borderLeft: '1px solid var(--border)', boxShadow: 'var(--shadow-lg)' }}
        >
          {/* Header */}
          <div
            className="px-5 py-4 flex justify-between items-center shrink-0"
            style={{ borderBottom: '1px solid var(--border)', background: 'var(--bg-deep)' }}
          >
            <div className="flex items-center gap-2.5 min-w-0">
              <Package size={16} style={{ color: 'var(--status-info)' }} className="shrink-0" />
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <h3 className="text-sm font-bold tracking-wider" style={{ color: 'var(--fg)', fontFamily: 'var(--font-display)' }}>
                    {t("Composer 控制台")}
                  </h3>
                  <span
                    className="px-2 py-0.5 rounded text-[10px] font-bold"
                    style={{ background: 'var(--status-info-bg)', color: 'var(--status-info)' }}
                  >
                    {project.name}
                  </span>
                </div>
                <p className="text-[10px] mt-0.5 truncate" style={{ color: 'var(--meta)', fontFamily: 'var(--font-mono)' }} title={project.root_path}>
                  {project.root_path}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <button
                type="button"
                onClick={handleClearHistory}
                disabled={!!activeRunId || historyTotal === 0}
                className="btn-danger-hover px-2.5 py-1.5 rounded-lg text-[11px] font-semibold flex items-center gap-1.5 transition disabled:opacity-40"
                style={{ background: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--fg-2)' }}
                title={t("清除此專案的所有指令歷史")}
              >
                <Trash2 size={12} />
                <span>{t("清除歷史")}</span>
              </button>
              <button
                type="button"
                onClick={onClose}
                className="p-1.5 rounded-lg transition hover:bg-[var(--surface-hover)]"
                style={{ color: 'var(--muted)' }}
              >
                <X size={16} />
              </button>
            </div>
          </div>

          {/* Environment Bar */}
          <div
            className="px-5 py-2 flex flex-wrap items-center gap-2 shrink-0 text-[11px]"
            style={{ borderBottom: '1px solid var(--border-soft)', background: 'var(--surface)' }}
          >
            {/* PHP 版本（跟隨專案） */}
            <div
              className="flex items-center gap-1.5 px-2.5 py-1 rounded-md"
              style={{
                background: matchedPHP ? 'var(--status-ok-bg)' : 'var(--status-error-bg)',
                color: matchedPHP ? 'var(--status-ok)' : 'var(--status-error)',
              }}
            >
              <span className="font-bold">
                {matchedPHP ? `PHP ${matchedPHP.Version}` : `PHP ${project.php_version || '?'} (${t("未安裝")})`}
              </span>
              <span className="text-[10px] opacity-80">({t("跟隨專案")})</span>
            </div>

            {/* Composer 版本（偵測多版本時可選） */}
            {composerList.length > 1 ? (
              <div className="flex items-center gap-1.5">
                <span className="font-semibold" style={{ color: 'var(--muted)' }}>Composer:</span>
                <select
                  value={selectedComposerVer}
                  disabled={!!activeRunId}
                  onChange={(e) => setSelectedComposerVer(e.target.value)}
                  className="px-2 py-1 rounded-md text-[11px] font-semibold cursor-pointer"
                  style={{
                    background: 'var(--input-bg)',
                    border: '1px solid var(--input-border)',
                    color: 'var(--fg)',
                  }}
                >
                  {composerList.map((c) => (
                    <option key={c.Version} value={c.Version}>
                      v{c.Version}
                    </option>
                  ))}
                </select>
              </div>
            ) : composerList.length === 1 ? (
              <div
                className="px-2.5 py-1 rounded-md font-bold"
                style={{ background: 'var(--status-info-bg)', color: 'var(--status-info)' }}
              >
                Composer v{composerList[0].Version}
              </div>
            ) : (
              <div
                className="px-2.5 py-1 rounded-md font-bold flex items-center gap-1"
                style={{ background: 'var(--status-error-bg)', color: 'var(--status-error)' }}
              >
                <AlertTriangle size={12} />
                <span>{t("尚未安裝 Composer")}</span>
              </div>
            )}
          </div>

          {/* Output & History Stream */}
          <div className="relative flex-1 overflow-hidden flex flex-col" style={{ background: 'var(--main-content-bg, var(--bg))' }}>
            <div
              ref={outputContainerRef}
              onScroll={handleScroll}
              className="flex-1 overflow-y-auto p-4 space-y-3 select-text"
            >
              {!canRun && (
                <div
                  className="p-3 rounded-xl text-xs flex items-start gap-2.5"
                  style={{ background: 'var(--status-warn-bg)', color: 'var(--status-warn)', border: '1px solid var(--status-warn-bg)' }}
                >
                  <AlertTriangle size={16} className="shrink-0 mt-0.5" />
                  <div>
                    <div className="font-bold">{t("環境尚未就緒")}</div>
                    <div className="text-[11px] opacity-90 mt-0.5">
                      {t("請確認「依賴庫管理」已安裝 Composer 以及此專案設定對應的 PHP 版本。")}
                    </div>
                  </div>
                </div>
              )}

              {hiddenCount > 0 && (
                <div className="flex justify-center">
                  <button
                    type="button"
                    onClick={handleLoadMore}
                    disabled={loadingMore}
                    className="btn-custom-hover rounded-full text-[11px] font-medium transition hover:opacity-90"
                    style={{
                      padding: '5px 14px',
                      background: 'var(--surface)',
                      border: '1px solid var(--border)',
                      color: 'var(--fg-2)',
                    }}
                  >
                    {loadingMore
                      ? t("正在載入更早記錄...")
                      : `${t("顯示更多")} (${hiddenCount})`}
                  </button>
                </div>
              )}

              {loadingHistory && (
                <div className="text-center py-8 text-xs" style={{ color: 'var(--muted)' }}>
                  {t("正在載入執行紀錄...")}
                </div>
              )}

              {!loadingHistory && visibleItems.length === 0 && !activeRunId && (
                <div className="text-center py-12 space-y-1.5" style={{ color: 'var(--muted)' }}>
                  <div className="text-xs font-semibold">{t("尚無 Composer 執行紀錄")}</div>
                  <div className="text-[11px]">
                    {t("點擊下方輸入框可瀏覽常用指令提示，或直接輸入 install、update、show、audit 執行。")}
                  </div>
                </div>
              )}

              {visibleItems.map((item) => {
              const statusColor =
                item.status === 'success'
                  ? 'var(--status-ok)'
                  : item.status === 'timeout'
                    ? 'var(--status-warn)'
                    : item.status === 'stopped'
                      ? 'var(--muted)'
                      : 'var(--status-error)';
              const statusLabel =
                item.status === 'success'
                  ? 'Exit 0'
                  : item.status === 'timeout'
                    ? `${t("逾時")} (${item.error_reason || 'timeout'})`
                    : item.status === 'stopped'
                      ? t("已停止")
                      : `Exit ${item.exit_code}`;

              const isExpanded = expandedIds.has(item.id);
              const isWrapped = wrappedIds.has(item.id);
              const isSessionFull = fullOutputIds.has(item.id);
              const toggleExpand = () => {
                setExpandedIds((prev) => {
                  const next = new Set(prev);
                  if (next.has(item.id)) next.delete(item.id);
                  else next.add(item.id);
                  return next;
                });
              };
              const toggleWrap = (e: React.MouseEvent) => {
                e.stopPropagation();
                setWrappedIds((prev) => {
                  const next = new Set(prev);
                  if (next.has(item.id)) next.delete(item.id);
                  else next.add(item.id);
                  return next;
                });
                if (!isExpanded) {
                  setExpandedIds((prev) => {
                    const next = new Set(prev);
                    next.add(item.id);
                    return next;
                  });
                }
              };
              const fillCommandToInput = (e: React.MouseEvent) => {
                e.stopPropagation();
                setCommandInput(item.command);
                setShowSuggestions(false);
                setHighlightedIndex(-1);
                requestAnimationFrame(() => inputRef.current?.focus());
              };

              return (
                <div
                  key={item.id}
                  className="rounded-xl overflow-hidden"
                  style={{ background: 'var(--card)', border: '1px solid var(--border)' }}
                >
                  {/* Record Header（收起時仍保留時間、狀態與執行指令，點擊可展開/收起） */}
                  <div
                    onClick={toggleExpand}
                    className="px-3 py-2 cursor-pointer hover:bg-[var(--surface-hover)] transition space-y-1"
                    style={{
                      background: 'var(--bg-deep)',
                      borderBottom: isExpanded ? '1px solid var(--border-soft)' : undefined,
                    }}
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2 text-[10px]" style={{ color: 'var(--meta)' }}>
                      <div className="flex flex-wrap items-center gap-2 font-mono">
                        <span className="inline-flex items-center" style={{ color: 'var(--fg-2)' }}>
                          {isExpanded ? <ChevronDown size={13} /> : <ChevronRight size={13} />}
                        </span>
                        <span className="flex items-center gap-1" style={{ color: 'var(--fg-2)' }}>
                          <Clock size={10} />
                          {formatDateTime(item.started_at)}
                        </span>
                        <span>PHP {item.php_version}</span>
                        <span>Composer {item.composer_version}</span>
                        <span>{formatDuration(item.duration_ms)}</span>
                      </div>
                      <div className="flex items-center gap-1.5">
                        <span className="font-bold font-mono mr-0.5" style={{ color: statusColor }}>
                          {statusLabel}
                        </span>
                        <button
                          type="button"
                          onClick={toggleWrap}
                          className="p-1 rounded hover:bg-[var(--surface-hover)] transition"
                          style={{
                            color: isWrapped ? 'var(--status-info)' : 'var(--muted)',
                            background: isWrapped ? 'var(--status-info-bg)' : undefined,
                          }}
                          title={isWrapped ? t("取消自動換行") : t("切換自動換行")}
                        >
                          <WrapText size={11} />
                        </button>
                        <button
                          type="button"
                          onClick={(e) => { e.stopPropagation(); handleCopyOutput(item.id, item.output); }}
                          className="p-1 rounded hover:bg-[var(--surface-hover)] transition"
                          style={{ color: 'var(--muted)' }}
                          title={t("複製輸出內容")}
                        >
                          {copiedId === item.id ? <Check size={11} style={{ color: 'var(--status-ok)' }} /> : <Copy size={11} />}
                        </button>
                        <button
                          type="button"
                          disabled={!canRun || !!activeRunId}
                          onClick={fillCommandToInput}
                          className="p-1 rounded hover:bg-[var(--surface-hover)] transition disabled:opacity-40"
                          style={{ color: 'var(--status-info)' }}
                          title={t("帶入此指令至輸入框")}
                        >
                          <RotateCw size={11} />
                        </button>
                      </div>
                    </div>
                    <div className="pl-5 font-mono text-[11px] font-bold truncate" style={{ color: 'var(--status-info)' }} title={`composer ${item.command}`}>
                      $ composer {item.command}
                    </div>
                  </div>

                  {/* Output Body（僅展開時顯示） */}
                  {isExpanded && (
                    <div className="p-3 font-mono text-[11px] leading-relaxed space-y-1.5">
                      {item.truncated && (
                        <div className="text-[10px]" style={{ color: isSessionFull ? 'var(--status-info)' : 'var(--status-warn)' }}>
                          {isSessionFull
                            ? t("本次顯示完整輸出（歷史僅保存頭尾 32 KB）")
                            : t("輸出已截斷保留頭尾")} · {t("原始大小")} {formatBytes(item.original_bytes)} / {item.original_lines} {t("行")}
                        </div>
                      )}
                      {item.output ? (
                        <div className={isWrapped ? '' : 'overflow-x-auto'}>
                          <pre
                            className={`${isWrapped ? 'whitespace-pre-wrap break-words' : 'whitespace-pre'} m-0 font-mono text-[11px]`}
                            style={{ color: 'var(--fg-2)' }}
                          >
                            {renderAnsiText(item.output)}
                          </pre>
                        </div>
                      ) : (
                        <div className="text-[10px] italic" style={{ color: 'var(--meta)' }}>
                          ({t("無輸出內容")})
                        </div>
                      )}
                    </div>
                  )}
                </div>
              );
            })}

            {/* Live Running Output */}
            {activeRunId && (
              <div
                className="rounded-xl overflow-hidden"
                style={{ background: 'var(--card)', border: '1px solid var(--status-info)' }}
              >
                <div
                  className="px-3 py-1.5 flex items-center justify-between text-[10px]"
                  style={{ background: 'var(--status-info-bg)', color: 'var(--status-info)' }}
                >
                  <div className="flex items-center gap-2 font-mono font-bold">
                    <span className="inline-block w-2 h-2 rounded-full animate-pulse" style={{ background: 'var(--status-info)' }} />
                    <span>{t("執行中...")} ({formatDateTime(activeStartedAt)})</span>
                  </div>
                  <span className="font-mono">Composer v{selectedComposerVer}</span>
                </div>
                <div className="p-3 font-mono text-[11px] leading-relaxed space-y-1.5">
                  <div className="font-bold" style={{ color: 'var(--status-info)' }}>
                    $ composer {activeCommand}
                  </div>
                  <div className="overflow-x-auto">
                    <pre className="whitespace-pre m-0 font-mono text-[11px]" style={{ color: 'var(--fg-2)' }}>
                      {liveOutput ? renderAnsiText(liveOutput) : t("正在啟動 Composer...")}
                    </pre>
                  </div>
                </div>
              </div>
            )}
            </div>

            {showScrollToBottom && (
              <button
                type="button"
                onClick={handleScrollToBottom}
                title={t("回到最底")}
                className="btn-custom-hover absolute right-5 bottom-4 z-10 rounded-full text-[11px] font-semibold flex items-center gap-1.5 shadow-lg transition hover:opacity-95"
                style={{
                  padding: '7px 12px',
                  background: 'var(--menu-bg, var(--card))',
                  border: '1px solid var(--menu-border, var(--border))',
                  boxShadow: 'var(--menu-shadow, 0 6px 16px rgba(0,0,0,0.25))',
                  color: 'var(--fg)',
                }}
              >
                <ArrowDown size={13} style={{ color: 'var(--status-info)' }} />
                <span>{t("回到最底")}</span>
              </button>
            )}
          </div>

          {/* Input Bar with Live Command Suggestions */}
          <div
            className="relative px-4 py-3 shrink-0"
            style={{ borderTop: '1px solid var(--border)', background: 'var(--bg-deep)' }}
          >
            {showSuggestions && !activeRunId && filteredSuggestions.length > 0 && (
              <div
                className="absolute left-4 right-4 bottom-full mb-1.5 rounded-xl border p-1.5 z-20 max-h-64 overflow-y-auto"
                style={{
                  backgroundColor: 'var(--menu-bg, var(--bg-deep))',
                  borderColor: 'var(--menu-border, var(--border))',
                  boxShadow: 'var(--menu-shadow, 0 12px 28px rgba(0,0,0,0.25))',
                }}
              >
                <div className="px-2 py-1 flex items-center justify-between text-[10px] font-bold tracking-wider" style={{ color: 'var(--muted)' }}>
                  <span>{t("可用指令與用法提示（點擊帶入）")}</span>
                  <span className="font-normal opacity-80">{t("↑↓ 選擇 · Enter / Tab 帶入")}</span>
                </div>
                {filteredSuggestions.map((s, idx) => {
                  const isActive = idx === highlightedIndex;
                  return (
                    <button
                      key={`${s.cmd}-${s.display || ''}`}
                      ref={(el) => { suggestionRefs.current[idx] = el; }}
                      type="button"
                      onMouseEnter={() => setHighlightedIndex(idx)}
                      onMouseDown={(e) => {
                        e.preventDefault();
                        applySuggestion(s);
                      }}
                      className="dropdown-menu-item w-full text-left rounded-lg flex items-center justify-between gap-3 text-xs transition"
                      style={{
                        backgroundColor: isActive ? 'var(--surface-hover)' : undefined,
                      }}
                    >
                      <span className="font-mono font-semibold shrink-0" style={{ color: 'var(--status-info)' }}>
                        {s.display || s.cmd.trim()}
                      </span>
                      <span className="text-[11px] truncate" style={{ color: isActive ? 'var(--fg)' : 'var(--muted)' }}>
                        {t(s.descKey)}
                      </span>
                    </button>
                  );
                })}
              </div>
            )}

            <div className="flex items-center gap-2">
              {(() => {
                const inputDisabled = !canRun || !!activeRunId;
                const inputPlaceholder = activeRunId
                  ? t("指令執行中，請稍候或點擊右側「停止」...")
                  : !canRun
                    ? t("請先安裝對應的 PHP 與 Composer 環境...")
                    : t("輸入指令，例如 install / show / audit / require laravel/sanctum");

                return (
                  <div
                    className={`flex-1 flex items-center gap-2 px-3 py-1.5 rounded-lg transition ${inputDisabled ? 'opacity-60 cursor-not-allowed' : ''}`}
                    style={{
                      background: inputDisabled ? 'var(--surface)' : 'var(--input-bg)',
                      border: '1px solid var(--input-border)',
                    }}
                  >
                    <span
                      className="font-mono text-xs font-bold shrink-0"
                      style={{ color: inputDisabled ? 'var(--muted)' : 'var(--status-info)' }}
                    >
                      composer &gt;
                    </span>
                    <input
                      ref={inputRef}
                      type="text"
                      disabled={inputDisabled}
                      value={commandInput}
                      onFocus={() => setShowSuggestions(true)}
                      onBlur={() => setTimeout(() => { setShowSuggestions(false); setHighlightedIndex(-1); }, 150)}
                      onChange={(e) => {
                        setCommandInput(e.target.value);
                        setShowSuggestions(true);
                        setHighlightedIndex(-1);
                      }}
                      onKeyDown={handleInputKeyDown}
                      placeholder={inputPlaceholder}
                      className={`flex-1 bg-transparent outline-none font-mono text-xs ${inputDisabled ? 'cursor-not-allowed' : ''}`}
                      style={{ color: inputDisabled ? 'var(--muted)' : 'var(--fg)' }}
                    />
                  </div>
                );
              })()}

              {activeRunId ? (
                <button
                  type="button"
                  onClick={handleStop}
                  className="px-3.5 py-2 rounded-lg text-xs font-bold flex items-center gap-1.5 transition shrink-0"
                  style={{ background: 'var(--status-error)', color: '#fff' }}
                >
                  <Square size={12} />
                  <span>{t("停止")}</span>
                </button>
              ) : (
                <button
                  type="button"
                  disabled={!canRun || !commandInput.trim()}
                  onClick={() => executeCommand(commandInput)}
                  className="px-3.5 py-2 rounded-lg text-xs font-bold flex items-center gap-1.5 transition shrink-0 disabled:opacity-40"
                  style={{ background: 'var(--status-info)', color: '#fff' }}
                >
                  <Play size={12} />
                  <span>{t("執行")}</span>
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

import React, { useEffect, useRef, useState } from 'react';
import { X, RefreshCw, Terminal as TermIcon, ShieldAlert } from 'lucide-react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import {
  StartTerminalSession,
  SendTerminalInput,
  ResizeTerminal,
  StopTerminalSession
} from '../../wailsjs/go/main/App';
import { t, useLanguage } from '../i18n';

import '@xterm/xterm/css/xterm.css';

interface ProjectTerminalProps {
  projectName: string;
  isOpen: boolean;
  onClose: () => void;
}

export default function ProjectTerminal({ projectName, isOpen, onClose }: ProjectTerminalProps) {
  useLanguage(); // 訂閱語系變更
  const terminalRef = useRef<HTMLDivElement>(null);
  const termInstance = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const sessionIDRef = useRef<string | null>(null);
  const containerResizeObserver = useRef<ResizeObserver | null>(null);
  
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [isReady, setIsReady] = useState(false);

  // 初始化並啟動終端
  useEffect(() => {
    if (!isOpen || !terminalRef.current) return;

    setErrorMsg(null);
    setIsReady(false);

    let unsubOutput: (() => void) | null = null;
    let unsubExit: (() => void) | null = null;

    // 1. 初始化 xterm.js
    const term = new Terminal({
      cursorBlink: true,
      fontFamily: '"Fira Code", Consolas, Menlo, Monaco, "Courier New", monospace',
      fontSize: 12,
      lineHeight: 1.2,
      theme: {
        background: '#08080a',
        foreground: '#d4d4d8',
        cursor: '#3b82f6',
        cursorAccent: '#08080a',
        selectionBackground: '#1e3a8a',
        black: '#000000',
        red: '#ef4444',
        green: '#22c55e',
        yellow: '#eab308',
        blue: '#3b82f6',
        magenta: '#a855f7',
        cyan: '#06b6d4',
        white: '#d4d4d8',
        brightBlack: '#71717a',
        brightRed: '#f87171',
        brightGreen: '#4ade80',
        brightYellow: '#facc15',
        brightBlue: '#60a5fa',
        brightMagenta: '#c084fc',
        brightCyan: '#22d3ee',
        brightWhite: '#ffffff'
      }
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    
    termInstance.current = term;
    fitAddonRef.current = fitAddon;

    // 2. 打開 xterm.js
    term.open(terminalRef.current);
    
    // 渲染提示資訊
    term.writeln('\x1b[38;5;39m🚀 [WinCMP] ' + t("正在啟動專案終端會話...") + '\x1b[0m');

    // 3. 延遲執行 fit 避免 DOM 尺寸未就緒導致計算為 0
    setTimeout(async () => {
      try {
        fitAddon.fit();
        const cols = term.cols || 80;
        const rows = term.rows || 24;

        // 4. 呼叫後端啟動 PTY 進程
        const sID = await StartTerminalSession(projectName, cols, rows);
        sessionIDRef.current = sID;
        setIsReady(true);

        // 5. 綁定終端輸入事件
        term.onData((data) => {
          SendTerminalInput(sID, data).catch((err) => {
            console.error("發送終端輸入失敗:", err);
          });
        });

        // 6. 監聽後端輸出事件
        unsubOutput = EventsOn('terminal_output', (res: { sessionId: string; data: string }) => {
          if (res.sessionId === sID) {
            term.write(res.data);
          }
        });

        // 7. 監聽後端進程關閉事件
        unsubExit = EventsOn('terminal_exit', (res: { sessionId: string }) => {
          if (res.sessionId === sID) {
            term.writeln('\r\n\x1b[38;5;203m🚫 [WinCMP] ' + t("終端會話已中斷或關閉") + '\x1b[0m\r\n');
          }
        });

        // 8. 監聽 DOM 容器大小變化，隨時通知後端 PTY Resize
        containerResizeObserver.current = new ResizeObserver(() => {
          if (!fitAddonRef.current || !termInstance.current || !sessionIDRef.current) return;
          try {
            fitAddonRef.current.fit();
            const currentCols = termInstance.current.cols;
            const currentRows = termInstance.current.rows;
            ResizeTerminal(sessionIDRef.current, currentCols, currentRows).catch((e) => {
              console.error("調整終端視窗尺寸失敗:", e);
            });
          } catch (e) {
            // 忽略因 DOM 不在文件流中導致的 fit 錯誤
          }
        });
        if (terminalRef.current) {
          containerResizeObserver.current.observe(terminalRef.current);
        }

      } catch (err: any) {
        console.error("無法建立終端會話:", err);
        setErrorMsg(err.toString() || "無法建立 PTY 進程");
        term.writeln(`\r\n\x1b[31m❌ ${t("啟動失敗")}: ${err}\x1b[0m\r\n`);
      }
    }, 150);

    // 清理資源
    return () => {
      // 關閉監聽器
      if (unsubOutput) unsubOutput();
      if (unsubExit) unsubExit();

      // 銷毀 Observer
      if (containerResizeObserver.current) {
        containerResizeObserver.current.disconnect();
      }

      // 呼叫後端終止會話
      if (sessionIDRef.current) {
        StopTerminalSession(sessionIDRef.current);
      }

      // 銷毀 xterm
      if (termInstance.current) {
        termInstance.current.dispose();
      }
    };
  }, [isOpen, projectName]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 overflow-hidden select-none">
      {/* 半透明背景 */}
      <div 
        className="absolute inset-0 bg-black/45 backdrop-blur-[1px] transition-opacity duration-300"
        onClick={onClose}
      />

      <div className="absolute inset-y-0 right-0 pl-10 max-w-full flex">
        {/* Drawer 容器 */}
        <div className="w-screen max-w-xl bg-darkCard border-l border-darkBorder shadow-2xl flex flex-col h-full overflow-hidden animate-slide-in">
          
          {/* Header */}
          <div className="px-6 py-5 border-b border-darkBorder flex justify-between items-center bg-[#0d0d10] shrink-0">
            <div className="flex items-center gap-2">
              <TermIcon size={14} className="text-blue-500" />
              <div>
                <h3 className="text-sm font-bold uppercase tracking-wider text-gray-200">
                  {t("專案開發終端 (Terminal)")}
                </h3>
                <p className="text-[10px] text-gray-500 font-mono mt-0.5">{projectName}</p>
              </div>
            </div>
            <button onClick={onClose} className="text-gray-400 hover:text-white transition">
              <X size={16} />
            </button>
          </div>

          {/* Terminal Area */}
          <div className="flex-1 bg-[#08080a] p-4 relative overflow-hidden flex flex-col justify-end">
            <div 
              ref={terminalRef} 
              id="terminal-container"
              className="w-full h-full text-left"
            />

            {errorMsg && (
              <div className="absolute inset-0 flex flex-col items-center justify-center bg-[#08080a]/90 text-red-400 px-8 py-4 gap-3 text-center">
                <ShieldAlert size={36} className="text-red-500" />
                <div className="text-xs font-semibold">{t("無法啟動終端")}</div>
                <div className="text-[11px] font-mono text-gray-500 bg-black/40 px-3 py-2 rounded-lg border border-darkBorder max-w-full truncate">
                  {errorMsg}
                </div>
              </div>
            )}
          </div>

          {/* Footer */}
          <div className="px-6 py-3 border-t border-darkBorder flex justify-between items-center bg-[#0d0d10] shrink-0 text-[10px] text-gray-500">
            <div>
              {t("💡 支援完整互動指令、Ctrl+C 中斷與 TAB 自動補齊。")}
            </div>
            <button
              onClick={() => {
                // 重新載入 Session
                onClose();
                setTimeout(() => {
                  // 這邊可以透過外部狀態控制觸發重新開啟
                  onClose();
                }, 100);
              }}
              className="px-2.5 py-1 hover:bg-darkBorder border border-darkBorder rounded text-gray-400 hover:text-white transition flex items-center gap-1 font-semibold"
            >
              <RefreshCw size={10} /> {t("重啟")}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

import React, { useEffect, useRef, useState } from 'react';
import { X, RefreshCw, Terminal as TermIcon, ShieldAlert } from 'lucide-react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import { StartTerminalSession, SendTerminalInput, ResizeTerminal, StopTerminalSession } from '../../wailsjs/go/main/App';
import { t, useLanguage } from '../i18n';
import '@xterm/xterm/css/xterm.css';

interface ProjectTerminalProps { projectName: string; isOpen: boolean; onClose: () => void; }

export default function ProjectTerminal({ projectName, isOpen, onClose }: ProjectTerminalProps) {
  useLanguage();
  const terminalRef = useRef<HTMLDivElement>(null);
  const termInstance = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const sessionIDRef = useRef<string | null>(null);
  const containerResizeObserver = useRef<ResizeObserver | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [isReady, setIsReady] = useState(false);

  useEffect(() => {
    if (!isOpen || !terminalRef.current) return;
    setErrorMsg(null); setIsReady(false);
    let unsubOutput: (() => void) | null = null;
    let unsubExit: (() => void) | null = null;

    const term = new Terminal({
      cursorBlink: true,
      fontFamily: '"Fira Code", Consolas, Menlo, Monaco, "Courier New", monospace',
      fontSize: 12, lineHeight: 1.2,
      theme: {
        background: '#08080a', foreground: '#d4d4d8', cursor: '#3b82f6',
        cursorAccent: '#08080a', selectionBackground: '#1e3a8a',
        black: '#000000', red: '#ef4444', green: '#22c55e', yellow: '#eab308',
        blue: '#3b82f6', magenta: '#a855f7', cyan: '#06b6d4', white: '#d4d4d8',
        brightBlack: '#71717a', brightRed: '#f87171', brightGreen: '#4ade80',
        brightYellow: '#facc15', brightBlue: '#60a5fa', brightMagenta: '#c084fc',
        brightCyan: '#22d3ee', brightWhite: '#ffffff'
      }
    });

    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    termInstance.current = term;
    fitAddonRef.current = fitAddon;
    term.open(terminalRef.current);
    term.writeln('\x1b[38;5;39m🚀 [WinCMP] ' + t("正在啟動專案終端會話...") + '\x1b[0m');

    setTimeout(async () => {
      try {
        fitAddon.fit();
        const cols = term.cols || 80; const rows = term.rows || 24;
        const sID = await StartTerminalSession(projectName, cols, rows);
        sessionIDRef.current = sID; setIsReady(true);
        term.onData((data) => { SendTerminalInput(sID, data).catch((err) => { console.error("發送終端輸入失敗:", err); }); });
        unsubOutput = EventsOn('terminal_output', (res: { sessionId: string; data: string }) => { if (res.sessionId === sID) term.write(res.data); });
        unsubExit = EventsOn('terminal_exit', (res: { sessionId: string }) => { if (res.sessionId === sID) term.writeln('\r\n\x1b[38;5;203m🚫 [WinCMP] ' + t("終端會話已中斷或關閉") + '\x1b[0m\r\n'); });
        containerResizeObserver.current = new ResizeObserver(() => {
          if (!fitAddonRef.current || !termInstance.current || !sessionIDRef.current) return;
          try {
            fitAddonRef.current.fit();
            ResizeTerminal(sessionIDRef.current, termInstance.current.cols, termInstance.current.rows).catch((e) => { console.error("調整終端視窗尺寸失敗:", e); });
          } catch (e) {}
        });
        if (terminalRef.current) containerResizeObserver.current.observe(terminalRef.current);
      } catch (err: any) {
        console.error("無法建立終端會話:", err);
        setErrorMsg(err.toString() || "無法建立 PTY 進程");
        term.writeln(`\r\n\x1b[31m❌ ${t("啟動失敗")}: ${err}\x1b[0m\r\n`);
      }
    }, 150);

    return () => {
      if (unsubOutput) unsubOutput();
      if (unsubExit) unsubExit();
      if (containerResizeObserver.current) containerResizeObserver.current.disconnect();
      if (sessionIDRef.current) StopTerminalSession(sessionIDRef.current);
      if (termInstance.current) termInstance.current.dispose();
    };
  }, [isOpen, projectName]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 overflow-hidden select-none">
      <div className="absolute inset-0 transition-opacity duration-300" style={{ background: 'var(--overlay-bg)', backdropFilter: 'blur(1px)' }} onClick={onClose} />
      <div className="absolute inset-y-0 right-0 pl-10 max-w-full flex">
        <div className="w-screen max-w-xl flex flex-col h-full overflow-hidden animate-slide-in" style={{ background: 'var(--card)', borderLeft: '1px solid var(--border)', boxShadow: 'var(--shadow-lg)' }}>
          {/* Header */}
          <div className="px-6 py-5 flex justify-between items-center shrink-0" style={{ borderBottom: '1px solid var(--border)', background: 'var(--bg-deep)' }}>
            <div className="flex items-center gap-2">
              <TermIcon size={14} style={{ color: 'var(--status-info)' }} />
              <div>
                <h3 className="text-sm font-bold uppercase tracking-wider" style={{ color: 'var(--fg)', fontFamily: 'var(--font-display)' }}>{t("專案開發終端 (Terminal)")}</h3>
                <p className="text-[10px] mt-0.5" style={{ color: 'var(--meta)', fontFamily: 'var(--font-mono)' }}>{projectName}</p>
              </div>
            </div>
            <button onClick={onClose} className="transition" style={{ color: 'var(--muted)' }}><X size={16} /></button>
          </div>

          {/* Terminal Area */}
          <div className="flex-1 p-4 relative overflow-hidden flex flex-col justify-end" style={{ background: '#08080a' }}>
            <div ref={terminalRef} id="terminal-container" className="w-full h-full text-left" />
            {errorMsg && (
              <div className="absolute inset-0 flex flex-col items-center justify-center px-8 py-4 gap-3 text-center" style={{ background: 'rgba(8,8,10,0.9)' }}>
                <ShieldAlert size={36} style={{ color: 'var(--status-error)' }} />
                <div className="text-xs font-semibold" style={{ color: 'var(--status-error)' }}>{t("無法啟動終端")}</div>
                <div className="text-[11px] px-3 py-2 rounded-lg max-w-full truncate" style={{ color: 'var(--meta)', background: 'rgba(0,0,0,0.4)', border: '1px solid var(--border)', fontFamily: 'var(--font-mono)' }}>
                  {errorMsg}
                </div>
              </div>
            )}
          </div>

          {/* Footer */}
          <div className="px-6 py-3 flex justify-between items-center shrink-0 text-[10px]" style={{ borderTop: '1px solid var(--border)', background: 'var(--bg-deep)', color: 'var(--meta)' }}>
            <div>{t("💡 支援完整互動指令、Ctrl+C 中斷與 TAB 自動補齊。")}</div>
            <button onClick={() => { onClose(); setTimeout(() => onClose(), 100); }} className="px-2.5 py-1 border rounded transition flex items-center gap-1 font-semibold" style={{ borderColor: 'var(--border)', color: 'var(--fg-2)' }}>
              <RefreshCw size={10} /> {t("重啟")}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

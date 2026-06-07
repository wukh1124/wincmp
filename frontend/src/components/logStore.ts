import { EventsOn } from '../../wailsjs/runtime/runtime';
import { GetConfig, GetCategoryLogs } from '../../wailsjs/go/main/App';

export interface LogLine {
  text: string;
  time: string;
}

export interface LogData {
  system: LogLine[];
  caddy: LogLine[];
  mariadb: LogLine[];
  mailpit: LogLine[];
  php: LogLine[];
  runtime: Record<string, LogLine[]>;
}

type LogListener = (logs: LogData) => void;

class LogStore {
  private logs: LogData = {
    system: [],
    caddy: [],
    mariadb: [],
    mailpit: [],
    php: [],
    runtime: {
      System: []
    }
  };

  private listeners: Set<LogListener> = new Set();
  private initialized = false;
  private maxLogLines = 500; // 預設保留 500 行，後續會從設定檔動態載入

  constructor() {
    this.init();
  }

  private async loadMaxLogLines() {
    try {
      const cfg = await GetConfig();
      if (cfg && cfg.global && cfg.global.max_log_lines > 0) {
        this.maxLogLines = cfg.global.max_log_lines;
        this.applyMaxLogLines();
      }
    } catch (err) {
      console.error('[logStore] 載入 max_log_lines 失敗:', err);
    }
  }

  private async loadHistoryLogs() {
    try {
      const categories = ['system', 'caddy', 'mariadb', 'mailpit', 'php'];
      
      // 1. 載入系統分類歷史日誌
      for (const cat of categories) {
        const entries = await GetCategoryLogs(cat, "");
        if (entries && entries.length > 0) {
          const existing = this.logs[cat as keyof Omit<LogData, 'runtime'>] || [];
          const combined = [...entries.map((e: any) => ({ text: e.text || '', time: e.time || '' }))];
          for (const item of existing) {
            const isDup = combined.some(c => c.time === item.time && c.text === item.text);
            if (!isDup) {
              combined.push(item);
            }
          }
          this.logs[cat as keyof Omit<LogData, 'runtime'>] = combined;
        }
      }

      // 2. 載入 runtime 分類歷史日誌 (System 和各專案)
      const systemRuntimeEntries = await GetCategoryLogs("runtime", "System");
      if (systemRuntimeEntries && systemRuntimeEntries.length > 0) {
        const existing = this.logs.runtime["System"] || [];
        const combined = [...systemRuntimeEntries.map((e: any) => ({ text: e.text || '', time: e.time || '' }))];
        for (const item of existing) {
          const isDup = combined.some(c => c.time === item.time && c.text === item.text);
          if (!isDup) {
            combined.push(item);
          }
        }
        this.logs.runtime["System"] = combined;
      }

      const cfg = await GetConfig();
      if (cfg && cfg.projects) {
        for (const proj of cfg.projects) {
          if (proj.name) {
            const entries = await GetCategoryLogs("runtime", proj.name);
            if (entries && entries.length > 0) {
              const existing = this.logs.runtime[proj.name] || [];
              const combined = [...entries.map((e: any) => ({ text: e.text || '', time: e.time || '' }))];
              for (const item of existing) {
                const isDup = combined.some(c => c.time === item.time && c.text === item.text);
                if (!isDup) {
                  combined.push(item);
                }
              }
              this.logs.runtime[proj.name] = combined;
            }
          }
        }
      }

      this.applyMaxLogLines();
      this.notify();
    } catch (err) {
      console.error('[logStore] 載入歷史日誌失敗:', err);
    }
  }

  public setMaxLogLines(lines: number) {
    if (lines > 0 && lines !== this.maxLogLines) {
      this.maxLogLines = lines;
      this.applyMaxLogLines();
      this.notify();
    }
  }

  private applyMaxLogLines() {
    const max = this.maxLogLines;
    
    // 裁切系統分類日誌
    const categories: Array<keyof Omit<LogData, 'runtime'>> = ['system', 'caddy', 'mariadb', 'mailpit', 'php'];
    for (const cat of categories) {
      if (this.logs[cat].length > max) {
        this.logs[cat] = this.logs[cat].slice(-max);
      }
    }

    // 裁切專案 Runtime 日誌
    for (const proj in this.logs.runtime) {
      if (this.logs.runtime[proj].length > max) {
        this.logs.runtime[proj] = this.logs.runtime[proj].slice(-max);
      }
    }
  }

  public init() {
    if (this.initialized) return;

    // 檢查 Wails runtime 是否已經就緒，若尚未就緒則延遲 100ms 再次檢查
    if (!(window as any).runtime) {
      setTimeout(() => this.init(), 100);
      return;
    }

    this.initialized = true;
    console.log('[logStore] window.runtime detected. Binding log listener...');
    
    // 異步載入最大保留行數設定
    this.loadMaxLogLines();

    // 異步載入歷史日誌並去重合併
    this.loadHistoryLogs();

    EventsOn('log', (data: any) => {
      console.log('[logStore] Received log payload:', data);
      if (!data || !data.category) return;
      const category = data.category === 'node' ? 'runtime' : data.category;
      const isValidCategory = ['system', 'caddy', 'mariadb', 'mailpit', 'php', 'runtime'].includes(category);
      
      if (isValidCategory) {
        const time = data.time || new Date().toLocaleTimeString();
        const text = data.message || '';
        
        if (category === 'runtime') {
          const proj = data.projectName || 'System';
          if (!this.logs.runtime[proj]) {
            this.logs.runtime[proj] = [];
          }
          this.logs.runtime[proj].push({ text, time });
          if (this.logs.runtime[proj].length > this.maxLogLines) {
            this.logs.runtime[proj].shift();
          }
        } else {
          const cat = category as keyof Omit<LogData, 'runtime'>;
          this.logs[cat].push({ text, time });
          if (this.logs[cat].length > this.maxLogLines) {
            this.logs[cat].shift();
          }
        }
        
        this.notify();
      }
    });
  }

  public subscribe(listener: LogListener) {
    this.init();
    this.listeners.add(listener);
    // 立即傳遞當前歷史日誌
    listener(this.logs);
    return () => {
      this.listeners.delete(listener);
    };
  }

  public getLogs(): LogData {
    return this.logs;
  }

  public clearLogs(category: string, subCategory?: string) {
    if (category === 'runtime') {
      const proj = subCategory || 'System';
      this.logs.runtime[proj] = [];
    } else if (category in this.logs) {
      this.logs[category as keyof Omit<LogData, 'runtime'>] = [];
    }
    this.notify();
  }

  private notify() {
    // 徹底複製所有分類日誌陣列，確保 React 100% 觸發更新
    const currentLogs: LogData = {
      system: [...this.logs.system],
      caddy: [...this.logs.caddy],
      mariadb: [...this.logs.mariadb],
      mailpit: [...this.logs.mailpit],
      php: [...this.logs.php],
      runtime: {}
    };
    for (const key in this.logs.runtime) {
      currentLogs.runtime[key] = [...this.logs.runtime[key]];
    }
    this.listeners.forEach(l => l(currentLogs));
  }
}

// 綁定到 window 全域對象，抵禦 Vite HMR 熱更新導致單例失效
if (!(window as any)._logStore) {
  (window as any)._logStore = new LogStore();
}

export const logStore = (window as any)._logStore as LogStore;

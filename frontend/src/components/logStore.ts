import { EventsOn } from '../../wailsjs/runtime/runtime';

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

  constructor() {
    this.init();
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
          if (this.logs.runtime[proj].length > 1000) {
            this.logs.runtime[proj].shift();
          }
        } else {
          const cat = category as keyof Omit<LogData, 'runtime'>;
          this.logs[cat].push({ text, time });
          if (this.logs[cat].length > 1000) {
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

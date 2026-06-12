import React, { useState, useEffect } from 'react';
import { Database, RefreshCw, ExternalLink, AlertTriangle, Layers, Table } from 'lucide-react';
import { IsMariaDBRunning, QueryDatabases, QueryTables, OpenInHeidiSQL } from '../../wailsjs/go/main/App';
import { t, useLanguage } from '../i18n';

export default function DBExplorer() {
  useLanguage(); // 訂閱語系變更
  const [isRunning, setIsRunning] = useState(false);
  const [databases, setDatabases] = useState<string[]>([]);
  const [selectedSchema, setSelectedSchema] = useState<string | null>(null);
  const [tables, setTables] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isTablesLoading, setIsTablesLoading] = useState(false);

  useEffect(() => {
    checkDBStatus();
  }, []);

  const checkDBStatus = async () => {
    setIsLoading(true);
    try {
      const running = await IsMariaDBRunning();
      setIsRunning(running);
      if (running) {
        const dbs = await QueryDatabases();
        setDatabases(dbs || []);
      }
    } catch (err) {
      console.error("檢查資料庫失敗:", err);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSelectSchema = async (schema: string) => {
    setSelectedSchema(schema);
    setIsTablesLoading(true);
    try {
      const tbs = await QueryTables(schema);
      setTables(tbs || []);
    } catch (err) {
      console.error("載入資料表失敗:", err);
      setTables([`${t("載入失敗")}: ${err}`]);
    } finally {
      setIsTablesLoading(false);
    }
  };

  const handleOpenHeidiSQL = async () => {
    try {
      await OpenInHeidiSQL();
    } catch (err) {
      (window as any).customAlert(`${t("開啟 HeidiSQL 失敗")}: ${err}`);
    }
  };

  return (
    <div className="p-6 h-full flex flex-col space-y-6">
      {/* 標頭 */}
      <div className="flex justify-between items-center select-none">
        <div>
          <h1 className="text-xl font-bold tracking-tight" style={{ color: 'var(--fg)' }}>{t("資料庫瀏覽器")}</h1>
          <p className="text-xs mt-1" style={{ color: 'var(--muted)' }}>{t("內建極簡 Schema / 資料表結構速覽，或一鍵透過外部工具管理")}</p>
        </div>
        <div className="flex gap-2.5">
          <button
            onClick={checkDBStatus}
            disabled={isLoading}
            className="px-3.5 py-2 rounded-lg text-xs font-semibold border flex items-center gap-1.5 transition duration-200"
            style={{ borderColor: 'var(--border)', backgroundColor: 'var(--card)', color: 'var(--fg-2)' }}
          >
            <RefreshCw size={13} className={isLoading ? 'animate-spin' : ''} />
            {t("重新整理")}
          </button>
          <button
            onClick={handleOpenHeidiSQL}
            disabled={!isRunning}
            className="px-3.5 py-2 disabled:opacity-50 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition duration-200"
            style={{ backgroundColor: 'var(--accent)', color: 'var(--accent-on)' }}
          >
            <ExternalLink size={13} /> {t("Open in HeidiSQL")}
          </button>
        </div>
      </div>

      {/* 資料庫運行狀態判斷 */}
      {!isRunning ? (
        <div className="flex-1 border rounded-xl p-8 flex flex-col items-center justify-center text-center space-y-4 select-none" style={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)' }}>
          <div className="p-4 rounded-full" style={{ backgroundColor: 'var(--status-warn-bg)', color: 'var(--status-warn)' }}>
            <AlertTriangle size={36} />
          </div>
          <div>
            <h3 className="text-sm font-bold" style={{ color: 'var(--fg)' }}>{t("MariaDB 尚未啟動")}</h3>
            <p className="text-xs mt-1.5 max-w-sm leading-relaxed" style={{ color: 'var(--muted)' }}>
              {t("請先前往 **Dashboard** 頁面啟動 MariaDB 資料庫服務，再使用 Database Explorer 進行瀏覽。")}
            </p>
          </div>
        </div>
      ) : (
        <div className="flex-1 border rounded-xl flex overflow-hidden min-h-[400px]" style={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)' }}>
          {/* 左側：Databases 列表 */}
          <div className="w-1/3 border-r flex flex-col select-none" style={{ borderColor: 'var(--border)', backgroundColor: 'var(--bg-deep)' }}>
            <div className="px-5 py-3.5 border-b font-bold text-[10px] tracking-wider uppercase" style={{ borderColor: 'var(--border)', backgroundColor: 'var(--bg-deep)', color: 'var(--meta)' }}>
              📁 Databases ({databases.length})
            </div>
            <div className="flex-1 overflow-y-auto p-2.5 space-y-0.5">
              {databases.map(db => (
                <button
                  key={db}
                  onClick={() => handleSelectSchema(db)}
                  className={`w-full text-left px-3.5 py-2 rounded-lg text-xs font-semibold flex items-center gap-2.5 transition duration-150 ${selectedSchema === db
                    ? 'border'
                    : ''
                    }`}
                  style={
                    selectedSchema === db
                      ? { backgroundColor: 'var(--accent-muted)', color: 'var(--accent)', borderColor: 'var(--border-soft)' }
                      : { color: 'var(--muted)' }
                  }
                >
                  <Database size={13} style={{ color: selectedSchema === db ? 'var(--accent)' : 'var(--meta)' }} />
                  <span className="truncate">{db}</span>
                </button>
              ))}
            </div>
          </div>

          {/* 右側：Tables 列表 */}
          <div className="w-2/3 flex flex-col" style={{ backgroundColor: 'var(--surface)' }}>
            <div className="px-5 py-3.5 border-b font-bold text-[10px] tracking-wider uppercase select-none" style={{ borderColor: 'var(--border)', backgroundColor: 'var(--bg-deep)', color: 'var(--meta)' }}>
              Tables {selectedSchema ? `(in ${selectedSchema})` : ''}
            </div>

            <div className="flex-1 overflow-y-auto p-5 font-mono text-xs">
              {selectedSchema ? (
                isTablesLoading ? (
                  <div className="h-full flex items-center justify-center" style={{ color: 'var(--muted)' }}>
                    <RefreshCw size={20} className="animate-spin" style={{ color: 'var(--accent)' }} />
                  </div>
                ) : tables.length > 0 ? (
                  <div className="space-y-3">
                    <div className="font-bold mb-3 select-none flex items-center gap-1.5 border-b pb-2 text-[11px]" style={{ color: 'var(--muted)', borderColor: 'var(--border)' }}>
                      <Table size={12} style={{ color: 'var(--accent)' }} /> {t("資料庫")} '{selectedSchema}' {t("的資料表：")}
                    </div>
                    <div className="max-h-[60vh] overflow-y-auto" style={{ borderTopColor: 'var(--border)', borderTopWidth: 1, borderTopStyle: 'solid' }}>
                      {tables.map((tb, idx) => {
                        const name = tb.split('  ')[0];
                        const rowInfo = tb.split('  ')[1] || '';

                        return (
                          <div key={idx} className="py-2.5 px-2 rounded transition flex items-center justify-between" style={{ color: 'var(--fg-2)', borderBottomColor: 'var(--border-soft)', borderBottomWidth: 1, borderBottomStyle: 'solid' }}>
                            <div className="flex items-center gap-2">
                              <span className="select-none text-[10px]" style={{ color: 'var(--meta)' }}>{String(idx + 1).padStart(2, '0')}.</span>
                              <span className="text-xs font-semibold" style={{ color: 'var(--fg-2)' }}>{name}</span>
                            </div>
                            {rowInfo.includes('rows') && (
                              <span className="text-[10px] px-2 py-0.5 rounded-full font-sans font-bold" style={{ color: 'var(--accent)', backgroundColor: 'var(--accent-muted)', borderColor: 'var(--border-soft)', borderWidth: 1, borderStyle: 'solid' }}>
                                {rowInfo}
                              </span>
                            )}
                          </div>
                        );
                      })}
                    </div>
                  </div>
                ) : (
                  <div className="h-full flex items-center justify-center italic select-none text-xs" style={{ color: 'var(--meta)' }}>
                    {t("（此資料庫沒有資料表）")}
                  </div>
                )
              ) : (
                <div className="h-full flex items-center justify-center italic select-none text-xs" style={{ color: 'var(--meta)' }}>
                  {t("請選擇左側的資料庫以檢視資料表")}
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

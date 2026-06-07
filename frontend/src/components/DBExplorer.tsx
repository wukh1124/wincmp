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
          <h1 className="text-xl font-bold tracking-tight text-white">🗄️ {t("資料庫瀏覽器 (DB Explorer)")}</h1>
          <p className="text-xs text-gray-400 mt-1">{t("內建極簡 Schema / 資料表結構速覽，或一鍵透過外部工具管理")}</p>
        </div>
        <div className="flex gap-2.5">
          <button
            onClick={checkDBStatus}
            disabled={isLoading}
            className="px-3.5 py-2 rounded-lg text-xs font-semibold border border-darkBorder flex items-center gap-1.5 bg-darkCard hover:bg-opacity-80 transition duration-200 text-gray-200"
          >
            <RefreshCw size={13} className={isLoading ? 'animate-spin' : ''} />
            {t("重新整理")}
          </button>
          <button
            onClick={handleOpenHeidiSQL}
            disabled={!isRunning}
            className="px-3.5 py-2 bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 transition duration-200"
          >
            <ExternalLink size={13} /> {t("Open in HeidiSQL")}
          </button>
        </div>
      </div>

      {/* 資料庫運行狀態判斷 */}
      {!isRunning ? (
        <div className="flex-1 bg-darkCard border border-darkBorder rounded-xl p-8 flex flex-col items-center justify-center text-center space-y-4 select-none">
          <div className="p-4 bg-yellow-500 bg-opacity-10 text-yellow-400 rounded-full">
            <AlertTriangle size={36} />
          </div>
          <div>
            <h3 className="text-sm font-bold text-gray-100">{t("MariaDB 尚未啟動")}</h3>
            <p className="text-xs text-gray-400 mt-1.5 max-w-sm leading-relaxed">
              {t("請先前往 **Dashboard** 頁面啟動 MariaDB 資料庫服務，再使用 Database Explorer 進行瀏覽。")}
            </p>
          </div>
        </div>
      ) : (
        <div className="flex-1 bg-darkCard border border-darkBorder rounded-xl flex overflow-hidden min-h-[400px]">
          {/* 左側：Databases 列表 */}
          <div className="w-1/3 border-r border-darkBorder flex flex-col bg-[#0b0b0e]/30 select-none">
            <div className="px-5 py-3.5 border-b border-darkBorder bg-[#0f0f12] font-bold text-[10px] tracking-wider uppercase text-gray-500">
              📁 Databases ({databases.length})
            </div>
            <div className="flex-1 overflow-y-auto p-2.5 space-y-0.5">
              {databases.map(db => (
                <button
                  key={db}
                  onClick={() => handleSelectSchema(db)}
                  className={`w-full text-left px-3.5 py-2 rounded-lg text-xs font-semibold flex items-center gap-2.5 transition duration-150 ${
                    selectedSchema === db
                      ? 'bg-emerald-600/10 text-emerald-400 border border-emerald-500/10'
                      : 'text-gray-400 hover:bg-white/5 hover:text-white'
                  }`}
                >
                  <Database size={13} className={selectedSchema === db ? 'text-emerald-400' : 'text-gray-500'} />
                  <span className="truncate">{db}</span>
                </button>
              ))}
            </div>
          </div>

          {/* 右側：Tables 列表 */}
          <div className="w-2/3 flex flex-col bg-darkCard bg-opacity-40">
            <div className="px-5 py-3.5 border-b border-darkBorder bg-[#0f0f12] font-bold text-[10px] tracking-wider uppercase text-gray-500 select-none">
              📊 Tables {selectedSchema ? `(in ${selectedSchema})` : ''}
            </div>

            <div className="flex-1 overflow-y-auto p-5 font-mono text-xs">
              {selectedSchema ? (
                isTablesLoading ? (
                  <div className="h-full flex items-center justify-center text-gray-400">
                    <RefreshCw size={20} className="animate-spin text-emerald-400" />
                  </div>
                ) : tables.length > 0 ? (
                  <div className="space-y-3">
                    <div className="text-gray-400 font-bold mb-3 select-none flex items-center gap-1.5 border-b border-darkBorder pb-2 text-[11px]">
                      <Table size={12} className="text-emerald-400" /> {t("資料庫")} '{selectedSchema}' {t("的資料表：")}
                    </div>
                    <div className="divide-y divide-darkBorder/40 max-h-[60vh] overflow-y-auto">
                      {tables.map((tb, idx) => {
                        const name = tb.split('  ')[0];
                        const rowInfo = tb.split('  ')[1] || '';
                        
                        return (
                          <div key={idx} className="py-2.5 px-2 hover:bg-white/[0.015] rounded transition flex items-center justify-between text-gray-300">
                            <div className="flex items-center gap-2">
                              <span className="text-gray-600 select-none text-[10px]">{String(idx + 1).padStart(2, '0')}.</span>
                              <span className="text-gray-200 text-xs font-semibold">{name}</span>
                            </div>
                            {rowInfo.includes('rows') && (
                              <span className="text-[10px] text-emerald-400 bg-emerald-500/10 border border-emerald-500/10 px-2 py-0.5 rounded-full font-sans font-bold">
                                {rowInfo}
                              </span>
                            )}
                          </div>
                        );
                      })}
                    </div>
                  </div>
                ) : (
                  <div className="h-full flex items-center justify-center text-gray-500 italic select-none text-xs">
                    {t("（此資料庫沒有資料表）")}
                  </div>
                )
              ) : (
                <div className="h-full flex items-center justify-center text-gray-500 italic select-none text-xs">
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

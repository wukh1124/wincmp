import React, { useState, useEffect } from 'react';
import { Database, RefreshCw, ExternalLink, AlertTriangle, Layers, Table, Zap, Search, Trash2, Key, Clock, FileText } from 'lucide-react';
import { IsMariaDBRunning, QueryDatabases, QueryTables, OpenInHeidiSQL } from '../../wailsjs/go/main/App';
import { t, useLanguage } from '../i18n';

interface RedisKeyItem {
  key: string;
  type: string;
  ttl: number;
}

interface RedisKeyDetail {
  key: string;
  type: string;
  ttl: number;
  value: string;
  size: number;
}

interface RedisDBInfo {
  db: number;
  keys: number;
}

export default function DBExplorer() {
  useLanguage(); // 訂閱語系變更

  const [activeTab, setActiveTab] = useState<'mariadb' | 'redis'>('mariadb');

  // MariaDB 狀態
  const [isMariaDBRunning, setIsMariaDBRunning] = useState(false);
  const [databases, setDatabases] = useState<string[]>([]);
  const [selectedSchema, setSelectedSchema] = useState<string | null>(null);
  const [tables, setTables] = useState<string[]>([]);
  const [isLoadingMariaDB, setIsLoadingMariaDB] = useState(false);
  const [isTablesLoading, setIsTablesLoading] = useState(false);

  // Redis 狀態
  const [isRedisRunning, setIsRedisRunning] = useState(false);
  const [redisDBList, setRedisDBList] = useState<RedisDBInfo[]>([]);
  const [selectedDB, setSelectedDB] = useState<number>(0);
  const [redisKeys, setRedisKeys] = useState<RedisKeyItem[]>([]);
  const [redisCursor, setRedisCursor] = useState<number>(0);
  const [searchMatch, setSearchMatch] = useState<string>('');
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [keyDetail, setKeyDetail] = useState<RedisKeyDetail | null>(null);
  const [isLoadingRedis, setIsLoadingRedis] = useState(false);
  const [isLoadingKeyDetail, setIsLoadingKeyDetail] = useState(false);

  useEffect(() => {
    checkMariaDBStatus();
    checkRedisStatus();
  }, []);

  // ─── MariaDB 相關方法 ───
  const checkMariaDBStatus = async () => {
    setIsLoadingMariaDB(true);
    try {
      const running = await IsMariaDBRunning();
      setIsMariaDBRunning(running);
      if (running) {
        const dbs = await QueryDatabases();
        setDatabases(dbs || []);
      }
    } catch (err) {
      console.error("檢查 MariaDB 失敗:", err);
    } finally {
      setIsLoadingMariaDB(false);
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

  // ─── Redis 相關方法 ───
  const checkRedisStatus = async () => {
    setIsLoadingRedis(true);
    try {
      let isAlive = false;
      if ((window as any).go?.main?.App?.RedisPing) {
        isAlive = await (window as any).go.main.App.RedisPing();
      }
      setIsRedisRunning(isAlive);
      if (isAlive) {
        await fetchRedisDBList();
        await scanRedisKeys(selectedDB, 0, searchMatch, true);
      }
    } catch (err) {
      setIsRedisRunning(false);
    } finally {
      setIsLoadingRedis(false);
    }
  };

  const fetchRedisDBList = async () => {
    try {
      if ((window as any).go?.main?.App?.RedisGetDBList) {
        const dbs = await (window as any).go.main.App.RedisGetDBList();
        setRedisDBList(dbs || []);
      }
    } catch (err) {
      console.error("獲取 Redis DB 列表失敗:", err);
    }
  };

  const scanRedisKeys = async (db: number, cursor: number, match: string, reset: boolean = false) => {
    setIsLoadingRedis(true);
    try {
      if ((window as any).go?.main?.App?.RedisScanKeys) {
        const filter = match.trim() ? match.trim() : '*';
        const res = await (window as any).go.main.App.RedisScanKeys(db, cursor, filter, 50);
        if (res) {
          const newKeys = res.keys || [];
          setRedisKeys(prev => reset ? newKeys : [...prev, ...newKeys]);
          setRedisCursor(res.next_cursor || 0);
          if (reset) {
            setSelectedKey(null);
            setKeyDetail(null);
          }
        }
      }
    } catch (err) {
      console.error("掃描 Redis 鍵值失敗:", err);
    } finally {
      setIsLoadingRedis(false);
    }
  };

  const handleSelectKey = async (keyName: string) => {
    setSelectedKey(keyName);
    setIsLoadingKeyDetail(true);
    try {
      if ((window as any).go?.main?.App?.RedisGetKeyDetail) {
        const detail = await (window as any).go.main.App.RedisGetKeyDetail(selectedDB, keyName);
        setKeyDetail(detail);
      }
    } catch (err) {
      console.error("讀取 Key 詳細資料失敗:", err);
      (window as any).customAlert(`${t("讀取失敗")}: ${err}`);
    } finally {
      setIsLoadingKeyDetail(false);
    }
  };

  const handleDeleteKey = async () => {
    if (!selectedKey) return;
    const confirmed = await (window as any).customConfirm(t("確定要刪除鍵值 '%s' 嗎？", selectedKey));
    if (!confirmed) return;

    try {
      if ((window as any).go?.main?.App?.RedisDeleteKey) {
        await (window as any).go.main.App.RedisDeleteKey(selectedDB, selectedKey);
        await (window as any).customAlert(t("刪除成功"));
        setSelectedKey(null);
        setKeyDetail(null);
        await fetchRedisDBList();
        await scanRedisKeys(selectedDB, 0, searchMatch, true);
      }
    } catch (err: any) {
      (window as any).customAlert(`${t("刪除失敗")}: ${err}`);
    }
  };

  const handleFlushDB = async () => {
    const confirmed = await (window as any).customConfirm(t("確定要清空 DB %s 的所有快取鍵值嗎？此操作不可還原！", selectedDB));
    if (!confirmed) return;

    try {
      if ((window as any).go?.main?.App?.RedisFlushDB) {
        await (window as any).go.main.App.RedisFlushDB(selectedDB);
        await (window as any).customAlert(t("清空成功"));
        setSelectedKey(null);
        setKeyDetail(null);
        await fetchRedisDBList();
        await scanRedisKeys(selectedDB, 0, searchMatch, true);
      }
    } catch (err: any) {
      (window as any).customAlert(`${t("清空失敗")}: ${err}`);
    }
  };

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    scanRedisKeys(selectedDB, 0, searchMatch, true);
  };

  const handleDBChange = (newDB: number) => {
    setSelectedDB(newDB);
    scanRedisKeys(newDB, 0, searchMatch, true);
  };

  const formatTTL = (ttl: number) => {
    if (ttl === -1) return t("無過期時間");
    if (ttl === -2) return t("已過期");
    if (ttl < 60) return `${ttl}s`;
    if (ttl < 3600) return `${Math.floor(ttl / 60)}m ${ttl % 60}s`;
    return `${Math.floor(ttl / 3600)}h ${Math.floor((ttl % 3600) / 60)}m`;
  };

  return (
    <div className="p-6 h-full flex flex-col space-y-4">
      {/* 頂部導航與標頭 */}
      <div className="flex justify-between items-center select-none">
        <div className="flex items-baseline gap-3">
          <h1 className="text-xl font-bold tracking-tight" style={{ color: 'var(--fg)' }}>{t("資料庫瀏覽器")}</h1>
          <p className="text-xs" style={{ color: 'var(--muted)' }}>{t("內建極簡 Schema / 資料表結構速覽，或一鍵透過外部工具管理")}</p>
        </div>

        {/* 雙分頁按鈕切換 */}
        <div className="flex items-center gap-1.5 p-1 rounded-xl border" style={{ backgroundColor: 'var(--surface)', borderColor: 'var(--border)' }}>
          <button
            onClick={() => setActiveTab('mariadb')}
            className={`px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition ${activeTab === 'mariadb' ? 'shadow-sm' : ''}`}
            style={{
              backgroundColor: activeTab === 'mariadb' ? 'var(--accent)' : 'transparent',
              color: activeTab === 'mariadb' ? 'var(--accent-on)' : 'var(--muted)',
            }}
          >
            <Database size={13} /> {t("MariaDB (MySQL)")}
          </button>
          <button
            onClick={() => { setActiveTab('redis'); checkRedisStatus(); }}
            className={`px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition ${activeTab === 'redis' ? 'shadow-sm' : ''}`}
            style={{
              backgroundColor: activeTab === 'redis' ? 'var(--accent)' : 'transparent',
              color: activeTab === 'redis' ? 'var(--accent-on)' : 'var(--muted)',
            }}
          >
            <Zap size={13} /> {t("Redis (NoSQL)")}
          </button>
        </div>
      </div>

      {/* ─── MariaDB 檢視視圖 ─── */}
      {activeTab === 'mariadb' && (
        <div className="flex-1 flex flex-col space-y-4 min-h-0">
          <div className="flex justify-end gap-2.5">
            <button
              onClick={checkMariaDBStatus}
              disabled={isLoadingMariaDB}
              className="btn-custom-hover px-3.5 py-2 rounded-lg text-xs font-semibold border flex items-center gap-1.5 transition duration-200"
              style={{ borderColor: 'var(--border)', backgroundColor: 'var(--card)', color: 'var(--fg-2)' }}
            >
              <RefreshCw size={13} className={isLoadingMariaDB ? 'animate-spin' : ''} />
              {t("重新整理")}
            </button>
            <button
              onClick={handleOpenHeidiSQL}
              disabled={!isMariaDBRunning}
              className="px-3.5 py-2 disabled:opacity-50 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition duration-200"
              style={{ backgroundColor: 'var(--accent)', color: 'var(--accent-on)' }}
            >
              <ExternalLink size={13} /> {t("Open in HeidiSQL")}
            </button>
          </div>

          {!isMariaDBRunning ? (
            <div className="flex-1 border rounded-xl p-8 flex flex-col items-center justify-center text-center space-y-4 select-none" style={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)' }}>
              <div className="p-4 rounded-full" style={{ backgroundColor: 'var(--status-warn-bg)', color: 'var(--status-warn)' }}>
                <AlertTriangle size={36} />
              </div>
              <div>
                <h3 className="text-sm font-bold" style={{ color: 'var(--fg)' }}>{t("MariaDB 尚未啟動")}</h3>
                <p className="text-xs mt-1.5 max-w-sm leading-relaxed" style={{ color: 'var(--muted)' }}>
                  {t("請先前往 儀表板 頁面啟動 MariaDB 資料庫服務，再使用 Database Explorer 進行瀏覽。")}
                </p>
              </div>
            </div>
          ) : (
            <div className="flex-1 border rounded-xl flex overflow-hidden min-h-[400px]" style={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)' }}>
              {/* 左側：Databases 列表 */}
              <div className="w-1/3 border-r flex flex-col select-none" style={{ borderColor: 'var(--border)', backgroundColor: 'var(--bg-deep)' }}>
                <div className="px-5 py-3.5 border-b font-bold text-[10px] tracking-wider uppercase" style={{ borderColor: 'var(--border)', backgroundColor: 'var(--bg-deep)', color: 'var(--meta)' }}>
                  Databases ({databases.length})
                </div>
                <div className="flex-1 overflow-y-auto p-2.5 space-y-0.5">
                  {databases.map(db => (
                    <button
                      key={db}
                      onClick={() => handleSelectSchema(db)}
                      className={`w-full text-left px-3.5 py-2 rounded-lg text-xs font-semibold flex items-center gap-2.5 transition duration-150 db-list-item-btn ${selectedSchema === db ? 'db-list-item-btn--active' : ''}`}
                      style={{ color: selectedSchema === db ? undefined : 'var(--muted)' }}
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
      )}

      {/* ─── Redis (NoSQL) 檢視視圖 ─── */}
      {activeTab === 'redis' && (
        <div className="flex-1 flex flex-col space-y-4 min-h-0">
          {!isRedisRunning ? (
            <div className="flex-1 border rounded-xl p-8 flex flex-col items-center justify-center text-center space-y-4 select-none" style={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)' }}>
              <div className="p-4 rounded-full" style={{ backgroundColor: 'var(--status-warn-bg)', color: 'var(--status-warn)' }}>
                <AlertTriangle size={36} />
              </div>
              <div>
                <h3 className="text-sm font-bold" style={{ color: 'var(--fg)' }}>{t("Redis 尚未啟動")}</h3>
                <p className="text-xs mt-1.5 max-w-sm leading-relaxed" style={{ color: 'var(--muted)' }}>
                  {t("請先前往 儀表板 頁面啟動 Redis 快取服務，再使用快取瀏覽器。")}
                </p>
              </div>
              <button
                onClick={checkRedisStatus}
                className="px-4 py-2 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition"
                style={{ backgroundColor: 'var(--accent)', color: 'var(--accent-on)' }}
              >
                <RefreshCw size={13} className={isLoadingRedis ? 'animate-spin' : ''} /> {t("重新整理連線")}
              </button>
            </div>
          ) : (
            <div className="flex-1 flex flex-col space-y-3 min-h-0">
              {/* 控制工具列 */}
              <div className="flex flex-wrap items-center justify-between gap-3 p-3 rounded-xl border" style={{ backgroundColor: 'var(--surface)', borderColor: 'var(--border)' }}>
                <div className="flex items-center gap-3">
                  <div className="flex items-center gap-1.5 text-xs font-semibold px-2.5 py-1.5 rounded-lg" style={{ backgroundColor: 'var(--status-ok-bg)', color: 'var(--status-ok)', border: '1px solid var(--status-ok)' }}>
                    <span className="w-2 h-2 rounded-full" style={{ backgroundColor: 'var(--status-ok)' }} />
                    <span>127.0.0.1:6379</span>
                  </div>

                  {/* DB 選擇器 */}
                  <div className="flex items-center gap-1.5 text-xs">
                    <span className="font-semibold" style={{ color: 'var(--muted)' }}>DB:</span>
                    <select
                      value={selectedDB}
                      onChange={(e) => handleDBChange(parseInt(e.target.value))}
                      className="rounded-lg px-2.5 py-1 text-xs font-semibold outline-none"
                      style={{ backgroundColor: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--fg)' }}
                    >
                      {Array.from({ length: 16 }).map((_, i) => {
                        const dbInfo = redisDBList.find(d => d.db === i);
                        const count = dbInfo ? dbInfo.keys : 0;
                        return (
                          <option key={i} value={i}>
                            DB {i} ({count} keys)
                          </option>
                        );
                      })}
                    </select>
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  <form onSubmit={handleSearchSubmit} className="relative flex items-center">
                    <input
                      type="text"
                      placeholder={t("搜尋鍵名 (例如 laravel_cache:*)...")}
                      value={searchMatch}
                      onChange={(e) => setSearchMatch(e.target.value)}
                      className="pl-8 pr-3 py-1.5 rounded-lg text-xs outline-none w-56 font-mono"
                      style={{ backgroundColor: 'var(--input-bg)', border: '1px solid var(--border)', color: 'var(--fg)' }}
                    />
                    <Search size={13} className="absolute left-2.5" style={{ color: 'var(--muted)' }} />
                  </form>

                  <button
                    onClick={() => scanRedisKeys(selectedDB, 0, searchMatch, true)}
                    disabled={isLoadingRedis}
                    className="btn-custom-hover px-3 py-1.5 rounded-lg text-xs font-semibold border flex items-center gap-1.5 transition"
                    style={{ borderColor: 'var(--border)', backgroundColor: 'var(--card)', color: 'var(--fg-2)' }}
                    title={t("重新整理")}
                  >
                    <RefreshCw size={12} className={isLoadingRedis ? 'animate-spin' : ''} />
                  </button>

                  <button
                    onClick={handleFlushDB}
                    className="px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition text-white"
                    style={{ backgroundColor: 'var(--status-error)' }}
                    title={t("清空當前 DB")}
                  >
                    <Trash2 size={12} /> {t("清空當前 DB")}
                  </button>
                </div>
              </div>

              {/* 雙欄主視圖：左側 Key 列表，右側 Key 詳情 */}
              <div className="flex-1 border rounded-xl flex overflow-hidden min-h-[400px]" style={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)' }}>
                {/* 左側 Keys */}
                <div className="w-1/3 border-r flex flex-col select-none" style={{ borderColor: 'var(--border)', backgroundColor: 'var(--bg-deep)' }}>
                  <div className="px-5 py-3 border-b font-bold text-[10px] tracking-wider uppercase flex justify-between items-center" style={{ borderColor: 'var(--border)', backgroundColor: 'var(--bg-deep)', color: 'var(--meta)' }}>
                    <span>Keys ({redisKeys.length})</span>
                    {redisCursor !== 0 && (
                      <button
                        onClick={() => scanRedisKeys(selectedDB, redisCursor, searchMatch, false)}
                        disabled={isLoadingRedis}
                        className="text-[10px] px-2 py-0.5 rounded border transition hover:opacity-80"
                        style={{ borderColor: 'var(--border)', color: 'var(--accent)' }}
                      >
                        {t("載入更多")}
                      </button>
                    )}
                  </div>

                  <div className="flex-1 overflow-y-auto p-2 space-y-1">
                    {redisKeys.length > 0 ? (
                      redisKeys.map((item) => (
                        <button
                          key={item.key}
                          onClick={() => handleSelectKey(item.key)}
                          className={`w-full text-left px-3 py-2 rounded-lg text-xs font-mono flex items-center justify-between transition ${selectedKey === item.key ? 'db-list-item-btn--active' : 'hover:bg-[var(--surface)]'}`}
                          style={{ color: selectedKey === item.key ? undefined : 'var(--fg-2)' }}
                        >
                          <div className="flex items-center gap-2 truncate flex-1 min-w-0">
                            <Key size={12} style={{ color: selectedKey === item.key ? 'var(--accent)' : 'var(--meta)' }} />
                            <span className="truncate">{item.key}</span>
                          </div>
                          <span className="text-[10px] uppercase font-sans font-bold px-1.5 py-0.5 rounded shrink-0 ml-1.5" style={{ backgroundColor: 'var(--surface-warm)', color: 'var(--muted)', border: '1px solid var(--border-soft)' }}>
                            {item.type}
                          </span>
                        </button>
                      ))
                    ) : (
                      <div className="h-full flex items-center justify-center italic text-xs" style={{ color: 'var(--meta)' }}>
                        {searchMatch ? t("未找到匹配的 Key") : t("暫無鍵值")}
                      </div>
                    )}
                  </div>
                </div>

                {/* 右側 Key 詳情 */}
                <div className="w-2/3 flex flex-col" style={{ backgroundColor: 'var(--surface)' }}>
                  <div className="px-5 py-3 border-b font-bold text-[10px] tracking-wider uppercase flex justify-between items-center select-none" style={{ borderColor: 'var(--border)', backgroundColor: 'var(--bg-deep)', color: 'var(--meta)' }}>
                    <span>{t("鍵值內容")}</span>
                    {selectedKey && (
                      <button
                        onClick={handleDeleteKey}
                        className="text-[11px] font-semibold px-2 py-1 rounded flex items-center gap-1 transition"
                        style={{ color: 'var(--status-error)', backgroundColor: 'var(--status-error-bg)', border: '1px solid var(--status-error)' }}
                      >
                        <Trash2 size={11} /> {t("刪除此鍵")}
                      </button>
                    )}
                  </div>

                  <div className="flex-1 overflow-y-auto p-5 font-mono text-xs">
                    {isLoadingKeyDetail ? (
                      <div className="h-full flex items-center justify-center" style={{ color: 'var(--muted)' }}>
                        <RefreshCw size={20} className="animate-spin" style={{ color: 'var(--accent)' }} />
                      </div>
                    ) : keyDetail ? (
                      <div className="space-y-4">
                        {/* Meta 資訊欄 */}
                        <div className="grid grid-cols-3 gap-3 p-3 rounded-lg border font-sans" style={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)' }}>
                          <div>
                            <span className="text-[10px] uppercase block font-bold" style={{ color: 'var(--meta)' }}>{t("資料型態")}</span>
                            <span className="text-xs font-bold uppercase" style={{ color: 'var(--accent)' }}>{keyDetail.type}</span>
                          </div>
                          <div>
                            <span className="text-[10px] uppercase block font-bold" style={{ color: 'var(--meta)' }}>{t("剩餘 TTL")}</span>
                            <span className="text-xs font-semibold flex items-center gap-1" style={{ color: 'var(--fg)' }}>
                              <Clock size={11} style={{ color: 'var(--muted)' }} /> {formatTTL(keyDetail.ttl)}
                            </span>
                          </div>
                          <div>
                            <span className="text-[10px] uppercase block font-bold" style={{ color: 'var(--meta)' }}>Size</span>
                            <span className="text-xs font-semibold" style={{ color: 'var(--fg)' }}>{keyDetail.size} bytes</span>
                          </div>
                        </div>

                        {/* 名稱 */}
                        <div className="p-2.5 rounded-lg border flex items-center gap-2" style={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)' }}>
                          <span className="font-bold select-none text-[11px] font-sans" style={{ color: 'var(--meta)' }}>Key:</span>
                          <span className="font-semibold select-all text-xs" style={{ color: 'var(--fg)' }}>{keyDetail.key}</span>
                        </div>

                        {/* Value 內容預覽 */}
                        <div>
                          <pre
                            className="p-4 rounded-lg border overflow-x-auto whitespace-pre-wrap leading-relaxed max-h-[50vh]"
                            style={{ backgroundColor: 'var(--bg-deep)', borderColor: 'var(--border)', color: 'var(--fg)' }}
                          >
                            {keyDetail.value}
                          </pre>
                        </div>
                      </div>
                    ) : (
                      <div className="h-full flex items-center justify-center italic select-none text-xs font-sans" style={{ color: 'var(--meta)' }}>
                        {t("請從左側選擇一個 Key 以檢視內容")}
                      </div>
                    )}
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

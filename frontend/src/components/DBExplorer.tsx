import React, { useState, useEffect, useMemo, useCallback } from 'react';
import { Database, RefreshCw, ExternalLink, AlertTriangle, Table, Zap, Search, Key, Clock, ChevronRight, ChevronDown, Folder, Copy, Check } from 'lucide-react';
import { IsMariaDBRunning, QueryDatabases, QueryTables, OpenInHeidiSQL, GetConfig } from '../../wailsjs/go/main/App';
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

interface KeyTreeNode {
  name: string;
  path: string;
  children: Map<string, KeyTreeNode>;
  leaf?: RedisKeyItem;
}

function buildKeyTree(keys: RedisKeyItem[]): KeyTreeNode {
  const root: KeyTreeNode = { name: '', path: '', children: new Map() };
  for (const item of keys) {
    const segments = item.key.split(':');
    let node = root;
    let acc = '';
    segments.forEach((part, i) => {
      acc = acc === '' ? part : `${acc}:${part}`;
      if (!node.children.has(part)) {
        node.children.set(part, { name: part, path: acc, children: new Map() });
      }
      const next = node.children.get(part);
      if (!next) return;
      node = next;
      if (i === segments.length - 1) {
        node.leaf = item;
        node.path = item.key;
      }
    });
  }
  return root;
}

/** Redis 搜尋：無萬用字元時自動包成 *query*，避免 SCAN MATCH 精確比對導致搜尋無結果 */
function normalizeRedisMatch(raw: string): string {
  const q = (raw || '').trim();
  if (!q) return '*';
  if (q.includes('*') || q.includes('?')) return q;
  return `*${q}*`;
}

const REDIS_SCAN_PAGE = 50;
const REDIS_SCAN_MAX_PAGES = 20;
const REDIS_SCAN_MAX_KEYS = 2000;
/** Redis 面板左右標題列共用固定高度，避免有無按鈕時高度不一致 */
const REDIS_PANEL_HEADER_H = 48;

function tryParseJSON(raw: string): unknown {
  try {
    return JSON.parse(raw);
  } catch {
    return undefined;
  }
}

function deepParseJSONStrings(value: unknown): unknown {
  if (typeof value === 'string') {
    const s = value.trim();
    if ((s.startsWith('{') && s.endsWith('}')) || (s.startsWith('[') && s.endsWith(']'))) {
      try {
        return deepParseJSONStrings(JSON.parse(s));
      } catch {
        return value;
      }
    }
    return value;
  }
  if (Array.isArray(value)) {
    return value.map((v) => deepParseJSONStrings(v));
  }
  if (value && typeof value === 'object') {
    const out: Record<string, unknown> = {};
    for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
      out[k] = deepParseJSONStrings(v);
    }
    return out;
  }
  return value;
}

function formatRedisValue(raw: string): { isJSON: boolean; data: unknown; text: string } {
  const parsed = tryParseJSON(raw);
  if (parsed !== undefined) {
    const deep = deepParseJSONStrings(parsed);
    return { isJSON: true, data: deep, text: JSON.stringify(deep, null, 2) };
  }
  return { isJSON: false, data: raw, text: raw };
}

function JsonNodeView({ name, value, depth = 0, defaultOpen = true }: { name?: string; value: unknown; depth?: number; defaultOpen?: boolean }) {
  const [open, setOpen] = useState(defaultOpen && depth < 6);
  const isObj = value !== null && typeof value === 'object';
  const isArray = Array.isArray(value);

  if (!isObj) {
    let display: string;
    let color = 'var(--fg)';
    if (value === null) {
      display = 'null';
      color = 'var(--muted)';
    } else if (typeof value === 'number') {
      display = String(value);
      color = 'var(--status-info)';
    } else if (typeof value === 'boolean') {
      display = String(value);
      color = 'var(--status-warn)';
    } else {
      display = JSON.stringify(value);
      color = 'var(--status-ok)';
    }
    return (
      <div className="flex items-start gap-2 py-0.5" style={{ paddingLeft: depth * 12 }}>
        {name !== undefined && (
          <span className="shrink-0 font-semibold" style={{ color: 'var(--accent)' }}>{name}:</span>
        )}
        <span className="break-all" style={{ color, userSelect: 'text', WebkitUserSelect: 'text' }}>{display}</span>
      </div>
    );
  }

  const entries = isArray
    ? (value as unknown[]).map((v, i) => [String(i), v] as const)
    : Object.entries(value as Record<string, unknown>);

  return (
    <div style={{ paddingLeft: depth * 12 }}>
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex items-center gap-1 py-0.5 text-left select-none"
        style={{ color: 'var(--fg-2)' }}
      >
        {open ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
        {name !== undefined && (
          <span className="font-semibold" style={{ color: 'var(--accent)' }}>{name}:</span>
        )}
        <span className="text-[11px]" style={{ color: 'var(--meta)' }}>
          {isArray ? `Array(${entries.length})` : `Object{${entries.length}}`}
        </span>
      </button>
      {open && (
        <div className="border-l ml-1.5" style={{ borderColor: 'var(--border)' }}>
          {entries.map(([k, v]) => (
            <JsonNodeView key={k} name={k} value={v} depth={0} defaultOpen={depth < 4} />
          ))}
        </div>
      )}
    </div>
  );
}

function formatCellDisplay(value: unknown): string {
  if (value === null || value === undefined) return '';
  if (typeof value === 'string') {
    const s = value.trim();
    if ((s.startsWith('{') && s.endsWith('}')) || (s.startsWith('[') && s.endsWith(']'))) {
      try {
        return JSON.stringify(JSON.parse(s));
      } catch {
        return value;
      }
    }
    return value;
  }
  return JSON.stringify(value);
}

function normalizeZsetRows(data: unknown): { score: number | string; member: string }[] {
  if (!Array.isArray(data)) return [];
  return data.map((item, idx) => {
    if (item && typeof item === 'object') {
      const o = item as Record<string, unknown>;
      const memberRaw = o.member ?? o.Member ?? o.value ?? '';
      const scoreRaw = o.score ?? o.Score ?? idx;
      return {
        score: typeof scoreRaw === 'number' ? scoreRaw : String(scoreRaw),
        member: typeof memberRaw === 'string' ? memberRaw : JSON.stringify(memberRaw),
      };
    }
    return { score: idx, member: String(item) };
  });
}

function RedisValueTable({
  type,
  data,
}: {
  type: string;
  data: unknown;
}) {
  const tableWrap = 'rounded-lg border overflow-hidden';
  const thStyle: React.CSSProperties = {
    backgroundColor: 'var(--bg-deep)',
    color: 'var(--meta)',
    borderColor: 'var(--border)',
  };
  const tdStyle: React.CSSProperties = {
    borderColor: 'var(--border-soft)',
    color: 'var(--fg-2)',
    userSelect: 'text',
    WebkitUserSelect: 'text',
  };

  if (type === 'list' || type === 'set') {
    const items = Array.isArray(data) ? data : [];
    return (
      <div className={tableWrap} style={{ borderColor: 'var(--border)', backgroundColor: 'var(--card)' }}>
        <table className="w-full text-xs font-mono border-collapse">
          <thead>
            <tr>
              <th className="text-left px-3 py-2 font-bold text-[10px] uppercase border-b w-14" style={thStyle}>#</th>
              <th className="text-left px-3 py-2 font-bold text-[10px] uppercase border-b" style={thStyle}>
                {type === 'set' ? 'Member' : 'Value'}
              </th>
            </tr>
          </thead>
          <tbody>
            {items.map((item, i) => (
              <tr key={i}>
                <td className="px-3 py-2 border-b font-sans" style={{ ...tdStyle, color: 'var(--meta)' }}>{i}</td>
                <td className="px-3 py-2 border-b break-all" style={tdStyle}>{formatCellDisplay(item)}</td>
              </tr>
            ))}
            {items.length === 0 && (
              <tr>
                <td colSpan={2} className="px-3 py-4 text-center italic" style={{ color: 'var(--meta)' }}>
                  {t('暫無鍵值')}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    );
  }

  if (type === 'zset') {
    const rows = normalizeZsetRows(data);
    return (
      <div className={tableWrap} style={{ borderColor: 'var(--border)', backgroundColor: 'var(--card)' }}>
        <table className="w-full text-xs font-mono border-collapse">
          <thead>
            <tr>
              <th className="text-left px-3 py-2 font-bold text-[10px] uppercase border-b w-14" style={thStyle}>#</th>
              <th className="text-left px-3 py-2 font-bold text-[10px] uppercase border-b w-28" style={thStyle}>Score</th>
              <th className="text-left px-3 py-2 font-bold text-[10px] uppercase border-b" style={thStyle}>Member</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row, i) => (
              <tr key={`${row.member}-${i}`}>
                <td className="px-3 py-2 border-b font-sans" style={{ ...tdStyle, color: 'var(--meta)' }}>{i}</td>
                <td className="px-3 py-2 border-b" style={{ ...tdStyle, color: 'var(--status-info)' }}>{row.score}</td>
                <td className="px-3 py-2 border-b break-all" style={tdStyle}>{row.member}</td>
              </tr>
            ))}
            {rows.length === 0 && (
              <tr>
                <td colSpan={3} className="px-3 py-4 text-center italic" style={{ color: 'var(--meta)' }}>
                  {t('暫無鍵值')}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    );
  }

  if (type === 'hash' && data && typeof data === 'object' && !Array.isArray(data)) {
    const entries = Object.entries(data as Record<string, unknown>);
    return (
      <div className={tableWrap} style={{ borderColor: 'var(--border)', backgroundColor: 'var(--card)' }}>
        <table className="w-full text-xs font-mono border-collapse">
          <thead>
            <tr>
              <th className="text-left px-3 py-2 font-bold text-[10px] uppercase border-b w-1/3" style={thStyle}>Field</th>
              <th className="text-left px-3 py-2 font-bold text-[10px] uppercase border-b" style={thStyle}>Value</th>
            </tr>
          </thead>
          <tbody>
            {entries.map(([field, val]) => (
              <tr key={field}>
                <td className="px-3 py-2 border-b font-semibold break-all" style={{ ...tdStyle, color: 'var(--accent)' }}>{field}</td>
                <td className="px-3 py-2 border-b break-all" style={tdStyle}>{formatCellDisplay(val)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  }

  return null;
}

function KeyTreeRows({
  node,
  expanded,
  onToggle,
  selectedKey,
  onSelect,
  depth = 0,
}: {
  node: KeyTreeNode;
  expanded: Set<string>;
  onToggle: (path: string) => void;
  selectedKey: string | null;
  onSelect: (key: string) => void;
  depth?: number;
}) {
  const folders = [...node.children.values()]
    .filter((n) => !n.leaf)
    .sort((a, b) => a.name.localeCompare(b.name));
  const leaves = [...node.children.values()]
    .filter((n) => n.leaf)
    .sort((a, b) => a.name.localeCompare(b.name));
  const indent = depth * 12;

  return (
    <div className={depth > 0 ? 'border-l ml-2' : ''} style={depth > 0 ? { borderColor: 'var(--border-soft)', paddingLeft: 2 } : undefined}>
      {folders.map((child) => {
        const isOpen = expanded.has(child.path);
        return (
          <div key={`dir-${child.path}`} style={{ marginLeft: indent }}>
            <button
              type="button"
              onClick={() => onToggle(child.path)}
              className="w-full text-left px-2 py-1.5 rounded-lg text-[11px] flex items-center gap-1.5 transition hover:bg-[var(--surface)]"
              style={{ color: 'var(--fg-2)', letterSpacing: '-0.01em' }}
            >
              {isOpen ? <ChevronDown size={12} style={{ color: 'var(--meta)' }} /> : <ChevronRight size={12} style={{ color: 'var(--meta)' }} />}
              <Folder size={12} style={{ color: 'var(--status-warn)' }} />
              <span className="break-all flex-1 min-w-0" title={child.path}>{child.name}</span>
              <span className="text-[10px] font-sans" style={{ color: 'var(--meta)' }}>
                {child.children.size}
              </span>
            </button>
            {isOpen && (
              <KeyTreeRows
                node={child}
                expanded={expanded}
                onToggle={onToggle}
                selectedKey={selectedKey}
                onSelect={onSelect}
                depth={depth + 1}
              />
            )}
          </div>
        );
      })}
      {leaves.map((child) => {
        const item = child.leaf;
        if (!item) return null;
        const active = selectedKey === item.key;
        return (
          <div key={item.key} style={{ marginLeft: indent }}>
            <button
              type="button"
              onClick={() => onSelect(item.key)}
              className={`w-full text-left px-2 py-1.5 rounded-lg text-[11px] flex items-center justify-between gap-1.5 transition ${active ? 'db-list-item-btn--active' : 'hover:bg-[var(--surface)]'}`}
              style={{ color: active ? undefined : 'var(--fg-2)', letterSpacing: '-0.01em' }}
            >
              <div className="flex items-center gap-1.5 break-all flex-1 min-w-0">
                <Key size={12} className="shrink-0" style={{ color: active ? 'var(--accent)' : 'var(--meta)' }} />
                <span title={item.key}>{child.name}</span>
              </div>
              <span
                className="text-[10px] uppercase font-sans font-bold px-1.5 py-0.5 rounded shrink-0"
                style={{ backgroundColor: 'var(--surface-warm)', color: 'var(--muted)', border: '1px solid var(--border-soft)' }}
              >
                {item.type}
              </span>
            </button>
          </div>
        );
      })}
    </div>
  );
}

export default function DBExplorer() {
  useLanguage();

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
  const [redisPort, setRedisPort] = useState<number>(6379);
  const [mariaDBPort, setMariaDBPort] = useState<number>(3306);
  const [redisDBList, setRedisDBList] = useState<RedisDBInfo[]>([]);
  const [selectedDB, setSelectedDB] = useState<number>(0);
  const [redisKeys, setRedisKeys] = useState<RedisKeyItem[]>([]);
  const [redisCursor, setRedisCursor] = useState<number>(0);
  const [searchMatch, setSearchMatch] = useState<string>('');
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [keyDetail, setKeyDetail] = useState<RedisKeyDetail | null>(null);
  const [isLoadingRedis, setIsLoadingRedis] = useState(false);
  const [isLoadingKeyDetail, setIsLoadingKeyDetail] = useState(false);
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set());
  const [copied, setCopied] = useState(false);
  const [partialSearch, setPartialSearch] = useState(false);

  const redisAddrDisplay = `127.0.0.1:${redisPort}`;
  const keyTree = useMemo(() => buildKeyTree(redisKeys), [redisKeys]);
  const valueView = useMemo(() => (keyDetail ? formatRedisValue(keyDetail.value) : null), [keyDetail]);

  const loadPorts = useCallback(async () => {
    try {
      const cfg = await GetConfig();
      const rawRedis = cfg?.global?.redis_port;
      const rPort = typeof rawRedis === 'string' ? parseInt(rawRedis, 10) : Number(rawRedis);
      setRedisPort(Number.isFinite(rPort) && rPort > 0 ? rPort : 6379);

      const rawMaria = cfg?.global?.mariadb_port;
      const mPort = typeof rawMaria === 'string' ? parseInt(rawMaria, 10) : Number(rawMaria);
      setMariaDBPort(Number.isFinite(mPort) && mPort > 0 ? mPort : 3306);
    } catch (err) {
      console.error('讀取資料庫與快取端口失敗:', err);
      setRedisPort(6379);
      setMariaDBPort(3306);
    }
  }, []);

  useEffect(() => {
    checkMariaDBStatus();
    loadPorts().then(() => checkRedisStatus());
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const next = new Set<string>();
    for (const item of redisKeys) {
      const segments = item.key.split(':');
      let acc = '';
      for (let i = 0; i < segments.length - 1; i += 1) {
        acc = acc === '' ? segments[i] : `${acc}:${segments[i]}`;
        next.add(acc);
      }
    }
    setExpandedPaths(next);
  }, [redisKeys]);

  const togglePath = (path: string) => {
    setExpandedPaths((prev) => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  };

  const handleCopyValue = async () => {
    if (!valueView) return;
    try {
      await navigator.clipboard.writeText(valueView.text);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch (err) {
      console.error('複製失敗:', err);
    }
  };

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
      console.error('檢查 MariaDB 失敗:', err);
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
      console.error('載入資料表失敗:', err);
      setTables([`${t('載入失敗')}: ${err}`]);
    } finally {
      setIsTablesLoading(false);
    }
  };

  const handleOpenHeidiSQL = async () => {
    try {
      await OpenInHeidiSQL();
    } catch (err) {
      (window as any).customAlert(`${t('開啟 HeidiSQL 失敗')}: ${err}`);
    }
  };

  const checkRedisStatus = async () => {
    // 連線探測不視為載入中，避免 Redis 未啟動時重新整理圖示一直轉動
    try {
      let isAlive = false;
      if ((window as any).go?.main?.App?.RedisPing) {
        isAlive = await (window as any).go.main.App.RedisPing();
      }
      setIsRedisRunning(isAlive);
      if (isAlive) {
        await fetchRedisDBList();
        await scanRedisKeys(selectedDB, 0, searchMatch, true);
      } else {
        setIsLoadingRedis(false);
        setRedisKeys([]);
        setRedisCursor(0);
        setPartialSearch(false);
        setKeyDetail(null);
        setSelectedKey(null);
      }
    } catch (err) {
      console.error('檢查 Redis 失敗:', err);
      setIsRedisRunning(false);
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
      console.error('獲取 Redis DB 列表失敗:', err);
    }
  };

  const fetchRedisKeysPage = async (db: number, cursor: number, match: string): Promise<{ keys: RedisKeyItem[]; nextCursor: number }> => {
    if (!(window as any).go?.main?.App?.RedisScanKeys) {
      return { keys: [], nextCursor: 0 };
    }
    const filter = normalizeRedisMatch(match);
    const res = await (window as any).go.main.App.RedisScanKeys(db, cursor, filter, REDIS_SCAN_PAGE);
    return { keys: res?.keys || [], nextCursor: res?.next_cursor || 0 };
  };

  /** 多頁 SCAN：避免單頁空結果被誤判為「找不到 Key」 */
  const scanRedisKeys = async (db: number, cursor: number, match: string, reset = false) => {
    setIsLoadingRedis(true);
    try {
      const batch: RedisKeyItem[] = [];
      let nextCursor = cursor;
      let pages = 0;
      let truncated = false;

      do {
        const page = await fetchRedisKeysPage(db, nextCursor, match);
        batch.push(...page.keys);
        nextCursor = page.nextCursor;
        pages += 1;
        if (batch.length >= REDIS_SCAN_MAX_KEYS || pages >= REDIS_SCAN_MAX_PAGES) {
          if (nextCursor !== 0) truncated = true;
          break;
        }
      } while (nextCursor !== 0);

      setRedisKeys(prev => (reset ? batch : [...prev, ...batch]));
      setRedisCursor(nextCursor);
      setPartialSearch(truncated);
    } catch (err) {
      console.error('掃描 Redis 鍵值失敗:', err);
    } finally {
      setIsLoadingRedis(false);
    }
  };

  const handleSelectKey = async (keyName: string) => {
    setSelectedKey(keyName);
    setIsLoadingKeyDetail(true);
    setKeyDetail(null);
    try {
      if ((window as any).go?.main?.App?.RedisGetKeyDetail) {
        const detail = await (window as any).go.main.App.RedisGetKeyDetail(selectedDB, keyName);
        setKeyDetail(detail);
      }
    } catch (err) {
      console.error('獲取 Key 詳情失敗:', err);
    } finally {
      setIsLoadingKeyDetail(false);
    }
  };

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    scanRedisKeys(selectedDB, 0, searchMatch, true);
  };

  const handleDBChange = (newDB: number) => {
    setSelectedDB(newDB);
    setSelectedKey(null);
    setKeyDetail(null);
    scanRedisKeys(newDB, 0, searchMatch, true);
  };

  const formatTTL = (ttl: number) => {
    if (ttl === -1) return t('無過期時間');
    if (ttl === -2) return t('已過期');
    if (ttl < 60) return `${ttl}s`;
    if (ttl < 3600) return `${Math.floor(ttl / 60)}m ${ttl % 60}s`;
    return `${Math.floor(ttl / 3600)}h ${Math.floor((ttl % 3600) / 60)}m`;
  };

  return (
    <div className="p-6 h-full flex flex-col space-y-4">
      {/* 標題列：標題與分頁切換器同行，節省縱向空間 */}
      <div className="flex items-center justify-between gap-3 min-w-0 select-none">
        <div className="flex items-baseline gap-3 min-w-0">
          <h1 className="text-xl font-bold tracking-tight shrink-0" style={{ color: 'var(--fg)' }}>{t('資料庫瀏覽器')}</h1>
          <p className="text-xs truncate min-w-0" style={{ color: 'var(--muted)' }}>{t('內建極簡 Schema / 資料表結構速覽，或一鍵透過外部工具管理')}</p>
        </div>

        <div className="flex items-center gap-1 p-1 rounded-xl border shrink-0" style={{ backgroundColor: 'var(--surface)', borderColor: 'var(--border)' }}>
          <button
            onClick={() => setActiveTab('mariadb')}
            title={isMariaDBRunning ? `127.0.0.1:${mariaDBPort} (${t('運行中')})` : `127.0.0.1:${mariaDBPort} (${t('未啟動')})`}
            className={`px-2.5 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition ${activeTab === 'mariadb' ? 'shadow-sm' : ''}`}
            style={{
              backgroundColor: activeTab === 'mariadb' ? 'var(--accent)' : 'transparent',
              color: activeTab === 'mariadb' ? 'var(--accent-on)' : 'var(--muted)',
            }}
          >
            <Database size={13} /> {t('MariaDB (MySQL)')}
          </button>
          <button
            onClick={() => {
              setActiveTab('redis');
              checkRedisStatus();
            }}
            title={isRedisRunning ? `127.0.0.1:${redisPort} (${t('運行中')})` : `127.0.0.1:${redisPort} (${t('未啟動')})`}
            className={`px-2.5 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition ${activeTab === 'redis' ? 'shadow-sm' : ''}`}
            style={{
              backgroundColor: activeTab === 'redis' ? 'var(--accent)' : 'transparent',
              color: activeTab === 'redis' ? 'var(--accent-on)' : 'var(--muted)',
            }}
          >
            <Zap size={13} /> {t('Redis (NoSQL)')}
          </button>
        </div>
      </div>

      {/* ─── MariaDB ─── */}
      {activeTab === 'mariadb' && (
        <div className="flex-1 flex flex-col space-y-4 min-h-0">
          {!isMariaDBRunning ? (
            <div className="flex-1 border rounded-xl p-8 flex flex-col items-center justify-center text-center space-y-4 select-none" style={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)' }}>
              <div className="p-4 rounded-full" style={{ backgroundColor: 'var(--status-warn-bg)', color: 'var(--status-warn)' }}>
                <AlertTriangle size={36} />
              </div>
              <div>
                <h3 className="text-sm font-bold" style={{ color: 'var(--fg)' }}>{t('MariaDB 尚未啟動')}</h3>
                <p className="text-xs mt-1.5 max-w-sm leading-relaxed" style={{ color: 'var(--muted)' }}>
                  {t('請先前往 儀表板 頁面啟動 MariaDB 資料庫服務，再使用 Database Explorer 進行瀏覽。')}
                </p>
              </div>
            </div>
          ) : (
            <div className="flex-1 border rounded-xl flex overflow-hidden min-h-[400px]" style={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)' }}>
              <div className="w-1/3 border-r flex flex-col select-none" style={{ borderColor: 'var(--border)', backgroundColor: 'var(--bg-deep)' }}>
                <div
                  className="px-4 border-b font-bold text-[10px] tracking-wider uppercase flex items-center justify-between shrink-0"
                  style={{ borderColor: 'var(--border)', backgroundColor: 'var(--bg-deep)', color: 'var(--meta)', height: 48, boxSizing: 'border-box' }}
                >
                  <span>Databases ({databases.length})</span>
                  <button
                    onClick={checkMariaDBStatus}
                    disabled={isLoadingMariaDB}
                    title={t('重新整理')}
                    className="btn-custom-hover px-2 py-1 rounded-md text-[11px] font-semibold border flex items-center gap-1 transition duration-200 shrink-0"
                    style={{ borderColor: 'var(--border)', backgroundColor: 'var(--card)', color: 'var(--fg-2)' }}
                  >
                    <RefreshCw size={11} className={isLoadingMariaDB ? 'animate-spin' : ''} />
                    <span>{t('重新整理')}</span>
                  </button>
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

              <div className="w-2/3 flex flex-col" style={{ backgroundColor: 'var(--surface)' }}>
                <div
                  className="px-4 border-b font-bold text-[10px] tracking-wider uppercase select-none flex items-center justify-between shrink-0"
                  style={{ borderColor: 'var(--border)', backgroundColor: 'var(--bg-deep)', color: 'var(--meta)', height: 48, boxSizing: 'border-box' }}
                >
                  <span className="truncate">Tables {selectedSchema ? `(in ${selectedSchema})` : ''}</span>
                  <button
                    onClick={handleOpenHeidiSQL}
                    disabled={!isMariaDBRunning}
                    title={t('Open in HeidiSQL')}
                    className="px-2 py-1 disabled:opacity-40 rounded-md text-[11px] font-semibold flex items-center gap-1 transition duration-200 shrink-0"
                    style={{ backgroundColor: 'var(--accent)', color: 'var(--accent-on)' }}
                  >
                    <ExternalLink size={11} /> {t('Open in HeidiSQL')}
                  </button>
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
                          <Table size={12} style={{ color: 'var(--accent)' }} /> {t('資料庫')} '{selectedSchema}' {t('的資料表：')}
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
                        {t('（此資料庫沒有資料表）')}
                      </div>
                    )
                  ) : (
                    <div className="h-full flex items-center justify-center italic select-none text-xs" style={{ color: 'var(--meta)' }}>
                      {t('請選擇左側的資料庫以檢視資料表')}
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>
      )}

      {/* ─── Redis ─── */}
      {activeTab === 'redis' && (
        <div className="flex-1 flex flex-col space-y-4 min-h-0">
          {!isRedisRunning ? (
            <div className="flex-1 border rounded-xl p-8 flex flex-col items-center justify-center text-center space-y-4 select-none" style={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)' }}>
              <div className="p-4 rounded-full" style={{ backgroundColor: 'var(--status-warn-bg)', color: 'var(--status-warn)' }}>
                <AlertTriangle size={36} />
              </div>
              <div>
                <h3 className="text-sm font-bold" style={{ color: 'var(--fg)' }}>{t('Redis 尚未啟動')}</h3>
                <p className="text-xs mt-1.5 max-w-sm leading-relaxed" style={{ color: 'var(--muted)' }}>
                  {t('請先前往 儀表板 頁面啟動 Redis 快取服務，再使用快取瀏覽器。')}
                </p>
              </div>
            </div>
          ) : (
            <div className="flex-1 border rounded-xl flex overflow-hidden min-h-[400px]" style={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)' }}>
              <div className="w-1/3 border-r flex flex-col min-h-0" style={{ borderColor: 'var(--border)', backgroundColor: 'var(--bg-deep)' }}>
                <div
                  className="px-4 border-b font-bold text-[10px] tracking-wider uppercase flex items-center justify-between select-none shrink-0"
                  style={{
                    borderColor: 'var(--border)',
                    backgroundColor: 'var(--bg-deep)',
                    color: 'var(--meta)',
                    height: REDIS_PANEL_HEADER_H,
                    boxSizing: 'border-box',
                  }}
                >
                  <span className="shrink-0">Keys ({redisKeys.length})</span>
                  <div className="flex items-center gap-1.5 shrink-0">
                    {redisCursor !== 0 && (
                      <button
                        onClick={() => scanRedisKeys(selectedDB, redisCursor, searchMatch, false)}
                        disabled={isLoadingRedis}
                        className="text-[10px] px-1.5 py-0.5 rounded border transition hover:opacity-80 shrink-0"
                        style={{ borderColor: 'var(--border)', color: 'var(--accent)' }}
                      >
                        {t('載入更多')}
                      </button>
                    )}
                    <button
                      onClick={() => checkRedisStatus()}
                      disabled={isLoadingRedis && isRedisRunning}
                      title={t('重新整理')}
                      className="btn-custom-hover px-2 py-1 rounded-md text-[11px] font-semibold border flex items-center gap-1 transition duration-200"
                      style={{ borderColor: 'var(--border)', backgroundColor: 'var(--card)', color: 'var(--fg-2)' }}
                    >
                      <RefreshCw size={11} className={isLoadingRedis && isRedisRunning ? 'animate-spin' : ''} />
                      <span>{t('重新整理')}</span>
                    </button>
                  </div>
                </div>

                {/* 搜尋鍵名：移至 KEYS 下方內容區最上方 */}
                <div className="p-2 border-b shrink-0" style={{ borderColor: 'var(--border)' }}>
                  <form onSubmit={handleSearchSubmit} className="relative flex items-center w-full">
                    <input
                      type="text"
                      placeholder={t('搜尋鍵名')}
                      value={searchMatch}
                      onChange={(e) => setSearchMatch(e.target.value)}
                      className="pl-7 pr-2 rounded-md text-[11px] outline-none w-full font-mono"
                      style={{
                        backgroundColor: 'var(--input-bg)',
                        border: '1px solid var(--border)',
                        color: 'var(--fg)',
                        letterSpacing: '-0.01em',
                        height: 28,
                      }}
                      title={t('支援 Redis 萬用字元，例如 cache:*')}
                    />
                    <Search size={12} className="absolute left-2 pointer-events-none" style={{ color: 'var(--muted)' }} />
                  </form>
                </div>

                <div className="flex-1 overflow-y-auto p-2 space-y-0.5">
                  {partialSearch && redisKeys.length > 0 && (
                    <div className="text-[10px] px-2 py-1.5 mb-1 rounded border" style={{ color: 'var(--status-warn)', borderColor: 'var(--status-warn)', backgroundColor: 'var(--status-warn-bg)' }}>
                      {t('僅顯示部分搜尋結果，可按載入更多')}
                    </div>
                  )}
                  {redisKeys.length > 0 ? (
                    <KeyTreeRows
                      node={keyTree}
                      expanded={expandedPaths}
                      onToggle={togglePath}
                      selectedKey={selectedKey}
                      onSelect={handleSelectKey}
                    />
                  ) : (
                    <div className="h-full flex items-center justify-center italic text-xs select-none" style={{ color: 'var(--meta)' }}>
                      {searchMatch ? t('未找到匹配的 Key') : t('暫無鍵值')}
                    </div>
                  )}
                </div>
              </div>

              <div className="w-2/3 flex flex-col min-h-0" style={{ backgroundColor: 'var(--surface)' }}>
                <div
                  className="px-4 border-b font-bold text-[10px] tracking-wider uppercase flex justify-between items-center select-none shrink-0"
                  style={{
                    borderColor: 'var(--border)',
                    backgroundColor: 'var(--bg-deep)',
                    color: 'var(--meta)',
                    height: REDIS_PANEL_HEADER_H,
                    boxSizing: 'border-box',
                  }}
                >
                  <div className="flex items-center gap-2.5">
                    <div className="flex items-center gap-1 text-[11px] font-normal shrink-0">
                      <span className="font-bold text-[10px]" style={{ color: 'var(--meta)' }}>DB</span>
                      <select
                        value={selectedDB}
                        onChange={(e) => handleDBChange(parseInt(e.target.value))}
                        disabled={!isRedisRunning}
                        className="rounded-md px-1.5 py-0.5 text-[11px] font-semibold outline-none disabled:opacity-50 cursor-pointer"
                        style={{ backgroundColor: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--fg)' }}
                      >
                        {Array.from({ length: 16 }).map((_, i) => {
                          const dbInfo = redisDBList.find(d => d.db === i);
                          const count = dbInfo ? dbInfo.keys : 0;
                          return (
                            <option key={i} value={i}>
                              {i} ({count})
                            </option>
                          );
                        })}
                      </select>
                    </div>
                    <span className="border-l pl-2.5" style={{ borderColor: 'var(--border)' }}>{t('鍵值內容')}</span>
                  </div>
                  {keyDetail && (
                    <button
                      onClick={handleCopyValue}
                      className="text-[10px] font-semibold px-2 py-1 rounded flex items-center gap-1 transition shrink-0"
                      style={{ color: copied ? 'var(--status-ok)' : 'var(--fg-2)', backgroundColor: 'var(--card)', border: '1px solid var(--border)' }}
                      title={t('複製內容')}
                    >
                      {copied ? <Check size={11} /> : <Copy size={11} />}
                      {copied ? t('已複製') : t('複製')}
                    </button>
                  )}
                </div>

                <div
                  className="flex-1 min-h-0 overflow-y-auto p-5 text-[11px]"
                  style={{ userSelect: 'text', WebkitUserSelect: 'text', letterSpacing: '-0.01em' }}
                >
                  {isLoadingKeyDetail ? (
                    <div className="h-full flex items-center justify-center" style={{ color: 'var(--muted)' }}>
                      <RefreshCw size={20} className="animate-spin" style={{ color: 'var(--accent)' }} />
                    </div>
                  ) : keyDetail ? (
                    <div className="space-y-4">
                      <div className="grid grid-cols-3 gap-3 p-3 rounded-lg border font-sans select-none" style={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)' }}>
                        <div>
                          <span className="text-[10px] uppercase block font-bold" style={{ color: 'var(--meta)' }}>{t('資料型態')}</span>
                          <span className="text-xs font-bold uppercase" style={{ color: 'var(--accent)' }}>{keyDetail.type}</span>
                        </div>
                        <div>
                          <span className="text-[10px] uppercase block font-bold" style={{ color: 'var(--meta)' }}>{t('剩餘 TTL')}</span>
                          <span className="text-xs font-semibold flex items-center gap-1" style={{ color: 'var(--fg)' }}>
                            <Clock size={11} style={{ color: 'var(--muted)' }} /> {formatTTL(keyDetail.ttl)}
                          </span>
                        </div>
                        <div>
                          <span className="text-[10px] uppercase block font-bold" style={{ color: 'var(--meta)' }}>Size</span>
                          <span className="text-xs font-semibold" style={{ color: 'var(--fg)' }}>{keyDetail.size} bytes</span>
                        </div>
                      </div>

                      <div className="p-2.5 rounded-lg border flex items-start gap-2" style={{ backgroundColor: 'var(--card)', borderColor: 'var(--border)' }}>
                        <span className="font-bold select-none text-[11px] font-sans shrink-0" style={{ color: 'var(--meta)' }}>Key:</span>
                        <span className="font-semibold select-all text-xs break-all" style={{ color: 'var(--fg)' }}>{keyDetail.key}</span>
                      </div>

                      <div>
                        {['list', 'set', 'zset', 'hash'].includes(keyDetail.type) && valueView?.isJSON ? (
                          <RedisValueTable type={keyDetail.type} data={valueView.data} />
                        ) : valueView?.isJSON ? (
                          <div
                            className="p-4 rounded-lg border"
                            style={{ backgroundColor: 'var(--bg-deep)', borderColor: 'var(--border)', color: 'var(--fg)', userSelect: 'text', WebkitUserSelect: 'text' }}
                          >
                            <JsonNodeView value={valueView.data} depth={0} defaultOpen />
                          </div>
                        ) : (
                          <pre
                            className="p-4 rounded-lg border overflow-x-auto whitespace-pre-wrap leading-relaxed"
                            style={{ backgroundColor: 'var(--bg-deep)', borderColor: 'var(--border)', color: 'var(--fg)', userSelect: 'text', WebkitUserSelect: 'text', letterSpacing: '-0.02em', fontFamily: 'var(--font-mono)', fontSize: '11px' }}
                          >
                            {valueView?.text ?? keyDetail.value}
                          </pre>
                        )}
                      </div>
                    </div>
                  ) : (
                    <div className="h-full flex items-center justify-center italic select-none text-xs font-sans" style={{ color: 'var(--meta)' }}>
                      {t('請從左側選擇一個 Key 以檢視內容')}
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

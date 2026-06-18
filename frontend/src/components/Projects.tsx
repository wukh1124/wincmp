import React, { useState, useEffect } from 'react';
import { Play, Square, Plus, Edit, FolderOpen, Link, Check, X, Shield, Settings, Trash2, Copy, Globe, Terminal } from 'lucide-react';
import ProjectTerminal from './ProjectTerminal';
import {
  GetConfig, SaveConfig, GetScanResult, GetServicesStatus,
  StartProjectRuntime, StopProjectRuntime, ReloadCaddy,
  OpenFolder, SelectFolder, DetectProjectPath, OpenProjectCaddyfile
} from '../../wailsjs/go/main/App';
import { t, useLanguage } from '../i18n';

interface Project {
  name: string;
  domains: string[];
  type?: string;
  runtime_type?: string;
  php_version: string;
  root_path: string;
  ssl_crt: string;
  ssl_key: string;
  use_ssl: boolean;
  enabled: boolean;
  runtime_port?: number;
  runtime_mode?: string;
  runtime_version?: string;
  command?: string;
  use_wincmp_bin?: boolean;
}

export default function Projects({ highlightedProjectName, clearHighlight }: { highlightedProjectName?: string | null; clearHighlight?: () => void }) {
  useLanguage();
  const [config, setConfig] = useState<any>(null);
  const [scanResult, setScanResult] = useState<any>(null);
  const [servicesStatus, setServicesStatus] = useState<Record<string, boolean>>({});
  const [loadingProjects, setLoadingProjects] = useState<Record<string, boolean>>({});
  const [editingProject, setEditingProject] = useState<Project | null>(null);
  const [editIndex, setEditIndex] = useState<number | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isDetecting, setIsDetecting] = useState(false);
  const [detected, setDetected] = useState(false);
  const [terminalProject, setTerminalProject] = useState<string | null>(null);
  const [highlightedRow, setHighlightedRow] = useState<string | null>(null);
  const [showGuide, setShowGuide] = useState(false);

  useEffect(() => {
    if (config?.projects && config.projects.length > 0) {
      const isShown = localStorage.getItem('wincmp_onboarding_shown') === 'true';
      if (!isShown) {
        setShowGuide(true);
      }
    }
  }, [config]);

  const dismissGuide = () => {
    localStorage.setItem('wincmp_onboarding_shown', 'true');
    setShowGuide(false);
  };

  useEffect(() => {
    if (highlightedProjectName && config?.projects) {
      setTimeout(() => {
        const element = document.getElementById(`project-row-${highlightedProjectName}`);
        if (element) {
          element.scrollIntoView({ behavior: 'smooth', block: 'center' });
          setHighlightedRow(highlightedProjectName);
          setTimeout(() => { setHighlightedRow(null); if (clearHighlight) clearHighlight(); }, 2000);
        } else { if (clearHighlight) clearHighlight(); }
      }, 100);
    }
  }, [highlightedProjectName, config]);

  const projectTypes = [
    { value: 'static', label: t('Static HTML') },
    { value: 'php', label: t('純 PHP') },
    { value: 'laravel', label: t('Laravel PHP') },
    { value: 'vite', label: t('Vite React/Vue') },
    { value: 'next', label: t('Next.js') },
    { value: 'nuxt', label: t('Nuxt') },
    { value: 'astro', label: t('Astro') },
    { value: 'python_fastapi', label: t('Python FastAPI') },
    { value: 'python_django', label: t('Python Django') },
    { value: 'python_flask', label: t('Python Flask') },
    { value: 'go_api', label: t('Go Web API') },
    { value: 'pocketbase', label: t('PocketBase') },
    { value: 'custom', label: t('Custom Command') }
  ];

  const runtimeTypes = [
    { value: 'none', label: t('無 Runtime') },
    { value: 'auto', label: t('Auto (Node/Bun)') },
    { value: 'node', label: t('Node.js') },
    { value: 'bun', label: t('Bun') },
    { value: 'python', label: t('Python') },
    { value: 'go_air', label: t('Go + Air (Hot Reload)') },
    { value: 'go_run', label: t('Go Run') },
    { value: 'custom', label: t('Custom Command') }
  ];

  const hasBundledRuntime = (rt?: string) => {
    if (!rt || rt === 'none' || rt === 'custom') return false;
    if (rt === 'node') return !!(scanResult?.NodeList && scanResult.NodeList.length > 0);
    if (rt === 'bun') return !!(scanResult?.BunList && scanResult.BunList.length > 0);
    if (rt === 'auto') return !!(scanResult?.NodeList && scanResult.NodeList.length > 0) || !!(scanResult?.BunList && scanResult.BunList.length > 0);
    return false;
  };

  const shouldDefaultUseWinCMPBin = (rt?: string) => {
    if (!rt || rt === 'none' || rt === 'custom') return false;
    const hasBin = hasBundledRuntime(rt);
    if (!hasBin) return false;

    // 若系統已包含全域 Node 或 Bun 環境，則預設不幫用戶勾選「使用內建 bin」
    if (rt === 'node' && (scanResult as any)?.has_global_node) return false;
    if (rt === 'bun' && (scanResult as any)?.has_global_bun) return false;
    if (rt === 'auto' && ((scanResult as any)?.has_global_node || (scanResult as any)?.has_global_bun)) return false;

    return true;
  };

  useEffect(() => {
    async function initData() {
      try {
        setConfig(await GetConfig());
        setScanResult(await GetScanResult());
        await updateStatus();
      } catch (err) { console.error("載入專案資料失敗:", err); }
    }
    initData();
  }, []);

  useEffect(() => {
    const timer = setInterval(() => updateStatus(), 2000);
    return () => clearInterval(timer);
  }, []);

  const updateStatus = async () => {
    try { setServicesStatus(await GetServicesStatus()); } catch (err) { console.error("更新狀態失敗:", err); }
  };

  const isRuntimeProject = (type?: string) => type && !['static', 'laravel', 'php'].includes(type);

  const handleToggleEnable = async (idx: number) => {
    if (!config) return;
    const newCfg = { ...config };
    newCfg.projects[idx].enabled = !newCfg.projects[idx].enabled;
    try { await SaveConfig(newCfg); setConfig(newCfg); await ReloadCaddy(); await updateStatus(); }
    catch (err) { (window as any).customAlert(`${t("儲存設定失敗")}: ${err}`); }
  };

  const handleStartRuntime = async (name: string) => {
    setLoadingProjects(prev => ({ ...prev, [name]: true }));
    try { await StartProjectRuntime(name); await updateStatus(); }
    catch (err) { (window as any).customAlert(`${t("啟動 Runtime 失敗")}: ${err}`); }
    finally { setLoadingProjects(prev => ({ ...prev, [name]: false })); }
  };

  const handleStopRuntime = async (name: string) => {
    setLoadingProjects(prev => ({ ...prev, [name]: true }));
    try { await StopProjectRuntime(name); await updateStatus(); }
    catch (err) { (window as any).customAlert(`${t("停止 Runtime 失敗")}: ${err}`); }
    finally { setLoadingProjects(prev => ({ ...prev, [name]: false })); }
  };

  const handleOpenFolder = async (path: string) => {
    try { await OpenFolder(path); }
    catch (err) { (window as any).customAlert(`${t("無法開啟目錄")}: ${err}`); }
  };

  const handleCopyLink = (domain: string, useSSL: boolean) => {
    const link = `${useSSL ? 'https' : 'http'}://${domain}`;
    navigator.clipboard.writeText(link);
    (window as any).customAlert(`${t("已複製連結")}: ${link}`);
  };

  const handleOpenEditModal = (proj: Project | null, idx: number | null) => {
    if (proj) {
      const hasBin = hasBundledRuntime(proj.runtime_type);
      setEditingProject({ ...proj, use_wincmp_bin: hasBin ? proj.use_wincmp_bin : false });
      setDetected(true);
    } else {
      setEditingProject({
        name: '', domains: [''], type: 'static', runtime_type: 'none',
        php_version: scanResult?.PHPList?.[0]?.MajorMin || '', root_path: '',
        ssl_crt: '', ssl_key: '', use_ssl: true, enabled: true,
        runtime_port: 3000, runtime_mode: 'Background',
        runtime_version: scanResult?.NodeList?.[0]?.Version || '',
        command: '', use_wincmp_bin: false
      });
      setDetected(false);
    }
    setEditIndex(idx);
    setIsModalOpen(true);
    setIsDetecting(false);
  };

  const runAutoDetection = async (path: string) => {
    if (!editingProject || !path.trim()) return;
    setIsDetecting(true);
    try {
      const res = await DetectProjectPath(path);
      if (res) {
        const hasBin = hasBundledRuntime(res.runtime_type);
        setEditingProject({
          ...editingProject, root_path: path, name: res.name,
          domains: res.domains?.length > 0 ? res.domains : [`local-${res.name.toLowerCase().replace(/_/g, '-')}.test`],
          type: res.type || 'static', runtime_type: res.runtime_type || 'none',
          runtime_port: res.runtime_port || 3000,
          php_version: res.php_version || scanResult?.PHPList?.[0]?.MajorMin || '',
          runtime_version: res.runtime_type === 'bun' ? scanResult?.BunList?.[0]?.Version : scanResult?.NodeList?.[0]?.Version || '',
          use_wincmp_bin: shouldDefaultUseWinCMPBin(res.runtime_type)
        });
        setDetected(true);
      }
    } catch (err) { console.error("自動偵測專案失敗:", err); (window as any).customAlert(`${t("自動偵測專案失敗")}: ${err}`); }
    finally { setIsDetecting(false); }
  };

  const handleSelectRootPath = async () => {
    if (!editingProject) return;
    try {
      const path = await SelectFolder();
      if (path) {
        setEditingProject(prev => prev ? { ...prev, root_path: path } : null);
        if (editIndex === null) await runAutoDetection(path);
      }
    } catch (err) { console.error("選擇目錄失敗:", err); }
  };

  const handleSaveProject = async () => {
    if (!editingProject || !config) return;
    const trimName = editingProject.name.trim();
    if (!trimName) { (window as any).customAlert(t("專案名稱不能為空")); return; }
    const nameRegex = /^[a-zA-Z0-9_-]+$/;
    if (!nameRegex.test(trimName)) { (window as any).customAlert(t("專案名稱僅能包含英數字、連字號(-)與底線(_)喔！")); return; }
    if (editingProject.use_wincmp_bin && !hasBundledRuntime(editingProject.runtime_type)) {
      (window as any).customAlert(t("儲存失敗：您勾選了使用 WinCMP 內建執行檔，但系統未在 ./bin/ 下偵測到可用的 Node.js 或 Bun 執行檔。請先下載並放置於對應目錄，或取消勾選此選項以使用系統全域執行檔。"));
      return;
    }
    const newCfg = { ...config };
    const cleanProj = { ...editingProject }; cleanProj.name = trimName;
    if (cleanProj.type !== 'laravel' && cleanProj.type !== 'php') cleanProj.php_version = '';
    cleanProj.domains = cleanProj.domains.filter(d => d.trim() !== "");
    if (cleanProj.domains.length === 0) cleanProj.domains = [`local-${cleanProj.name.toLowerCase().replace(/_/g, '-')}.test`];
    const domainRegex = /^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$/;
    let hasInvalidDomain = false;
    let invalidDomainName = "";
    for (const d of cleanProj.domains) {
      if (!domainRegex.test(d)) {
        hasInvalidDomain = true;
        invalidDomainName = d;
        break;
      }
    }
    if (hasInvalidDomain) {
      const confirmSave = await (window as any).customConfirm(t("網域 '%s' 格式不正確喔！請確認是否要繼續保存設定？（正確格式例如 my-site.test，且不能包含底線、埠號或路徑。）", invalidDomainName));
      if (!confirmSave) return;
    }
    const duplicate = newCfg.projects?.find((p: any, idx: number) => idx !== editIndex && p.name.trim().toLowerCase() === cleanProj.name.trim().toLowerCase());
    if (duplicate) { (window as any).customAlert(t("專案名稱已存在，請使用其他名稱喔！")); return; }
    let oldName = ''; let isNameChanged = false;
    if (editIndex !== null) { oldName = config.projects[editIndex].name; isNameChanged = oldName !== cleanProj.name; }
    if (isNameChanged && oldName) {
      const isOldRunning = !!servicesStatus[`runtime_${oldName}`];
      if (isOldRunning) { try { await StopProjectRuntime(oldName); } catch (e) { } }
    }
    if (editIndex === null) { newCfg.projects = [...(newCfg.projects || []), cleanProj]; }
    else { newCfg.projects[editIndex] = cleanProj; }
    try {
      await SaveConfig(newCfg); setConfig(newCfg); setIsModalOpen(false);
      await ReloadCaddy(); await updateStatus();
      if (isNameChanged && !cleanProj.root_path) { (window as any).customAlert(t("偵測到您更改了專案名稱，且此專案使用的是預設路徑。請記得將 www/ 目錄下對應的資料夾名稱也改為新的名稱，以避免網站無法訪問喔！")); }
    } catch (err) { (window as any).customAlert(`${t("保存專案設定失敗")}: ${err}`); }
  };

  const handleDeleteProject = async (idx: number) => {
    if (!await (window as any).customConfirm(t("確定要刪除此專案嗎？這只會從 WinCMP 面板移除，不會刪除硬碟上的專案代碼喔！"))) return;
    const newCfg = { ...config }; newCfg.projects.splice(idx, 1);
    try { await SaveConfig(newCfg); setConfig(newCfg); await ReloadCaddy(); await updateStatus(); }
    catch (err) { (window as any).customAlert(`${t("刪除專案失敗")}: ${err}`); }
  };

  // ─── Styles ─────────────────────────────────────────────
  const thStyle: React.CSSProperties = {
    padding: '10px 16px', fontWeight: 700, fontSize: 10,
    letterSpacing: '0.05em', textTransform: 'uppercase',
    color: 'var(--muted)', background: 'var(--surface)',
    borderBottom: '1px solid var(--border)',
    position: 'sticky',
    top: 0,
    zIndex: 10,
  };

  const tdStyle: React.CSSProperties = {
    padding: '10px 16px', fontSize: 12,
    borderBottom: '1px solid var(--border-soft)',
  };

  const inputStyle: React.CSSProperties = {
    backgroundColor: 'var(--input-bg)', border: '1px solid var(--input-border)',
    color: 'var(--fg)', borderRadius: 'var(--radius-md)',
    padding: '8px 12px', outline: 'none', fontFamily: 'var(--font-mono)', fontSize: 12,
  };

  const labelStyle: React.CSSProperties = {
    display: 'block', fontSize: 10, fontWeight: 700,
    textTransform: 'uppercase', letterSpacing: '0.05em', color: 'var(--meta)',
  };

  return (
    <div className="flex flex-col h-full overflow-hidden select-none" style={{ background: 'var(--main-content-bg, var(--bg))' }}>
      {/* Header */}
      <div className="flex justify-between items-center px-4 py-2.5 shrink-0" style={{ borderBottom: '1px solid var(--border)', background: 'var(--bg-deep)' }}>
        <div className="flex items-baseline gap-2">
          <h1 className="text-xs font-bold" style={{ color: 'var(--fg)' }}>{t("專案管理")}</h1>
          <span className="text-[10px] hidden sm:inline" style={{ color: 'var(--meta)' }}> {t("管理與運行網頁專案，支援靜態、PHP 及 Node/Python/Go 自訂專案")}</span>
        </div>
        <button id="btn-add-project" onClick={() => handleOpenEditModal(null, null)} className="px-2.5 py-1 rounded-lg text-[11px] font-bold flex items-center gap-1 transition duration-200" style={{ background: 'var(--status-info)', color: '#fff' }}>
          <Plus size={12} /> {t("新增專案")}
        </button>
      </div>

      {/* Project List */}
      <div className="flex-1 overflow-auto" style={{ background: 'var(--main-content-bg, var(--bg))' }}>
        {config?.projects && config.projects.length > 0 ? (
          <table className="w-full text-left text-xs table-auto">
            <thead>
              <tr>
                <th style={thStyle}>{t("專案名稱")}</th>
                <th style={thStyle}>{t("類型 / 框架")}</th>
                <th style={thStyle}>{t("本機網域")}</th>
                <th style={thStyle}>{t("狀態")}</th>
                <th style={thStyle}>{t("啟用")}</th>
                <th style={{ ...thStyle, textAlign: 'center', position: 'relative' }}>
                  {t("操作")}
                  {showGuide && (
                    <div className="absolute right-0 top-10 z-50 animate-fade-in w-80 text-left p-4 rounded-xl border font-normal" style={{
                      background: 'var(--bg-deep)',
                      borderColor: 'var(--border)',
                      boxShadow: 'var(--shadow-lg)',
                      color: 'var(--fg)',
                      textTransform: 'none',
                      letterSpacing: 'normal',
                    }}>
                      {/* 氣泡小箭頭 */}
                      <div className="absolute -top-1.5 right-6 w-3 h-3 rotate-45 border-t border-l" style={{
                        background: 'var(--bg-deep)',
                        borderColor: 'var(--border)'
                      }} />

                      <div className="space-y-3">
                        <div className="font-bold text-xs flex items-center gap-1.5 pb-1.5" style={{ color: 'var(--status-info)', borderBottom: '1px solid var(--border-soft)' }}>
                          <span>💡 {t("操作按鈕快速指南")}</span>
                        </div>
                        <div className="space-y-2.5 text-[11px]" style={{ color: 'var(--fg-2)' }}>
                          <div className="flex items-start gap-2">
                            <span className="p-1 rounded text-white flex items-center shrink-0" style={{ background: 'var(--status-ok)' }}><Play size={10} /></span>
                            <div className="leading-tight">
                              <strong>{t("啟動專案 Runtime")}</strong>
                              <span className="block text-[10px]" style={{ color: 'var(--meta)' }}>{t("啟動專案的 Node/Python/Go 運行環境")}</span>
                            </div>
                          </div>
                          <div className="flex items-start gap-2">
                            <span className="p-1 rounded flex items-center shrink-0" style={{ background: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--status-info)' }}><Terminal size={10} /></span>
                            <div className="leading-tight">
                              <strong>{t("開啟專案終端")}</strong>
                              <span className="block text-[10px]" style={{ color: 'var(--meta)' }}>{t("進入專案的 CLI 交互終端偵錯")}</span>
                            </div>
                          </div>
                          <div className="flex items-start gap-2">
                            <span className="p-1 rounded flex items-center shrink-0" style={{ background: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--fg-2)' }}><FolderOpen size={10} /></span>
                            <div className="leading-tight">
                              <strong>{t("開啟專案資料夾")}</strong>
                              <span className="block text-[10px]" style={{ color: 'var(--meta)' }}>{t("開啟專案在硬碟上的物理根目錄")}</span>
                            </div>
                          </div>
                          <div className="flex items-start gap-2">
                            <span className="p-1 rounded flex items-center shrink-0" style={{ background: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--status-info)' }}><Edit size={10} /></span>
                            <div className="leading-tight">
                              <strong>{t("編輯專案設定")}</strong>
                              <span className="block text-[10px]" style={{ color: 'var(--meta)' }}>{t("調整網域、SSL 憑證、連接埠與啟動指令")}</span>
                            </div>
                          </div>
                          <div className="flex items-start gap-2">
                            <span className="p-1 rounded flex items-center shrink-0" style={{ background: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--status-error)' }}><Trash2 size={10} /></span>
                            <div className="leading-tight">
                              <strong>{t("刪除專案")}</strong>
                              <span className="block text-[10px]" style={{ color: 'var(--meta)' }}>{t("從面板移除（不會刪除硬碟檔案喔）")}</span>
                            </div>
                          </div>
                        </div>

                        <div className="flex justify-end pt-1">
                          <button onClick={(e) => { e.stopPropagation(); dismissGuide(); }} className="px-2.5 py-1 rounded text-[10px] font-bold text-white transition hover:opacity-90" style={{ background: 'var(--status-info)' }}>
                            {t("好的，我知道了")}
                          </button>
                        </div>
                      </div>
                    </div>
                  )}
                </th>
              </tr>
            </thead>
            <tbody>
              {config.projects.map((proj: Project, idx: number) => {
                const hasRuntime = isRuntimeProject(proj.type);
                const runtimeKey = `runtime_${proj.name}`;
                const running = hasRuntime && !!servicesStatus[runtimeKey];
                const loading = loadingProjects[proj.name];
                const isHighlighted = proj.name === highlightedRow;

                return (
                  <tr
                    key={idx}
                    id={`project-row-${proj.name}`}
                    className={isHighlighted ? 'animate-highlight' : ''}
                    style={{
                      opacity: proj.enabled ? 1 : 0.5,
                      background: isHighlighted ? 'var(--status-info-bg)' : 'var(--table-row-bg, transparent)',
                      transition: 'all 0.3s',
                    }}
                  >
                    <td style={tdStyle}>
                      <div className="space-y-0.5">
                        <div className="text-sm font-bold" style={{ color: 'var(--fg)' }}>{proj.name}</div>
                        <div className="text-[11px] max-w-[400px] truncate" style={{ color: 'var(--meta)', fontFamily: 'var(--font-mono)' }} title={proj.root_path}>{proj.root_path}</div>
                      </div>
                    </td>
                    <td style={tdStyle}>
                      <div className="flex flex-col gap-1 items-start">
                        <div className="flex items-center gap-1.5">
                          <span className="inline-block px-2 py-0.5 rounded text-[10px] font-bold" style={{ background: 'var(--status-info-bg)', color: 'var(--status-info)', border: '1px solid var(--status-info-bg)' }}>
                            {projectTypes.find(t => t.value === proj.type)?.label || proj.type}
                          </span>
                          {(proj.type === 'laravel' || proj.type === 'php') && proj.php_version && (
                            <span className="inline-block px-2 py-0.5 rounded text-[10px] font-bold" style={{ background: 'var(--status-ok-bg)', color: 'var(--status-ok)', border: '1px solid var(--status-ok-bg)' }}>
                              PHP {proj.php_version}
                            </span>
                          )}
                        </div>
                        {hasRuntime && (
                          <div className="text-[10px]" style={{ color: 'var(--meta)', fontFamily: 'var(--font-mono)' }}>
                            Port: {proj.runtime_port || 3000} | {proj.runtime_mode}
                          </div>
                        )}
                      </div>
                    </td>
                    <td style={tdStyle}>
                      <div className="flex flex-col gap-1.5">
                        {proj.domains.map((dom, dIdx) => (
                          <div key={dIdx} className="flex items-center gap-1.5 font-medium" style={{ color: 'var(--fg-2)' }}>
                            <Globe size={11} style={{ color: 'var(--meta)' }} />
                            <span className="hover:underline cursor-pointer" style={{ color: 'var(--fg-2)' }} onClick={() => handleCopyLink(dom, proj.use_ssl)}>{dom}</span>
                            {proj.use_ssl && <Shield size={11} style={{ color: 'var(--status-info)' }} />}
                          </div>
                        ))}
                      </div>
                    </td>
                    <td style={tdStyle}>
                      {hasRuntime ? (
                        running ? (
                          <span className="flex items-center gap-1.5 font-semibold text-xs" style={{ color: 'var(--status-ok)' }}>
                            <span className="relative flex h-2 w-2">
                              <span className="animate-ping absolute inline-flex h-full w-full rounded-full opacity-75" style={{ background: 'var(--status-ok)' }}></span>
                              <span className="relative inline-flex rounded-full h-2 w-2" style={{ background: 'var(--status-ok)' }}></span>
                            </span>
                            <span>{t("運行中")}</span>
                          </span>
                        ) : (
                          <span className="flex items-center gap-1.5 font-semibold text-xs" style={{ color: 'var(--meta)' }}>
                            <span className="relative inline-flex rounded-full h-2 w-2" style={{ background: 'var(--meta)' }}></span>
                            <span>{t("已停止")}</span>
                          </span>
                        )
                      ) : (
                        <span className="text-[10px] font-medium" style={{ color: 'var(--meta)' }}>{t("Caddy/PHP 靜態託管")}</span>
                      )}
                    </td>
                    <td style={tdStyle}>
                      <input type="checkbox" checked={proj.enabled} onChange={() => handleToggleEnable(idx)} className="w-3.5 h-3.5 cursor-pointer accent-blue-500" />
                    </td>
                    <td style={{ ...tdStyle, textAlign: 'center' }}>
                      <div className="flex gap-1.5 justify-center items-center">
                        {hasRuntime && proj.enabled && (
                          !running ? (
                            <button onClick={() => handleStartRuntime(proj.name)} disabled={loading} className="p-1.5 rounded-lg transition" style={{ background: 'var(--status-ok)', color: '#fff' }} title={t("啟動專案 Runtime")}>
                              <Play size={11} />
                            </button>
                          ) : (
                            <button onClick={() => handleStopRuntime(proj.name)} disabled={loading} className="p-1.5 rounded-lg transition" style={{ background: 'var(--status-error)', color: '#fff' }} title={t("停止專案 Runtime")}>
                              <Square size={11} />
                            </button>
                          )
                        )}
                        <button onClick={() => setTerminalProject(proj.name)} className="p-1.5 rounded-lg transition btn-open-terminal" style={{ background: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--status-info)' }} title={t("開啟專案終端")}>
                          <Terminal size={11} />
                        </button>
                        <button onClick={() => handleOpenFolder(proj.root_path)} className="p-1.5 rounded-lg transition" style={{ background: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--fg-2)' }} title={t("開啟專案資料夾")}>
                          <FolderOpen size={11} />
                        </button>
                        <button onClick={() => handleOpenEditModal(proj, idx)} className="p-1.5 rounded-lg transition" style={{ background: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--status-info)' }} title={t("編輯專案設定")}>
                          <Edit size={11} />
                        </button>
                        <button onClick={() => handleDeleteProject(idx)} className="p-1.5 rounded-lg transition" style={{ background: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--status-error)' }} title={t("刪除專案")}>
                          <Trash2 size={11} />
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        ) : (
          <div className="p-12 text-center space-y-3 select-none" style={{ color: 'var(--muted)' }}>
            <div className="text-xs">{t("目前尚未加入任何開發專案喔！")}</div>
            <button onClick={() => handleOpenEditModal(null, null)} className="px-3.5 py-1.5 rounded-lg text-xs font-semibold transition" style={{ background: 'var(--status-info)', color: '#fff' }}>
              {t("快速新增首個專案")}
            </button>
          </div>
        )}
      </div>

      {/* ─── Edit Drawer ───────────────────────────────────── */}
      {isModalOpen && editingProject && (
        <div className="fixed inset-0 z-50 overflow-hidden select-none">
          <div className="absolute inset-0 transition-opacity duration-300" style={{ background: 'var(--overlay-bg)', backdropFilter: 'blur(1px)' }} onClick={() => setIsModalOpen(false)} />
          <div className="absolute inset-y-0 right-0 pl-10 max-w-full flex">
            <div className="w-screen max-w-md flex flex-col h-full overflow-hidden animate-slide-in" style={{ background: 'var(--card)', borderLeft: '1px solid var(--border)', boxShadow: 'var(--shadow-lg)' }}>
              {/* Header */}
              <div className="px-6 py-5 flex justify-between items-center shrink-0" style={{ borderBottom: '1px solid var(--border)', background: 'var(--bg-deep)' }}>
                <div>
                  <h3 className="text-sm font-bold uppercase tracking-wider">
                    {editIndex === null ? t('新增開發專案') : t('編輯專案屬性')}
                  </h3>
                  {editIndex !== null && <p className="text-[11px] mt-0.5" style={{ color: 'var(--meta)', fontFamily: 'var(--font-mono)' }}>{editingProject.name}</p>}
                </div>
                <button onClick={() => setIsModalOpen(false)} className="transition" style={{ color: 'var(--muted)' }}><X size={16} /></button>
              </div>

              {/* Content */}
              <div className="flex-1 p-6 space-y-5 overflow-y-auto text-xs" style={{ color: 'var(--fg-2)' }}>
                {/* General */}
                <div className="space-y-4">
                  <h4 className="text-[11px] font-bold uppercase tracking-wider" style={{ color: 'var(--status-info)' }}>{t("基本設定 (General)")}</h4>
                  <div className="space-y-1.5">
                    <label style={labelStyle}>{t("專案物理根目錄")}</label>
                    <div className="flex gap-2">
                      <input type="text" value={editingProject.root_path} onChange={(e) => setEditingProject({ ...editingProject, root_path: e.target.value })}
                        onBlur={(e) => { if (editIndex === null && e.target.value.trim()) runAutoDetection(e.target.value.trim()); }}
                        onKeyDown={(e) => { if (e.key === 'Enter' && editIndex === null && (e.target as HTMLInputElement).value.trim()) runAutoDetection((e.target as HTMLInputElement).value.trim()); }}
                        placeholder={t("請選擇或填寫完整目錄路徑...")} className="flex-1" style={inputStyle} />
                      <button onClick={handleSelectRootPath} className="px-3 py-2 font-semibold transition" style={{ background: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--fg-2)', borderRadius: 'var(--radius-md)' }}>{t("選擇")}</button>
                    </div>
                  </div>
                  {(editIndex !== null || detected) && (
                    <div className="space-y-1.5 animate-fade-in">
                      <label style={labelStyle}>{t("專案名稱")}</label>
                      <input type="text" value={editingProject.name} onChange={(e) => setEditingProject({ ...editingProject, name: e.target.value })}
                        placeholder={t("例如: my-laravel-app")} className="w-full" style={inputStyle} />
                      <div className="text-[10px] italic mt-0.5" style={{ color: 'var(--meta)' }}>* {t("名稱不可重複，僅限英數字、連字號(-)與底線(_)")}</div>
                    </div>
                  )}
                </div>

                {isDetecting && (
                  <div className="p-4 rounded-xl flex items-center justify-center gap-3" style={{ background: 'var(--input-bg)', border: '1px solid var(--input-border)' }}>
                    <div className="animate-spin rounded-full h-4 w-4 border-2 border-t-transparent" style={{ borderColor: 'var(--status-info)', borderTopColor: 'transparent' }}></div>
                    <span className="text-[11px] font-medium" style={{ color: 'var(--muted)' }}>{t("🔄 正在偵測專案結構與配置，請稍候...")}</span>
                  </div>
                )}

                {editIndex === null && !detected && !isDetecting && (
                  <div className="p-4 border-dashed rounded-xl text-center py-8 select-none" style={{ borderColor: 'var(--border)', background: 'var(--surface)', color: 'var(--muted)' }}>
                    <span className="text-[11px]">{t("💡 請選擇專案根目錄，系統將自動偵測並帶入配置。")}</span>
                  </div>
                )}

                {(editIndex !== null || detected) && (
                  <div className="space-y-5 animate-fade-in">
                    {/* Runtime */}
                    <div className="space-y-4 pt-4" style={{ borderTop: '1px solid var(--border-soft)' }}>
                      <h4 className="text-[11px] font-bold uppercase tracking-wider" style={{ color: 'var(--accent)' }}>{t("執行環境 (Runtime)")}</h4>
                      <div className="grid grid-cols-2 gap-4">
                        <div className="space-y-1.5">
                          <label style={labelStyle}>{t("專案框架 / 類型")}</label>
                          <select value={editingProject.type} onChange={(e) => handleTypeChange(e.target.value)} className="w-full cursor-pointer font-semibold" style={inputStyle}>
                            {projectTypes.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
                          </select>
                        </div>
                        <div className="space-y-1.5">
                          <label style={labelStyle}>{t("執行器 (Runtime)")}</label>
                          <select value={editingProject.runtime_type} onChange={(e) => { const newRt = e.target.value; setEditingProject({ ...editingProject, runtime_type: newRt, use_wincmp_bin: hasBundledRuntime(newRt) }); }} className="w-full cursor-pointer font-semibold" style={inputStyle}>
                            {runtimeTypes.map(t => <option key={t.value} value={t.value}>{t.label}</option>)}
                          </select>
                        </div>
                      </div>

                      {(editingProject.type === 'laravel' || editingProject.type === 'php') && (
                        <div className="space-y-1.5">
                          <label style={labelStyle}>{t("PHP 執行版本")}</label>
                          <select value={editingProject.php_version} onChange={(e) => setEditingProject({ ...editingProject, php_version: e.target.value })} className="w-full cursor-pointer font-semibold" style={inputStyle}>
                            <option value="">{t("請選擇對應 PHP 版本...")}</option>
                            {scanResult?.PHPList?.map((p: any) => <option key={p.MajorMin} value={p.MajorMin}>PHP {p.MajorMin} ({t("偵測到")} {p.Version})</option>)}
                          </select>
                        </div>
                      )}

                      {isRuntimeProject(editingProject.type) && (
                        <div className="rounded-xl p-4 space-y-4" style={{ border: '1px solid var(--border)', background: 'var(--surface)' }}>
                          <div className="font-semibold text-xs flex items-center gap-1.5 pb-2" style={{ color: 'var(--fg)', borderBottom: '1px solid var(--border)' }}>
                            <Settings size={12} style={{ color: 'var(--status-info)' }} /> {t("Runtime 運行細節設定")}
                          </div>
                          <div className="grid grid-cols-2 gap-3">
                            <div className="space-y-1">
                              <label style={labelStyle}>{t("執行 Port")}</label>
                              <input type="number" value={editingProject.runtime_port || 3000} onChange={(e) => setEditingProject({ ...editingProject, runtime_port: parseInt(e.target.value) })} className="w-full" style={inputStyle} />
                            </div>
                            <div className="space-y-1">
                              <label style={labelStyle}>{t("運行模式")}</label>
                              <select value={editingProject.runtime_mode || 'Background'} onChange={(e) => setEditingProject({ ...editingProject, runtime_mode: e.target.value })} className="w-full cursor-pointer font-semibold" style={inputStyle}>
                                <option value="Background">{t("背景執行 (Background)")}</option>
                                <option value="Terminal">{t("終端執行 (Terminal)")}</option>
                              </select>
                            </div>
                          </div>
                          {['node', 'bun', 'auto'].includes(editingProject.runtime_type || '') && (
                            <div className="space-y-3 pt-2">
                              <div className="flex items-center gap-2">
                                <input type="checkbox" id="useWinCMPBin" checked={editingProject.use_wincmp_bin} onChange={(e) => setEditingProject({ ...editingProject, use_wincmp_bin: e.target.checked })} className="w-3.5 h-3.5 cursor-pointer accent-blue-500" />
                                <label htmlFor="useWinCMPBin" className="text-[11px] cursor-pointer font-medium" style={{ color: 'var(--fg-2)' }}>{t("使用 WinCMP 內建執行檔 (Bundled Runtime)")}</label>
                              </div>
                              {editingProject.use_wincmp_bin && (
                                <div className="space-y-1">
                                  <label style={labelStyle}>{t("選擇內建版本")}</label>
                                  <select value={editingProject.runtime_version || ''} onChange={(e) => setEditingProject({ ...editingProject, runtime_version: e.target.value })} className="w-full text-[11px] cursor-pointer" style={inputStyle}>
                                    {(editingProject.runtime_type === 'bun' ? scanResult?.BunList : scanResult?.NodeList)?.map((r: any) => <option key={r.Version} value={r.Version}>{r.Version}</option>)}
                                    {!(editingProject.runtime_type === 'bun' ? scanResult?.BunList : scanResult?.NodeList)?.length && <option value="">{t("無可用版本 (請確認 ./bin/)")}</option>}
                                  </select>
                                </div>
                              )}
                            </div>
                          )}
                          <div className="space-y-1.5">
                            <label style={labelStyle}>{t("自訂啟動指令 (可選，空白將使用預設)")}</label>
                            <input type="text" value={editingProject.command || ''} onChange={(e) => setEditingProject({ ...editingProject, command: e.target.value })} placeholder={t("例如: npm run dev -- --port 3000")} className="w-full" style={inputStyle} />
                          </div>
                        </div>
                      )}
                    </div>

                    {/* Domains */}
                    <div className="space-y-4 pt-4" style={{ borderTop: '1px solid var(--border-soft)' }}>
                      <h4 className="text-[11px] font-bold uppercase tracking-wider" style={{ color: 'var(--status-ok)' }}>{t("網域別名 (Domains)")}</h4>
                      <div className="space-y-2">
                        {editingProject.domains.map((dom, dIdx) => (
                          <div key={dIdx} className="flex gap-2">
                            <input type="text" value={dom} onChange={(e) => handleDomainChange(dIdx, e.target.value)} placeholder={t("例如: my-site.test")} className="flex-1" style={inputStyle} />
                            {editingProject.domains.length > 1 && (
                              <button onClick={() => handleRemoveDomain(dIdx)} className="px-3 py-2 font-semibold transition" style={{ background: 'var(--status-error-bg)', color: 'var(--status-error)', border: '1px solid var(--status-error-bg)', borderRadius: 'var(--radius-md)' }}>{t("移除")}</button>
                            )}
                          </div>
                        ))}
                        <button onClick={handleAddDomain} className="text-[11px] font-semibold flex items-center gap-1 transition" style={{ color: 'var(--status-info)' }}>{t("+ 新增別名網域")}</button>
                        <div className="text-[10px] italic mt-0.5" style={{ color: 'var(--meta)' }}>* {t("不可包含底線(_)、埠號(:)或路徑(/)，僅限英數字、連字號(-)與點(.)")}</div>
                      </div>
                    </div>

                    {/* SSL */}
                    <div className="flex items-center justify-between p-4 rounded-xl" style={{ border: '1px solid var(--border)', background: 'var(--surface)' }}>
                      <div className="flex items-center gap-3">
                        <Shield size={16} style={{ color: 'var(--status-info)' }} />
                        <div>
                          <div className="font-semibold text-[11px]" style={{ color: 'var(--fg)' }}>{t("啟用 HTTPS 安全憑證")}</div>
                          <div className="text-[10px] mt-0.5" style={{ color: 'var(--meta)' }}>{t("自動套用 Caddy 內部自簽憑證")}</div>
                        </div>
                      </div>
                      <input type="checkbox" checked={editingProject.use_ssl} onChange={(e) => setEditingProject({ ...editingProject, use_ssl: e.target.checked })} className="w-3.5 h-3.5 cursor-pointer accent-blue-500" />
                    </div>

                    {/* Caddyfile */}
                    <div className="p-4 rounded-xl space-y-3" style={{ border: '1px solid var(--border)', background: 'var(--surface)' }}>
                      <div className="flex items-center gap-3">
                        <Settings size={16} style={{ color: 'var(--status-info)' }} />
                        <div>
                          <div className="font-semibold text-[11px]" style={{ color: 'var(--fg)' }}>{t("Caddy 配置文件路徑")}</div>
                          <div className="text-[10px] mt-0.5" style={{ color: 'var(--meta)' }}>{t("編輯專案 Caddyfile 設定")}</div>
                        </div>
                      </div>
                      <div className="flex gap-2 items-center">
                        <input type="text" readOnly value={editingProject.name ? `conf\\sites\\${editingProject.name}.caddy` : ''} className="flex-1 text-[11px]" style={{ ...inputStyle, color: 'var(--meta)' }} />
                        <button type="button" disabled={editIndex === null} onClick={async () => { try { await OpenProjectCaddyfile(editingProject.name); } catch (err) { (window as any).customAlert(`${t("無法開啟設定檔")}: ${err}`); } }}
                          className="px-3.5 py-2 transition font-semibold text-[11px] whitespace-nowrap" style={{ background: 'var(--status-info)', color: '#fff', borderRadius: 'var(--radius-md)', opacity: editIndex === null ? 0.4 : 1 }}>
                          {t("開啟檔案")}
                        </button>
                      </div>
                      {editIndex === null && <div className="text-[10px] italic" style={{ color: 'var(--meta)' }}>* {t("儲存專案後將自動建立 Caddyfile")}</div>}
                    </div>
                  </div>
                )}
              </div>

              {/* Footer */}
              <div className="px-6 py-4 flex justify-end gap-3 shrink-0" style={{ borderTop: '1px solid var(--border)', background: 'var(--bg-deep)' }}>
                <button id="btn-cancel-add" onClick={() => setIsModalOpen(false)} className="px-4 py-2 rounded-lg text-xs font-semibold transition" style={{ border: '1px solid var(--border)', color: 'var(--fg-2)' }}>{t("取消")}</button>
                {(editIndex !== null || detected) && (
                  <button onClick={handleSaveProject} className="px-4 py-2 rounded-lg text-xs font-semibold transition" style={{ background: 'var(--status-info)', color: '#fff' }}>{t("儲存設定")}</button>
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      <ProjectTerminal projectName={terminalProject || ''} isOpen={terminalProject !== null} onClose={() => setTerminalProject(null)} />
    </div>
  );

  function handleDomainChange(idx: number, val: string) {
    if (!editingProject) return;
    const newDoms = [...editingProject.domains]; newDoms[idx] = val;
    setEditingProject({ ...editingProject, domains: newDoms });
  }
  function handleAddDomain() { if (!editingProject) return; setEditingProject({ ...editingProject, domains: [...editingProject.domains, ''] }); }
  function handleRemoveDomain(idx: number) { if (!editingProject) return; const newDoms = [...editingProject.domains]; newDoms.splice(idx, 1); setEditingProject({ ...editingProject, domains: newDoms }); }
  async function handleRootPathSelect() { await handleSelectRootPath(); }
  function handleTypeChange(type: string) {
    if (!editingProject) return;
    let rt = 'none'; let port = 0;
    if (['next', 'nuxt', 'astro', 'vite'].includes(type)) { rt = 'auto'; port = 3000; }
    else if (type.startsWith('python')) { rt = 'python'; port = 8000; }
    else if (type === 'go_api') { rt = 'go_air'; port = 8080; }
    else if (type === 'pocketbase') { rt = 'go_run'; port = 8090; }
    else if (type === 'custom') { rt = 'custom'; port = 3000; }
    const hasBin = hasBundledRuntime(rt);
    setEditingProject({ ...editingProject, type, runtime_type: rt, runtime_port: port, runtime_version: rt === 'bun' ? scanResult?.BunList?.[0]?.Version : scanResult?.NodeList?.[0]?.Version, use_wincmp_bin: shouldDefaultUseWinCMPBin(rt) });
  }
}

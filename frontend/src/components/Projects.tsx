import React, { useState, useEffect } from 'react';
import { Play, Square, Plus, Edit, FolderOpen, Link, Check, X, Shield, Settings, Trash2, Copy, Globe, Terminal } from 'lucide-react';
import ProjectTerminal from './ProjectTerminal';
import {
  GetConfig,
  SaveConfig,
  GetScanResult,
  GetServicesStatus,
  StartProjectRuntime,
  StopProjectRuntime,
  ReloadCaddy,
  OpenFolder,
  SelectFolder,
  DetectProjectPath,
  OpenProjectCaddyfile
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
  useLanguage(); // 訂閱語系變更
  const [config, setConfig] = useState<any>(null);
  const [scanResult, setScanResult] = useState<any>(null);
  const [servicesStatus, setServicesStatus] = useState<Record<string, boolean>>({});
  const [loadingProjects, setLoadingProjects] = useState<Record<string, boolean>>({});
  const [editingProject, setEditingProject] = useState<Project | null>(null);
  const [editIndex, setEditIndex] = useState<number | null>(null); // null 代表新增
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isDetecting, setIsDetecting] = useState(false);
  const [detected, setDetected] = useState(false);
  const [terminalProject, setTerminalProject] = useState<string | null>(null);

  // 用於記錄當前高亮閃爍的專案行
  const [highlightedRow, setHighlightedRow] = useState<string | null>(null);

  // 監聽來自 Topbar 的搜尋跳轉高亮滾動需求
  useEffect(() => {
    if (highlightedProjectName && config?.projects) {
      // 確保 DOM 渲染完畢
      setTimeout(() => {
        const element = document.getElementById(`project-row-${highlightedProjectName}`);
        if (element) {
          element.scrollIntoView({ behavior: 'smooth', block: 'center' });
          setHighlightedRow(highlightedProjectName);
          const timer = setTimeout(() => {
            setHighlightedRow(null);
            if (clearHighlight) clearHighlight();
          }, 2000);
        } else {
          if (clearHighlight) clearHighlight();
        }
      }, 100);
    }
  }, [highlightedProjectName, config]);

  // 初始化專案類型與 Runtime 類型對照表
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
    if (rt === 'node') {
      return !!(scanResult?.NodeList && scanResult.NodeList.length > 0);
    }
    if (rt === 'bun') {
      return !!(scanResult?.BunList && scanResult.BunList.length > 0);
    }
    if (rt === 'auto') {
      const hasNode = !!(scanResult?.NodeList && scanResult.NodeList.length > 0);
      const hasBun = !!(scanResult?.BunList && scanResult.BunList.length > 0);
      return hasNode || hasBun;
    }
    return false;
  };

  useEffect(() => {
    async function initData() {
      try {
        const cfg = await GetConfig();
        setConfig(cfg);
        const scan = await GetScanResult();
        setScanResult(scan);
        await updateStatus();
      } catch (err) {
        console.error("載入專案資料失敗:", err);
      }
    }
    initData();
  }, []);

  // 定時輪詢專案 Runtime 狀態
  useEffect(() => {
    const timer = setInterval(() => {
      updateStatus();
    }, 2000);
    return () => clearInterval(timer);
  }, []);

  const updateStatus = async () => {
    try {
      const status = await GetServicesStatus();
      setServicesStatus(status);
    } catch (err) {
      console.error("更新狀態失敗:", err);
    }
  };

  const isRuntimeProject = (type?: string) => {
    if (!type) return false;
    return !['static', 'laravel', 'php'].includes(type);
  };

  const handleToggleEnable = async (idx: number) => {
    if (!config) return;
    const newCfg = { ...config };
    newCfg.projects[idx].enabled = !newCfg.projects[idx].enabled;

    try {
      await SaveConfig(newCfg);
      setConfig(newCfg);
      // 一切啟用/禁用後自動重載 Caddy 設定
      await ReloadCaddy();
      await updateStatus();
    } catch (err) {
      (window as any).customAlert(`${t("儲存設定失敗")}: ${err}`);
    }
  };

  const handleStartRuntime = async (name: string) => {
    setLoadingProjects(prev => ({ ...prev, [name]: true }));
    try {
      await StartProjectRuntime(name);
      await updateStatus();
    } catch (err) {
      (window as any).customAlert(`${t("啟動 Runtime 失敗")}: ${err}`);
    } finally {
      setLoadingProjects(prev => ({ ...prev, [name]: false }));
    }
  };

  const handleStopRuntime = async (name: string) => {
    setLoadingProjects(prev => ({ ...prev, [name]: true }));
    try {
      await StopProjectRuntime(name);
      await updateStatus();
    } catch (err) {
      (window as any).customAlert(`${t("停止 Runtime 失敗")}: ${err}`);
    } finally {
      setLoadingProjects(prev => ({ ...prev, [name]: false }));
    }
  };

  const handleOpenFolder = async (path: string) => {
    try {
      await OpenFolder(path);
    } catch (err) {
      (window as any).customAlert(`${t("無法開啟目錄")}: ${err}`);
    }
  };

  const handleCopyLink = (domain: string, useSSL: boolean) => {
    const link = `${useSSL ? 'https' : 'http'}://${domain}`;
    navigator.clipboard.writeText(link);
    (window as any).customAlert(`${t("已複製連結")}: ${link}`);
  };

  const handleOpenEditModal = (proj: Project | null, idx: number | null) => {
    if (proj) {
      const hasBin = hasBundledRuntime(proj.runtime_type);
      setEditingProject({
        ...proj,
        use_wincmp_bin: hasBin ? proj.use_wincmp_bin : false
      });
      setDetected(true);
    } else {
      // 預設全新專案結構
      setEditingProject({
        name: '',
        domains: [''],
        type: 'static',
        runtime_type: 'none',
        php_version: scanResult?.PHPList?.[0]?.MajorMin || '',
        root_path: '',
        ssl_crt: '',
        ssl_key: '',
        use_ssl: true,
        enabled: true,
        runtime_port: 3000,
        runtime_mode: 'Background',
        runtime_version: scanResult?.NodeList?.[0]?.Version || '',
        command: '',
        use_wincmp_bin: false
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
          ...editingProject,
          root_path: path,
          name: res.name,
          domains: res.domains && res.domains.length > 0 ? res.domains : [`local-${res.name.toLowerCase().replace(/_/g, '-')}.test`],
          type: res.type || 'static',
          runtime_type: res.runtime_type || 'none',
          runtime_port: res.runtime_port || 3000,
          php_version: res.php_version || scanResult?.PHPList?.[0]?.MajorMin || '',
          runtime_version: res.runtime_type === 'bun' ? scanResult?.BunList?.[0]?.Version : scanResult?.NodeList?.[0]?.Version || '',
          use_wincmp_bin: hasBin
        });
        setDetected(true);
      }
    } catch (err) {
      console.error("自動偵測專案失敗:", err);
      (window as any).customAlert(`${t("自動偵測專案失敗")}: ${err}`);
    } finally {
      setIsDetecting(false);
    }
  };

  const handleSelectRootPath = async () => {
    if (!editingProject) return;
    try {
      const path = await SelectFolder();
      if (path) {
        setEditingProject(prev => prev ? { ...prev, root_path: path } : null);
        if (editIndex === null) {
          await runAutoDetection(path);
        }
      }
    } catch (err) {
      console.error("選擇目錄失敗:", err);
    }
  };

  const handleSaveProject = async () => {
    if (!editingProject || !config) return;
    
    const trimName = editingProject.name.trim();
    if (!trimName) {
      (window as any).customAlert(t("專案名稱不能為空"));
      return;
    }

    // 專案名稱格式驗證 (只允許英數字、連字號與底線)
    const nameRegex = /^[a-zA-Z0-9_-]+$/;
    if (!nameRegex.test(trimName)) {
      (window as any).customAlert(t("專案名稱僅能包含英數字、連字號(-)與底線(_)喔！"));
      return;
    }

    if (editingProject.use_wincmp_bin && !hasBundledRuntime(editingProject.runtime_type)) {
      (window as any).customAlert(t("儲存失敗：您勾選了使用 WinCMP 內建執行檔，但系統未在 ./bin/ 下偵測到可用的 Node.js 或 Bun 執行檔。請先下載並放置於對應目錄，或取消勾選此選項以使用系統全域執行檔。"));
      return;
    }

    const newCfg = { ...config };
    const cleanProj = { ...editingProject };
    cleanProj.name = trimName;
    if (cleanProj.type !== 'laravel' && cleanProj.type !== 'php') {
      cleanProj.php_version = '';
    }

    // 清理 domains 空白
    cleanProj.domains = cleanProj.domains.filter(d => d.trim() !== "");
    if (cleanProj.domains.length === 0) {
      cleanProj.domains = [`local-${cleanProj.name.toLowerCase().replace(/_/g, '-')}.test`];
    }

    // 網域格式驗證 (比照後端 hosts.IsValidDomain 規則)
    const domainRegex = /^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*$/;
    for (const d of cleanProj.domains) {
      if (!domainRegex.test(d)) {
        (window as any).customAlert(t("網域 '%s' 格式不正確喔！請輸入正確的網域格式（例如 my-site.test），且不能包含底線、埠號或路徑。", d));
        return;
      }
    }

    // 重名檢測
    const duplicate = newCfg.projects?.find((p: any, idx: number) => 
      idx !== editIndex && p.name.trim().toLowerCase() === cleanProj.name.trim().toLowerCase()
    );
    if (duplicate) {
      (window as any).customAlert(t("專案名稱已存在，請使用其他名稱喔！"));
      return;
    }

    let oldName = '';
    let isNameChanged = false;
    if (editIndex !== null) {
      oldName = config.projects[editIndex].name;
      isNameChanged = oldName !== cleanProj.name;
    }

    // 專案改名時，若舊進程正在運行中，先自動停止
    if (isNameChanged && oldName) {
      const oldRuntimeKey = `runtime_${oldName}`;
      const isOldRunning = !!servicesStatus[oldRuntimeKey];
      if (isOldRunning) {
        try {
          await StopProjectRuntime(oldName);
        } catch (stopErr) {
          console.error("自動停止舊專案 Runtime 失敗:", stopErr);
        }
      }
    }

    if (editIndex === null) {
      // 新增
      newCfg.projects = [...(newCfg.projects || []), cleanProj];
    } else {
      // 修改
      newCfg.projects[editIndex] = cleanProj;
    }

    try {
      await SaveConfig(newCfg);
      setConfig(newCfg);
      setIsModalOpen(false);
      await ReloadCaddy(); // 自動重載 Caddyfile 與 Hosts 檔
      await updateStatus();

      // 改名後的預設路徑貼心提示
      if (isNameChanged && !cleanProj.root_path) {
        (window as any).customAlert(t("偵測到您更改了專案名稱，且此專案使用的是預設路徑。請記得將 www/ 目錄下對應的資料夾名稱也改為新的名稱，以避免網站無法訪問喔！"));
      }
    } catch (err) {
      (window as any).customAlert(`${t("保存專案設定失敗")}: ${err}`);
    }
  };

  const handleDeleteProject = async (idx: number) => {
    if (!await (window as any).customConfirm(t("確定要刪除此專案嗎？這只會從 WinCMP 面板移除，不會刪除硬碟上的專案代碼喔！"))) {
      return;
    }

    const newCfg = { ...config };
    newCfg.projects.splice(idx, 1);

    try {
      await SaveConfig(newCfg);
      setConfig(newCfg);
      await ReloadCaddy();
      await updateStatus();
    } catch (err) {
      (window as any).customAlert(`${t("刪除專案失敗")}: ${err}`);
    }
  };

  return (
    <div className="flex flex-col h-full bg-[#08080a] overflow-hidden select-none">
      {/* 標頭 */}
      <div className="flex justify-between items-center select-none px-4 py-2.5 border-b border-darkBorder bg-[#0b0b0e] shrink-0">
        <div className="flex items-center gap-2">
          <h1 className="text-xs font-bold text-gray-200">📁 {t("專案管理面板")}</h1>
          <span className="text-[10px] text-gray-500 hidden sm:inline">| {t("管理與運行網頁專案，支援靜態、PHP 及 Node/Python/Go 自訂專案")}</span>
        </div>
        <button
          onClick={() => handleOpenEditModal(null, null)}
          className="px-2.5 py-1 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-[11px] font-bold flex items-center gap-1 transition duration-200"
        >
          <Plus size={12} /> {t("新增專案")}
        </button>
      </div>

      {/* 專案列表 */}
      <div className="flex-1 overflow-auto bg-[#060608]">
        {config?.projects && config.projects.length > 0 ? (
          <table className="w-full text-left text-xs table-auto">
            <thead className="bg-[#0f0f12] text-gray-400 uppercase text-[10px] tracking-wider border-b border-darkBorder select-none sticky top-0 z-10">
              <tr>
                <th className="px-4 py-2.5 font-bold">{t("專案名稱")}</th>
                <th className="px-4 py-2.5 font-bold">{t("類型 / 框架")}</th>
                <th className="px-4 py-2.5 font-bold">{t("本機網域")}</th>
                <th className="px-4 py-2.5 font-bold">{t("可用性")}</th>
                <th className="px-4 py-2.5 font-bold">{t("可用性")}</th>
                <th className="px-4 py-2.5 text-center font-bold">{t("操作")}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-darkBorder/40">
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
                    className={`hover:bg-white/[0.015] transition-all duration-300 ${proj.enabled ? '' : 'opacity-50'} ${isHighlighted ? 'animate-highlight bg-blue-600/10' : ''}`}
                  >
                    <td className="px-4 py-2.5">
                      <div className="space-y-0.5">
                        <div className="text-sm font-bold text-gray-100">{proj.name}</div>
                        <div className="text-[11px] text-gray-500 max-w-[400px] truncate font-mono" title={proj.root_path}>
                          {proj.root_path}
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-2.5">
                      <div className="flex flex-col gap-1 items-start">
                        <div className="flex items-center gap-1.5">
                          <span className="inline-block px-2 py-0.5 rounded text-[10px] font-bold bg-blue-500/10 text-blue-400 border border-blue-500/10">
                            {projectTypes.find(t => t.value === proj.type)?.label || proj.type}
                          </span>
                          {(proj.type === 'laravel' || proj.type === 'php') && proj.php_version && (
                            <span className="inline-block px-2 py-0.5 rounded text-[10px] font-bold bg-emerald-500/10 text-emerald-400 border border-emerald-500/10">
                              PHP {proj.php_version}
                            </span>
                          )}
                        </div>
                        {hasRuntime && (
                          <div className="text-[10px] text-gray-500 font-mono">
                            Port: {proj.runtime_port || 3000} | {proj.runtime_mode}
                          </div>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-2.5">
                      <div className="flex flex-col gap-1.5">
                        {proj.domains.map((dom, dIdx) => (
                          <div key={dIdx} className="flex items-center gap-1.5 text-gray-300 font-medium">
                            <Globe size={11} className="text-gray-500" />
                            <span className="hover:underline hover:text-blue-400 cursor-pointer" onClick={() => handleCopyLink(dom, proj.use_ssl)}>
                              {dom}
                            </span>
                            {proj.use_ssl && <Shield size={11} className="text-blue-500" />}
                          </div>
                        ))}
                      </div>
                    </td>
                    <td className="px-4 py-2.5 select-none">
                      {hasRuntime ? (
                        running ? (
                          <span className="flex items-center gap-1.5 text-green-400 font-semibold text-xs">
                            <span className="relative flex h-2 w-2">
                              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                              <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>
                            </span>
                            <span>{t("運行中")}</span>
                          </span>
                        ) : (
                          <span className="flex items-center gap-1.5 text-gray-500 font-semibold text-xs">
                            <span className="relative inline-flex rounded-full h-2 w-2 bg-gray-600"></span>
                            <span>{t("已停止")}</span>
                          </span>
                        )
                      ) : (
                        <span className="text-[10px] text-gray-500 font-medium">{t("Caddy/PHP 靜態託管")}</span>
                      )}
                    </td>
                    <td className="px-4 py-2.5 select-none">
                      <input
                        type="checkbox"
                        checked={proj.enabled}
                        onChange={() => handleToggleEnable(idx)}
                        className="w-3.5 h-3.5 bg-darkInput border-darkBorder rounded text-blue-500 accent-blue-500 cursor-pointer"
                      />
                    </td>
                    <td className="px-4 py-2.5">
                      <div className="flex gap-1.5 justify-center items-center select-none">
                        {/* Runtime 啟停控制 */}
                        {hasRuntime && proj.enabled && (
                          <>
                            {!running ? (
                              <button
                                onClick={() => handleStartRuntime(proj.name)}
                                disabled={loading}
                                className="p-1.5 bg-green-600/90 hover:bg-green-600 text-white rounded-lg transition"
                                title={t("啟動專案 Runtime")}
                              >
                                <Play size={11} />
                              </button>
                            ) : (
                              <button
                                onClick={() => handleStopRuntime(proj.name)}
                                disabled={loading}
                                className="p-1.5 bg-red-600/90 hover:bg-red-700 text-white rounded-lg transition"
                                title={t("停止專案 Runtime")}
                              >
                                <Square size={11} />
                              </button>
                            )}
                          </>
                        )}
                        {/* 常規按鈕 */}
                        <button
                          onClick={() => setTerminalProject(proj.name)}
                          className="p-1.5 bg-darkInput border border-darkBorder hover:border-blue-500 rounded-lg text-blue-400 transition"
                          title={t("開啟專案終端")}
                        >
                          <Terminal size={11} />
                        </button>
                        <button
                          onClick={() => handleOpenFolder(proj.root_path)}
                          className="p-1.5 bg-darkInput border border-darkBorder hover:border-gray-500 rounded-lg text-gray-300 transition"
                          title={t("開啟專案資料夾")}
                        >
                          <FolderOpen size={11} />
                        </button>
                        <button
                          onClick={() => handleOpenEditModal(proj, idx)}
                          className="p-1.5 bg-darkInput border border-darkBorder hover:border-blue-500 rounded-lg text-blue-400 transition"
                          title={t("編輯專案設定")}
                        >
                          <Edit size={11} />
                        </button>
                        <button
                          onClick={() => handleDeleteProject(idx)}
                          className="p-1.5 bg-darkInput border border-darkBorder hover:border-red-500 rounded-lg text-red-500 transition"
                          title={t("刪除專案")}
                        >
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
          <div className="p-12 text-center text-gray-500 space-y-3 select-none">
            <div className="text-xs">{t("目前尚未加入任何開發專案喔！")}</div>
            <button
              onClick={() => handleOpenEditModal(null, null)}
              className="px-3.5 py-1.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-xs font-semibold transition"
            >
              {t("快速新增首個專案")}
            </button>
          </div>
        )}
      </div>

      {/* 右側滑出式設定 Drawer */}
      {isModalOpen && editingProject && (
        <div className="fixed inset-0 z-50 overflow-hidden select-none">
          {/* 半透明遮罩背景 */}
          <div
            className="absolute inset-0 bg-black/45 backdrop-blur-[1px] transition-opacity duration-300"
            onClick={() => setIsModalOpen(false)}
          />

          <div className="absolute inset-y-0 right-0 pl-10 max-w-full flex">
            {/* Drawer 容器 */}
            <div className="w-screen max-w-md bg-darkCard border-l border-darkBorder shadow-2xl flex flex-col h-full overflow-hidden animate-slide-in">

              {/* Header */}
              <div className="px-6 py-5 border-b border-darkBorder flex justify-between items-center bg-[#0d0d10]">
                <div>
                  <h3 className="text-sm font-bold uppercase tracking-wider text-gray-400">
                    {editIndex === null ? t('✨ 新增開發專案') : t('⚙️ 編輯專案屬性')}
                  </h3>
                  {editIndex !== null && <p className="text-[11px] text-gray-500 font-mono mt-0.5">{editingProject.name}</p>}
                </div>
                <button onClick={() => setIsModalOpen(false)} className="text-gray-400 hover:text-white transition">
                  <X size={16} />
                </button>
              </div>

              {/* Drawer Content */}
              <div className="flex-1 p-6 space-y-5 overflow-y-auto text-xs text-gray-300">
                {/* 1. 基本設定 */}
                <div className="space-y-4">
                  <h4 className="text-[11px] font-bold text-blue-400 uppercase tracking-wider select-none">{t("基本設定 (General)")}</h4>

                  {/* 專案目錄 */}
                  <div className="space-y-1.5">
                    <label className="block text-[10px] font-bold uppercase tracking-wider text-gray-500">{t("專案物理根目錄")}</label>
                    <div className="flex gap-2">
                      <input
                        type="text"
                        value={editingProject.root_path}
                        onChange={(e) => setEditingProject({ ...editingProject, root_path: e.target.value })}
                        onBlur={(e) => {
                          if (editIndex === null && e.target.value.trim()) {
                            runAutoDetection(e.target.value.trim());
                          }
                        }}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' && editIndex === null && (e.target as HTMLInputElement).value.trim()) {
                            runAutoDetection((e.target as HTMLInputElement).value.trim());
                          }
                        }}
                        placeholder={t("請選擇或填寫完整目錄路徑...")}
                        className="flex-1 bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-3 py-2 outline-none focus:border-blue-500 transition font-mono"
                      />
                      <button
                        onClick={handleRootPathSelect}
                        className="px-3 py-2 bg-darkInput border border-darkBorder hover:border-gray-500 rounded-lg transition font-semibold"
                      >
                        {t("選擇")}
                      </button>
                    </div>
                  </div>

                  {/* 專案名稱 */}
                  {(editIndex !== null || detected) && (
                    <div className="space-y-1.5 animate-fade-in">
                      <label className="block text-[10px] font-bold uppercase tracking-wider text-gray-500">{t("專案名稱")}</label>
                      <input
                        type="text"
                        value={editingProject.name}
                        onChange={(e) => setEditingProject({ ...editingProject, name: e.target.value })}
                        placeholder={t("例如: my-laravel-app")}
                        className="w-full bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-3 py-2 outline-none focus:border-blue-500 disabled:opacity-50 transition"
                      />
                      <div className="text-[10px] text-gray-500 italic mt-0.5">
                        * {t("名稱不可重複，僅限英數字、連字號(-)與底線(_)")}
                      </div>
                    </div>
                  )}
                </div>

                {/* 偵測中 Loading 骨架屏 / 動畫 */}
                {isDetecting && (
                  <div className="p-4 bg-darkInput border border-darkBorder rounded-xl flex items-center justify-center gap-3">
                    <div className="animate-spin rounded-full h-4 w-4 border-2 border-blue-500 border-t-transparent"></div>
                    <span className="text-[11px] text-gray-400 font-medium">{t("🔄 正在偵測專案結構與配置，請稍候...")}</span>
                  </div>
                )}

                {/* 新增專案時，若尚未偵測，顯示引導文字 */}
                {editIndex === null && !detected && !isDetecting && (
                  <div className="p-4 border border-dashed border-darkBorder rounded-xl bg-white/[0.01] text-center text-gray-500 py-8 select-none">
                    <span className="text-[11px]">{t("💡 請選擇專案根目錄，系統將自動偵測並帶入配置。")}</span>
                  </div>
                )}

                {/* 偵測成功或編輯模式下才顯示的欄位 */}
                {(editIndex !== null || detected) && (
                  <div className="space-y-5 animate-fade-in">
                    {/* 2. 類型與執行環境 */}
                    <div className="space-y-4 border-t border-darkBorder/40 pt-4">
                      <h4 className="text-[11px] font-bold text-indigo-400 uppercase tracking-wider select-none">{t("執行環境 (Runtime)")}</h4>

                      <div className="grid grid-cols-2 gap-4">
                        <div className="space-y-1.5">
                          <label className="block text-[10px] font-bold uppercase tracking-wider text-gray-500">{t("專案框架 / 類型")}</label>
                          <select
                            value={editingProject.type}
                            onChange={(e) => handleTypeChange(e.target.value)}
                            className="w-full bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-3 py-2 outline-none focus:border-blue-500 transition cursor-pointer font-semibold"
                          >
                            {projectTypes.map(t => (
                              <option key={t.value} value={t.value}>{t.label}</option>
                            ))}
                          </select>
                        </div>
                        <div className="space-y-1.5">
                          <label className="block text-[10px] font-bold uppercase tracking-wider text-gray-500">{t("執行器 (Runtime)")}</label>
                          <select
                            value={editingProject.runtime_type}
                            onChange={(e) => {
                              const newRt = e.target.value;
                              const hasBin = hasBundledRuntime(newRt);
                              setEditingProject({
                                ...editingProject,
                                runtime_type: newRt,
                                use_wincmp_bin: hasBin
                              });
                            }}
                            className="w-full bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-3 py-2 outline-none focus:border-blue-500 transition cursor-pointer font-semibold"
                          >
                            {runtimeTypes.map(t => (
                              <option key={t.value} value={t.value}>{t.label}</option>
                            ))}
                          </select>
                        </div>
                      </div>

                      {/* PHP 版本 */}
                      {(editingProject.type === 'laravel' || editingProject.type === 'php') && (
                        <div className="space-y-1.5">
                          <label className="block text-[10px] font-bold uppercase tracking-wider text-gray-500">{t("PHP 執行版本")}</label>
                          <select
                            value={editingProject.php_version}
                            onChange={(e) => setEditingProject({ ...editingProject, php_version: e.target.value })}
                            className="w-full bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-3 py-2 outline-none focus:border-blue-500 transition cursor-pointer font-semibold"
                          >
                            <option value="">{t("請選擇對應 PHP 版本...")}</option>
                            {scanResult?.PHPList?.map((p: any) => (
                              <option key={p.MajorMin} value={p.MajorMin}>PHP {p.MajorMin} ({t("偵測到")} {p.Version})</option>
                            ))}
                          </select>
                        </div>
                      )}

                      {/* Runtime 進階配置 */}
                      {isRuntimeProject(editingProject.type) && (
                        <div className="border border-darkBorder rounded-xl p-4 bg-[#0a0a0c]/40 space-y-4">
                          <div className="font-semibold text-gray-200 text-xs flex items-center gap-1.5 border-b border-darkBorder pb-2">
                            <Settings size={12} className="text-blue-400" /> {t("Runtime 運行細節設定")}
                          </div>

                          <div className="grid grid-cols-2 gap-3">
                            <div className="space-y-1">
                              <label className="block text-[10px] text-gray-500 font-bold uppercase">{t("執行 Port")}</label>
                              <input
                                type="number"
                                value={editingProject.runtime_port || 3000}
                                onChange={(e) => setEditingProject({ ...editingProject, runtime_port: parseInt(e.target.value) })}
                                className="w-full bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-2.5 py-1.5 outline-none focus:border-blue-500 transition font-mono"
                              />
                            </div>
                            <div className="space-y-1">
                              <label className="block text-[10px] text-gray-500 font-bold uppercase">{t("運行模式")}</label>
                              <select
                                value={editingProject.runtime_mode || 'Background'}
                                onChange={(e) => setEditingProject({ ...editingProject, runtime_mode: e.target.value })}
                                className="w-full bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-2.5 py-1.5 outline-none focus:border-blue-500 transition cursor-pointer font-semibold"
                              >
                                <option value="Background">{t("背景執行 (Background)")}</option>
                                <option value="Terminal">{t("終端執行 (Terminal)")}</option>
                              </select>
                            </div>
                          </div>

                          {/* Node/Bun 專屬配置 */}
                          {['node', 'bun', 'auto'].includes(editingProject.runtime_type || '') && (
                            <div className="space-y-3 pt-2">
                              <div className="flex items-center gap-2 select-none">
                                <input
                                  type="checkbox"
                                  id="useWinCMPBin"
                                  checked={editingProject.use_wincmp_bin}
                                  onChange={(e) => setEditingProject({ ...editingProject, use_wincmp_bin: e.target.checked })}
                                  className="w-3.5 h-3.5 bg-darkInput border-darkBorder rounded text-blue-500 accent-blue-500 cursor-pointer"
                                />
                                <label htmlFor="useWinCMPBin" className="text-[11px] text-gray-400 cursor-pointer font-medium">{t("使用 WinCMP 內建執行檔 (Bundled Runtime)")}</label>
                              </div>

                              {editingProject.use_wincmp_bin && (
                                <div className="space-y-1">
                                  <label className="block text-[10px] text-gray-500 font-bold uppercase">{t("選擇內建版本")}</label>
                                  <select
                                    value={editingProject.runtime_version || ''}
                                    onChange={(e) => setEditingProject({ ...editingProject, runtime_version: e.target.value })}
                                    className="w-full bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-2.5 py-1.5 outline-none focus:border-blue-500 transition text-[11px] cursor-pointer"
                                  >
                                    {(editingProject.runtime_type === 'bun' ? scanResult?.BunList : scanResult?.NodeList)?.map((r: any) => (
                                      <option key={r.Version} value={r.Version}>{r.Version}</option>
                                    ))}
                                    {!(editingProject.runtime_type === 'bun' ? scanResult?.BunList : scanResult?.NodeList)?.length && (
                                      <option value="">{t("無可用版本 (請確認 ./bin/)")}</option>
                                    )}
                                  </select>
                                </div>
                              )}
                            </div>
                          )}

                          {/* 自訂啟動命令 */}
                          <div className="space-y-1.5">
                            <label className="block text-[10px] text-gray-500 font-bold uppercase">{t("自訂啟動指令 (可選，空白將使用預設)")}</label>
                            <input
                              type="text"
                              value={editingProject.command || ''}
                              onChange={(e) => setEditingProject({ ...editingProject, command: e.target.value })}
                              placeholder={t("例如: npm run dev -- --port 3000")}
                              className="w-full bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-2.5 py-1.5 outline-none focus:border-blue-500 transition font-mono"
                            />
                          </div>
                        </div>
                      )}
                    </div>

                    {/* 3. 網域別名設定 */}
                    <div className="space-y-4 border-t border-darkBorder/40 pt-4">
                      <h4 className="text-[11px] font-bold text-teal-400 uppercase tracking-wider select-none">{t("網域別名 (Domains)")}</h4>

                      <div className="space-y-2">
                        {editingProject.domains.map((dom, dIdx) => (
                          <div key={dIdx} className="flex gap-2">
                            <input
                              type="text"
                              value={dom}
                              onChange={(e) => handleDomainChange(dIdx, e.target.value)}
                              placeholder={t("例如: my-site.test")}
                              className="flex-1 bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-3 py-2 outline-none focus:border-blue-500 transition font-mono"
                            />
                            {editingProject.domains.length > 1 && (
                              <button
                                onClick={() => handleRemoveDomain(dIdx)}
                                className="px-3 py-2 bg-red-900 bg-opacity-25 hover:bg-opacity-50 text-red-400 border border-red-900 border-opacity-40 rounded-lg transition font-semibold"
                              >
                                {t("移除")}
                              </button>
                            )}
                          </div>
                        ))}
                        <button
                          onClick={handleAddDomain}
                          className="text-[11px] text-blue-400 hover:text-blue-300 font-semibold flex items-center gap-1 transition"
                        >
                          {t("+ 新增別名網域")}
                        </button>
                        <div className="text-[10px] text-gray-500 italic mt-0.5">
                          * {t("不可包含底線(_)、埠號(:)或路徑(/)，僅限英數字、連字號(-)與點(.)")}
                        </div>
                      </div>
                    </div>

                    {/* 4. SSL 憑證選項 */}
                    <div className="flex items-center justify-between border border-darkBorder p-4 rounded-xl bg-[#0a0a0c]/40 space-y-0 border-t border-darkBorder/40">
                      <div className="flex items-center gap-3">
                        <Shield size={16} className="text-blue-500" />
                        <div>
                          <div className="font-semibold text-gray-200 text-[11px]">{t("啟用 HTTPS 安全憑證")}</div>
                          <div className="text-[10px] text-gray-500 mt-0.5">{t("自動套用 Caddy 內部自簽憑證")}</div>
                        </div>
                      </div>
                      <input
                        type="checkbox"
                        checked={editingProject.use_ssl}
                        onChange={(e) => setEditingProject({ ...editingProject, use_ssl: e.target.checked })}
                        className="w-3.5 h-3.5 bg-darkInput border-darkBorder rounded text-blue-500 accent-blue-500 cursor-pointer"
                      />
                    </div>

                    {/* 5. Caddyfile 路徑與開啟 */}
                    <div className="border border-darkBorder p-4 rounded-xl bg-[#0a0a0c]/40 space-y-3 border-t border-darkBorder/40">
                      <div className="flex items-center gap-3">
                        <Settings size={16} className="text-blue-500" />
                        <div>
                          <div className="font-semibold text-gray-200 text-[11px]">{t("Caddy 配置文件路徑")}</div>
                          <div className="text-[10px] text-gray-500 mt-0.5">{t("編輯專案 Caddyfile 設定")}</div>
                        </div>
                      </div>
                      <div className="flex gap-2 items-center">
                        <input
                          type="text"
                          readOnly
                          value={editingProject.name ? `conf\\sites\\${editingProject.name}.caddy` : ''}
                          className="flex-1 bg-darkInput border border-darkBorder text-gray-400 rounded-lg px-3 py-2 outline-none font-mono text-[11px]"
                        />
                        <button
                          type="button"
                          disabled={editIndex === null}
                          onClick={async () => {
                            try {
                              await OpenProjectCaddyfile(editingProject.name);
                            } catch (err) {
                              (window as any).customAlert(`${t("無法開啟設定檔")}: ${err}`);
                            }
                          }}
                          className="px-3.5 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-40 disabled:hover:bg-blue-600 text-white rounded-lg transition font-semibold text-[11px] whitespace-nowrap"
                        >
                          {t("開啟檔案")}
                        </button>
                      </div>
                      {editIndex === null && (
                        <div className="text-[10px] text-gray-500 italic">
                          * {t("儲存專案後將自動建立 Caddyfile")}
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>

              {/* Drawer Footer */}
              <div className="px-6 py-4.5 border-t border-darkBorder flex justify-end gap-3 bg-[#0d0d10]">
                <button
                  onClick={() => setIsModalOpen(false)}
                  className="px-4 py-2 border border-darkBorder rounded-lg text-xs font-semibold text-gray-300 hover:bg-darkBorder transition"
                >
                  {t("取消")}
                </button>
                {(editIndex !== null || detected) && (
                  <button
                    onClick={handleSaveProject}
                    className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-xs font-semibold transition"
                  >
                    {t("儲存設定")}
                  </button>
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 專案終端 Drawer */}
      <ProjectTerminal
        projectName={terminalProject || ''}
        isOpen={terminalProject !== null}
        onClose={() => setTerminalProject(null)}
      />
    </div>
  );

  function handleDomainChange(idx: number, val: string) {
    if (!editingProject) return;
    const newDoms = [...editingProject.domains];
    newDoms[idx] = val;
    setEditingProject({ ...editingProject, domains: newDoms });
  }

  function handleAddDomain() {
    if (!editingProject) return;
    setEditingProject({ ...editingProject, domains: [...editingProject.domains, ''] });
  }

  function handleRemoveDomain(idx: number) {
    if (!editingProject) return;
    const newDoms = [...editingProject.domains];
    newDoms.splice(idx, 1);
    setEditingProject({ ...editingProject, domains: newDoms });
  }

  async function handleRootPathSelect() {
    await handleSelectRootPath();
  }

  // 輔助方法：自動適配與更改類型
  function handleTypeChange(type: string) {
    if (!editingProject) return;
    let rt = 'none';
    let port = 0;

    if (['next', 'nuxt', 'astro', 'vite'].includes(type)) {
      rt = 'auto';
      port = 3000;
    } else if (type.startsWith('python')) {
      rt = 'python';
      port = 8000;
    } else if (type === 'go_api') {
      rt = 'go_air';
      port = 8080;
    } else if (type === 'pocketbase') {
      rt = 'go_run';
      port = 8090;
    } else if (type === 'custom') {
      rt = 'custom';
      port = 3000;
    }

    const hasBin = hasBundledRuntime(rt);
    setEditingProject({
      ...editingProject,
      type: type,
      runtime_type: rt,
      runtime_port: port,
      // 預設為該類型配置對應的 version
      runtime_version: rt === 'bun' ? scanResult?.BunList?.[0]?.Version : scanResult?.NodeList?.[0]?.Version,
      use_wincmp_bin: hasBin
    });
  }
}

import React, { useState, useEffect } from 'react';
import { Settings as SettingsIcon, Save, FolderOpen, Shield, Database, Mail, Languages, Info, FileText, Package } from 'lucide-react';
import { GetConfig, SaveConfig, SelectFolder, OpenFolder } from '../../wailsjs/go/main/App';
import DependencyManager from './DependencyManager';
import { logStore } from './logStore';
import { t, useLanguage, setLanguage } from '../i18n';

export default function Settings() {
  useLanguage(); // 訂閱語系變更

  const [config, setConfig] = useState<any>(null);
  const [originalConfig, setOriginalConfig] = useState<any>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [showDepManager, setShowDepManager] = useState(false);

  useEffect(() => {
    async function loadConfig() {
      try {
        const cfg = await GetConfig();
        setConfig(cfg);
        // 深度複製設定作為比對基準線
        setOriginalConfig(JSON.parse(JSON.stringify(cfg)));
        (window as any).isSettingsDirty = false;
      } catch (err) {
        console.error("載入設定檔失敗:", err);
      }
    }
    loadConfig();
  }, []);

  // 監聽設定變更，判斷是否與原始設定不同
  useEffect(() => {
    if (config && originalConfig) {
      const isDirty = JSON.stringify(config.global) !== JSON.stringify(originalConfig.global);
      (window as any).isSettingsDirty = isDirty;
    } else {
      (window as any).isSettingsDirty = false;
    }
  }, [config, originalConfig]);

  // 元件卸載時，自動清理髒狀態標記
  useEffect(() => {
    return () => {
      (window as any).isSettingsDirty = false;
    };
  }, []);

  const handleSelectFolder = async (field: 'default_www' | 'default_ssl' | 'mariadb_basedir' | 'mariadb_datadir') => {
    if (!config) return;
    try {
      const path = await SelectFolder();
      if (path) {
        const newCfg = { ...config };
        if (field === 'default_www') newCfg.global.default_www = path;
        if (field === 'default_ssl') newCfg.global.default_ssl = path;
        if (field === 'mariadb_basedir') newCfg.global.mariadb_basedir = path;
        if (field === 'mariadb_datadir') newCfg.global.mariadb_datadir = path;
        setConfig(newCfg);
      }
    } catch (err) {
      console.error("選擇目錄失敗:", err);
    }
  };

  const handleSave = async () => {
    if (!config) return;
    setIsSaving(true);
    try {
      // 確保 Port 數字格式正確
      const newCfg = { ...config };
      newCfg.global.mariadb_port = parseInt(newCfg.global.mariadb_port || "3306");
      newCfg.global.mailpit_smtp_port = parseInt(newCfg.global.mailpit_smtp_port || "1025");
      newCfg.global.mailpit_http_port = parseInt(newCfg.global.mailpit_http_port || "8025");
      newCfg.global.max_log_retention = parseInt(newCfg.global.max_log_retention || "30");
      newCfg.global.max_log_lines = parseInt(newCfg.global.max_log_lines || "500");

      await SaveConfig(newCfg);
      logStore.setMaxLogLines(newCfg.global.max_log_lines);
      setConfig(newCfg);
      setOriginalConfig(JSON.parse(JSON.stringify(newCfg)));
      (window as any).isSettingsDirty = false;
      // 即時生效設定前端語系
      setLanguage(newCfg.global.language || 'zh-TW');
      (window as any).customAlert(t("設定儲存成功！"));
    } catch (err) {
      (window as any).customAlert(`${t("儲存設定失敗")}: ${err}`);
    } finally {
      setIsSaving(false);
    }
  };

  const handleOpenLocalPath = async (type: 'hosts' | 'phpini' | 'wincmpjson') => {
    try {
      if (type === 'hosts') {
        // 在 Windows 下打開 hosts
        await OpenFolder('C:\\Windows\\System32\\drivers\\etc\\hosts');
      } else if (type === 'phpini') {
        await OpenFolder('./conf/php/php.ini');
      } else if (type === 'wincmpjson') {
        await OpenFolder('./conf/wincmp.json');
      }
    } catch (err) {
      (window as any).customAlert(`${t("無法開啟設定檔")}: ${err}`);
    }
  };

  const handleGlobalFieldChange = (field: string, val: any) => {
    if (!config) return;
    const newCfg = { ...config };
    newCfg.global[field] = val;
    setConfig(newCfg);
  };

  if (!config) {
    return <div className="p-8 text-center text-gray-400 select-none text-xs font-semibold">{t("載入設定中...")}</div>;
  }

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* 標頭 */}
      <div className="p-6 pb-4 flex justify-between items-center select-none border-b border-darkBorder/40 shrink-0">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-white">⚙️ {t("系統全域設定")}</h1>
          <p className="text-xs text-gray-400 mt-1">{t("配置開發路徑、資料庫參數以及 WinCMP 全域行為")}</p>
        </div>
        <div className="flex gap-2.5">
          <button
            onClick={() => setShowDepManager(true)}
            className="px-3.5 py-2.5 bg-darkCard border border-darkBorder hover:border-gray-600 text-gray-200 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition duration-200"
          >
            <Package size={14} className="text-blue-400" />
            <span>{t("依賴庫管理")}</span>
          </button>
          <button
            onClick={handleSave}
            disabled={isSaving}
            className="px-4 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 transition duration-200"
          >
            <Save size={14} />
            <span>{isSaving ? t("儲存中...") : t("儲存全域設定")}</span>
          </button>
        </div>
      </div>

      {/* 滾動內容區域 */}
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 text-xs text-gray-300">
          {/* 1. 基本路徑與行為 */}
          <div className="bg-darkCard border border-darkBorder rounded-xl p-5 space-y-4">
            <h3 className="font-bold text-sm text-gray-200 flex items-center gap-2 border-b border-darkBorder/40 pb-3 select-none">
              <SettingsIcon size={14} className="text-blue-400" /> {t("基本路徑與行為")}
            </h3>

            {/* WWW 根目錄 */}
            <div className="space-y-1.5">
              <label className="text-[10px] text-gray-500 font-bold uppercase">{t("預設 Web 專案目錄 (WWW Dir)")}</label>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={config.global.default_www}
                  onChange={(e) => handleGlobalFieldChange('default_www', e.target.value)}
                  className="flex-1 bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-3 py-1.5 outline-none focus:border-blue-500 transition font-mono"
                />
                <button
                  onClick={() => handleSelectFolder('default_www')}
                  className="px-3 py-1.5 bg-darkInput border border-darkBorder hover:border-gray-500 rounded-lg transition font-semibold"
                >
                  {t("選擇")}
                </button>
              </div>
            </div>

            {/* SSL 根目錄 */}
            <div className="space-y-1.5">
              <label className="text-[10px] text-gray-500 font-bold uppercase">{t("預設 SSL 憑證存放目錄 (SSL Dir)")}</label>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={config.global.default_ssl}
                  onChange={(e) => handleGlobalFieldChange('default_ssl', e.target.value)}
                  className="flex-1 bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-3 py-1.5 outline-none focus:border-blue-500 transition font-mono"
                />
                <button
                  onClick={() => handleSelectFolder('default_ssl')}
                  className="px-3 py-1.5 bg-darkInput border border-darkBorder hover:border-gray-500 rounded-lg transition font-semibold"
                >
                  {t("選擇")}
                </button>
              </div>
            </div>

            {/* 系統開關組 */}
            <div className="space-y-3 pt-2 select-none">
              <div className="flex items-center justify-between">
                <span className="font-semibold text-gray-300">{t("恢復上次關閉時的服務狀態")}</span>
                <input
                  type="checkbox"
                  checked={config.global.restore_last_state}
                  onChange={(e) => handleGlobalFieldChange('restore_last_state', e.target.checked)}
                  className="w-3.5 h-3.5 bg-darkInput border-darkBorder rounded text-blue-500 accent-blue-500 cursor-pointer"
                />
              </div>
              <div className="flex items-center justify-between">
                <span className="font-semibold text-gray-300">{t("自動向 Windows Hosts 檔更新域名")}</span>
                <input
                  type="checkbox"
                  checked={config.global.auto_update_hosts}
                  onChange={(e) => handleGlobalFieldChange('auto_update_hosts', e.target.checked)}
                  className="w-3.5 h-3.5 bg-darkInput border-darkBorder rounded text-blue-500 accent-blue-500 cursor-pointer"
                />
              </div>
              <div className="flex items-center justify-between">
                <span className="font-semibold text-gray-300">{t("點擊關閉視窗時縮小至系統托盤 (Minimize to Tray)")}</span>
                <input
                  type="checkbox"
                  checked={config.global.minimize_to_tray}
                  onChange={(e) => handleGlobalFieldChange('minimize_to_tray', e.target.checked)}
                  className="w-3.5 h-3.5 bg-darkInput border-darkBorder rounded text-blue-500 accent-blue-500 cursor-pointer"
                />
              </div>
            </div>
          </div>

          {/* 2. 資料庫與郵件伺服器 */}
          <div className="bg-darkCard border border-darkBorder rounded-xl p-5 space-y-4">
            <h3 className="font-bold text-sm text-gray-200 flex items-center gap-2 border-b border-darkBorder/40 pb-3 select-none">
              <Database size={14} className="text-teal-400" /> {t("MariaDB 資料庫 & Mailpit 服務設定")}
            </h3>

            <div className="flex items-center justify-between select-none">
              <div>
                <span className="font-semibold text-gray-300 block">{t("使用外部自訂 MariaDB/MySQL")}</span>
                <span className="text-[10px] text-gray-500 mt-0.5 block">{t("手動指定資料庫二進位目錄與數據目錄")}</span>
              </div>
              <input
                type="checkbox"
                checked={config.global.mariadb_external}
                onChange={(e) => handleGlobalFieldChange('mariadb_external', e.target.checked)}
                className="w-3.5 h-3.5 bg-darkInput border-darkBorder rounded text-teal-500 accent-teal-500 cursor-pointer"
              />
            </div>

            {/* 外部資料庫路徑 */}
            {config.global.mariadb_external && (
              <div className="space-y-3.5 p-4 border border-darkBorder rounded-xl bg-[#0a0a0c]/40 transition">
                <div className="grid grid-cols-2 gap-3 select-none">
                  <div className="space-y-1">
                    <label className="text-[10px] text-gray-500 font-bold uppercase">{t("資料庫引擎類型")}</label>
                    <select
                      value={config.global.mariadb_type || 'MariaDB'}
                      onChange={(e) => handleGlobalFieldChange('mariadb_type', e.target.value)}
                      className="w-full bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-2.5 py-1.5 outline-none focus:border-blue-500 transition cursor-pointer font-semibold"
                    >
                      <option value="MariaDB">MariaDB</option>
                      <option value="MySQL">MySQL</option>
                    </select>
                  </div>
                  <div className="space-y-1">
                    <label className="text-[10px] text-gray-500 font-bold uppercase">{t("執行 Port")}</label>
                    <input
                      type="number"
                      value={config.global.mariadb_port || 3306}
                      onChange={(e) => handleGlobalFieldChange('mariadb_port', e.target.value)}
                      className="w-full bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-2.5 py-1.5 outline-none focus:border-blue-500 transition font-mono"
                    />
                  </div>
                </div>

                <div className="space-y-1.5">
                  <label className="text-[10px] text-gray-500 font-bold uppercase block select-none">{t("Binary Path (含 bin 資料夾的根目錄)")}</label>
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={config.global.mariadb_basedir || ""}
                      onChange={(e) => handleGlobalFieldChange('mariadb_basedir', e.target.value)}
                      className="flex-1 bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-2.5 py-1 text-xs outline-none focus:border-blue-500 transition font-mono"
                    />
                    <button
                      onClick={() => handleSelectFolder('mariadb_basedir')}
                      className="px-3 py-1 bg-darkInput border border-darkBorder hover:border-gray-500 rounded-lg transition font-semibold"
                    >
                      {t("選擇")}
                    </button>
                  </div>
                </div>

                <div className="space-y-1.5">
                  <label className="text-[10px] text-gray-500 font-bold uppercase block select-none">{t("Data Path (資料存放目錄)")}</label>
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={config.global.mariadb_datadir || ""}
                      onChange={(e) => handleGlobalFieldChange('mariadb_datadir', e.target.value)}
                      className="flex-1 bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-2.5 py-1 text-xs outline-none focus:border-blue-500 transition font-mono"
                    />
                    <button
                      onClick={() => handleSelectFolder('mariadb_datadir')}
                      className="px-3 py-1 bg-darkInput border border-darkBorder hover:border-gray-500 rounded-lg transition font-semibold"
                    >
                      {t("選擇")}
                    </button>
                  </div>
                </div>
              </div>
            )}

            {/* Mailpit 端口與設定 */}
            <div className="border-t border-darkBorder/40 pt-4 space-y-3.5">
              <div className="font-bold text-[10px] text-purple-400 uppercase tracking-wider flex items-center gap-1 select-none">
                <Mail size={12} /> {t("Mailpit 端口配置")}
              </div>
              <div className="grid grid-cols-2 gap-3 select-none">
                <div className="space-y-1">
                  <label className="text-[10px] text-gray-500 font-bold uppercase">SMTP Port</label>
                  <input
                    type="number"
                    value={config.global.mailpit_smtp_port || 1025}
                    onChange={(e) => handleGlobalFieldChange('mailpit_smtp_port', e.target.value)}
                    className="w-full bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-2.5 py-1.5 outline-none focus:border-blue-500 transition font-mono"
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-[10px] text-gray-500 font-bold uppercase">HTTP Port</label>
                  <input
                    type="number"
                    value={config.global.mailpit_http_port || 8025}
                    onChange={(e) => handleGlobalFieldChange('mailpit_http_port', e.target.value)}
                    className="w-full bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-2.5 py-1.5 outline-none focus:border-blue-500 transition font-mono"
                  />
                </div>
              </div>
              <div className="flex items-center justify-between pt-1 select-none">
                <span className="font-semibold text-gray-300">{t("使用內置數據庫持久化保存信箱數據")}</span>
                <input
                  type="checkbox"
                  checked={config.global.mailpit_use_db}
                  onChange={(e) => handleGlobalFieldChange('mailpit_use_db', e.target.checked)}
                  className="w-3.5 h-3.5 bg-darkInput border-darkBorder rounded text-purple-500 accent-purple-500 cursor-pointer"
                />
              </div>
            </div>
          </div>

          {/* 3. 本地化語系與日誌設定 */}
          <div className="bg-darkCard border border-darkBorder rounded-xl p-5 space-y-4">
            <h3 className="font-bold text-sm text-gray-200 flex items-center gap-2 border-b border-darkBorder/40 pb-3 select-none">
              <Languages size={14} className="text-purple-400" /> {t("本地化語言與日誌設定")}
            </h3>

            {/* 語言 */}
            <div className="space-y-1.5">
              <label className="text-[10px] text-gray-500 font-bold uppercase select-none">{t("顯示語言 (Language)")}</label>
              <select
                value={config.global.language || 'zh-TW'}
                onChange={(e) => handleGlobalFieldChange('language', e.target.value)}
                className="w-full bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-3 py-1.5 outline-none focus:border-blue-500 transition cursor-pointer font-semibold"
              >
                <option value="zh-TW">繁體中文 (zh-TW)</option>
                <option value="en-US">English (en-US)</option>
              </select>
            </div>

            {/* 預設終端 Shell */}
            <div className="space-y-1.5">
              <label className="text-[10px] text-gray-500 font-bold uppercase select-none">{t("預設專案終端 (Terminal Shell)")}</label>
              <select
                value={config.global.terminal_shell || 'powershell.exe'}
                onChange={(e) => handleGlobalFieldChange('terminal_shell', e.target.value)}
                className="w-full bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-3 py-1.5 outline-none focus:border-blue-500 transition cursor-pointer font-semibold"
              >
                <option value="powershell.exe">PowerShell (powershell.exe)</option>
                <option value="cmd.exe">Command Prompt (cmd.exe)</option>
                <option value="C:\Program Files\Git\bin\bash.exe">Git Bash (bash.exe)</option>
                <option value="wsl.exe">WSL (wsl.exe)</option>
              </select>
            </div>

            <div className="grid grid-cols-2 gap-4">
              {/* 日誌天數 */}
              <div className="space-y-1.5">
                <label className="text-[10px] text-gray-500 font-bold uppercase select-none">{t("檔案日誌保存期限 (天)")}</label>
                <input
                  type="number"
                  value={config.global.max_log_retention || 30}
                  onChange={(e) => handleGlobalFieldChange('max_log_retention', e.target.value)}
                  className="w-full bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-3 py-1.5 outline-none focus:border-blue-500 transition font-mono"
                />
              </div>
              {/* 最大日誌行數 */}
              <div className="space-y-1.5">
                <label className="text-[10px] text-gray-500 font-bold uppercase select-none">{t("終端保留最大行數")}</label>
                <input
                  type="number"
                  value={config.global.max_log_lines || 500}
                  onChange={(e) => handleGlobalFieldChange('max_log_lines', e.target.value)}
                  className="w-full bg-darkInput border border-darkBorder text-gray-100 rounded-lg px-3 py-1.5 outline-none focus:border-blue-500 transition font-mono"
                />
              </div>
            </div>
          </div>

          {/* 4. 快速編輯系統設定檔 */}
          <div className="bg-darkCard border border-darkBorder rounded-xl p-5 space-y-4">
            <h3 className="font-bold text-sm text-gray-200 flex items-center gap-2 border-b border-darkBorder/40 pb-3 select-none">
              <FileText size={14} className="text-orange-400" /> {t("系統設定檔案快速捷徑")}
            </h3>
            <p className="text-[11px] text-gray-500 select-none font-medium">{t("可以在這裡直接使用系統預設編輯器開啟核心設定檔，進行進階手動編輯：")}</p>
            <div className="grid grid-cols-3 gap-3 select-none">
              <button
                onClick={() => handleOpenLocalPath('hosts')}
                className="py-3 px-2 bg-darkInput/40 border border-darkBorder hover:border-orange-500/40 hover:bg-orange-500/[0.02] rounded-xl flex flex-col items-center gap-2 transition"
              >
                <Shield size={16} className="text-orange-400" />
                <span className="font-bold text-gray-300 text-xs">{t("Hosts 檔案")}</span>
                <span className="text-[10px] text-gray-500 font-mono">{t("(Hosts)")}</span>
              </button>
              <button
                onClick={() => handleOpenLocalPath('phpini')}
                className="py-3 px-2 bg-darkInput/40 border border-darkBorder hover:border-emerald-500/40 hover:bg-emerald-500/[0.02] rounded-xl flex flex-col items-center gap-2 transition"
              >
                <SettingsIcon size={16} className="text-emerald-400" />
                <span className="font-bold text-gray-300 text-xs">{t("php.ini 設定")}</span>
                <span className="text-[10px] text-gray-500 font-mono">{t("(PHP 全域)")}</span>
              </button>
              <button
                onClick={() => handleOpenLocalPath('wincmpjson')}
                className="py-3 px-2 bg-darkInput/40 border border-darkBorder hover:border-blue-500/40 hover:bg-blue-500/[0.02] rounded-xl flex flex-col items-center gap-2 transition"
              >
                <Info size={16} className="text-blue-400" />
                <span className="font-bold text-gray-300 text-xs">{t("WinCMP Json")}</span>
                <span className="text-[10px] text-gray-500 font-mono">{t("(核心配置)")}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
      <DependencyManager isOpen={showDepManager} onClose={() => setShowDepManager(false)} />
    </div>
  );
}

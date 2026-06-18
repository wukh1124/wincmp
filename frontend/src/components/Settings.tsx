import React, { useState, useEffect } from 'react';
import { Settings as SettingsIcon, Save, FolderOpen, Shield, Database, Mail, Languages, Info, FileText, Package, Palette, Check } from 'lucide-react';
import { GetConfig, SaveConfig, SelectFolder, OpenFolder } from '../../wailsjs/go/main/App';
import DependencyManager from './DependencyManager';
import { logStore } from './logStore';
import { t, useLanguage, setLanguage, getLanguage } from '../i18n';
import { useTheme, THEMES, ThemeId } from './ThemeContext';

export default function Settings() {
  useLanguage(); // 訂閱語系變更
  const { theme, setTheme, fontSize, setFontSize } = useTheme();

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

  // 監聽 Sidebar 或全域快速設定的儲存同步事件，自動重新讀取 Config 以同步頁面狀態
  useEffect(() => {
    const handleConfigSynced = async () => {
      try {
        const cfg = await GetConfig();
        setConfig(cfg);
        // 如果此時 Settings 頁面是乾淨的，就把 originalConfig 也刷新
        // 這樣就不會讓 (window as any).isSettingsDirty 變成 true
        if (!(window as any).isSettingsDirty) {
          setOriginalConfig(JSON.parse(JSON.stringify(cfg)));
        }
      } catch (err) {
        console.error("同步設定失敗:", err);
      }
    };

    window.addEventListener('wincmp_config_synced', handleConfigSynced);
    return () => {
      window.removeEventListener('wincmp_config_synced', handleConfigSynced);
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

  const handleSave = async (silent = false) => {
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
      // 即時生效設定前端語系與字型大小
      setLanguage(newCfg.global.language || 'zh-TW');
      setFontSize(newCfg.global.font_size || 'small');
      if (!silent) {
        await (window as any).customAlert(t("設定儲存成功！"));
      }
    } catch (err) {
      (window as any).customAlert(`${t("儲存設定失敗")}: ${err}`);
      throw err;
    } finally {
      setIsSaving(false);
    }
  };

  // 每次 render 都更新 window.saveSettings 參照，確保在外部呼叫時能以最新設定值儲存
  useEffect(() => {
    (window as any).saveSettings = handleSave;
    return () => {
      (window as any).saveSettings = undefined;
    };
  });

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
    return <div className="p-8 text-center select-none text-xs font-semibold" style={{ color: 'var(--muted)' }}>{t("載入設定中...")}</div>;
  }

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* 標頭 */}
      <div className="p-6 pb-4 flex justify-between items-center select-none shrink-0" style={{ borderBottom: '1px solid var(--border)' }}>
        <div className="flex items-baseline gap-3">
          <h1 className="text-xl font-bold tracking-tight" style={{ color: 'var(--fg)' }}>{t("系統設定")}</h1>
          <p className="text-xs" style={{ color: 'var(--muted)' }}>{t("配置開發路徑、資料庫參數以及 WinCMP 全域行為")}</p>
        </div>
        <div className="flex gap-2.5">
          <button
            onClick={() => setShowDepManager(true)}
            className="px-3.5 py-2.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition duration-200 hover:border-gray-600"
            style={{ backgroundColor: 'var(--card)', border: '1px solid var(--border)', color: 'var(--fg-2)' }}
          >
            <Package size={14} style={{ color: 'var(--accent)' }} />
            <span>{t("依賴庫管理")}</span>
          </button>
          <button
            onClick={() => handleSave()}
            disabled={isSaving}
            className="px-4 py-2.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition duration-200 hover:bg-blue-700 disabled:opacity-50"
            style={{ backgroundColor: 'var(--accent)', color: 'var(--accent-on)' }}
          >
            <Save size={14} />
            <span>{isSaving ? t("儲存中...") : t("儲存設定")}</span>
          </button>
        </div>
      </div>

      {/* 滾動內容區域 */}
      <div className="flex-1 overflow-y-auto p-6 space-y-6">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 text-xs" style={{ color: 'var(--fg-2)' }}>
          {/* 1. 基本路徑與行為 */}
          <div className="rounded-xl p-5 space-y-4" style={{ backgroundColor: 'var(--card)', border: '1px solid var(--border)' }}>
            <h3 className="font-bold text-sm flex items-center gap-2 pb-3 select-none" style={{ color: 'var(--fg)', borderBottom: '1px solid var(--border)' }}>
              <SettingsIcon size={14} style={{ color: 'var(--accent)' }} /> {t("基本路徑與行為")}
            </h3>

            {/* WWW 根目錄 */}
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold uppercase" style={{ color: 'var(--meta)' }}>{t("預設 Web 專案目錄 (WWW Dir)")}</label>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={config.global.default_www}
                  onChange={(e) => handleGlobalFieldChange('default_www', e.target.value)}
                  className="flex-1 rounded-lg px-3 py-1.5 outline-none transition font-mono focus:border-blue-500"
                  style={{ backgroundColor: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--fg)' }}
                />
                <button
                  onClick={() => handleSelectFolder('default_www')}
                  className="px-3 py-1.5 rounded-lg transition font-semibold hover:border-gray-500"
                  style={{ backgroundColor: 'var(--input-bg)', border: '1px solid var(--input-border)' }}
                >
                  {t("選擇")}
                </button>
              </div>
            </div>

            {/* SSL 根目錄 */}
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold uppercase" style={{ color: 'var(--meta)' }}>{t("預設 SSL 憑證存放目錄 (SSL Dir)")}</label>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={config.global.default_ssl}
                  onChange={(e) => handleGlobalFieldChange('default_ssl', e.target.value)}
                  className="flex-1 rounded-lg px-3 py-1.5 outline-none transition font-mono focus:border-blue-500"
                  style={{ backgroundColor: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--fg)' }}
                />
                <button
                  onClick={() => handleSelectFolder('default_ssl')}
                  className="px-3 py-1.5 rounded-lg transition font-semibold hover:border-gray-500"
                  style={{ backgroundColor: 'var(--input-bg)', border: '1px solid var(--input-border)' }}
                >
                  {t("選擇")}
                </button>
              </div>
            </div>

            {/* 系統開關組 */}
            <div className="space-y-3 pt-2 select-none">
              <div className="flex items-center justify-between">
                <span className="font-semibold" style={{ color: 'var(--fg-2)' }}>{t("恢復上次關閉時的服務狀態")}</span>
                <input
                  type="checkbox"
                  checked={config.global.restore_last_state}
                  onChange={(e) => handleGlobalFieldChange('restore_last_state', e.target.checked)}
                  className="w-3.5 h-3.5 rounded cursor-pointer"
                  style={{ backgroundColor: 'var(--input-bg)', borderColor: 'var(--input-border)', accentColor: 'var(--accent)' }}
                />
              </div>
              <div className="flex items-center justify-between">
                <span className="font-semibold" style={{ color: 'var(--fg-2)' }}>{t("自動向 Windows Hosts 檔更新域名")}</span>
                <input
                  type="checkbox"
                  checked={config.global.auto_update_hosts}
                  onChange={(e) => handleGlobalFieldChange('auto_update_hosts', e.target.checked)}
                  className="w-3.5 h-3.5 rounded cursor-pointer"
                  style={{ backgroundColor: 'var(--input-bg)', borderColor: 'var(--input-border)', accentColor: 'var(--accent)' }}
                />
              </div>
              <div className="flex items-center justify-between">
                <span className="font-semibold" style={{ color: 'var(--fg-2)' }}>{t("點擊關閉視窗時縮小至系統托盤 (Minimize to Tray)")}</span>
                <input
                  type="checkbox"
                  checked={config.global.minimize_to_tray}
                  onChange={(e) => handleGlobalFieldChange('minimize_to_tray', e.target.checked)}
                  className="w-3.5 h-3.5 rounded cursor-pointer"
                  style={{ backgroundColor: 'var(--input-bg)', borderColor: 'var(--input-border)', accentColor: 'var(--accent)' }}
                />
              </div>
              <div className="flex items-center justify-between">
                <div>
                  <span className="font-semibold block" style={{ color: 'var(--fg-2)' }}>{t("定時自動檢查新版本")}</span>
                  <span className="text-[10px] mt-0.5 block" style={{ color: 'var(--meta)' }}>{t("每 6 小時自動檢查新版本")}</span>
                </div>
                <input
                  type="checkbox"
                  checked={config.global.auto_check_update}
                  onChange={(e) => handleGlobalFieldChange('auto_check_update', e.target.checked)}
                  className="w-3.5 h-3.5 rounded cursor-pointer"
                  style={{ backgroundColor: 'var(--input-bg)', borderColor: 'var(--input-border)', accentColor: 'var(--accent)' }}
                />
              </div>
            </div>
          </div>

          {/* 2. 本地化語系與日誌設定 */}
          <div className="rounded-xl p-5 space-y-4" style={{ backgroundColor: 'var(--card)', border: '1px solid var(--border)' }}>
            <h3 className="font-bold text-sm flex items-center gap-2 pb-3 select-none" style={{ color: 'var(--fg)', borderBottom: '1px solid var(--border)' }}>
              <Languages size={14} style={{ color: 'var(--status-info)' }} /> {t("本地化語言與日誌設定")}
            </h3>

            {/* 語言 */}
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold uppercase select-none" style={{ color: 'var(--meta)' }}>{t("顯示語言 (Language)")}</label>
              <select
                value={config.global.language || 'zh-TW'}
                onChange={(e) => handleGlobalFieldChange('language', e.target.value)}
                className="w-full rounded-lg px-3 py-1.5 outline-none transition cursor-pointer font-semibold focus:border-blue-500"
                style={{ backgroundColor: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--fg)' }}
              >
                <option value="zh-TW">繁體中文 (zh-TW)</option>
                <option value="en-US">English (en-US)</option>
              </select>
            </div>

            {/* 字型大小 */}
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold uppercase select-none" style={{ color: 'var(--meta)' }}>{t("字型大小")}</label>
              <select
                value={config.global.font_size || 'small'}
                onChange={(e) => handleGlobalFieldChange('font_size', e.target.value)}
                className="w-full rounded-lg px-3 py-1.5 outline-none transition cursor-pointer font-semibold focus:border-blue-500"
                style={{ backgroundColor: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--fg)' }}
              >
                <option value="small">{t("小")}</option>
                <option value="medium">{t("中")}</option>
                <option value="large">{t("大")}</option>
              </select>
            </div>

            {/* 預設終端 Shell */}
            <div className="space-y-1.5">
              <label className="text-[10px] font-bold uppercase select-none" style={{ color: 'var(--meta)' }}>{t("預設專案終端 (Terminal Shell)")}</label>
              <select
                value={config.global.terminal_shell || 'powershell.exe'}
                onChange={(e) => handleGlobalFieldChange('terminal_shell', e.target.value)}
                className="w-full rounded-lg px-3 py-1.5 outline-none transition cursor-pointer font-semibold focus:border-blue-500"
                style={{ backgroundColor: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--fg)' }}
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
                <label className="text-[10px] font-bold uppercase select-none" style={{ color: 'var(--meta)' }}>{t("檔案日誌保存期限 (天)")}</label>
                <input
                  type="number"
                  value={config.global.max_log_retention || 30}
                  onChange={(e) => handleGlobalFieldChange('max_log_retention', e.target.value)}
                  className="w-full rounded-lg px-3 py-1.5 outline-none transition font-mono focus:border-blue-500"
                  style={{ backgroundColor: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--fg)' }}
                />
              </div>
              {/* 最大日誌行數 */}
              <div className="space-y-1.5">
                <label className="text-[10px] font-bold uppercase select-none" style={{ color: 'var(--meta)' }}>{t("終端保留最大行數")}</label>
                <input
                  type="number"
                  value={config.global.max_log_lines || 500}
                  onChange={(e) => handleGlobalFieldChange('max_log_lines', e.target.value)}
                  className="w-full rounded-lg px-3 py-1.5 outline-none transition font-mono focus:border-blue-500"
                  style={{ backgroundColor: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--fg)' }}
                />
              </div>
            </div>
          </div>

          {/* 3. 資料庫與郵件伺服器 */}
          <div className="rounded-xl p-5 space-y-4" style={{ backgroundColor: 'var(--card)', border: '1px solid var(--border)' }}>
            <h3 className="font-bold text-sm flex items-center gap-2 pb-3 select-none" style={{ color: 'var(--fg)', borderBottom: '1px solid var(--border)' }}>
              <Database size={14} style={{ color: 'var(--status-ok)' }} /> {t("MariaDB 資料庫 & Mailpit 服務設定")}
            </h3>

            <div className="flex items-center justify-between select-none">
              <div>
                <span className="font-semibold block" style={{ color: 'var(--fg-2)' }}>{t("使用外部自訂 MariaDB/MySQL")}</span>
                <span className="text-[10px] mt-0.5 block" style={{ color: 'var(--meta)' }}>{t("手動指定資料庫二進位目錄與數據目錄")}</span>
              </div>
              <input
                type="checkbox"
                checked={config.global.mariadb_external}
                onChange={(e) => handleGlobalFieldChange('mariadb_external', e.target.checked)}
                className="w-3.5 h-3.5 rounded cursor-pointer"
                style={{ backgroundColor: 'var(--input-bg)', borderColor: 'var(--input-border)', accentColor: 'var(--status-ok)' }}
              />
            </div>

            {/* 外部資料庫路徑 */}
            {config.global.mariadb_external && (
              <div className="space-y-3.5 p-4 rounded-xl transition" style={{ border: '1px solid var(--border)', backgroundColor: 'var(--bg-deep)' }}>
                <div className="grid grid-cols-2 gap-3 select-none">
                  <div className="space-y-1">
                    <label className="text-[10px] font-bold uppercase" style={{ color: 'var(--meta)' }}>{t("資料庫引擎類型")}</label>
                    <select
                      value={config.global.mariadb_type || 'MariaDB'}
                      onChange={(e) => handleGlobalFieldChange('mariadb_type', e.target.value)}
                      className="w-full rounded-lg px-2.5 py-1.5 outline-none transition cursor-pointer font-semibold focus:border-blue-500"
                      style={{ backgroundColor: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--fg)' }}
                    >
                      <option value="MariaDB">MariaDB</option>
                      <option value="MySQL">MySQL</option>
                    </select>
                  </div>
                  <div className="space-y-1">
                    <label className="text-[10px] font-bold uppercase" style={{ color: 'var(--meta)' }}>{t("執行 Port")}</label>
                    <input
                      type="number"
                      value={config.global.mariadb_port || 3306}
                      onChange={(e) => handleGlobalFieldChange('mariadb_port', e.target.value)}
                      className="w-full rounded-lg px-2.5 py-1.5 outline-none transition font-mono focus:border-blue-500"
                      style={{ backgroundColor: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--fg)' }}
                    />
                  </div>
                </div>

                <div className="space-y-1.5">
                  <label className="text-[10px] font-bold uppercase block select-none" style={{ color: 'var(--meta)' }}>{t("Binary Path (含 bin 資料夾的根目錄)")}</label>
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={config.global.mariadb_basedir || ""}
                      onChange={(e) => handleGlobalFieldChange('mariadb_basedir', e.target.value)}
                      className="flex-1 rounded-lg px-2.5 py-1 text-xs outline-none transition font-mono focus:border-blue-500"
                      style={{ backgroundColor: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--fg)' }}
                    />
                    <button
                      onClick={() => handleSelectFolder('mariadb_basedir')}
                      className="px-3 py-1 rounded-lg transition font-semibold hover:border-gray-500"
                      style={{ backgroundColor: 'var(--input-bg)', border: '1px solid var(--input-border)' }}
                    >
                      {t("選擇")}
                    </button>
                  </div>
                </div>

                <div className="space-y-1.5">
                  <label className="text-[10px] font-bold uppercase block select-none" style={{ color: 'var(--meta)' }}>{t("Data Path (資料存放目錄)")}</label>
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={config.global.mariadb_datadir || ""}
                      onChange={(e) => handleGlobalFieldChange('mariadb_datadir', e.target.value)}
                      className="flex-1 rounded-lg px-2.5 py-1 text-xs outline-none transition font-mono focus:border-blue-500"
                      style={{ backgroundColor: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--fg)' }}
                    />
                    <button
                      onClick={() => handleSelectFolder('mariadb_datadir')}
                      className="px-3 py-1 rounded-lg transition font-semibold hover:border-gray-500"
                      style={{ backgroundColor: 'var(--input-bg)', border: '1px solid var(--input-border)' }}
                    >
                      {t("選擇")}
                    </button>
                  </div>
                </div>
              </div>
            )}

            {/* Mailpit 端口與設定 */}
            <div className="pt-4 space-y-3.5" style={{ borderTop: '1px solid var(--border)' }}>
              <div className="font-bold text-[10px] uppercase tracking-wider flex items-center gap-1 select-none" style={{ color: 'var(--status-info)' }}>
                <Mail size={12} /> {t("Mailpit 端口配置")}
              </div>
              <div className="grid grid-cols-2 gap-3 select-none">
                <div className="space-y-1">
                  <label className="text-[10px] font-bold uppercase" style={{ color: 'var(--meta)' }}>SMTP Port</label>
                  <input
                    type="number"
                    value={config.global.mailpit_smtp_port || 1025}
                    onChange={(e) => handleGlobalFieldChange('mailpit_smtp_port', e.target.value)}
                    className="w-full rounded-lg px-2.5 py-1.5 outline-none transition font-mono focus:border-blue-500"
                    style={{ backgroundColor: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--fg)' }}
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-[10px] font-bold uppercase" style={{ color: 'var(--meta)' }}>HTTP Port</label>
                  <input
                    type="number"
                    value={config.global.mailpit_http_port || 8025}
                    onChange={(e) => handleGlobalFieldChange('mailpit_http_port', e.target.value)}
                    className="w-full rounded-lg px-2.5 py-1.5 outline-none transition font-mono focus:border-blue-500"
                    style={{ backgroundColor: 'var(--input-bg)', border: '1px solid var(--input-border)', color: 'var(--fg)' }}
                  />
                </div>
              </div>
              <div className="flex items-center justify-between pt-1 select-none">
                <span className="font-semibold" style={{ color: 'var(--fg-2)' }}>{t("使用內置數據庫持久化保存信箱數據")}</span>
                <input
                  type="checkbox"
                  checked={config.global.mailpit_use_db}
                  onChange={(e) => handleGlobalFieldChange('mailpit_use_db', e.target.checked)}
                  className="w-3.5 h-3.5 rounded cursor-pointer"
                  style={{ backgroundColor: 'var(--input-bg)', borderColor: 'var(--input-border)', accentColor: 'var(--status-info)' }}
                />
              </div>
            </div>
          </div>

          {/* 4. 快速編輯系統設定檔 */}
          <div className="rounded-xl p-5 space-y-4" style={{ backgroundColor: 'var(--card)', border: '1px solid var(--border)' }}>
            <h3 className="font-bold text-sm flex items-center gap-2 pb-3 select-none" style={{ color: 'var(--fg)', borderBottom: '1px solid var(--border)' }}>
              <FileText size={14} style={{ color: 'var(--status-warn)' }} /> {t("系統設定檔案快速捷徑")}
            </h3>
            <p className="text-[11px] select-none font-medium" style={{ color: 'var(--meta)' }}>{t("可以在這裡直接使用系統預設編輯器開啟核心設定檔，進行進階手動編輯：")}</p>
            <div className="grid grid-cols-3 gap-3 select-none">
              <button
                onClick={() => handleOpenLocalPath('hosts')}
                className="py-3 px-2 rounded-xl flex flex-col items-center justify-center gap-2 transition hover:border-orange-500/40 hover:bg-orange-500/[0.02]"
                style={{ backgroundColor: 'var(--surface)', border: '1px solid var(--border)' }}
              >
                <Shield size={16} style={{ color: 'var(--status-warn)' }} />
                <span className="font-bold text-xs" style={{ color: 'var(--fg-2)' }}>{t("Hosts 檔案")}</span>
                <span className="text-[10px] font-mono" style={{ color: 'var(--meta)' }}>{t("(Hosts)")}</span>
              </button>
              <button
                onClick={() => handleOpenLocalPath('phpini')}
                className="py-3 px-2 rounded-xl flex flex-col items-center justify-center gap-2 transition hover:border-emerald-500/40 hover:bg-emerald-500/[0.02]"
                style={{ backgroundColor: 'var(--surface)', border: '1px solid var(--border)' }}
              >
                <SettingsIcon size={16} style={{ color: 'var(--status-ok)' }} />
                <span className="font-bold text-xs" style={{ color: 'var(--fg-2)' }}>{t("php.ini 設定")}</span>
                <span className="text-[10px] font-mono" style={{ color: 'var(--meta)' }}>{t("(PHP 全域)")}</span>
              </button>
              <button
                onClick={() => handleOpenLocalPath('wincmpjson')}
                className="py-3 px-2 rounded-xl flex flex-col items-center justify-center gap-2 transition hover:border-blue-500/40 hover:bg-blue-500/[0.02]"
                style={{ backgroundColor: 'var(--surface)', border: '1px solid var(--border)' }}
              >
                <Info size={16} style={{ color: 'var(--accent)' }} />
                <span className="font-bold text-xs" style={{ color: 'var(--fg-2)' }}>{t("WinCMP Json")}</span>
                <span className="text-[10px] font-mono" style={{ color: 'var(--meta)' }}>{t("(核心配置)")}</span>
              </button>
            </div>
          </div>

          {/* 5. 主題選擇 */}
          <div className="rounded-xl p-5 space-y-4" style={{ backgroundColor: 'var(--card)', border: '1px solid var(--border)' }}>
            <h3 className="font-bold text-sm flex items-center gap-2 pb-3 select-none" style={{ color: 'var(--fg)', borderBottom: '1px solid var(--border)' }}>
              <Palette size={14} style={{ color: 'var(--accent)' }} /> {t("外觀主題 (Theme)")}
            </h3>
            <p className="text-[11px] select-none font-medium" style={{ color: 'var(--meta)' }}>{t("選擇您偏好的視覺風格，切換後立即套用。")}</p>
            <div className="grid grid-cols-3 gap-3 select-none">
              {THEMES.map((th) => {
                const isActive = theme === th.id;
                const themeColors: Record<ThemeId, { bg: string; fg: string; accent: string; border: string }> = {
                  carbon: { bg: '#1f2228', fg: '#ffffff', accent: '#ffffff', border: 'rgba(255,255,255,0.2)' },
                  cream: { bg: '#faf9f7', fg: '#1a1916', accent: '#c96442', border: '#e5e0d8' },
                  sketch: { bg: '#f0ebe0', fg: '#2d2b28', accent: '#2b6cb0', border: '#c8c0b0' },
                };
                const tc = themeColors[th.id];
                return (
                  <button
                    key={th.id}
                    onClick={() => {
                      setTheme(th.id);
                      if ((window as any).saveQuickSettingsDebounced) {
                        (window as any).saveQuickSettingsDebounced(th.id, getLanguage());
                      }
                    }}
                    className="py-4 px-3 rounded-xl flex flex-col items-center gap-2 transition relative"
                    style={{
                      background: tc.bg,
                      border: isActive ? `2px solid ${tc.accent}` : `1px solid ${tc.border}`,
                      boxShadow: isActive ? `0 0 0 1px ${tc.accent}` : 'none',
                    }}
                  >
                    {isActive && (
                      <div className="absolute top-2 right-2 w-4 h-4 rounded-full flex items-center justify-center" style={{ background: tc.accent }}>
                        <Check size={10} style={{ color: tc.bg }} />
                      </div>
                    )}
                    <span className="text-sm font-bold" style={{ color: tc.fg, fontFamily: 'var(--font-display)' }}>{th.name}</span>
                    <span className="text-[10px]" style={{ color: tc.accent }}>{th.description}</span>
                  </button>
                );
              })}
            </div>
          </div>
        </div>
      </div>
      <DependencyManager isOpen={showDepManager} onClose={() => setShowDepManager(false)} />
    </div>
  );
}

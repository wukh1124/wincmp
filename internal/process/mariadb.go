package process

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// MariaDBServiceKey 產生 MariaDB 服務的唯一識別 key
func MariaDBServiceKey(version string) string {
	return "mariadb-" + version
}

// StartMariaDB 啟動 MariaDB 服務
// 對應 bat 指令: bin\mariadb\mariadb-11.4.10\bin\mariadbd.exe --defaults-file=conf\my.ini --console
func (m *Manager) StartMariaDB(version string) error {
	serviceKey := MariaDBServiceKey(version)

	if m.IsRunning(serviceKey) {
		return fmt.Errorf("MariaDB %s 已經在運行中", version)
	}

	// 驗證設定檔是否存在
	myIniPath := filepath.Join(m.baseDir, "conf", "my.ini")

	// 執行檔路徑 (根據使用者結構: bin/mariadb/mariadb-version/bin/mariadbd.exe)
	exePath := filepath.Join(m.baseDir, "bin", "mariadb", "mariadb-"+version, "bin", "mariadbd.exe")

	// 檢查資料目錄是否已初始化 (檢查 data/mariadb/mysql 是否存在)
	dataDir := filepath.Join(m.baseDir, "data", "mariadb")
	mysqlDBPath := filepath.Join(dataDir, "mysql")
	if _, err := os.Stat(mysqlDBPath); os.IsNotExist(err) {
		m.log("mariadb", "目錄 %s 不存在，正在初始化 MariaDB 資料庫...", mysqlDBPath)

		if err := os.RemoveAll(dataDir); err != nil {
			return fmt.Errorf("清除資料目錄失敗: %w", err)
		}
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return fmt.Errorf("建立資料目錄失敗: %w", err)
		}

		installDbExe := filepath.Join(m.baseDir, "bin", "mariadb", "mariadb-"+version, "bin", "mariadb-install-db.exe")
		initCmd := m.createCommand(installDbExe,
			"--datadir="+dataDir,
		)

		// 擷取初始化程序的輸出
		m.pipeOutput(initCmd, "mariadb-init", "MariaDB-Init")

		if err := initCmd.Run(); err != nil {
			m.errorLog("mariadb", "MariaDB 資料庫初始化失敗", err)
			return fmt.Errorf("MariaDB 初始化失敗: %w", err)
		}
		m.log("mariadb", "✅ MariaDB 資料庫初始化完成")
	}

	cmd := m.createCommand(exePath,
		"--defaults-file="+myIniPath,
		"--console",
	)

	// 擷取輸出
	m.pipeOutput(cmd, "mariadb", "MariaDB")

	m.log("mariadb", "🚀 啟動 MariaDB %s...", version)
	m.log("mariadb", "  執行檔: %s", exePath)
	m.log("mariadb", "  設定檔: %s", myIniPath)

	if err := cmd.Start(); err != nil {
		m.errorLog("mariadb", fmt.Sprintf("MariaDB %s 啟動失敗", version), err)
		return fmt.Errorf("MariaDB %s 啟動失敗: %w", version, err)
	}

	m.register(serviceKey, "MariaDB "+version, exePath, []*exec.Cmd{cmd})
	m.log("mariadb", "✅ MariaDB %s 已啟動 (PID: %d)", version, cmd.Process.Pid)

	// 監控程序退出
	go m.waitForExit(cmd, serviceKey, "mariadb", "MariaDB "+version)

	return nil
}

// StopMariaDB 停止 MariaDB 服務
func (m *Manager) StopMariaDB(version string) error {
	serviceKey := MariaDBServiceKey(version)

	if !m.IsRunning(serviceKey) {
		return fmt.Errorf("MariaDB %s 未在運行", version)
	}

	m.log("mariadb", "🛑 停止 MariaDB %s...", version)

	// 嘗試優雅關閉：使用 mariadb-admin shutdown (取代 mysqladmin)
	adminExe := filepath.Join(m.baseDir, "bin", "mariadb", "mariadb-"+version, "bin", "mariadb-admin.exe")
	shutdownCmd := m.createCommand(adminExe, "-u", "root", "shutdown")
	if err := shutdownCmd.Run(); err != nil {
		// mariadb-admin 失敗，改用強制終止
		m.errorLog("mariadb", "mariadb-admin shutdown 失敗，改用強制終止", err)
		if err := m.stopService(serviceKey); err != nil {
			m.errorLog("mariadb", fmt.Sprintf("MariaDB %s 停止失敗", version), err)
			return err
		}
	} else {
		// mariadb-admin 成功，等待程序退出
		m.unregister(serviceKey)
	}

	m.log("mariadb", "✅ MariaDB %s 已停止", version)
	return nil
}

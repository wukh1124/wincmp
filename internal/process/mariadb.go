package process

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const MariaDBExternalServiceKey = "mariadb-external"

// safeCleanDataDir 安全清除 data 目錄中的已知暫存檔
// 只刪除 mariadb 初始化已知的暫存檔，避免 RemoveAll 清空整個目錄
func safeCleanDataDir(dataDir string) error {
	// MariaDB 初始化已知的子目錄和暫存檔
	knownEntries := []string{
		"mysql",
		"performance_schema",
		"test",
		"ibdata1",
		"ib_logfile0",
		"ib_logfile1",
		"ib_buffer_pool",
		"auto.cnf",
		"multi-master.info",
		"aria_log_control",
		"aria_log.%",
	}
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		for _, known := range knownEntries {
			if entry.Name() == known || strings.HasPrefix(entry.Name(), strings.TrimSuffix(known, "%")) {
				path := filepath.Join(dataDir, entry.Name())
				os.RemoveAll(path)
				break
			}
		}
	}
	return nil
}

func MariaDBServiceKey(version string) string {
	return "mariadb-" + version
}

func (m *Manager) getDBExecutableName(dbType string) string {
	if strings.EqualFold(dbType, "mysql") {
		return "mysqld.exe"
	}
	return "mariadbd.exe"
}

func (m *Manager) getAdminExecutableName(dbType string) string {
	if strings.EqualFold(dbType, "mysql") {
		return "mysqladmin.exe"
	}
	return "mariadb-admin.exe"
}

func (m *Manager) StartMariaDBAsync(
	version string,
	external bool,
	externalBasedir, externalDatadir, externalType string,
	port int,
) (done chan struct{}, errCh chan error) {
	done = make(chan struct{})
	errCh = make(chan error, 1)

	go func() {
		defer close(done)

		serviceKey := MariaDBExternalServiceKey
		displayName := "外部 " + externalType
		if !external {
			serviceKey = MariaDBServiceKey(version)
			displayName = "MariaDB " + version
		}

		if m.IsRunning(serviceKey) {
			errCh <- fmt.Errorf("%s 已經在運行中", displayName)
			return
		}

		var exePath, dataDir, myIniPath string
		var dbExeName string

		if external {
			dbExeName = m.getDBExecutableName(externalType)
			exePath = filepath.Join(externalBasedir, "bin", dbExeName)
			dataDir = externalDatadir

			if _, err := os.Stat(exePath); os.IsNotExist(err) {
				errCh <- fmt.Errorf("執行檔不存在: %s，請確認 Basedir 路徑正確", exePath)
				return
			}
			if _, err := os.Stat(dataDir); os.IsNotExist(err) {
				errCh <- fmt.Errorf("資料目錄不存在: %s，請先初始化資料庫", dataDir)
				return
			}
		} else {
			dbExeName = m.getDBExecutableName("mariadb")
			myIniPath = filepath.Join(m.baseDir, "conf", "my.ini")
			exePath = filepath.Join(m.baseDir, "bin", "mariadb", "mariadb-"+version, "bin", dbExeName)
			dataDir = filepath.Join(m.baseDir, "data", "mariadb")
			mysqlDBPath := filepath.Join(dataDir, "mysql")

			if _, err := os.Stat(mysqlDBPath); os.IsNotExist(err) {
				m.log("mariadb", "目錄 %s 不存在，正在初始化 MariaDB 資料庫...", mysqlDBPath)

				// 安全清除：只刪除已知暫存檔，避免 RemoveAll 清空整個目錄
				if err := safeCleanDataDir(dataDir); err != nil {
					m.errorLog("mariadb", "清除資料目錄暫存檔失敗", err)
				}
				if err := os.MkdirAll(dataDir, 0700); err != nil {
					errCh <- fmt.Errorf("建立資料目錄失敗: %w", err)
					return
				}

				installDbExe := filepath.Join(m.baseDir, "bin", "mariadb", "mariadb-"+version, "bin", "mariadb-install-db.exe")
				initCmd := m.createCommand(installDbExe, "--datadir="+dataDir)
				m.pipeOutput(initCmd, "mariadb-init", "MariaDB-Init")

				if err := initCmd.Run(); err != nil {
					m.errorLog("mariadb", "MariaDB 資料庫初始化失敗", err)
					errCh <- fmt.Errorf("MariaDB 初始化失敗: %w", err)
					return
				}
				m.log("mariadb", "✅ MariaDB 資料庫初始化完成")
			}
		}

		var cmd *exec.Cmd
		var portLabel string
		if port > 0 {
			portLabel = fmt.Sprintf("%d", port)
		} else {
			portLabel = "3306 (預設)"
		}

		if external {
			args := []string{"--basedir=" + externalBasedir, "--datadir=" + dataDir}
			if port > 0 {
				args = append(args, "--port="+fmt.Sprintf("%d", port))
			}
			args = append(args, "--console")
			cmd = m.createCommand(exePath, args...)
		} else {
			args := []string{"--defaults-file=" + myIniPath}
			if port > 0 {
				args = append(args, "--port="+fmt.Sprintf("%d", port))
			}
			args = append(args, "--console")
			cmd = m.createCommand(exePath, args...)
		}
		m.pipeOutput(cmd, "mariadb", "MariaDB")

		m.log("mariadb", "🚀 啟動 %s...", displayName)
		m.log("mariadb", "  執行檔: %s", exePath)
		m.log("mariadb", "  Port: %s", portLabel)
		if external {
			m.log("mariadb", "  Basedir: %s", externalBasedir)
			m.log("mariadb", "  Datadir: %s", dataDir)
		} else {
			m.log("mariadb", "  設定檔: %s", myIniPath)
		}

		if err := cmd.Start(); err != nil {
			m.errorLog("mariadb", fmt.Sprintf("%s 啟動失敗", displayName), err)
			errCh <- fmt.Errorf("%s 啟動失敗: %w", displayName, err)
			return
		}

		m.register(serviceKey, displayName, exePath, []*exec.Cmd{cmd})
		m.log("mariadb", "✅ %s 已啟動 (PID: %d)", displayName, cmd.Process.Pid)

		go m.waitForExit(cmd, serviceKey, "mariadb", displayName)
		errCh <- nil
	}()

	return done, errCh
}

func (m *Manager) StopMariaDB(
	version string,
	external bool,
	externalBasedir, externalType string,
	port int,
) error {
	serviceKey := MariaDBExternalServiceKey
	displayName := "外部 " + externalType
	if !external {
		serviceKey = MariaDBServiceKey(version)
		displayName = "MariaDB " + version
	}

	if !m.IsRunning(serviceKey) {
		return fmt.Errorf("%s 未在運行", displayName)
	}

	m.log("mariadb", "🛑 停止 %s...", displayName)

	var adminExe string
	if external {
		adminExe = filepath.Join(externalBasedir, "bin", m.getAdminExecutableName(externalType))
	} else {
		adminExe = filepath.Join(m.baseDir, "bin", "mariadb", "mariadb-"+version, "bin", m.getAdminExecutableName("mariadb"))
	}

	shutdownCmd := m.createCommand(adminExe, "-u", "root", "-P", fmt.Sprintf("%d", port), "shutdown")
	if err := shutdownCmd.Run(); err != nil {
		m.errorLog("mariadb", "admin shutdown 失敗，改用強制終止", err)
		if err := m.stopService(serviceKey); err != nil {
			m.errorLog("mariadb", fmt.Sprintf("%s 停止失敗", displayName), err)
			return err
		}
	} else {
		m.unregister(serviceKey)
	}

	m.log("mariadb", "✅ %s 已停止", displayName)
	return nil
}

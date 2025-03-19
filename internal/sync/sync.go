package sync

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"
)

type Config struct {
	AppPackage     string
	DBName         string
	SyncInterval   int
	BackupEnabled  bool
	BackupInterval int
	BackupMaxCount int
}

var (
	lastBackupTime time.Time
	run            = true
)

func RunSync(config Config, sigCh <-chan os.Signal) {
	syncInterval := time.Duration(config.SyncInterval) * time.Second
	backupInterval := time.Duration(config.BackupInterval) * time.Second

	go func() {
		<-sigCh
		fmt.Println("\nStopping database sync...")
		run = false
	}()

	for run {
		syncDatabase(config.AppPackage, config.DBName, config.BackupEnabled, backupInterval, config.BackupMaxCount)
		time.Sleep(syncInterval)
	}

	fmt.Println("Database sync stopped")
}

func syncDatabase(appPackage, dbName string, backupEnabled bool, backupInterval time.Duration, backupMaxCount int) {
	currentTime := time.Now()

	cmd := exec.Command("sh", "-c", fmt.Sprintf("adb shell run-as %s cat databases/%s > %s", appPackage, dbName, dbName))
	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("Error syncing database: %s\n", string(output))
		return
	}

	fileInfo, err := os.Stat(dbName)
	if err != nil {
		fmt.Printf("Sync failed: %s\n", err)
		return
	}

	timestamp := currentTime.Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] Database synced successfully (%d bytes)\n", timestamp, fileInfo.Size())

	if backupEnabled && time.Since(lastBackupTime) > backupInterval {
		createBackup(dbName, backupMaxCount)
		lastBackupTime = currentTime
	}
}

func createBackup(dbPath string, maxCount int) {
	if err := os.MkdirAll("backups", 0755); err != nil {
		fmt.Printf("Error creating backup directory: %s\n", err)
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join("backups", fmt.Sprintf("%s_%s.db", filepath.Base(dbPath), timestamp))

	input, err := os.ReadFile(dbPath)
	if err != nil {
		fmt.Printf("Error reading DB file: %s\n", err)
		return
	}

	if err := os.WriteFile(backupPath, input, 0644); err != nil {
		fmt.Printf("Error creating backup: %s\n", err)
		return
	}

	fmt.Printf("Created backup: %s\n", backupPath)

	files, err := filepath.Glob(filepath.Join("backups", fmt.Sprintf("%s_*.db", filepath.Base(dbPath))))
	if err != nil {
		fmt.Printf("Error listing backup files: %s\n", err)
		return
	}

	if len(files) > maxCount {
		sort.Strings(files)
		for _, oldFile := range files[:len(files)-maxCount] {
			os.Remove(oldFile)
			fmt.Printf("Removed old backup: %s\n", oldFile)
		}
	}
}

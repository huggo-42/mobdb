package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/huggo-42/mobdb/internal/sync"
	"github.com/huggo-42/mobdb/internal/viewer"
)

const (
	DBName = "questionarios.db"
)

func main() {
	port := flag.String("port", "6969", "Port for web interface")
	appPackage := flag.String("app", "androidAppPackageName", "Android app package name")
	syncInterval := flag.Int("sync-interval", 5, "Sync interval in seconds")
	backupEnabled := flag.Bool("backup", true, "Enable database backups")
	backupInterval := flag.Int("backup-interval", 3600, "Backup interval in seconds")
	backupMaxCount := flag.Int("backup-max", 24, "Maximum number of backups to keep")
	flag.Parse()

	syncConfig := sync.Config{
		AppPackage:     *appPackage,
		DBName:         DBName,
		SyncInterval:   *syncInterval,
		BackupEnabled:  *backupEnabled,
		BackupInterval: *backupInterval,
		BackupMaxCount: *backupMaxCount,
	}

	viewerConfig := viewer.Config{
		DBPath: DBName,
		Port:   *port,
	}

	fmt.Printf(`
 _______  _______  ______   ______   ______  
(       )(  ___  )(  ___ \ (  __  \ (  ___ \ 
| () () || (   ) || (   ) )| (  \  )| (   ) )
| || || || |   | || (__/ / | |   ) || (__/ / 
| |(_)| || |   | ||  __ (  | |   | ||  __ (  
| |   | || |   | || (  \ \ | |   ) || (  \ \ 
| )   ( || (___) || )___) )| (__/  )| )___) )
|/     \|(_______)|/ \___/ (______/ |/ \___/ 
    `)
	fmt.Printf("\nDB Manager - Combined Sync & Viewer\n")
	fmt.Printf("-----------------------------------\n")
	fmt.Printf("Database: %s\n", DBName)
	fmt.Printf("App Package: %s\n", *appPackage)
	fmt.Printf("Sync interval: %d seconds\n", *syncInterval)
	fmt.Printf("Backup enabled: %v\n", *backupEnabled)
	if *backupEnabled {
		fmt.Printf("Backup interval: %d seconds\n", *backupInterval)
		fmt.Printf("Max backups: %d\n", *backupMaxCount)
	}
	fmt.Printf("Web interface: http://localhost:%s\n", *port)
	fmt.Printf("-----------------------------------\n")
	fmt.Println("Press Ctrl+C to stop")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	syncDone := make(chan struct{})
	go func() {
		sync.RunSync(syncConfig, sigCh)
		close(syncDone)
	}()

	go func() {
		if err := viewer.StartServer(viewerConfig); err != nil {
			log.Printf("Web server error: %v\n", err)
		}
	}()

	<-sigCh
	fmt.Println("\nShutting down...")
	<-syncDone
	fmt.Println("Shutdown complete")
}

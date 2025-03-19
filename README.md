# mobdb

> [!WARNING]
> This software is unfinished. Keep your expectations low.

The project aims to make a tool for visualizing the database structure and data of a mobile app in real time.

## Demo
![image](https://github.com/user-attachments/assets/056be9ce-8c8e-43a1-9367-8ee23c470e44)


## Supported environments
- [Flutter](https://flutter.dev) with [sqflite](https://pub.dev/packages/sqflite)

## Build
```console
$ make build-static
```

## Running
```console
$ bin/mobdb
 _______  _______  ______   ______   ______
(       )(  ___  )(  ___ \ (  __  \ (  ___ \
| () () || (   ) || (   ) )| (  \  )| (   ) )
| || || || |   | || (__/ / | |   ) || (__/ /
| |(_)| || |   | ||  __ (  | |   | ||  __ (
| |   | || |   | || (  \ \ | |   ) || (  \ \
| )   ( || (___) || )___) )| (__/  )| )___) )
|/     \|(_______)|/ \___/ (______/ |/ \___/
-----------------------------------
Database: database_name.db
App Package: com.app_package
Sync interval: 5 seconds
Backup enabled: true
Backup interval: 3600 seconds
Max backups: 24
Web interface: http://localhost:6969
-----------------------------------
Press Ctrl+C to stop
2025/03/18 21:35:48 Starting DB Viewer on port 6969
[2025-03-18 21:35:48] Database synced successfully (98304 bytes)
Created backup: backups/database_name.db_20250318_213548.db
[2025-03-18 21:35:53] Database synced successfully (98304 bytes)
```

## Flags
| Flags          | Description                       |
| -------------- | --------------------------------- |
| port           | Port for web interface            |
| appPackage     | Android app package name          |
| syncInterval   | Sync interval in seconds          |
| backupEnabled  | Enable database backups           |
| backupInterval | Backup interval in seconds        |
| backupMaxCount | Maximum number of backups to keep |

## Examples
- Running with flag
```console
$ bin/mobdb -port=4242
```


If you have any questions or feedback, please open an issue on this repository.

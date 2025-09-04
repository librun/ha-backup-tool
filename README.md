# ha-backup-tool
Tool for work with Home Assistant Backup

# Install

## Way 1 - Download by bash (for Linux and MacOS)

```bash
wget -qO- https://github.com/librun/ha-backup-tool/releases/latest/download/ha-backup-tool-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m).tar.gz | tar -xz
```

## Way 2 - Download binary (for Linux, MacOS, Windows)
Open [link](https://github.com/librun/ha-backup-tool/releases) choose you OS & platrom and download file
Unpack file and use

## Way 3 - Build yourself (for All)

run comand:
```bash
go install github.com/librun/ha-backup-tool@latest
```

# Use

## NAME

ha-backup-tool - Home Assistant Tool for work with backup

## SYNOPSIS

ha-backup-tool

```
[--emergency|-e]=[value]
[--max-archive-size]=[value]
[--password|-p]=[value]
[--verbose]
```

**Usage**:

```
ha-backup-tool [GLOBAL OPTIONS] [command [COMMAND OPTIONS]] [ARGUMENTS...]
```

## GLOBAL OPTIONS

**--emergency, -e**="": Filepath for emergency text file

**--max-archive-size**="": Max size for extract archive (default size 500GB)

**--password, -p**="": Password for decrypt backup

**--verbose**: Verbose mode for output more information


## COMMANDS

### extract, unpack, e, u

command for decrypt and extract one or more backups

> :warning: **If you are using Windows OS**: For correct work with symlinks and hard links you must run this command with **administrator rights** or change _Policy management_ from this [article](https://learn.microsoft.com/en-us/previous-versions/windows/it-pro/windows-10/security/threat-protection/security-policy-settings/create-symbolic-links)

**Usage**:
    ha-backup-tool extract [command [command options]] files for extract backup home assistant in tar format

#### OPTIONS

**--exclude, --ec**="": Exclude files (split value by ,)

**--include, --ic**="": Include files (split value by ,)

**--crypto string, -c**="": Type cryptography for decode archive (support values: aes128)

**--output, -o**="": Directory for unpack files

**--skip-create-links**: Skip create symlinks and hard links

#### Example

##### Extract full
Extract N archives by password to same location current files
```bash
ha-backup-tool extract -p XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX dir1/backup1.tar dir2/backup2.tar dir3/backupN.tar
```

Extract N archives by emergency file to different location dir
```bash
ha-backup-tool extract -e dir/emergency_file.txt -o dir/extract_backup dir1/backup1.tar dir2/backup2.tar dir3/backupN.tar
```

##### Extract part
Extract only media archive:
```bash
ha-backup-tool extract -e dir/emergency_file.txt -ic media.tar.gz dir1/backup1.tar
```

Extract media and share archive:
```bash
ha-backup-tool extract -e dir/emergency_file.txt -ic media*,share* dir1/backup1.tar
```

extract archive whose file name starts with core:
```bash
ha-backup-tool extract -e dir/emergency_file.txt -ic core* dir1/backup1.tar
```

Extract archive whose file name have influxdb
```bash
ha-backup-tool extract -e dir/emergency_file.txt -ic *influxdb* dir1/backup1.tar
```

extract archive whose file name starts with core and exclude archive whose file name end with server.tar.gz
```bash
ha-backup-tool extract -e dir/emergency_file.txt -ic core* -ec *server.tar.gz dir1/backup1.tar
```

## Shell Completions

For install completions run command
```
ha-backup-tool completion --help
```
And read and run instruction

# Related projects

* https://github.com/sabeechen/decrypt-ha-backup 
* https://github.com/cogneato/ha-decrypt-backup-tool - this tool was reference for text messages and base AES-SBS decrypt.
* https://github.com/azzieg/ha-backup-tool - this tool was reference for new decrypt method.
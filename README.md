# ha-backup-tool
Tool for work with Home Assistant Backup

# Install

## Way 1 - Download binary
Open [link](https://github.com/librun/ha-backup-tool/releases) choose you platrom and os and download file

## Way 2 - Build yourself

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
[-b|--backup]=[value]
[-e|--emergency]=[value]
[-o|--output]=[value]
[-p|--password]=[value]
```

**Usage**:

```
ha-backup-tool [GLOBAL OPTIONS] [command [COMMAND OPTIONS]] [ARGUMENTS...]
```

## GLOBAL OPTIONS

**-b, --backup**="": Filepath for backup home assistant in tar format

**-e, --emergency**="": Filepath for emergency text file

**-o, --output**="": Directory for unpack files

**-p, --password**="": Password for decrypt backup

## Shell Completions

For install completions run command
```
ha-backup-tool completion bash
```
And read and run instruction

# Related projects

* https://github.com/sabeechen/decrypt-ha-backup 
* https://github.com/cogneato/ha-decrypt-backup-tool - this tool was reference for text messages and base AES-SBS decrypt.
* https://github.com/azzieg/ha-backup-tool - this tool was reference for new decrypt method.
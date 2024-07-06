# miservice

## Install

`go install github.com/caeret/miservice/cmd/micli@latest`

## Command Line

```bash
NAME:
   micli - cli for xiao mi devices

USAGE:
   micli [global options] command [command options]

COMMANDS:
   action   send action to specified device
   devices  list devices
   spec     get spec for specified device type
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --username username  xiaomi account's username [$MI_USER]
   --password password  xiaomi account's password [$MI_PASS]
   --help, -h           show help
```

### List Devices

```bash
export MI_USER=小米账号
export MI_PASS=账号密码
micli devices
```

### Send Action to Device

```bash
micli action [DID] [iid] [args...]
```
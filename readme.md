## `init-exporter` [![Build Status](https://travis-ci.org/funbox/init-exporter.svg?branch=master)](https://travis-ci.org/funbox/init-exporter) [![Go Report Card](https://goreportcard.com/badge/github.com/funbox/init-exporter)](https://goreportcard.com/report/github.com/funbox/init-exporter) [![License](https://gh.kaos.io/mit.svg)](LICENSE)

Utility for exporting services described by Procfile to init system.
Supported init systems: upstart and systemd

* [Installation](#installation)
* [Configuration](#configuration)
* [Usage](#usage)
  * [Procfile v.1](#procfile-v1)
  * [Procfile v.2](#procfile-v2)
* [Exporting](#exporting)
* [Command options](#command-options)
* [Build status](#build-status)
* [License](#license)

### Installation

#### From source

Before the initial install allows git to use redirects for [pkg.re](https://github.com/essentialkaos/pkgre) service (_reason why you should do this described [here](https://github.com/essentialkaos/pkgre#git-support)_):

```
git config --global http.https://pkg.re.followRedirects true
```

To build the init-exporter from scratch, make sure you have a working Go 1.5+ workspace ([instructions](https://golang.org/doc/install)), then:

```bash
go get -d github.com/funbox/init-exporter
cd $GOPATH/src/github.com/funbox/init-exporter
make deps
make all
[sudo] make install
```

#### From ESSENTIAL KAOS Public repo for RHEL6/CentOS6

```
[sudo] yum install -y https://yum.kaos.io/6/release/x86_64/kaos-repo-8.0-0.el6.noarch.rpm
[sudo] yum install init-exporter
```

#### From ESSENTIAL KAOS Public repo for RHEL7/CentOS7

```
[sudo] yum install -y https://yum.kaos.io/7/release/x86_64/kaos-repo-8.0-0.el7.noarch.rpm
[sudo] yum install init-exporter
```

### Configuration

The export process can be configured through the config `/etc/init-exporter.conf`:

```ini
# Default configuration for init-exporter

[main]

  # Default run user
  run-user: service

  # Default run group
  run-group: service

  # Prefix used for exported units and helpers
  prefix: fb-

[paths]

  # Working dir
  working-dir: /tmp

  # Path to directory with helpers
  helper-dir: /var/local/init-exporter/helpers

  # Path to directory with systemd configs
  systemd-dir: /etc/systemd/system

  # Path to directory with upstart configs
  upstart-dir: /etc/init

[defaults]

  # Number of Processes (0 - disabled)
  nproc: 10240

  # Number of File Descriptors (0 - disabled)
  nofile: 10240

  # Enable or disable respawn by default here
  respawn: true

  # Respawn count
  respawn-count: 10

  # Respawn interval
  respawn-interval: 15

  # Kill timeout (0 - disabled)
  kill-timeout: 60

[log]

  # Enable or disable logging here
  enabled: true

  # Log file directory
  dir: /var/log/init-exporter

  # Path to log file
  file: {log:dir}/init-exporter.log

  # Default log file permissions
  perms: 0644

  # Minimal log level (debug/info/warn/error/crit)
  level: info
```

To give a certain user (i.e. `deployuser`) the ability to use this script, you can place the following lines into `sudoers` file:

```bash
# Commands required for manipulating jobs
Cmnd_Alias UPSTART = /sbin/start, /sbin/stop, /sbin/restart
Cmnd_Alias SYSTEMD = /usr/bin/systemctl
Cmnd_Alias EXPORTER = /usr/local/bin/init-exporter

...

# Allow deploy user to manipulate jobs
deployuser        ALL=(deployuser) NOPASSWD: ALL, (root) NOPASSWD: UPSTART, SYSTEMD, EXPORTER
```

### Usage

`init-exporter` is able to process two versions of Procfiles. Utility automatically recognise used format.

#### Procfile v.1

After init-exporter is installed and configured, you may export background jobs
from an arbitrary Procfile-like file of the following format:

```yaml
cmdlabel1: cmd1
cmdlabel2: cmd2
```

i.e. a file `./myprocfile` containing:

```yaml
my_tail_cmd: /usr/bin/tail -F /var/log/messages
my_another_tail_cmd: /usr/bin/tail -F /var/log/messages
```

For security purposes, command labels are allowed to contain only letters, digits, and underscores.

#### Procfile v.2

Another format of Procfile scripts is YAML config. A configuration script may
look like this:

```yaml
version: 2

start_on_runlevel: 3
stop_on_runlevel: 3

env:
  RAILS_ENV: production
  TEST: true

working_directory: /srv/projects/my_website/current

commands:
  my_tail_cmd:
    command: /usr/bin/tail -F /var/log/messages
    respawn:
      count: 5
      interval: 10
    env:
      RAILS_ENV: staging # if needs to be redefined or extended
    working_directory: '/var/...' # if needs to be redefined
  
  my_another_tail_cmd:
    command: /usr/bin/tail -F /var/log/messages
    kill_timeout: 60
    kill_signal: SIGQUIT
    reload_signal: SIGUSR2
    env_file: shared/staging.env
    respawn: false # by default respawn option is enabled
  
  my_one_another_tail_cmd:
    pre: /usr/bin/echo pre_command
    command: /usr/bin/tail -F /var/log/messages
    post: /usr/bin/echo post_command
    log: /var/log/messages_copy
  
  my_multi_tail_cmd:
    command: /usr/bin/tail -F /var/log/messages
    count: 2
```

`start_on_runlevel` and `stop_on_runlevel` are two global options that can't be
redefined per-command.

`working_directory` will generate the following line:

```bash
cd 'your/working/directory' && your_command
```

`env` params can be redefined and extended in per-command options. Note that
you can't remove a globally defined `env` variable.
For Procfile example given earlier the generated command will look like:

```bash
env RAILS_ENV=staging TEST=true your_command
```

`log` option lets you override the default log location (`/var/log/fb-my_website/my_one_another_tail_cmd.log`).

`kill_timeout` option lets you override the default process kill timeout of 30 seconds.

`kill_signal` specifies which signal to use when killing a service.

`reload_signal` specifies which signal to use when reloading a service.

`env_file` absolute or relative path to file with environment variables.

`respawn` option controls how often the job can fail. If the job restarts more
often than `count` times in `interval`, it won't be restarted anymore.

Options `working_directory`, `env`, `log`, `respawn` can be
defined both as global and as per-command options.

### Exporting

To export a Procfile you should run

```bash
sudo init-exporter -p ./myprocfile -f format myapp
```
Where `myapp` is the application name. This name only affects the names of generated files. For security purposes, app name is also allowed to contain only letters, digits and underscores.

Format is name of init system `(upstart | systemd)`.

Assuming that default options are used, the following files and folders will be generated (in case of upstart format):

in `/etc/init/`:

```
fb-myapp-my_another_tail_cmd.conf
fb-myapp-my_tail_cmd.conf
fb-myapp.conf
```

in `/var/local/init-exporter/helpers`:

```
fb-myapp-my_another_tail_cmd.sh
fb-myapp-my_tail_cmd.sh
```

Prefix `fb-` (which can be customised through config) is added to avoid collisions with other jobs.
After this `my_tail_cmd`, for example, will be able to be started as an Upstart job:

```bash
sudo start fb-myapp-my_tail_cmd
...
sudo stop fb-myapp-my_tail_cmd
```

It's stdout/stderr will be redirected to `/var/log/fb-myapp/my_tail_cmd.log`.

To start/stop all application commands at once, you can run:

```bash
sudo start fb-myapp
...
sudo stop fb-myapp
```

To remove init scripts and helpers for a particular application you can run

```bash
sudo init-exporter -u -f upstart myapp
```

The logs are not cleared in this case. Also, all old application scripts are cleared before each export.

### Command options

```
Usage: init-exporter {options} app-name

Options

  --procfile, -p file             Path to procfile
  --dry-start, -d                 Dry start (don't export anything, just parse and test procfile)
  --disable-validation, -D        Disable application validation
  --unistall, -u                  Remove scripts and helpers for a particular application
  --format, -f upstart|systemd    Format of generated configs
  --no-colors, -nc                Disable colors in output
  --help, -h                      Show this help message
  --version, -v                   Show version

Examples

  init-exporter -p ./myprocfile -f systemd myapp
  Export given procfile to systemd as myapp

  init-exporter -u -f systemd myapp
  Uninstall myapp from systemd

  init-exporter -p ./myprocfile -f upstart myapp
  Export given procfile to upstart as myapp

  init-exporter -u -f upstart myapp
  Uninstall myapp from upstart

```

### Build status

| Repository | Status |
|------------|--------|
| Stable | [![Build Status](https://travis-ci.org/funbox/init-exporter.svg?branch=master)](https://travis-ci.org/funbox/init-exporter) |
| Unstable | [![Build Status](https://travis-ci.org/funbox/init-exporter.svg?branch=develop)](https://travis-ci.org/funbox/init-exporter) |

### License

init-exporter is released under the MIT license (see [LICENSE](LICENSE))

[![Sponsored by FunBox](https://funbox.ru/badges/sponsored_by_funbox_grayscale.svg)](https://funbox.ru)

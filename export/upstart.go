package export

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                           Copyright (c) 2006-2023 FUNBOX                           //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v12/strutil"
	"github.com/essentialkaos/ek/v12/timeutil"
	"github.com/essentialkaos/ek/v12/version"

	"github.com/funbox/init-exporter/procfile"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// UpstartProvider is upstart export provider
type UpstartProvider struct{}

// ////////////////////////////////////////////////////////////////////////////////// //

// TEMPLATE_UPSTART_HELPER contains default helper template
const TEMPLATE_UPSTART_HELPER = `#!/bin/bash

# This helper generated {{.ExportDate}} by init-exporter/upstart for {{.Application.Name}} application

[[ -r /etc/profile.d/rbenv.sh ]] && source /etc/profile.d/rbenv.sh
[[ -r /etc/profile.d/pyenv.sh ]] && source /etc/profile.d/pyenv.sh

cd {{.Service.Options.WorkingDir}} && {{ if .Service.HasPreCmd }}{{.Service.GetCommandExec "pre"}} && {{ end }}{{.Service.GetCommandExec ""}}{{ if .Service.HasPostCmd }} && {{.Service.GetCommandExec "post"}}{{ end }}
`

// TEMPLATE_UPSTART_APP contains default application template
const TEMPLATE_UPSTART_APP = `# This unit generated {{.ExportDate}} by init-exporter/upstart for {{.Application.Name}} application

start on {{.StartLevel}}
stop on {{.StopLevel}}

pre-start script

bash << "EOF"
  mkdir -p /var/log/{{.Application.Name}}
  chown -R {{.Application.User}} /var/log/{{.Application.Name}}
  chgrp -R {{.Application.Group}} /var/log/{{.Application.Name}}
  chmod -R g+w /var/log/{{.Application.Name}}
EOF

end script
`

// TEMPLATE_UPSTART_SERVICE contains default service template
const TEMPLATE_UPSTART_SERVICE = `# This unit generated {{.ExportDate}} by init-exporter/upstart for {{.Application.Name}} application

start on {{.StartLevel}}
stop on {{.StopLevel}}

{{ if .Service.Options.IsRespawnEnabled }}respawn{{ end }}
{{ if .Service.Options.IsRespawnLimitSet }}respawn limit {{.Service.Options.RespawnCount}} {{.Service.Options.RespawnInterval}}{{ end }}
{{ if and .Service.Options.IsRespawnLimitSet (gt .Service.Options.RespawnDelay 0) }}post-stop exec sleep {{.Service.Options.RespawnDelay}}{{ end }}

kill timeout {{.Service.Options.KillTimeout}}
{{ if .Service.Options.IsKillSignalSet }}kill signal {{.Service.Options.KillSignal}}{{ end }}
{{ if .Service.Options.IsReloadSignalSet }}reload signal {{.Service.Options.ReloadSignal}}{{ end }}

{{ if .Service.Options.IsFileLimitSet }}limit nofile {{.Service.Options.LimitFile}} {{.Service.Options.LimitFile}}{{ end }}
{{ if .Service.Options.IsProcLimitSet }}limit nproc {{.Service.Options.LimitProc}} {{.Service.Options.LimitProc}}{{ end }}
{{ if .Service.Options.IsMemlockLimitSet }}limit memlock {{.GetMemlockLimit}} {{.GetMemlockLimit}}{{ end }}

script
  touch /var/log/{{.Application.Name}}/{{.Service.Name}}.log
  chown {{.Application.User}} /var/log/{{.Application.Name}}/{{.Service.Name}}.log
  chgrp {{.Application.Group}} /var/log/{{.Application.Name}}/{{.Service.Name}}.log
  chmod g+w /var/log/{{.Application.Name}}/{{.Service.Name}}.log
  exec sudo -u {{.Application.User}} /bin/bash {{.Service.HelperPath}} &>>/var/log/{{.Application.Name}}/{{.Service.Name}}.log
end script
`

// ////////////////////////////////////////////////////////////////////////////////// //

type upstartAppData struct {
	Application *procfile.Application
	ExportDate  string
	StartLevel  string
	StopLevel   string
}

type upstartServiceData struct {
	Application *procfile.Application
	Service     *procfile.Service
	ExportDate  string
	StartLevel  string
	StopLevel   string
}

// ////////////////////////////////////////////////////////////////////////////////// //

var upstartVersionCache version.Version

// ////////////////////////////////////////////////////////////////////////////////// //

// NewUpstart creates new UpstartProvider struct
func NewUpstart() *UpstartProvider {
	return &UpstartProvider{}
}

// ////////////////////////////////////////////////////////////////////////////////// //

// CheckRequirements checks provider requirements for given application
func (up *UpstartProvider) CheckRequirements(app *procfile.Application) error {
	return checkReloadSignalSupport(app)
}

// UnitName returns unit name with extension
func (up *UpstartProvider) UnitName(name string) string {
	return name + ".conf"
}

// EnableService enables service with given name
func (up *UpstartProvider) EnableService(appName string) error {
	return nil
}

// DisableService disables service with given name
func (up *UpstartProvider) DisableService(appName string) error {
	return nil
}

// Reload reloads service units
func (up *UpstartProvider) Reload() error {
	return nil
}

// RenderAppTemplate renders unit template data with given app data and return
// app unit code
func (up *UpstartProvider) RenderAppTemplate(app *procfile.Application) (string, error) {
	data := &upstartAppData{
		Application: app,
		StartLevel:  up.renderStartLevel(app.StartLevel, app.StartDevice, app.Depends),
		StopLevel:   up.renderStopLevel(app.StopLevel, app.Depends),
		ExportDate:  timeutil.Format(time.Now(), "%Y/%m/%d %H:%M:%S"),
	}

	return renderTemplate("upstart-app-template", TEMPLATE_UPSTART_APP, data)
}

// RenderServiceTemplate renders unit template data with given service data and
// return service unit code
func (up *UpstartProvider) RenderServiceTemplate(service *procfile.Service) (string, error) {
	data := &upstartServiceData{
		Application: service.Application,
		Service:     service,
		StartLevel:  fmt.Sprintf("starting %s", service.Application.Name),
		StopLevel:   fmt.Sprintf("stopping %s", service.Application.Name),
		ExportDate:  timeutil.Format(time.Now(), "%Y/%m/%d %H:%M:%S"),
	}

	return renderTemplate("upstart-service-template", TEMPLATE_UPSTART_SERVICE, data)
}

// RenderHelperTemplate renders helper template data with given service data and
// return helper script code
func (up *UpstartProvider) RenderHelperTemplate(service *procfile.Service) (string, error) {
	data := &upstartServiceData{
		Application: service.Application,
		Service:     service,
		ExportDate:  timeutil.Format(time.Now(), "%Y/%m/%d %H:%M:%S"),
	}

	return renderTemplate("upstart-helper-template", TEMPLATE_UPSTART_HELPER, data)
}

// RenderReloadHelperTemplate renders helper template data for reloading services
func (up *UpstartProvider) RenderReloadHelperTemplate(app *procfile.Application) (string, error) {
	return "", nil
}

// ////////////////////////////////////////////////////////////////////////////////// //

// renderStartLevel converts level number to upstart start level name
func (up *UpstartProvider) renderStartLevel(level int, device string, deps []string) string {
	if device == "" && len(deps) == 0 {
		return fmt.Sprintf("runlevel [%d]", level)
	}

	var depsList []string

	if device != "" {
		depsList = append(depsList, "net-device-up IFACE="+device)
	}

	for _, dep := range deps {
		depsList = append(depsList, "started "+dep)
	}

	return strings.Join(depsList, " and ")
}

// renderStopLevel converts level number to upstart stop level name
func (up *UpstartProvider) renderStopLevel(level int, deps []string) string {
	if len(deps) == 0 {
		return fmt.Sprintf("runlevel [%d]", level)
	}

	var depsList []string

	for _, dep := range deps {
		depsList = append(depsList, "stopped "+dep)
	}

	return strings.Join(depsList, " and ")
}

// ////////////////////////////////////////////////////////////////////////////////// //

// GetMemlockLimit returns formatted memlock value
func (d *upstartServiceData) GetMemlockLimit() string {
	if d.Service.Options.LimitMemlock == -1 {
		return "unlimited"
	}

	return fmt.Sprintf("%d", d.Service.Options.LimitMemlock)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// checkReloadSignalSupport checks if app requires reload signal
// and if current upstart version supports it
func checkReloadSignalSupport(app *procfile.Application) error {
	if !app.IsReloadSignalSet() {
		return nil
	}

	if upstartVersionCache.IsZero() {
		upstartVersion, err := getUpstartVersion()

		if err != nil {
			return err
		}

		upstartVersionCache = upstartVersion
	}

	minReloadVer, _ := version.Parse("1.10.0")

	if upstartVersionCache.Less(minReloadVer) {
		return fmt.Errorf(
			"Upstart %s doesn't support reload signal. Upstart 1.10.0 is required.",
			upstartVersionCache.Simple(),
		)
	}

	return nil
}

// getUpstartVersion returns current upstart version
func getUpstartVersion() (version.Version, error) {
	cmd := exec.Command("init", "--version")
	output, err := cmd.Output()

	if err != nil {
		return version.Version{}, fmt.Errorf("Can't execute init binary")
	}

	return parseUpstartVersionData(string(output))
}

// parseUpstartVersionData parses upstart version data
func parseUpstartVersionData(data string) (version.Version, error) {
	line := strutil.ReadField(data, 0, false, '\n')
	verStr := strings.Trim(strutil.ReadField(line, 2, false, ' '), "()")

	return version.Parse(verStr)
}

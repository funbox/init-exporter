package export

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                       Copyright (c) 2006-2017 FB GROUP LLC                         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"pkg.re/essentialkaos/ek.v9/system/exec"
	"pkg.re/essentialkaos/ek.v9/timeutil"

	"github.com/funbox/init-exporter/procfile"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// SystemdProvider is systemd export provider
type SystemdProvider struct{}

// ////////////////////////////////////////////////////////////////////////////////// //

// TEMPLATE_SYSTEMD_HELPER contains default helper template
const TEMPLATE_SYSTEMD_HELPER = `#!/bin/bash

# This helper generated {{.ExportDate}} by init-exporter/systemd for {{.Application.Name}} application

[[ -r /etc/profile.d/rbenv.sh ]] && source /etc/profile.d/rbenv.sh

{{ if .Service.HasPreCmd }}{{.Service.GetCommandExec "pre"}} && {{ end }}{{.Service.GetCommandExec ""}}{{ if .Service.HasPostCmd }} && {{.Service.GetCommandExec "post"}}{{ end }}
`

// TEMPLATE_SYSTEMD_APP contains default application template
const TEMPLATE_SYSTEMD_APP = `# This unit generated {{.ExportDate}} by init-exporter/systemd for {{.Application.Name}} application

[Unit]

Description=Unit for {{.Application.Name}} application
After={{.StartLevel}}
{{.Wants}}

[Service]
Type=oneshot
RemainAfterExit=true

ExecStartPre=/bin/mkdir -p /var/log/{{.Application.Name}}
ExecStartPre=/bin/chown -R {{.Application.User}} /var/log/{{.Application.Name}}
ExecStartPre=/bin/chgrp -R {{.Application.Group}} /var/log/{{.Application.Name}}
ExecStartPre=/bin/chmod -R g+w /var/log/{{.Application.Name}}
ExecStart=/bin/echo "{{.Application.Name}} started"
ExecStop=/bin/echo "{{.Application.Name}} stopped"

[Install]
WantedBy={{.StartLevel}}
`

// TEMPLATE_SYSTEMD_SERVICE contains default service template
const TEMPLATE_SYSTEMD_SERVICE = `# This unit generated {{.ExportDate}} by init-exporter/systemd for {{.Application.Name}} application

[Unit]

Description=Unit for {{.Service.Name}} service (part of {{.Application.Name}} application)
PartOf={{.Application.Name}}.service

[Service]
Type=simple

{{ if .Service.Options.IsKillSignalSet }}KillSignal={{.Service.Options.KillSignal}}{{ end }}
TimeoutStopSec={{.Service.Options.KillTimeout}}
{{ if .Service.Options.IsRespawnEnabled }}Restart=on-failure{{ end }}
{{ if .Service.Options.IsRespawnLimitSet }}StartLimitInterval={{.Service.Options.RespawnInterval}}{{ end }}
{{ if .Service.Options.IsRespawnLimitSet }}StartLimitBurst={{.Service.Options.RespawnCount}}{{ end }}

{{ if .Service.Options.IsFileLimitSet }}LimitNOFILE={{.Service.Options.LimitFile}}{{ end }}
{{ if .Service.Options.IsProcLimitSet }}LimitNPROC={{.Service.Options.LimitProc}}{{ end }}

ExecStartPre=/bin/touch /var/log/{{.Application.Name}}/{{.Service.Name}}.log
ExecStartPre=/bin/chown {{.Application.User}} /var/log/{{.Application.Name}}/{{.Service.Name}}.log
ExecStartPre=/bin/chgrp {{.Application.Group}} /var/log/{{.Application.Name}}/{{.Service.Name}}.log
ExecStartPre=/bin/chmod g+w /var/log/{{.Application.Name}}/{{.Service.Name}}.log

User={{.Application.User}}
Group={{.Application.Group}}
WorkingDirectory={{.Service.Options.WorkingDir}}
ExecStart=/bin/sh -c '/bin/bash {{.Service.HelperPath}} &>>/var/log/{{.Application.Name}}/{{.Service.Name}}.log'
{{ if .Service.Options.IsReloadSignalSet }}ExecReload=/bin/kill -{{.Service.Options.ReloadSignal}} $MAINPID{{ end }}
`

// ////////////////////////////////////////////////////////////////////////////////// //

type systemdAppData struct {
	Application *procfile.Application
	ExportDate  string
	StartLevel  string
	StopLevel   string
	Wants       string
}

type systemdServiceData struct {
	Application *procfile.Application
	Service     *procfile.Service
	ExportDate  string
	StartLevel  string
	StopLevel   string
}

// ////////////////////////////////////////////////////////////////////////////////// //

// NewSystemd create new SystemdProvider struct
func NewSystemd() *SystemdProvider {
	return &SystemdProvider{}
}

// ////////////////////////////////////////////////////////////////////////////////// //

// UnitName return unit name with extension
func (sp *SystemdProvider) UnitName(name string) string {
	return name + ".service"
}

// EnableService enable service with given name
func (sp *SystemdProvider) EnableService(appName string) error {
	err := exec.Run("systemctl", "enable", sp.UnitName(appName))

	if err != nil {
		return errors.New("Can't enable service through systemctl")
	}

	return nil
}

// DisableService disable service with given name
func (sp *SystemdProvider) DisableService(appName string) error {
	err := exec.Run("systemctl", "disable", sp.UnitName(appName))

	if err != nil {
		return errors.New("Can't disable service through systemctl")
	}

	return nil
}

// Reload reload service units
func (sp *SystemdProvider) Reload() error {
	err := exec.Run("systemctl", "daemon-reload")

	if err != nil {
		return errors.New("Can't reload units through systemctl")
	}

	return nil
}

// RenderAppTemplate render unit template data with given app data and return
// app unit code
func (sp *SystemdProvider) RenderAppTemplate(app *procfile.Application) (string, error) {
	data := &systemdAppData{
		Application: app,
		Wants:       sp.renderWantsClause(app),
		StartLevel:  sp.renderLevel(app.StartLevel),
		StopLevel:   sp.renderLevel(app.StopLevel),
		ExportDate:  timeutil.Format(time.Now(), "%Y/%m/%d %H:%M:%S"),
	}

	return renderTemplate("systemd-app-template", TEMPLATE_SYSTEMD_APP, data)
}

// RenderServiceTemplate render unit template data with given service data and
// return service unit code
func (sp *SystemdProvider) RenderServiceTemplate(service *procfile.Service) (string, error) {
	data := systemdServiceData{
		Application: service.Application,
		Service:     service,
		StartLevel:  sp.renderLevel(service.Application.StartLevel),
		StopLevel:   sp.renderLevel(service.Application.StopLevel),
		ExportDate:  timeutil.Format(time.Now(), "%Y/%m/%d %H:%M:%S"),
	}

	return renderTemplate("systemd-service-template", TEMPLATE_SYSTEMD_SERVICE, data)
}

// RenderHelperTemplate render helper template data with given service data and
// return helper script code
func (sp *SystemdProvider) RenderHelperTemplate(service *procfile.Service) (string, error) {
	data := systemdServiceData{
		Application: service.Application,
		Service:     service,
		StartLevel:  sp.renderLevel(service.Application.StartLevel),
		StopLevel:   sp.renderLevel(service.Application.StopLevel),
		ExportDate:  timeutil.Format(time.Now(), "%Y/%m/%d %H:%M:%S"),
	}

	return renderTemplate("systemd-helper-template", TEMPLATE_SYSTEMD_HELPER, data)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// renderLevel convert level number to upstart level name
func (sp *SystemdProvider) renderLevel(level int) string {
	switch level {
	case 1:
		return "rescue.target"
	case 5:
		return "graphical.target"
	case 6:
		return "reboot.target"
	default:
		return "multi-user.target"
	}
}

// renderWantsClause render list of services in application for upstart config
func (sp *SystemdProvider) renderWantsClause(app *procfile.Application) string {
	var wants []string
	var buffer string

	for _, service := range app.Services {
		if service.Options.Count <= 0 {
			unitName := sp.UnitName(app.Name+"-"+service.Name) + " "

			if len(buffer)+len(unitName) >= 1536 {
				wants = append(wants, strings.TrimSpace(buffer))
				buffer = ""
			}

			buffer += unitName
		} else {
			for i := 1; i <= service.Options.Count; i++ {
				unitName := sp.UnitName(app.Name+"-"+service.Name+strconv.Itoa(i)) + " "

				if len(buffer)+len(unitName) >= 1536 {
					wants = append(wants, "Wants="+strings.TrimSpace(buffer))
					buffer = ""
				}

				buffer += unitName
			}
		}
	}

	wants = append(wants, "Wants="+strings.TrimSpace(buffer))

	return strings.Join(wants, "\n")
}

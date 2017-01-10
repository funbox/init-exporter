package export

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                       Copyright (c) 2006-2017 FB GROUP LLC                         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"strings"
	"time"

	"pkg.re/essentialkaos/ek.v6/system"
	"pkg.re/essentialkaos/ek.v6/timeutil"

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

exec {{.Service.Cmd}}
`

// TEMPLATE_SYSTEMD_APP contains default application template
const TEMPLATE_SYSTEMD_APP = `# This unit generated {{.ExportDate}} by init-exporter/systemd for {{.Application.Name}} application

[Unit]

Description=Unit for {{.Application.Name}} application
After={{.StartLevel}}
Wants={{.Wants}}

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

TimeoutStopSec={{.Service.Options.KillTimeout}}
{{ if .Service.Options.RespawnEnabled }}Restart=on-failure{{ end }}
{{ if .Service.Options.RespawnLimitSet }}StartLimitInterval={{.Service.Options.RespawnInterval}}{{ end }}
{{ if .Service.Options.RespawnLimitSet }}StartLimitBurst={{.Service.Options.RespawnCount}}{{ end }}

ExecStartPre=/bin/touch /var/log/{{.Application.Name}}/{{.Service.Name}}.log
ExecStartPre=/bin/chown {{.Application.User}} /var/log/{{.Application.Name}}/{{.Service.Name}}.log
ExecStartPre=/bin/chgrp {{.Application.Group}} /var/log/{{.Application.Name}}/{{.Service.Name}}.log
ExecStartPre=/bin/chmod g+w /var/log/{{.Application.Name}}/{{.Service.Name}}.log

User={{.Application.User}}
Group={{.Application.Group}}
WorkingDirectory={{.Service.Options.WorkingDir}}
{{ if .Service.Options.EnvSet }}Environment={{.Service.Options.EnvString}}{{ end }}
ExecStart=/bin/bash {{.Service.HelperPath}} {{ if .Service.Options.CustomLogEnabled }}>> {{.Service.Options.LogPath}} {{end}}>> /var/log/{{.Application.Name}}/{{.Service.Name}}.log 2>&1
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
	return system.Exec("systemctl", "enable", sp.UnitName(appName))
}

// DisableService disable service with given name
func (sp *SystemdProvider) DisableService(appName string) error {
	return system.Exec("systemctl", "disable", sp.UnitName(appName))
}

// RenderAppTemplate render unit template data with given app data and return
// app unit code
func (sp *SystemdProvider) RenderAppTemplate(app *procfile.Application) (string, error) {
	data := &systemdAppData{
		Application: app,
		Wants:       sp.renderWantsClause(app),
		StartLevel:  sp.randerLevel(app.StartLevel),
		StopLevel:   sp.randerLevel(app.StopLevel),
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
		StartLevel:  sp.randerLevel(service.Application.StartLevel),
		StopLevel:   sp.randerLevel(service.Application.StopLevel),
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
		StartLevel:  sp.randerLevel(service.Application.StartLevel),
		StopLevel:   sp.randerLevel(service.Application.StopLevel),
		ExportDate:  timeutil.Format(time.Now(), "%Y/%m/%d %H:%M:%S"),
	}

	return renderTemplate("systemd-helper-template", TEMPLATE_SYSTEMD_HELPER, data)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// randerLevel convert level number to upstart level name
func (sp *SystemdProvider) randerLevel(level int) string {
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

	for _, service := range app.Services {
		wants = append(wants, sp.UnitName(app.Name+"-"+service.Name))
	}

	return strings.Join(wants, " ")
}

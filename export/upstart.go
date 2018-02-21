package export

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                       Copyright (c) 2006-2018 FB GROUP LLC                         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"time"

	"pkg.re/essentialkaos/ek.v9/timeutil"

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

kill timeout {{.Service.Options.KillTimeout}}
{{ if .Service.Options.IsKillSignalSet }}kill signal {{.Service.Options.KillSignal}}{{ end }}
{{ if .Service.Options.IsReloadSignalSet }}reload signal {{.Service.Options.ReloadSignal}}{{ end }}

{{ if .Service.Options.IsFileLimitSet }}limit nofile {{.Service.Options.LimitFile}} {{.Service.Options.LimitFile}}{{ end }}
{{ if .Service.Options.IsProcLimitSet }}limit nproc {{.Service.Options.LimitProc}} {{.Service.Options.LimitProc}}{{ end }}

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

// NewUpstart create new UpstartProvider struct
func NewUpstart() *UpstartProvider {
	return &UpstartProvider{}
}

// ////////////////////////////////////////////////////////////////////////////////// //

// UnitName return unit name with extension
func (up *UpstartProvider) UnitName(name string) string {
	return name + ".conf"
}

// EnableService enable service with given name
func (up *UpstartProvider) EnableService(appName string) error {
	return nil
}

// DisableService disable service with given name
func (up *UpstartProvider) DisableService(appName string) error {
	return nil
}

// Reload reload service units
func (up *UpstartProvider) Reload() error {
	return nil
}

// RenderAppTemplate render unit template data with given app data and return
// app unit code
func (up *UpstartProvider) RenderAppTemplate(app *procfile.Application) (string, error) {
	data := &upstartAppData{
		Application: app,
		StartLevel:  fmt.Sprintf("runlevel [%d]", app.StartLevel),
		StopLevel:   fmt.Sprintf("runlevel [%d]", app.StopLevel),
		ExportDate:  timeutil.Format(time.Now(), "%Y/%m/%d %H:%M:%S"),
	}

	return renderTemplate("upstart-app-template", TEMPLATE_UPSTART_APP, data)
}

// RenderServiceTemplate render unit template data with given service data and
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

// RenderHelperTemplate render helper template data with given service data and
// return helper script code
func (up *UpstartProvider) RenderHelperTemplate(service *procfile.Service) (string, error) {
	data := &upstartServiceData{
		Application: service.Application,
		Service:     service,
		StartLevel:  fmt.Sprintf("[%d]", service.Application.StartLevel),
		StopLevel:   fmt.Sprintf("[%d]", service.Application.StopLevel),
		ExportDate:  timeutil.Format(time.Now(), "%Y/%m/%d %H:%M:%S"),
	}

	return renderTemplate("upstart-helper-template", TEMPLATE_UPSTART_HELPER, data)
}

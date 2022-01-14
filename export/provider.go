package export

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                           Copyright (c) 2006-2021 FUNBOX                           //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bytes"
	"fmt"
	"text/template"

	"pkg.re/essentialkaos/ek.v12/log"

	"github.com/funbox/init-exporter/procfile"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type Provider interface {
	// CheckRequirements checks provider requirements for given application
	CheckRequirements(app *procfile.Application) error

	// UnitName returns unit name with extension
	UnitName(name string) string

	// RenderAppTemplate renders unit template data with given app data and return
	// app unit code
	RenderAppTemplate(app *procfile.Application) (string, error)

	// RenderServiceTemplate renders unit template data with given service data and
	// return service unit code
	RenderServiceTemplate(service *procfile.Service) (string, error)

	// RenderHelperTemplate renders helper template data with given service data and
	// return helper script code
	RenderHelperTemplate(service *procfile.Service) (string, error)

	// RenderReloadHelperTemplate renders helper template data for reloading services
	RenderReloadHelperTemplate(app *procfile.Application) (string, error)

	// EnableService enables service with given name
	EnableService(appName string) error

	// DisableService disables service with given name
	DisableService(appName string) error

	// Reload reloads service units
	Reload() error
}

// ////////////////////////////////////////////////////////////////////////////////// //

// renderTemplate renders template data
func renderTemplate(name, templateData string, data interface{}) (string, error) {
	templ, err := template.New(name).Parse(templateData)

	if err != nil {
		log.Error(err.Error())
		return "", fmt.Errorf("Can't render template: %v", err)
	}

	var buffer bytes.Buffer

	ct := template.Must(templ, nil)
	err = ct.Execute(&buffer, data)

	if err != nil {
		log.Error(err.Error())
		return "", fmt.Errorf("Can't render template: %v", err)
	}

	return buffer.String(), nil
}

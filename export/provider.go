package export

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                       Copyright (c) 2006-2018 FB GROUP LLC                         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bytes"
	"fmt"
	"text/template"

	"pkg.re/essentialkaos/ek.v9/log"

	"github.com/funbox/init-exporter/procfile"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type Provider interface {
	// UnitName return unit name with extension
	UnitName(name string) string

	// RenderAppTemplate render unit template data with given app data and return
	// app unit code
	RenderAppTemplate(app *procfile.Application) (string, error)

	// RenderServiceTemplate render unit template data with given service data and
	// return service unit code
	RenderServiceTemplate(service *procfile.Service) (string, error)

	// RenderHelperTemplate render helper template data with given service data and
	// return helper script code
	RenderHelperTemplate(service *procfile.Service) (string, error)

	// EnableService enable service with given name
	EnableService(appName string) error

	// DisableService disable service with given name
	DisableService(appName string) error

	// Reload reload service units
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

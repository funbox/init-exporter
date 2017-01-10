package export

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                       Copyright (c) 2006-2017 FB GROUP LLC                         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"io/ioutil"
	"os"

	"pkg.re/essentialkaos/ek.v6/fsutil"
	"pkg.re/essentialkaos/ek.v6/log"
	"pkg.re/essentialkaos/ek.v6/path"

	"github.com/funbox/init-exporter/procfile"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type Config struct {
	HelperDir        string
	TargetDir        string
	DisableAutoStart bool
}

type Exporter struct {
	Provider Provider
	Config   *Config
}

// ////////////////////////////////////////////////////////////////////////////////// //

func NewExporter(config *Config, provider Provider) *Exporter {
	return &Exporter{Config: config, Provider: provider}
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Install install application to init system
func (e *Exporter) Install(app *procfile.Application) error {
	var err error

	if e.IsInstalled(app) {
		err = e.Uninstall(app)

		if err != nil {
			return err
		}
	}

	err = e.writeAppUnit(app)

	if err != nil {
		return err
	}

	err = e.writeServicesUnits(app)

	if err != nil {
		return err
	}

	if !e.Config.DisableAutoStart {
		err = e.Provider.EnableService(app.Name)

		if err != nil {
			return err
		}

		log.Debug("Service %s enabled", app.Name)
	}

	return nil
}

// Uninstall uninstall application from init system
func (e *Exporter) Uninstall(app *procfile.Application) error {
	var err error

	if !e.IsInstalled(app) {
		return fmt.Errorf("Application %s is not installed", app.Name)
	}

	if !e.Config.DisableAutoStart {
		err = e.Provider.DisableService(app.Name)

		if err != nil {
			return err
		}
	}

	log.Debug("Service %s disabled", app.Name)

	unitPath := e.unitPath(app.Name)
	err = os.Remove(unitPath)

	if err != nil {
		return err
	}

	log.Debug("Application unit %s deleted", unitPath)

	err = deleteByMask(e.Config.TargetDir, app.Name+"_*")

	if err != nil {
		return err
	}

	log.Debug("Service units deleted")

	err = deleteByMask(e.Config.HelperDir, app.Name+"_*.sh")

	if err != nil {
		return err
	}

	log.Debug("Helpers deleted")

	return nil
}

// IsInstalled return true if app already installed
func (e *Exporter) IsInstalled(app *procfile.Application) bool {
	return fsutil.IsExist(e.unitPath(app.Name))
}

// ////////////////////////////////////////////////////////////////////////////////// //

// writeAppUnit write app init to file
func (e *Exporter) writeAppUnit(app *procfile.Application) error {
	unitPath := e.unitPath(app.Name)
	data, err := e.Provider.RenderAppTemplate(app)

	if err != nil {
		return err
	}

	log.Debug("Application unit saved as %s", unitPath)

	err = ioutil.WriteFile(unitPath, []byte(data), 0644)

	return err
}

// writeAppUnit write services init to files
func (e *Exporter) writeServicesUnits(app *procfile.Application) error {
	err := os.MkdirAll(e.Config.HelperDir, 0755)

	if err != nil {
		return err
	}

	for _, service := range app.Services {
		fullServiceName := app.Name + "_" + service.Name

		service.HelperPath = e.helperPath(fullServiceName)

		helperData, err := e.Provider.RenderHelperTemplate(service)

		if err != nil {
			return err
		}

		unitData, err := e.Provider.RenderServiceTemplate(service)

		if err != nil {
			return err
		}

		unitPath := e.unitPath(fullServiceName)

		err = ioutil.WriteFile(unitPath, []byte(unitData), 0644)

		if err != nil {
			return err
		}

		log.Debug("Service unit saved as %s", unitPath)

		err = ioutil.WriteFile(service.HelperPath, []byte(helperData), 0755)

		if err != nil {
			return err
		}

		log.Debug("Helper saved as %s", service.HelperPath)
	}

	return nil
}

// unitPath return path for unit
func (e *Exporter) unitPath(name string) string {
	return path.Join(e.Config.TargetDir, e.Provider.UnitName(name))
}

// helperPath return path for helper
func (e *Exporter) helperPath(name string) string {
	return path.Join(e.Config.HelperDir, name+".sh")
}

// ////////////////////////////////////////////////////////////////////////////////// //

// deleteByMask delete all files witch
func deleteByMask(dir, mask string) error {
	files := fsutil.List(
		dir, true,
		&fsutil.ListingFilter{
			MatchPatterns: []string{mask},
		},
	)

	fsutil.ListToAbsolute(dir, files)

	if len(files) == 0 {
		return nil
	}

	for _, file := range files {
		log.Debug("File %s removed", file)

		err := os.Remove(file)

		if err != nil {
			return err
		}
	}

	return nil
}

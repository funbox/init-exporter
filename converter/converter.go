package converter

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                       Copyright (c) 2006-2017 FB GROUP LLC                         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bytes"
	"io/ioutil"
	"os"
	"runtime"
	"text/template"

	"pkg.re/essentialkaos/ek.v7/arg"
	"pkg.re/essentialkaos/ek.v7/fmtc"
	"pkg.re/essentialkaos/ek.v7/knf"
	"pkg.re/essentialkaos/ek.v7/usage"

	"github.com/funbox/init-exporter/procfile"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// App props
const (
	APP  = "init-exporter-converter"
	VER  = "0.4.0"
	DESC = "Utility for converting procfiles from v1 to v2 format"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Supported arguments
const (
	ARG_CONFIG    = "c:config"
	ARG_IN_PLACE  = "i:in-place"
	ARG_NO_COLORS = "nc:no-colors"
	ARG_HELP      = "h:help"
	ARG_VERSION   = "v:version"
)

// Config properies
const (
	MAIN_PREFIX               = "main:prefix"
	PATHS_WORKING_DIR         = "paths:working-dir"
	DEFAULTS_NPROC            = "defaults:nproc"
	DEFAULTS_NOFILE           = "defaults:nofile"
	DEFAULTS_RESPAWN          = "defaults:respawn"
	DEFAULTS_RESPAWN_COUNT    = "defaults:respawn-count"
	DEFAULTS_RESPAWN_INTERVAL = "defaults:respawn-interval"
	DEFAULTS_KILL_TIMEOUT     = "defaults:kill-timeout"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// PROCFILE_TEMPLATE is template used for generation v2 Procfile
const PROCFILE_TEMPLATE = `version: 2

start_on_runlevel: 2
stop_on_runlevel: 5

{{ if .Config.IsRespawnEnabled -}}
respawn:
  count: {{ .Config.RespawnCount }}
  interval: {{ .Config.RespawnInterval }}

{{ end -}}

limits:
  nofile: {{ .Config.LimitFile }}
  nproc: {{ .Config.LimitProc }}

{{ if not .HasCustomWorkingDirs -}}
working_directory: {{ .Config.WorkingDir }}

{{ end -}}

{{- $hasCustomWorkingDirs := .HasCustomWorkingDirs -}}

commands:
{{- range .Application.Services }}
  {{ .Name }}:
    {{ if .HasPreCmd -}}
    pre: {{ .PreCmd }}
    {{ end -}}
    command: {{ .Cmd }}
    {{ if .HasPostCmd -}}
    post: {{ .PostCmd }}
    {{ end -}}
    {{ if $hasCustomWorkingDirs -}}
    working_directory: {{ .Options.WorkingDir }}
    {{ end -}}
    {{ if .Options.IsCustomLogEnabled -}}
    log: {{ .Options.LogPath }}
    {{ end -}}
    {{ if .Options.IsEnvSet -}}
    env:
    {{- range $k, $v := .Options.Env }}
      {{ $k }}: {{ $v -}}
    {{ end }}
    {{ end -}}
{{ end -}}
`

// DEFAULT_WORKING_DIR is path to default working dir
const DEFAULT_WORKING_DIR = "/tmp"

// ////////////////////////////////////////////////////////////////////////////////// //

type templateData struct {
	Config               *procfile.Config
	Application          *procfile.Application
	HasCustomWorkingDirs bool
}

// ////////////////////////////////////////////////////////////////////////////////// //

var argMap = arg.Map{
	ARG_CONFIG:    {},
	ARG_IN_PLACE:  {Type: arg.BOOL},
	ARG_NO_COLORS: {Type: arg.BOOL},
	ARG_HELP:      {Type: arg.BOOL},
	ARG_VERSION:   {Type: arg.BOOL},
}

// ////////////////////////////////////////////////////////////////////////////////// //

func Init() {
	runtime.GOMAXPROCS(1)

	args, errs := arg.Parse(argMap)

	if len(errs) != 0 {
		fmtc.Println("Error while arguments parsing:")

		for _, err := range errs {
			fmtc.Printf("  %v\n", err)
		}

		os.Exit(1)
	}

	if arg.GetB(ARG_NO_COLORS) {
		fmtc.DisableColors = true
	}

	if arg.GetB(ARG_VERSION) {
		showAbout()
		return
	}

	if arg.GetB(ARG_HELP) || len(args) == 0 {
		showUsage()
		return
	}

	process(args[0])
}

// process start data processing
func process(file string) {
	var err error

	if arg.Has(ARG_CONFIG) {
		err = knf.Global(arg.GetS(ARG_CONFIG))

		if err != nil {
			printErrorAndExit(err.Error())
		}
	}

	err = convert(file)

	if err != nil {
		printErrorAndExit(err.Error())
	}
}

// convert read procfile in v1 format and print v2 data or save it to file
func convert(file string) error {
	var hasCustomWorkingDirs bool

	config := &procfile.Config{
		Name:             "",
		WorkingDir:       knf.GetS(PATHS_WORKING_DIR, "/tmp"),
		IsRespawnEnabled: knf.GetB(DEFAULTS_RESPAWN, true),
		RespawnInterval:  knf.GetI(DEFAULTS_RESPAWN_INTERVAL, 15),
		RespawnCount:     knf.GetI(DEFAULTS_RESPAWN_COUNT, 10),
		KillTimeout:      knf.GetI(DEFAULTS_KILL_TIMEOUT, 60),
		LimitFile:        knf.GetI(DEFAULTS_NOFILE, 10240),
		LimitProc:        knf.GetI(DEFAULTS_NPROC, 10240),
	}

	app, err := procfile.Read(file, config)

	if err != nil {
		return err
	}

	if app.ProcVersion != 1 {
		printErrorAndExit("Given procfile already converted to v2 format")
	}

	config.WorkingDir, hasCustomWorkingDirs = getWorkingDir(app)

	validateApplication(app)

	v2data, err := renderTemplate(
		"proc_v2", PROCFILE_TEMPLATE,
		&templateData{config, app, hasCustomWorkingDirs},
	)

	if err != nil {
		return err
	}

	if !arg.GetB(ARG_IN_PLACE) {
		fmtc.Println(v2data)
		return nil
	}

	return writeData(file, v2data)
}

// renderTemplate renders template data
func renderTemplate(name, templateData string, data interface{}) (string, error) {
	templ, err := template.New(name).Parse(templateData)

	if err != nil {
		return "", fmtc.Errorf("Can't render template: %v", err)
	}

	var buffer bytes.Buffer

	ct := template.Must(templ, nil)
	err = ct.Execute(&buffer, data)

	if err != nil {
		return "", fmtc.Errorf("Can't render template: %v", err)
	}

	return buffer.String(), nil
}

// getWorkingDir return path to default working dir and flag
// if custom working dirs is used
func getWorkingDir(app *procfile.Application) (string, bool) {
	var dir = DEFAULT_WORKING_DIR

	for _, service := range app.Services {
		if dir == DEFAULT_WORKING_DIR {
			if service.Options.WorkingDir != "" {
				dir = service.Options.WorkingDir
			}

			continue
		}

		if dir != service.Options.WorkingDir {
			return DEFAULT_WORKING_DIR, true
		}
	}

	return dir, false
}

// validateApplication validate application and all services
func validateApplication(app *procfile.Application) {
	errs := app.Validate()

	if len(errs) == 0 {
		return
	}

	printError("Errors while application validation:")

	for _, err := range errs {
		printError("  - %v", err)
	}

	os.Exit(1)
}

// writeData write procfile data to file
func writeData(file, data string) error {
	return ioutil.WriteFile(file, []byte(data), 0644)
}

// printError prints error message to console
func printError(f string, a ...interface{}) {
	fmtc.Fprintf(os.Stderr, "{r}"+f+"{!}\n", a...)
}

// printError prints warning message to console
func printWarn(f string, a ...interface{}) {
	fmtc.Fprintf(os.Stderr, "{y}"+f+"{!}\n", a...)
}

// printErrorAndExit print error mesage and exit with exit code 1
func printErrorAndExit(f string, a ...interface{}) {
	printError(f, a...)
	os.Exit(1)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// showUsage print usage info to console
func showUsage() {
	info := usage.NewInfo("", "procfile")

	info.AddOption(ARG_CONFIG, "Path to init-exporter config", "file")
	info.AddOption(ARG_IN_PLACE, "Edit procfile in place")
	info.AddOption(ARG_NO_COLORS, "Disable colors in output")
	info.AddOption(ARG_HELP, "Show this help message")
	info.AddOption(ARG_VERSION, "Show version")

	info.AddExample(
		"-i config/Procfile.production",
		"Convert Procfile.production to version 2 in-place",
	)

	info.AddExample(
		"config/Procfile.production -c /etc/init-exporter.conf Procfile.production",
		"Convert Procfile.production to version 2 with defaults from init-exporter config and print result to console",
	)

	info.Render()
}

// showAbout print version info to console
func showAbout() {
	about := &usage.About{
		App:     APP,
		Version: VER,
		Desc:    DESC,
		Year:    2006,
		Owner:   "FB Group",
		License: "MIT License",
	}

	about.Render()
}

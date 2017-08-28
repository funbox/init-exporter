// +build !windows
package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                       Copyright (c) 2006-2017 FB GROUP LLC                         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"runtime"

	"pkg.re/essentialkaos/ek.v9/env"
	"pkg.re/essentialkaos/ek.v9/fmtc"
	"pkg.re/essentialkaos/ek.v9/fsutil"
	"pkg.re/essentialkaos/ek.v9/knf"
	"pkg.re/essentialkaos/ek.v9/log"
	"pkg.re/essentialkaos/ek.v9/options"
	"pkg.re/essentialkaos/ek.v9/system"
	"pkg.re/essentialkaos/ek.v9/usage"
	"pkg.re/essentialkaos/ek.v9/usage/update"

	"github.com/funbox/init-exporter/export"
	"github.com/funbox/init-exporter/procfile"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// App props
const (
	APP  = "init-exporter"
	VER  = "0.15.2"
	DESC = "Utility for exporting services described by Procfile to init system"
)

// Supported arguments
const (
	OPT_PROCFILE           = "p:procfile"
	OPT_APP_NAME           = "n:appname"
	OPT_DRY_START          = "d:dry-start"
	OPT_DISABLE_VALIDATION = "D:disable-validation"
	OPT_UNINSTALL          = "u:uninstall"
	OPT_FORMAT             = "f:format"
	OPT_NO_COLORS          = "nc:no-colors"
	OPT_HELP               = "h:help"
	OPT_VERSION            = "v:version"
)

// Config properies
const (
	MAIN_RUN_USER             = "main:run-user"
	MAIN_RUN_GROUP            = "main:run-group"
	MAIN_PREFIX               = "main:prefix"
	PROCFILE_VERSION1         = "procfile:version1"
	PROCFILE_VERSION2         = "procfile:version2"
	PATHS_WORKING_DIR         = "paths:working-dir"
	PATHS_HELPER_DIR          = "paths:helper-dir"
	PATHS_SYSTEMD_DIR         = "paths:systemd-dir"
	PATHS_UPSTART_DIR         = "paths:upstart-dir"
	DEFAULTS_NPROC            = "defaults:nproc"
	DEFAULTS_NOFILE           = "defaults:nofile"
	DEFAULTS_RESPAWN          = "defaults:respawn"
	DEFAULTS_RESPAWN_COUNT    = "defaults:respawn-count"
	DEFAULTS_RESPAWN_INTERVAL = "defaults:respawn-interval"
	DEFAULTS_KILL_TIMEOUT     = "defaults:kill-timeout"
	LOG_ENABLED               = "log:enabled"
	LOG_DIR                   = "log:dir"
	LOG_FILE                  = "log:file"
	LOG_PERMS                 = "log:perms"
	LOG_LEVEL                 = "log:level"
)

const (
	// FORMAT_UPSTART contains name for upstart exporting format
	FORMAT_UPSTART = "upstart"
	// FORMAT_SYSTEMD contains name for systemd exporting format
	FORMAT_SYSTEMD = "systemd"
)

// CONFIG_FILE contains path to config file
const CONFIG_FILE = "/etc/init-exporter.conf"

// ////////////////////////////////////////////////////////////////////////////////// //

var optMap = options.Map{
	OPT_APP_NAME:           {},
	OPT_PROCFILE:           {},
	OPT_DRY_START:          {Type: options.BOOL},
	OPT_DISABLE_VALIDATION: {Type: options.BOOL},
	OPT_UNINSTALL:          {Type: options.BOOL, Alias: "c:clear"},
	OPT_FORMAT:             {},
	OPT_NO_COLORS:          {Type: options.BOOL},
	OPT_HELP:               {Type: options.BOOL},
	OPT_VERSION:            {Type: options.BOOL},
}

var user *system.User

// ////////////////////////////////////////////////////////////////////////////////// //

func Init() {
	runtime.GOMAXPROCS(1)

	args, errs := options.Parse(optMap)

	if len(errs) != 0 {
		fmt.Println("Error while arguments parsing:")

		for _, err := range errs {
			fmt.Printf("  %v\n", err)
		}

		os.Exit(1)
	}

	if options.GetB(OPT_NO_COLORS) {
		fmtc.DisableColors = true
	}

	if options.GetB(OPT_VERSION) {
		showAbout()
		return
	}

	if options.GetB(OPT_HELP) {
		showUsage()
		return
	}

	if len(args) == 0 && !options.Has(OPT_APP_NAME) {
		showUsage()
		return
	}

	checkForRoot()
	checkArguments()
	loadConfig()
	validateConfig()
	setupLogger()

	switch {
	case len(args) == 0:
		startProcessing(options.GetS(OPT_APP_NAME))
	default:
		startProcessing(args[0])
	}
}

// checkForRoot check superuser privileges
func checkForRoot() {
	var err error

	user, err = system.CurrentUser()

	if err != nil {
		printErrorAndExit(err.Error())
	}

	if !user.IsRoot() {
		printErrorAndExit("This utility must have superuser privileges (root)")
	}
}

// checkArguments check given arguments
func checkArguments() {
	if !options.GetB(OPT_UNINSTALL) {
		proc := options.GetS(OPT_PROCFILE)

		switch {
		case proc == "":
			printErrorAndExit("You should define path to procfile", proc)

		case fsutil.IsExist(proc) == false:
			printErrorAndExit("Procfile %s does not exist", proc)

		case fsutil.IsReadable(proc) == false:
			printErrorAndExit("Procfile %s is not readable", proc)

		case fsutil.IsNonEmpty(proc) == false:
			printErrorAndExit("Procfile %s is empty", proc)
		}
	}
}

// loadConfig check config path and load config
func loadConfig() {
	var err error

	switch {
	case !fsutil.IsExist(CONFIG_FILE):
		printErrorAndExit("Config %s is not exist", CONFIG_FILE)

	case !fsutil.IsReadable(CONFIG_FILE):
		printErrorAndExit("Config %s is not readable", CONFIG_FILE)

	case !fsutil.IsNonEmpty(CONFIG_FILE):
		printErrorAndExit("Config %s is empty", CONFIG_FILE)
	}

	err = knf.Global(CONFIG_FILE)

	if err != nil {
		printErrorAndExit(err.Error())
	}
}

// validateConfig validate config values
func validateConfig() {
	var permsChecker = func(config *knf.Config, prop string, value interface{}) error {
		if !fsutil.CheckPerms(value.(string), config.GetS(prop)) {
			switch value.(string) {
			case "DRX":
				return fmt.Errorf("Property %s must be path to readable directory", prop)
			case "DWX":
				return fmt.Errorf("Property %s must be path to writable directory", prop)
			case "DRWX":
				return fmt.Errorf("Property %s must be path to writable/readable directory", prop)
			case "FR":
				return fmt.Errorf("Property %s must be path to readable file", prop)
			}
		}

		return nil
	}

	var userChecker = func(config *knf.Config, prop string, value interface{}) error {
		if !system.IsUserExist(knf.GetS(prop)) {
			return fmt.Errorf("Property %s contains user which not exist on this system", prop)
		}

		return nil
	}

	var groupChecker = func(config *knf.Config, prop string, value interface{}) error {
		if !system.IsGroupExist(knf.GetS(prop)) {
			return fmt.Errorf("Property %s contains group which not exist on this system", prop)
		}

		return nil
	}

	validators := []*knf.Validator{
		{MAIN_RUN_USER, knf.Empty, nil},
		{MAIN_RUN_GROUP, knf.Empty, nil},
		{PATHS_WORKING_DIR, knf.Empty, nil},
		{PATHS_HELPER_DIR, knf.Empty, nil},
		{PATHS_SYSTEMD_DIR, knf.Empty, nil},
		{PATHS_UPSTART_DIR, knf.Empty, nil},
		{DEFAULTS_NPROC, knf.Empty, nil},
		{DEFAULTS_NOFILE, knf.Empty, nil},
		{DEFAULTS_RESPAWN_COUNT, knf.Empty, nil},
		{DEFAULTS_RESPAWN_INTERVAL, knf.Empty, nil},
		{DEFAULTS_KILL_TIMEOUT, knf.Empty, nil},

		{DEFAULTS_NPROC, knf.Less, 0},
		{DEFAULTS_NOFILE, knf.Less, 0},
		{DEFAULTS_RESPAWN_COUNT, knf.Less, 0},
		{DEFAULTS_RESPAWN_INTERVAL, knf.Less, 0},
		{DEFAULTS_KILL_TIMEOUT, knf.Less, 0},

		{MAIN_RUN_USER, userChecker, nil},
		{MAIN_RUN_GROUP, groupChecker, nil},

		{PATHS_WORKING_DIR, permsChecker, "DRWX"},
		{PATHS_HELPER_DIR, permsChecker, "DRWX"},
	}

	if knf.GetB(LOG_ENABLED, true) {
		validators = append(validators,
			&knf.Validator{LOG_DIR, knf.Empty, nil},
			&knf.Validator{LOG_FILE, knf.Empty, nil},
			&knf.Validator{LOG_DIR, permsChecker, "DWX"},
		)
	}

	errs := knf.Validate(validators)

	if len(errs) != 0 {
		printError("Errors while config validation:")

		for _, err := range errs {
			printError("  - %v", err)
		}

		os.Exit(1)
	}
}

// setupLogger configure logging subsystem
func setupLogger() {
	if !knf.GetB(LOG_ENABLED, true) {
		log.Set(os.DevNull, 0)
		return
	}

	log.Set(knf.GetS(LOG_FILE), knf.GetM(LOG_PERMS, 0644))
	log.MinLevel(knf.GetS(LOG_LEVEL, "info"))
}

// startProcessing start processing
func startProcessing(appName string) {
	if !options.GetB(OPT_UNINSTALL) {
		installApplication(appName)
	} else {
		uninstallApplication(appName)
	}
}

// installApplication install application to init system
func installApplication(appName string) {
	fullAppName := knf.GetS(MAIN_PREFIX) + appName

	app, err := procfile.Read(
		options.GetS(OPT_PROCFILE),
		&procfile.Config{
			Name:             fullAppName,
			User:             knf.GetS(MAIN_RUN_USER),
			Group:            knf.GetS(MAIN_RUN_GROUP),
			WorkingDir:       knf.GetS(PATHS_WORKING_DIR),
			IsRespawnEnabled: knf.GetB(DEFAULTS_RESPAWN, false),
			RespawnInterval:  knf.GetI(DEFAULTS_RESPAWN_INTERVAL),
			RespawnCount:     knf.GetI(DEFAULTS_RESPAWN_COUNT),
			KillTimeout:      knf.GetI(DEFAULTS_KILL_TIMEOUT, 0),
			LimitFile:        knf.GetI(DEFAULTS_NOFILE, 0),
			LimitProc:        knf.GetI(DEFAULTS_NPROC, 0),
		},
	)

	if err != nil {
		printErrorAndExit(err.Error())
	}

	validateApplication(app)

	if options.GetB(OPT_DRY_START) {
		os.Exit(0)
	}

	err = getExporter().Install(app)

	if err == nil {
		log.Info("User %s (%d) installed service %s", user.RealName, user.RealUID, app.Name)
	} else {
		log.Error(err.Error())
		printErrorAndExit(err.Error())
	}
}

// uninstallApplication uninstall application from init system
func uninstallApplication(appName string) {
	fullAppName := knf.GetS(MAIN_PREFIX) + appName
	app := &procfile.Application{Name: fullAppName}

	err := getExporter().Uninstall(app)

	if err == nil {
		log.Info("User %s (%d) uninstalled service %s", user.RealName, user.RealUID, app.Name)
	} else {
		log.Error(err.Error())
		printErrorAndExit(err.Error())
	}
}

// validateApplication validate application and all services
func validateApplication(app *procfile.Application) {
	if app.ProcVersion == 1 && !knf.GetB(PROCFILE_VERSION1, true) {
		printErrorAndExit("Proc format version 1 support is disabled")
	}

	if app.ProcVersion == 2 && !knf.GetB(PROCFILE_VERSION2, true) {
		printErrorAndExit("Proc format version 2 support is disabled")
	}

	if !options.GetB(OPT_DRY_START) && options.GetB(OPT_DISABLE_VALIDATION) {
		return
	}

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

// checkProviderTargetDir check permissions on target dir
func checkProviderTargetDir(dir string) error {
	if !fsutil.CheckPerms("DRWX", dir) {
		return fmt.Errorf("This utility require read/write access to directory %s", dir)
	}

	return nil
}

// getExporter create and configure exporter and return it
func getExporter() *export.Exporter {
	providerName, err := detectProvider(options.GetS(OPT_FORMAT))

	if err != nil {
		printErrorAndExit(err.Error())
	}

	var provider export.Provider

	exportConfig := &export.Config{HelperDir: knf.GetS(PATHS_HELPER_DIR)}

	switch providerName {
	case FORMAT_UPSTART:
		exportConfig.TargetDir = knf.GetS(PATHS_UPSTART_DIR)
		provider = export.NewUpstart()
	case FORMAT_SYSTEMD:
		exportConfig.TargetDir = knf.GetS(PATHS_SYSTEMD_DIR)
		provider = export.NewSystemd()
	}

	err = checkProviderTargetDir(exportConfig.TargetDir)

	if err != nil {
		printErrorAndExit(err.Error())
	}

	return export.NewExporter(exportConfig, provider)
}

// detectProvider try to detect provider
func detectProvider(format string) (string, error) {
	switch {
	case format == FORMAT_SYSTEMD:
		return FORMAT_SYSTEMD, nil
	case format == FORMAT_UPSTART:
		return FORMAT_UPSTART, nil
	case os.Args[0] == "systemd-exporter":
		return FORMAT_SYSTEMD, nil
	case os.Args[0] == "upstart-exporter":
		return FORMAT_UPSTART, nil
	case env.Which("systemctl") != "":
		return FORMAT_SYSTEMD, nil
	case env.Which("initctl") != "":
		return FORMAT_UPSTART, nil
	default:
		return "", fmt.Errorf("Can't find init provider")
	}
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
	info := usage.NewInfo("", "app-name")

	info.AddOption(OPT_PROCFILE, "Path to procfile", "file")
	info.AddOption(OPT_DRY_START, "Dry start {s-}(don't export anything, just parse and test procfile){!}")
	info.AddOption(OPT_DISABLE_VALIDATION, "Disable application validation")
	info.AddOption(OPT_UNINSTALL, "Remove scripts and helpers for a particular application")
	info.AddOption(OPT_FORMAT, "Format of generated configs", "?upstart|systemd")
	info.AddOption(OPT_NO_COLORS, "Disable colors in output")
	info.AddOption(OPT_HELP, "Show this help message")
	info.AddOption(OPT_VERSION, "Show version")

	info.AddExample("-p ./myprocfile -f systemd myapp", "Export given procfile to systemd as myapp")
	info.AddExample("-u -f systemd myapp", "Uninstall myapp from systemd")

	info.AddExample("-p ./myprocfile -f upstart myapp", "Export given procfile to upstart as myapp")
	info.AddExample("-u -f upstart myapp", "Uninstall myapp from upstart")

	info.Render()
}

// showAbout print version info to console
func showAbout() {
	about := &usage.About{
		App:           APP,
		Version:       VER,
		Desc:          DESC,
		Year:          2006,
		Owner:         "FB Group",
		License:       "MIT License",
		UpdateChecker: usage.UpdateChecker{"funbox/init-exporter", update.GitHubChecker},
	}

	about.Render()
}

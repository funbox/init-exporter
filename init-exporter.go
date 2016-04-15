package main

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                       Copyright (c) 2006-2016 FB GROUP LLC                         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"runtime"

	"pkg.re/essentialkaos/ek.v1/arg"
	"pkg.re/essentialkaos/ek.v1/env"
	"pkg.re/essentialkaos/ek.v1/fsutil"
	"pkg.re/essentialkaos/ek.v1/knf"
	"pkg.re/essentialkaos/ek.v1/log"
	"pkg.re/essentialkaos/ek.v1/system"
	"pkg.re/essentialkaos/ek.v1/usage"

	"github.com/funbox/init-exporter/export"
	"github.com/funbox/init-exporter/procfile"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// App props
const (
	APP  = "init-exporter"
	VER  = "0.2.0"
	DESC = "Utility for exporting services described by Procfile to init system"
)

// Supported arguments list
const (
	ARG_CONFIG    = "config"
	ARG_PROCFILE  = "p:procfile"
	ARG_APP_NAME  = "n:appname"
	ARG_DRY_START = "d:dry-start"
	ARG_UNINSTALL = "u:unistall"
	ARG_FORMAT    = "f:format"
	ARG_HELP      = "h:help"
	ARG_VERSION   = "v:version"
)

// Config properies list
const (
	MAIN_RUN_USER     = "main:run-user"
	MAIN_RUN_GROUP    = "main:run-group"
	MAIN_PREFIX       = "main:prefix"
	PATHS_WORKING_DIR = "paths:working-dir"
	PATHS_HELPER_DIR  = "paths:helper-dir"
	PATHS_SYSTEMD_DIR = "paths:systemd-dir"
	PATHS_UPSTART_DIR = "paths:upstart-dir"
	LOG_ENABLED       = "log:enabled"
	LOG_DIR           = "log:dir"
	LOG_FILE          = "log:file"
	LOG_PERMS         = "log:perms"
	LOG_LEVEL         = "log:level"
)

const (
	// FORMAT_UPSTART contains name for upstart exporting format
	FORMAT_UPSTART = "upstart"
	// FORMAT_SYSTEMD contains name for systemd exporting format
	FORMAT_SYSTEMD = "systemd"
)

// DEFAULT_CONFIG_FILE contains path to config file
const DEFAULT_CONFIG_FILE = "/etc/init-exporter.conf"

// ////////////////////////////////////////////////////////////////////////////////// //

var argMap = arg.Map{
	ARG_APP_NAME:  &arg.V{},
	ARG_CONFIG:    &arg.V{},
	ARG_PROCFILE:  &arg.V{},
	ARG_DRY_START: &arg.V{Type: arg.BOOL},
	ARG_UNINSTALL: &arg.V{Type: arg.BOOL, Alias: "c:clear"},
	ARG_FORMAT:    &arg.V{},
	ARG_HELP:      &arg.V{Type: arg.BOOL},
	ARG_VERSION:   &arg.V{Type: arg.BOOL},
}

var user *system.User

// ////////////////////////////////////////////////////////////////////////////////// //

func main() {
	runtime.GOMAXPROCS(1)

	args, errs := arg.Parse(argMap)

	if len(errs) != 0 {
		fmt.Println("Error while arguments parsing:")

		for _, err := range errs {
			fmt.Printf("  %v\n", err)
		}

		os.Exit(1)
	}

	if arg.GetB(ARG_VERSION) {
		showAbout()
		return
	}

	if arg.GetB(ARG_HELP) {
		showUsage()
		return
	}

	if len(args) == 0 && !arg.Has(ARG_APP_NAME) {
		showUsage()
		return
	}

	checkForRoot()
	checkArguments()
	loadConfig(configPath())
	validateConfig()
	setupLogger()

	switch {
	case len(args) == 0:
		startProcessing(arg.GetS(ARG_APP_NAME))
	default:
		startProcessing(args[0])
	}
}

// checkForRoot check superuser privileges
func checkForRoot() {
	var err error

	user, err = system.CurrentUser()

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if !user.IsRoot() {
		fmt.Println("This utility must have superuser privileges (root)")
		os.Exit(1)
	}
}

// checkArguments check given arguments
func checkArguments() {
	if !arg.GetB(ARG_UNINSTALL) {
		proc := arg.GetS(ARG_PROCFILE)

		switch {
		case fsutil.IsExist(proc) == false:
			printErrorAndExit("Procfile %s is not exist", proc)

		case fsutil.IsReadable(proc) == false:
			printErrorAndExit("Procfile %s is not readable", proc)

		case fsutil.IsNonEmpty(proc) == false:
			printErrorAndExit("Procfile %s is empty", proc)
		}
	}
}

// configPath returns path to config
func configPath() string {
	if config_file := arg.GetS(ARG_CONFIG); config_file != "" {
		return config_file
	} else {
		return DEFAULT_CONFIG_FILE
	}
}

// loadConfig check config path and load config
func loadConfig(path string) {
	var err error

	switch {
	case !fsutil.IsExist(path):
		printErrorAndExit("Config %s is not exist", path)

	case !fsutil.IsReadable(path):
		printErrorAndExit("Config %s is not readable", path)

	case !fsutil.IsNonEmpty(path):
		printErrorAndExit("Config %s is empty", path)
	}

	err = knf.Global(path)

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
		&knf.Validator{MAIN_RUN_USER, knf.Empty, nil},
		&knf.Validator{MAIN_RUN_GROUP, knf.Empty, nil},
		&knf.Validator{PATHS_WORKING_DIR, knf.Empty, nil},
		&knf.Validator{PATHS_HELPER_DIR, knf.Empty, nil},
		&knf.Validator{PATHS_SYSTEMD_DIR, knf.Empty, nil},
		&knf.Validator{PATHS_UPSTART_DIR, knf.Empty, nil},

		&knf.Validator{MAIN_RUN_USER, userChecker, nil},
		&knf.Validator{MAIN_RUN_GROUP, groupChecker, nil},

		&knf.Validator{PATHS_WORKING_DIR, permsChecker, "DRWX"},
		&knf.Validator{PATHS_HELPER_DIR, permsChecker, "DRWX"},
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
		fmt.Println("Errors while config validation:")

		for _, err := range errs {
			fmt.Printf("  %v\n", err)
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
	if !arg.GetB(ARG_UNINSTALL) {
		installApplication(appName)
	} else {
		uninstallApplication(appName)
	}
}

// installApplication install application to init system
func installApplication(appName string) {
	fullAppName := knf.GetS(MAIN_PREFIX) + appName
	app, err := procfile.Read(
		arg.GetS(ARG_PROCFILE),
		&procfile.Config{
			Name:       fullAppName,
			User:       knf.GetS(MAIN_RUN_USER),
			Group:      knf.GetS(MAIN_RUN_GROUP),
			WorkingDir: knf.GetS(PATHS_WORKING_DIR),
		},
	)

	if err != nil {
		printErrorAndExit(err.Error())
	}

	if arg.GetB(ARG_DRY_START) {
		os.Exit(0)
	}

	err = getExporter().Install(app)

	if err == nil {
		log.Aux("User %s (%d) installed service %s", user.RealName, user.RealUID, app.Name)
	} else {
		printErrorAndExit(err.Error())
	}
}

// uninstallApplication uninstall application from init system
func uninstallApplication(appName string) {
	fullAppName := knf.GetS(MAIN_PREFIX) + appName
	app := &procfile.Application{Name: fullAppName}

	err := getExporter().Uninstall(app)

	if err == nil {
		log.Aux("User %s (%d) uninstalled service %s", user.RealName, user.RealUID, app.Name)
	} else {
		printErrorAndExit(err.Error())
	}
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
	providerName, err := detectProvider(arg.GetS(ARG_FORMAT))

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

// printErrorAndExit print error mesage and exit with exit code 1
func printErrorAndExit(message string, a ...interface{}) {
	log.Crit(message)
	fmt.Printf(message+"\n", a...)
	os.Exit(1)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// showUsage print usage info to console
func showUsage() {
	info := usage.NewInfo("", "app-name")

	info.AddOption(ARG_CONFIG, "Path to config file", "file")
	info.AddOption(ARG_PROCFILE, "Path to procfile", "file")
	info.AddOption(ARG_DRY_START, "Dry start (don't export anything, just parse and test procfile)")
	info.AddOption(ARG_UNINSTALL, "Remove scripts and helpers for a particular application")
	info.AddOption(ARG_FORMAT, "Format of generated configs", "upstart|systemd")
	info.AddOption(ARG_HELP, "Show this help message")
	info.AddOption(ARG_VERSION, "Show version")

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

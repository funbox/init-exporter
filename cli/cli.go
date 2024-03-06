//go:build linux
// +build linux

package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                           Copyright (c) 2006-2024 FUNBOX                           //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"runtime"

	"github.com/essentialkaos/ek/v12/env"
	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fsutil"
	"github.com/essentialkaos/ek/v12/knf"
	"github.com/essentialkaos/ek/v12/log"
	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/system"
	"github.com/essentialkaos/ek/v12/terminal/tty"
	"github.com/essentialkaos/ek/v12/usage"
	"github.com/essentialkaos/ek/v12/usage/completion/bash"
	"github.com/essentialkaos/ek/v12/usage/completion/fish"
	"github.com/essentialkaos/ek/v12/usage/completion/zsh"
	"github.com/essentialkaos/ek/v12/usage/man"
	"github.com/essentialkaos/ek/v12/usage/update"

	knfv "github.com/essentialkaos/ek/v12/knf/validators"
	knff "github.com/essentialkaos/ek/v12/knf/validators/fs"
	knfs "github.com/essentialkaos/ek/v12/knf/validators/system"

	"github.com/funbox/init-exporter/cli/support"
	"github.com/funbox/init-exporter/export"
	"github.com/funbox/init-exporter/procfile"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// App props
const (
	APP  = "init-exporter"
	VER  = "0.25.1"
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
	OPT_NO_COLOR           = "nc:no-color"
	OPT_HELP               = "h:help"
	OPT_VER                = "v:version"

	OPT_VERB_VER     = "vv:verbose-version"
	OPT_COMPLETION   = "completion"
	OPT_GENERATE_MAN = "generate-man"
)

// Config properties
const (
	MAIN_RUN_USER  = "main:run-user"
	MAIN_RUN_GROUP = "main:run-group"
	MAIN_PREFIX    = "main:prefix"

	PROCFILE_VERSION1 = "procfile:version1"
	PROCFILE_VERSION2 = "procfile:version2"

	PATHS_WORKING_DIR = "paths:working-dir"
	PATHS_HELPER_DIR  = "paths:helper-dir"
	PATHS_SYSTEMD_DIR = "paths:systemd-dir"
	PATHS_UPSTART_DIR = "paths:upstart-dir"

	DEFAULTS_NPROC            = "defaults:nproc"
	DEFAULTS_NOFILE           = "defaults:nofile"
	DEFAULTS_RESPAWN          = "defaults:respawn"
	DEFAULTS_RESPAWN_COUNT    = "defaults:respawn-count"
	DEFAULTS_RESPAWN_INTERVAL = "defaults:respawn-interval"
	DEFAULTS_KILL_TIMEOUT     = "defaults:kill-timeout"

	LOG_ENABLED = "log:enabled"
	LOG_DIR     = "log:dir"
	LOG_FILE    = "log:file"
	LOG_PERMS   = "log:perms"
	LOG_LEVEL   = "log:level"
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
	OPT_NO_COLOR:           {Type: options.BOOL},
	OPT_HELP:               {Type: options.BOOL},
	OPT_VER:                {Type: options.BOOL},

	OPT_VERB_VER:     {Type: options.BOOL},
	OPT_COMPLETION:   {},
	OPT_GENERATE_MAN: {Type: options.BOOL},
}

var colorTagApp string
var colorTagVer string

var user *system.User

// ////////////////////////////////////////////////////////////////////////////////// //

func Run(gitRev string, gomod []byte) {
	runtime.GOMAXPROCS(1)

	preConfigureUI()

	args, errs := options.Parse(optMap)

	if len(errs) != 0 {
		printError(errs[0].Error())
		os.Exit(1)
	}

	configureUI()

	switch {
	case options.Has(OPT_COMPLETION):
		os.Exit(printCompletion())
	case options.Has(OPT_GENERATE_MAN):
		printMan()
		os.Exit(0)
	case options.GetB(OPT_VER):
		genAbout(gitRev).Print()
		os.Exit(0)
	case options.GetB(OPT_VERB_VER):
		support.Print(APP, VER, gitRev, gomod)
		os.Exit(0)
	case options.GetB(OPT_HELP),
		len(args) == 0 && !options.Has(OPT_APP_NAME):
		genUsage().Print()
		os.Exit(0)
	}

	checkForRoot()
	checkOptions()
	loadConfig()
	validateConfig()
	setupLogger()

	switch {
	case len(args) == 0:
		startProcessing(options.GetS(OPT_APP_NAME))
	default:
		startProcessing(args.Get(0).String())
	}
}

// preConfigureUI preconfigures UI based on information about user terminal
func preConfigureUI() {
	if !tty.IsTTY() {
		fmtc.DisableColors = true
	}
}

// configureUI configures user interface
func configureUI() {
	if options.GetB(OPT_NO_COLOR) {
		fmtc.DisableColors = true
	}

	switch {
	case fmtc.IsTrueColorSupported():
		colorTagApp, colorTagVer = "{#BCCF00}", "{#BCCF00}"
	case fmtc.Is256ColorsSupported():
		colorTagApp, colorTagVer = "{#148}", "{#148}"
	default:
		colorTagApp, colorTagVer = "{g}", "{g}"
	}
}

// checkForRoot checks superuser privileges
func checkForRoot() {
	var err error

	user, err = system.CurrentUser()

	if err != nil {
		printErrorAndExit(err.Error())
	}

	if !user.IsRoot() {
		printErrorAndExit("This utility requires superuser privileges (root)")
	}
}

// checkOptions checks given arguments
func checkOptions() {
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

// loadConfig checks configuration file path and loads it
func loadConfig() {
	var err error

	switch {
	case !fsutil.IsExist(CONFIG_FILE):
		printErrorAndExit("Configuration file %s does not exist", CONFIG_FILE)

	case !fsutil.IsReadable(CONFIG_FILE):
		printErrorAndExit("Configuration file %s is not readable", CONFIG_FILE)

	case !fsutil.IsNonEmpty(CONFIG_FILE):
		printErrorAndExit("Configuration file %s is empty", CONFIG_FILE)
	}

	err = knf.Global(CONFIG_FILE)

	if err != nil {
		printErrorAndExit(err.Error())
	}
}

// validateConfig validates configuration file values
func validateConfig() {
	validators := []*knf.Validator{
		{MAIN_RUN_USER, knfv.Empty, nil},
		{MAIN_RUN_GROUP, knfv.Empty, nil},
		{PATHS_WORKING_DIR, knfv.Empty, nil},
		{PATHS_HELPER_DIR, knfv.Empty, nil},
		{PATHS_SYSTEMD_DIR, knfv.Empty, nil},
		{PATHS_UPSTART_DIR, knfv.Empty, nil},
		{DEFAULTS_NPROC, knfv.Empty, nil},
		{DEFAULTS_NOFILE, knfv.Empty, nil},
		{DEFAULTS_RESPAWN_COUNT, knfv.Empty, nil},
		{DEFAULTS_RESPAWN_INTERVAL, knfv.Empty, nil},
		{DEFAULTS_KILL_TIMEOUT, knfv.Empty, nil},

		{DEFAULTS_NPROC, knfv.Less, 0},
		{DEFAULTS_NOFILE, knfv.Less, 0},
		{DEFAULTS_RESPAWN_COUNT, knfv.Less, 0},
		{DEFAULTS_RESPAWN_INTERVAL, knfv.Less, 0},
		{DEFAULTS_KILL_TIMEOUT, knfv.Less, 0},

		{MAIN_RUN_USER, knfs.User, nil},
		{MAIN_RUN_GROUP, knfs.Group, nil},

		{PATHS_WORKING_DIR, knff.Perms, "DRWX"},
		{PATHS_HELPER_DIR, knff.Perms, "DRWX"},
	}

	if knf.GetB(LOG_ENABLED, true) {
		validators = append(validators,
			&knf.Validator{LOG_DIR, knfv.Empty, nil},
			&knf.Validator{LOG_FILE, knfv.Empty, nil},
			&knf.Validator{LOG_DIR, knff.Perms, "DWX"},
		)
	}

	errs := knf.Validate(validators)

	if len(errs) != 0 {
		printError("Errors while configuration validation:")

		for _, err := range errs {
			printError("  - %v", err)
		}

		os.Exit(1)
	}
}

// setupLogger configures logging subsystem
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

// installApplication installs application to init system
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

// uninstallApplication uninstalls application from init system
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

// validateApplication validates application and all services
func validateApplication(app *procfile.Application) {
	if app.ProcVersion == 1 && !knf.GetB(PROCFILE_VERSION1, true) {
		printErrorAndExit("Procfile format version 1 support is disabled")
	}

	if app.ProcVersion == 2 && !knf.GetB(PROCFILE_VERSION2, true) {
		printErrorAndExit("Procfile format version 2 support is disabled")
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
		return fmt.Errorf("This utility requires read/write access to directory %s", dir)
	}

	return nil
}

// getExporter creates and configures exporter and return it
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

// detectProvider tries to detect provider
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
		return "", fmt.Errorf("Can't find init system provider")
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

// printErrorAndExit prints error message and exit with exit code 1
func printErrorAndExit(f string, a ...interface{}) {
	printError(f, a...)
	os.Exit(1)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// printCompletion prints completion for given shell
func printCompletion() int {
	info := genUsage()

	switch options.GetS(OPT_COMPLETION) {
	case "bash":
		fmt.Print(bash.Generate(info, "init-exporter"))
	case "fish":
		fmt.Print(fish.Generate(info, "init-exporter"))
	case "zsh":
		fmt.Print(zsh.Generate(info, optMap, "init-exporter"))
	default:
		return 1
	}

	return 0
}

// printMan prints man page
func printMan() {
	fmt.Println(
		man.Generate(
			genUsage(),
			genAbout(""),
		),
	)
}

// genUsage generates usage info
func genUsage() *usage.Info {
	info := usage.NewInfo("", "app-name")

	info.AppNameColorTag = "{*}" + colorTagApp

	info.AddOption(OPT_PROCFILE, "Path to procfile", "file")
	info.AddOption(OPT_DRY_START, "Dry start {s-}(don't export anything, just parse and test procfile){!}")
	info.AddOption(OPT_DISABLE_VALIDATION, "Disable application validation")
	info.AddOption(OPT_UNINSTALL, "Remove scripts and helpers for a particular application")
	info.AddOption(OPT_FORMAT, "Format of generated configs", "upstart|systemd")
	info.AddOption(OPT_NO_COLOR, "Disable colors in output")
	info.AddOption(OPT_HELP, "Show this help message")
	info.AddOption(OPT_VER, "Show version")

	info.AddExample("-p ./myprocfile -f systemd myapp", "Export given procfile to systemd as myapp")
	info.AddExample("-u -f systemd myapp", "Uninstall myapp from systemd")

	info.AddExample("-p ./myprocfile -f upstart myapp", "Export given procfile to upstart as myapp")
	info.AddExample("-u -f upstart myapp", "Uninstall myapp from upstart")

	return info
}

// genAbout generates info about version
func genAbout(gitRev string) *usage.About {
	about := &usage.About{
		App:           APP,
		Version:       VER,
		Desc:          DESC,
		Year:          2006,
		Owner:         "FunBox",
		License:       "MIT License",
		UpdateChecker: usage.UpdateChecker{"funbox/init-exporter", update.GitHubChecker},

		AppNameColorTag: "{*}" + colorTagApp,
		VersionColorTag: colorTagVer,
	}

	if gitRev != "" {
		about.Build = "git:" + gitRev
	}

	return about
}

package procfile

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                       Copyright (c) 2006-2017 FB GROUP LLC                         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"sort"
	"strings"

	"pkg.re/essentialkaos/ek.v7/errutil"
	"pkg.re/essentialkaos/ek.v7/fsutil"
	"pkg.re/essentialkaos/ek.v7/log"
	"pkg.re/essentialkaos/ek.v7/path"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	REGEXP_V1_LINE     = `^([A-z\d_]+):\s*(.+)`
	REGEXP_V2_VERSION  = `(?m)^\s*version:\s*2\s*$`
	REGEXP_PATH_CHECK  = `\A[A-Za-z0-9_\-./]+\z`
	REGEXP_VALUE_CHECK = `\A[A-Za-z0-9_\-]+\z`
)

// ////////////////////////////////////////////////////////////////////////////////// //

type Config struct {
	Name             string // Application name
	User             string // Working user
	Group            string // Working group
	WorkingDir       string // Working directory
	IsRespawnEnabled bool   // Global respawn enabled flag
	RespawnInterval  int    // Global respawn interval in seconds
	RespawnCount     int    // Global respawn count
	KillTimeout      int    // Global kill timeout in seconds
	LimitProc        int    // Global processes limit
	LimitFile        int    // Global descriptors limit
}

type Service struct {
	Name        string          // Service name
	Cmd         string          // Command
	PreCmd      string          // Pre command
	PostCmd     string          // Post command
	Options     *ServiceOptions // Service options
	Application *Application    // Pointer to parent application
	HelperPath  string          // Path to helper (will be set by exporter)
}

type ServiceOptions struct {
	Env              map[string]string // Environment variables
	WorkingDir       string            // Working directory
	LogPath          string            // Path to log file
	KillTimeout      int               // Kill timeout in seconds
	KillSignal       string            // Kill signal name
	ReloadSignal     string            // Reload signal name
	Count            int               // Exec count
	RespawnInterval  int               // Respawn interval in seconds
	RespawnCount     int               // Respawn count
	IsRespawnEnabled bool              // Respawn enabled flag
	LimitProc        int               // Processes limit
	LimitFile        int               // Descriptors limit
}

type Application struct {
	Name        string     // Name of application
	Services    []*Service // List of services in application
	User        string     // Working user
	Group       string     // Working group
	StartLevel  int        // Start level
	StopLevel   int        // Stop level
	WorkingDir  string     // Working directory
	ProcVersion int        // Proc version 1/2
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Read reads and parse procfile content
func Read(path string, config *Config) (*Application, error) {
	log.Debug("Processing file %s", path)

	if !fsutil.IsExist(path) {
		return nil, fmt.Errorf("Procfile %s is not exist", path)
	}

	if !fsutil.IsRegular(path) {
		return nil, fmt.Errorf("%s is not a file", path)
	}

	if !fsutil.IsNonEmpty(path) {
		return nil, fmt.Errorf("Procfile %s is empty", path)
	}

	if !fsutil.IsReadable(path) {
		return nil, fmt.Errorf("Procfile %s is not readable", path)
	}

	data, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, err
	}

	switch determineProcVersion(data) {

	case 1:
		return parseV1Procfile(data, config)

	case 2:
		return parseV2Procfile(data, config)

	}

	return nil, fmt.Errorf("Can't determine version for procfile %s", path)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Validate validate all services in application
func (a *Application) Validate() error {
	errs := errutil.NewErrors()

	errs.Add(checkRunLevel(a.StartLevel))
	errs.Add(checkRunLevel(a.StopLevel))

	for _, service := range a.Services {
		errs.Add(service.Validate())
	}

	return errs.Last()
}

// Validate validate service props and options
func (s *Service) Validate() error {
	errs := errutil.NewErrors()

	errs.Add(checkValue(s.Name))
	errs.Add(s.Options.Validate())

	return errs.Last()
}

// Validate validate service options
func (so *ServiceOptions) Validate() error {
	errs := errutil.NewErrors()

	errs.Add(checkPath(so.WorkingDir))
	errs.Add(checkPath(so.LogPath))

	for envName, envVal := range so.Env {
		errs.Add(checkEnv(envName, envVal))
	}

	return errs.Last()
}

// HasPreCmd return true if pre command is defined
func (s *Service) HasPreCmd() bool {
	return s.PreCmd != ""
}

// HasPostCmd return true if post command is defined
func (s *Service) HasPostCmd() bool {
	return s.PostCmd != ""
}

// GetCommandExecWithEnv return full command exec with env vars setting
func (s *Service) GetCommandExecWithEnv(command string) string {
	var result = "exec "

	if s.Options.IsEnvSet() {
		result += "env " + s.Options.EnvString() + " "
	}

	switch command {
	case "pre":
		result += s.PreCmd
	case "post":
		result += s.PostCmd
	default:
		result += s.Cmd
	}

	if s.Options.IsCustomLogEnabled() {
		result += " &>>" + s.Options.FullLogPath()
	}

	return result
}

// GetCommandExec return full command exec
func (s *Service) GetCommandExec(command string) string {
	var result = "exec "

	switch command {
	case "pre":
		result += s.PreCmd
	case "post":
		result += s.PostCmd
	default:
		result += s.Cmd
	}

	if s.Options.IsCustomLogEnabled() {
		result += " &>>" + s.Options.FullLogPath()
	}

	return result
}

// IsRespawnLimitSet return true if respawn options is set
func (so *ServiceOptions) IsRespawnLimitSet() bool {
	return so.RespawnCount != 0 || so.RespawnInterval != 0
}

// IsCustomLogEnabled return true if service have custom log
func (so *ServiceOptions) IsCustomLogEnabled() bool {
	return so.LogPath != ""
}

// IsEnvSet return true if service have custom env vars
func (so *ServiceOptions) IsEnvSet() bool {
	return len(so.Env) != 0
}

// IsFileLimitSet return true if descriptors limit is set
func (so *ServiceOptions) IsFileLimitSet() bool {
	return so.LimitFile != 0
}

// IsProcLimitSet return true if processes limit is set
func (so *ServiceOptions) IsProcLimitSet() bool {
	return so.LimitProc != 0
}

// IsKillSignalSet return true if custom kill signal set
func (so *ServiceOptions) IsKillSignalSet() bool {
	return so.KillSignal != ""
}

// IsReloadSignalSet return true if custom reload signal set
func (so *ServiceOptions) IsReloadSignalSet() bool {
	return so.ReloadSignal != ""
}

// EnvString return environment variables as string
func (so *ServiceOptions) EnvString() string {
	if len(so.Env) == 0 {
		return ""
	}

	var clauses []string

	for k, v := range so.Env {
		clauses = append(clauses, k+"="+v)
	}

	sort.Strings(clauses)

	return strings.Join(clauses, " ")
}

// FullLogPath return absolute path to service log
func (so *ServiceOptions) FullLogPath() string {
	if strings.HasPrefix(so.LogPath, "/") {
		return so.LogPath
	}

	return so.WorkingDir + "/" + so.LogPath
}

// ////////////////////////////////////////////////////////////////////////////////// //

// parseCommand parse shell command and extract command body, output redirection
// and environment variables
func parseCommand(command string) (string, string, map[string]string) {
	var (
		env map[string]string
		cmd []string
		log string

		isEnv bool
		isLog bool
	)

	cmdSlice := strings.Fields(command)

	for _, cmdPart := range cmdSlice {
		if strings.TrimSpace(cmdPart) == "" {
			continue
		}

		if strings.HasPrefix(cmdPart, "env") {
			env = make(map[string]string)
			isEnv = true
			continue
		}

		if isEnv {
			if strings.Contains(cmdPart, "=") {
				envSlice := strings.Split(cmdPart, "=")
				env[envSlice[0]] = envSlice[1]
				continue
			} else {
				isEnv = false
			}
		}

		if strings.Contains(cmdPart, ">>") {
			isLog = true
			continue
		}

		if isLog {
			log = cmdPart
			break
		}

		cmd = append(cmd, cmdPart)
	}

	return strings.Join(cmd, " "), log, env
}

// determineProcVersion process procfile data and return procfile version
func determineProcVersion(data []byte) int {
	if regexp.MustCompile(REGEXP_V2_VERSION).Match(data) {
		return 2
	}

	return 1
}

// convertMapType convert map with interface{} to map with string
func convertMapType(m map[interface{}]interface{}) map[string]string {
	result := make(map[string]string)

	for k, v := range m {
		result[k.(string)] = fmt.Sprint(v)
	}

	return result
}

// mergeServiceOptions merge two ServiceOptions structs
func mergeServiceOptions(dst, src *ServiceOptions) {

	mergeStringMaps(dst.Env, src.Env)

	if dst.WorkingDir == "" {
		dst.WorkingDir = src.WorkingDir
	}

	if dst.LogPath == "" {
		dst.LogPath = src.LogPath
	}

	if dst.KillTimeout == 0 {
		dst.KillTimeout = src.KillTimeout
	}

	if dst.RespawnInterval == 0 {
		dst.RespawnInterval = src.RespawnInterval
	}

	if dst.RespawnCount == 0 {
		dst.RespawnCount = src.RespawnCount
	}

	if dst.LimitFile == 0 {
		dst.LimitFile = src.LimitFile
	}

	if dst.LimitProc == 0 {
		dst.LimitProc = src.LimitProc
	}
}

// configureDefaults set options default values
func configureDefaults(serviceOptions *ServiceOptions, config *Config) {
	if serviceOptions.LimitFile == 0 && config.LimitFile != 0 {
		serviceOptions.LimitFile = config.LimitFile
	}

	if serviceOptions.LimitProc == 0 && config.LimitProc != 0 {
		serviceOptions.LimitProc = config.LimitProc
	}

	if serviceOptions.KillTimeout == 0 && config.KillTimeout != 0 {
		serviceOptions.KillTimeout = config.KillTimeout
	}

	if config.IsRespawnEnabled {
		serviceOptions.IsRespawnEnabled = true
	}

	if serviceOptions.IsRespawnEnabled {
		if serviceOptions.RespawnCount == 0 {
			serviceOptions.RespawnCount = config.RespawnCount
		}

		if serviceOptions.RespawnInterval == 0 {
			serviceOptions.RespawnInterval = config.RespawnInterval
		}
	}
}

// mergeStringMaps merges two maps
func mergeStringMaps(dest, src map[string]string) {
	for k, v := range src {
		if dest[k] == "" {
			dest[k] = v
		}
	}
}

// checkPath check path value and return error if value is insecure
func checkPath(value string) error {
	if value == "" {
		return nil
	}

	if !regexp.MustCompile(REGEXP_PATH_CHECK).MatchString(value) {
		return fmt.Errorf("Path %s is insecure and can't be accepted", value)
	}

	if !path.IsSafe(value) {
		return fmt.Errorf("Path %s is not safe and can't be accepted", value)
	}

	return nil
}

// checkValue check any value and return error if value is insecure
func checkValue(value string) error {
	if value == "" {
		return nil
	}

	if !regexp.MustCompile(REGEXP_VALUE_CHECK).MatchString(value) {
		return fmt.Errorf("Value %s is insecure and can't be accepted", value)
	}

	return nil
}

// checkEnv check given env variable and return error if name or value is insecure
func checkEnv(name, value string) error {
	if name == "" || value == "" {
		return nil
	}

	if !regexp.MustCompile(REGEXP_VALUE_CHECK).MatchString(name) {
		return fmt.Errorf("Environment variable name %s is insecure and can't be accepted", value)
	}

	if !regexp.MustCompile(REGEXP_VALUE_CHECK).MatchString(value) {
		return fmt.Errorf("Environment variable value %s is insecure and can't be accepted", value)
	}

	return nil
}

// checkRunLevel check run level value and return error if value is insecure
func checkRunLevel(value int) error {
	if value < 1 {
		return fmt.Errorf("Run level can't be less than 1")
	}

	if value > 6 {
		return fmt.Errorf("Run level can't be greater than 6")
	}

	return nil
}

// addCrossLink add to all service structs pointer
// to parent application struct
func addCrossLink(app *Application) {
	for _, service := range app.Services {
		service.Application = app
	}
}

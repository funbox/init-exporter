package procfile

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                       Copyright (c) 2006-2020 FB GROUP LLC                         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"sort"
	"strings"

	"pkg.re/essentialkaos/ek.v12/errutil"
	"pkg.re/essentialkaos/ek.v12/fsutil"
	"pkg.re/essentialkaos/ek.v12/log"
	"pkg.re/essentialkaos/ek.v12/path"
	"pkg.re/essentialkaos/ek.v12/sliceutil"
	"pkg.re/essentialkaos/ek.v12/strutil"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	REGEXP_V1_LINE            = `^([A-z\d_]+):\s*(.+)`
	REGEXP_V2_VERSION         = `(?m)^\s*version:\s*2\s*$`
	REGEXP_PATH_CHECK         = `\A[A-Za-z0-9_\-./]+\z`
	REGEXP_NAME_CHECK         = `\A[A-Za-z0-9_\-]+\z`
	REGEXP_NET_DEVICE_CHECK   = `eth[0-9]|e[nm][0-9]|p[0-9][ps][0-9]|wlan|wl[0-9]|wlp[0-9]|bond[0-9]`
	REGEXP_CPU_AFFINITY_CHECK = `^[\d\- ]+$`
)

// ////////////////////////////////////////////////////////////////////////////////// //

type Config struct {
	Name             string // Application name
	User             string // Working user
	Group            string // Working group
	WorkingDir       string // Working directory
	RespawnInterval  int    // Global respawn interval in seconds
	RespawnCount     int    // Global respawn count
	KillTimeout      int    // Global kill timeout in seconds
	LimitProc        int    // Global processes limit
	LimitFile        int    // Global descriptors limit
	LimitMemlock     int    // Global max locked memory limit
	IsRespawnEnabled bool   // Global respawn enabled flag
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
	EnvFile          string            // Path to file with environment variables
	WorkingDir       string            // Working directory
	LogFile          string            // Path to log file
	KillTimeout      int               // Kill timeout in seconds
	KillSignal       string            // Kill signal name
	KillMode         string            // Kill mode (systemd only)
	ReloadSignal     string            // Reload signal name (systemd only)
	Count            int               // Exec count
	RespawnInterval  int               // Respawn interval in seconds
	RespawnCount     int               // Respawn count
	LimitProc        int               // Processes limit
	LimitFile        int               // Descriptors limit
	LimitMemlock     int               // Max locked memory limit
	Resources        *Resources        // Resources limits (systemd only)
	IsRespawnEnabled bool              // Respawn enabled flag
}

type Resources struct {
	CPUWeight           int
	StartupCPUWeight    int
	CPUQuota            int
	CPUAffinity         string
	MemoryLow           string
	MemoryHigh          string
	MemoryMax           string
	MemorySwapMax       string
	TasksMax            int
	IOWeight            int
	StartupIOWeight     int
	IODeviceWeight      string
	IOReadBandwidthMax  string
	IOWriteBandwidthMax string
	IOReadIOPSMax       string
	IOWriteIOPSMax      string
	IPAddressAllow      string
	IPAddressDeny       string
}

type Application struct {
	Name               string     // Name of application
	Services           []*Service // List of services in application
	User               string     // Working user
	Group              string     // Working group
	StartLevel         int        // Start level
	StopLevel          int        // Stop level
	StartDevice        string     // Start on device activation
	Depends            []string   // Dependencies
	WorkingDir         string     // Working directory
	ReloadHelperPath   string     // Path to reload helper (will be set by exporter)
	ProcVersion        int        // Proc version 1/2
	StrongDependencies bool       // Use strong dependencies
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
func (a *Application) Validate() []error {
	errs := errutil.NewErrors()

	errs.Add(checkRunLevel(a.StartLevel))
	errs.Add(checkRunLevel(a.StopLevel))
	errs.Add(checkDependencies(a.Depends))

	if a.WorkingDir == "" {
		errs.Add(fmt.Errorf("Application working dir can't be empty"))
	}

	if a.StartDevice != "" && !regexp.MustCompile(REGEXP_NET_DEVICE_CHECK).MatchString(a.StartDevice) {
		errs.Add(fmt.Errorf("Name of device (%s) is not a valid", a.StartDevice))
	}

	for _, service := range a.Services {
		errs.Add(service.Validate())
	}

	return errs.All()
}

// IsReloadSignalSet returns true if any service contains reload signal
func (a *Application) IsReloadSignalSet() bool {
	for _, service := range a.Services {
		if service.Options.ReloadSignal != "" {
			return true
		}
	}

	return false
}

// Validate validate service props and options
func (s *Service) Validate() *errutil.Errors {
	errs := errutil.NewErrors()

	if !regexp.MustCompile(REGEXP_NAME_CHECK).MatchString(s.Name) {
		errs.Add(fmt.Errorf("Service name %s is misformatted and can't be accepted", s.Name))
	}

	errs.Add(s.Options.Validate())

	return errs
}

// Validate validate service options
func (so *ServiceOptions) Validate() *errutil.Errors {
	errs := errutil.NewErrors()

	errs.Add(checkPath(so.WorkingDir))

	if so.IsCustomLogEnabled() {
		errs.Add(checkPath(so.FullLogPath()))
	}

	if so.IsEnvFileSet() {
		errs.Add(checkPath(so.FullEnvFilePath()))
	}

	for envName, envVal := range so.Env {
		errs.Add(checkEnv(envName, envVal))
	}

	if so.Count < 0 {
		errs.Add(fmt.Errorf("Property \"count\" must be greater or equal 0"))
	}

	if so.KillTimeout < 0 {
		errs.Add(fmt.Errorf("Property \"kill_timeout\" must be greater or equal 0"))
	}

	if so.LimitFile < 0 {
		errs.Add(fmt.Errorf("Property \"nofile\" must be greater or equal 0"))
	}

	if so.LimitProc < 0 {
		errs.Add(fmt.Errorf("Property \"nproc\" must be greater or equal 0"))
	}

	if so.RespawnCount < 0 {
		errs.Add(fmt.Errorf("Property \"respawn:count\" must be greater or equal 0"))
	}

	if so.RespawnInterval < 0 {
		errs.Add(fmt.Errorf("Property \"respawn:interval\" must be greater or equal 0"))
	}

	if so.KillMode != "" && !sliceutil.Contains([]string{"control-group", "process", "mixed", "none"}, so.KillMode) {
		errs.Add(fmt.Errorf("Property \"kill_mode\" must contains 'control-group', 'process', 'mixed' or 'none'"))
	}

	if so.Resources != nil {
		if so.Resources.CPUWeight < 0 || so.Resources.CPUWeight > 10000 {
			errs.Add(fmt.Errorf("Property \"resources:cpu_weight\" must be greater or equal 0 and less or equal 10000"))
		}

		if so.Resources.CPUAffinity != "" && !regexp.MustCompile(REGEXP_CPU_AFFINITY_CHECK).MatchString(so.Resources.CPUAffinity) {
			errs.Add(fmt.Errorf("Property \"resources:cpu_affinity\" contains misformatted value"))
		}

		if so.Resources.StartupCPUWeight < 0 || so.Resources.StartupCPUWeight > 10000 {
			errs.Add(fmt.Errorf("Property \"resources:startup_cpu_weight\" must be greater or equal 0 and less or equal 10000"))
		}

		if so.Resources.CPUQuota < 0 {
			errs.Add(fmt.Errorf("Property \"resources:cpu_quota\" must be greater than 0"))
		}

		if so.Resources.IOWeight < 0 || so.Resources.IOWeight > 10000 {
			errs.Add(fmt.Errorf("Property \"resources:io_weight\" must be greater or equal 0 and less or equal 10000"))
		}

		if so.Resources.StartupIOWeight < 0 || so.Resources.StartupIOWeight > 10000 {
			errs.Add(fmt.Errorf("Property \"resources:startup_io_weight\" must be greater or equal 0 and less or equal 10000"))
		}
	}

	return errs
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
func (s *Service) GetCommandExec(command string) string {
	var result = "exec "

	if s.Options.IsEnvSet() || s.Options.IsEnvFileSet() {
		result += "env "

		if s.Options.IsEnvFileSet() {
			result += "$(cat " + s.Options.FullEnvFilePath() + " 2>/dev/null | xargs) "
		}

		if s.Options.IsEnvSet() {
			result += s.Options.EnvString() + " "
		}
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

// IsRespawnLimitSet returns true if respawn options is set
func (so *ServiceOptions) IsRespawnLimitSet() bool {
	return so.RespawnCount != 0 || so.RespawnInterval != 0
}

// IsCustomLogEnabled returns true if service have custom log
func (so *ServiceOptions) IsCustomLogEnabled() bool {
	return so.LogFile != ""
}

// IsEnvSet returns true if service have custom env vars
func (so *ServiceOptions) IsEnvSet() bool {
	return len(so.Env) != 0
}

// IsEnvFileSet returns true if service have file with env vars
func (so *ServiceOptions) IsEnvFileSet() bool {
	return so.EnvFile != ""
}

// IsFileLimitSet returns true if descriptors limit is set
func (so *ServiceOptions) IsFileLimitSet() bool {
	return so.LimitFile != 0
}

// IsProcLimitSet returns true if processes limit is set
func (so *ServiceOptions) IsProcLimitSet() bool {
	return so.LimitProc != 0
}

// IsMemlockLimitSet returns true if max memory limit is set
func (so *ServiceOptions) IsMemlockLimitSet() bool {
	return so.LimitMemlock != 0
}

// IsKillSignalSet returns true if custom kill signal set
func (so *ServiceOptions) IsKillSignalSet() bool {
	return so.KillSignal != ""
}

// IsKillModeSet returns true if custom kill mode set
func (so *ServiceOptions) IsKillModeSet() bool {
	return so.KillMode != ""
}

// IsResourcesSet returns true if resources limits are set
func (so *ServiceOptions) IsResourcesSet() bool {
	return so.Resources != nil
}

// IsReloadSignalSet returns true if custom reload signal set
func (so *ServiceOptions) IsReloadSignalSet() bool {
	return so.ReloadSignal != ""
}

// EnvString returns environment variables as string
func (so *ServiceOptions) EnvString() string {
	if len(so.Env) == 0 {
		return ""
	}

	var clauses []string

	for k, v := range so.Env {
		clauses = append(clauses, fmt.Sprintf("%s=%s", k, v))
	}

	sort.Strings(clauses)

	return strings.Join(clauses, " ")
}

// FullLogPath return absolute path to service log
func (so *ServiceOptions) FullLogPath() string {
	if strings.HasPrefix(so.LogFile, "/") {
		return so.LogFile
	}

	return so.WorkingDir + "/" + so.LogFile
}

// FullEnvFilePath return absolute path to file with env vars
func (so *ServiceOptions) FullEnvFilePath() string {
	if strings.HasPrefix(so.EnvFile, "/") {
		return so.EnvFile
	}

	return so.WorkingDir + "/" + so.EnvFile
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

	for index, token := range cmdSlice {
		if strings.TrimSpace(token) == "" {
			continue
		}

		if strings.HasPrefix(token, "env") {
			env = make(map[string]string)
			isEnv = true
			continue
		}

		if index == 0 && strings.Contains(token, "=") {
			env = make(map[string]string)
			isEnv = true
		}

		if isEnv {
			if strings.Contains(token, "=") {
				env[strutil.ReadField(token, 0, false, "=")] = strutil.ReadField(token, 1, false, "=")
				continue
			} else {
				isEnv = false
			}
		}

		if strings.Contains(token, ">>") {
			isLog = true
			continue
		}

		if isLog {
			log = token
			break
		}

		cmd = append(cmd, token)
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

	if src.IsEnvSet() {
		mergeStringMaps(dst.Env, src.Env)
	}

	if dst.EnvFile == "" {
		dst.EnvFile = src.EnvFile
	}

	if dst.WorkingDir == "" {
		dst.WorkingDir = src.WorkingDir
	}

	if dst.LogFile == "" {
		dst.LogFile = src.LogFile
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

	if dst.LimitProc == 0 {
		dst.LimitProc = src.LimitProc
	}

	if dst.LimitMemlock == 0 {
		dst.LimitMemlock = src.LimitMemlock
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

// checkPath checks path value and return error if value is insecure
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

// checkEnv checks given env variable and return error if name or value is insecure
func checkEnv(name, value string) error {
	if name == "" {
		return nil
	}

	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("Environment variable %s has empty value", name)
	}

	if strings.Contains(value, " ") {
		if isUnquotedValue(value) {
			return fmt.Errorf("Environment variable %s has unquoted value with spaces", name)
		}
	}

	if strings.Contains(value, "*") {
		if isUnquotedValue(value) {
			return fmt.Errorf("Environment variable %s has unquoted asterisk symbol", name)
		}
	}

	if !regexp.MustCompile(REGEXP_NAME_CHECK).MatchString(name) {
		return fmt.Errorf("Environment variable name %s is misformatted and can't be accepted", name)
	}

	return nil
}

// checkRunLevel checks run level value and return error if value is insecure
func checkRunLevel(value int) error {
	if value < 1 {
		return fmt.Errorf("Run level can't be less than 1")
	}

	if value > 6 {
		return fmt.Errorf("Run level can't be greater than 6")
	}

	return nil
}

// checkDependencies checks dependencies
func checkDependencies(deps []string) *errutil.Errors {
	if len(deps) == 0 {
		return nil
	}

	errs := errutil.NewErrors()

	for _, dep := range deps {
		if !regexp.MustCompile(REGEXP_NAME_CHECK).MatchString(dep) {
			errs.Add(fmt.Errorf("Dependency name %s is misformatted and can't be accepted", dep))
		}
	}

	return nil
}

// addCrossLink adds to all service structs pointer
// to parent application struct
func addCrossLink(app *Application) {
	for _, service := range app.Services {
		service.Application = app
	}
}

// isUnquotedValue returns true if given value is unquoted
func isUnquotedValue(value string) bool {
	if !strings.Contains(value, "\"") && !strings.Contains(value, "'") {
		return true
	}

	if strings.Contains(value, "\"") {
		if strings.Count(value, "\"")%2 != 0 {
			return true
		}
	}

	if strings.Contains(value, "'") {
		if strings.Count(value, "'")%2 != 0 {
			return true
		}
	}

	return false
}

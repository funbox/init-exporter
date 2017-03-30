package procfile

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                       Copyright (c) 2006-2017 FB GROUP LLC                         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"sort"
	"strings"

	"pkg.re/essentialkaos/ek.v7/errutil"
	"pkg.re/essentialkaos/ek.v7/fsutil"
	"pkg.re/essentialkaos/ek.v7/log"
	"pkg.re/essentialkaos/ek.v7/path"

	"pkg.re/essentialkaos/go-simpleyaml.v1"
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
		result += " >>" + s.Options.FullLogPath()
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
		result += " >>" + s.Options.FullLogPath()
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

// parseV1Procfile parse v1 procfile data
func parseV1Procfile(data []byte, config *Config) (*Application, error) {
	if config == nil {
		return nil, fmt.Errorf("Config is nil")
	}

	log.Debug("Parsing procfile as v1")

	var services []*Service

	reader := bytes.NewReader(data)
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		switch {
		case line == "":
			// Skip empty line
		case strings.HasPrefix(line, "#"):
			// Skip comment
		default:
			service, err := parseV1Line(line)

			if err != nil {
				return nil, err
			}

			if service.Options.LimitFile == 0 && config.LimitFile != 0 {
				service.Options.LimitFile = config.LimitFile
			}

			if service.Options.LimitProc == 0 && config.LimitProc != 0 {
				service.Options.LimitProc = config.LimitProc
			}

			services = append(services, service)
		}
	}

	err := scanner.Err()

	if err != nil {
		return nil, err
	}

	app := &Application{
		ProcVersion: 1,
		Name:        config.Name,
		User:        config.User,
		StartLevel:  3,
		StopLevel:   3,
		Group:       config.Group,
		WorkingDir:  config.WorkingDir,
		Services:    services,
	}

	addCrossLink(app)

	return app, nil
}

// parseV1Line parse v1 procfile line
func parseV1Line(line string) (*Service, error) {
	re := regexp.MustCompile(REGEXP_V1_LINE)
	matches := re.FindStringSubmatch(line)

	if len(matches) != 3 {
		return nil, fmt.Errorf("Procfile v1 should have format: 'some_label: command'")
	}

	cmd, options := parseV1Command(matches[2])

	return &Service{Name: matches[1], Cmd: cmd, Options: options}, nil
}

// parseV1Command parse command and extract command and working dir
func parseV1Command(cmd string) (string, *ServiceOptions) {
	var options = &ServiceOptions{}

	if !strings.HasPrefix(cmd, "cd ") && !strings.Contains(cmd, "&&") {
		return cmd, options
	}

	cmdSlice := strings.Split(cmd, "&&")
	command := strings.TrimSpace(cmdSlice[1])
	workingDir := strings.Replace(cmdSlice[0], "cd", "", -1)

	options.WorkingDir = strings.TrimSpace(workingDir)

	if strings.HasPrefix(command, "env ") {
		evMap := make(map[string]string)

		subCommandSlice := strings.Fields(command)

		for i, commandPart := range subCommandSlice {
			if commandPart == "env" {
				continue
			}

			if !strings.Contains(commandPart, "=") {
				command = strings.Join(subCommandSlice[i:], " ")
				break
			}

			envSlice := strings.Split(commandPart, "=")
			evMap[envSlice[0]] = envSlice[1]
		}

		options.Env = evMap
	}

	return command, options
}

// parseV2Procfile parse v2 procfile data
func parseV2Procfile(data []byte, config *Config) (*Application, error) {
	var err error

	log.Debug("Parsing procfile as v2")

	yaml, err := simpleyaml.NewYaml(data)

	if err != nil {
		return nil, err
	}

	commands, err := yaml.Get("commands").Map()

	if err != nil {
		return nil, fmt.Errorf("Commands missing in Procfile")
	}

	services, err := parseCommands(yaml, commands, config)

	if err != nil {
		return nil, err
	}

	app := &Application{
		ProcVersion: 2,
		Name:        config.Name,
		User:        config.User,
		Group:       config.Group,
		StartLevel:  3,
		StopLevel:   3,
		WorkingDir:  config.WorkingDir,
		Services:    services,
	}

	if yaml.IsExist("working_directory") {
		app.WorkingDir, err = yaml.Get("working_directory").String()

		if err != nil {
			return nil, fmt.Errorf("Can't parse working_directory value: %v", err)
		}
	}

	if yaml.IsExist("start_on_runlevel") {
		app.StartLevel, err = yaml.Get("start_on_runlevel").Int()

		if err != nil {
			return nil, fmt.Errorf("Can't parse start_on_runlevel value: %v", err)
		}
	}

	if yaml.IsExist("stop_on_runlevel") {
		app.StopLevel, err = yaml.Get("stop_on_runlevel").Int()

		if err != nil {
			return nil, fmt.Errorf("Can't parse stop_on_runlevel value: %v", err)
		}
	}

	addCrossLink(app)

	return app, nil
}

// parseCommands parse command section in yaml based procfile
func parseCommands(yaml *simpleyaml.Yaml, commands map[interface{}]interface{}, config *Config) ([]*Service, error) {
	var services []*Service

	commonOptions, err := parseOptions(yaml)

	if err != nil {
		return nil, err
	}

	for key := range commands {
		serviceName := fmt.Sprint(key)
		commandYaml := yaml.GetPath("commands", serviceName)
		serviceCmd, err := commandYaml.Get("command").String()

		if err != nil {
			return nil, err
		}

		servicePreCmd := commandYaml.Get("pre").MustString()
		servicePostCmd := commandYaml.Get("post").MustString()

		serviceOptions, err := parseOptions(commandYaml)

		if err != nil {
			return nil, err
		}

		mergeServiceOptions(serviceOptions, commonOptions)
		configureDefaults(serviceOptions, config)

		service := &Service{
			Name:    serviceName,
			Cmd:     serviceCmd,
			PreCmd:  servicePreCmd,
			PostCmd: servicePostCmd,
			Options: serviceOptions,
		}

		services = append(services, service)
	}

	return services, nil
}

// parseOptions parse service options in yaml based procfile
func parseOptions(yaml *simpleyaml.Yaml) (*ServiceOptions, error) {
	var err error

	options := &ServiceOptions{
		Env:              make(map[string]string),
		IsRespawnEnabled: true,
	}

	if yaml.IsExist("working_directory") {
		options.WorkingDir, err = yaml.Get("working_directory").String()

		if err != nil {
			return nil, fmt.Errorf("Can't parse working_directory value: %v", err)
		}
	}

	if yaml.IsExist("log") {
		options.LogPath, err = yaml.Get("log").String()

		if err != nil {
			return nil, fmt.Errorf("Can't parse log value: %v", err)
		}
	}

	if yaml.IsExist("kill_timeout") {
		options.KillTimeout, err = yaml.Get("kill_timeout").Int()

		if err != nil {
			return nil, fmt.Errorf("Can't parse kill_timeout value: %v", err)
		}
	}

	if yaml.IsExist("count") {
		options.Count, err = yaml.Get("count").Int()

		if err != nil {
			return nil, fmt.Errorf("Can't parse count value: %v", err)
		}
	}

	if yaml.IsExist("env") {
		env, err := yaml.Get("env").Map()

		if err != nil {
			return nil, fmt.Errorf("Can't parse env value: %v", err)
		}

		options.Env = convertMapType(env)
	}

	if yaml.IsPathExist("respawn", "count") || yaml.IsPathExist("respawn", "interval") {
		if yaml.IsPathExist("respawn", "count") {
			options.RespawnCount, err = yaml.Get("respawn").Get("count").Int()

			if err != nil {
				return nil, fmt.Errorf("Can't parse respawn.count value: %v", err)
			}
		}

		if yaml.IsPathExist("respawn", "interval") {
			options.RespawnInterval, err = yaml.Get("respawn").Get("interval").Int()

			if err != nil {
				return nil, fmt.Errorf("Can't parse respawn.interval value: %v", err)
			}
		}

	} else if yaml.IsExist("respawn") {
		options.IsRespawnEnabled, err = yaml.Get("respawn").Bool()

		if err != nil {
			return nil, fmt.Errorf("Can't parse respawn value: %v", err)
		}
	}

	if yaml.IsPathExist("limits", "nproc") || yaml.IsPathExist("limits", "nofile") {
		if yaml.IsPathExist("limits", "nofile") {
			options.LimitFile, err = yaml.Get("limits").Get("nofile").Int()

			if err != nil {
				return nil, fmt.Errorf("Can't parse limits.nofile value: %v", err)
			}
		}

		if yaml.IsPathExist("limits", "nproc") {
			options.LimitProc, err = yaml.Get("limits").Get("nproc").Int()

			if err != nil {
				return nil, fmt.Errorf("Can't parse limits.nproc value: %v", err)
			}
		}
	}

	return options, nil
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

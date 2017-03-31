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
	"regexp"
	"strings"

	"pkg.re/essentialkaos/ek.v7/log"
)

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

	return parseV1Command(matches[1], matches[2]), nil
}

// parseV1Command parse command and extract command and working dir
func parseV1Command(name, command string) *Service {
	var service = &Service{Name: name, Options: &ServiceOptions{}}

	cmdSlice := splitV1Command(command)

	if strings.HasPrefix(cmdSlice[0], "cd") {
		service.Options.WorkingDir = strings.Replace(cmdSlice[0], "cd ", "", -1)
		cmdSlice = cmdSlice[1:]
	}

	var (
		env  map[string]string
		cmd  string
		pre  string
		post string
		log  string
	)

	switch len(cmdSlice) {
	case 3:
		pre, _, _ = extractV1Data(cmdSlice[0])
		cmd, log, env = extractV1Data(cmdSlice[1])
		post, _, _ = extractV1Data(cmdSlice[2])
	case 2:
		pre, _, _ = extractV1Data(cmdSlice[0])
		cmd, log, env = extractV1Data(cmdSlice[1])
	default:
		cmd, log, env = extractV1Data(cmdSlice[0])
	}

	service.Cmd = cmd
	service.PreCmd = pre
	service.PostCmd = post
	service.Options.Env = env
	service.Options.LogPath = log

	return service
}

// extractV1Data extract data from command
func extractV1Data(command string) (string, string, map[string]string) {
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

// splitV1Cmd split command and format command
func splitV1Command(cmd string) []string {
	var result []string

	for _, cmdPart := range strings.Split(cmd, "&&") {
		result = append(result, strings.TrimSpace(cmdPart))
	}

	return result
}

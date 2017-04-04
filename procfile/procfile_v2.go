package procfile

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                       Copyright (c) 2006-2017 FB GROUP LLC                         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"

	"pkg.re/essentialkaos/ek.v7/log"

	"pkg.re/essentialkaos/go-simpleyaml.v1"
)

// ////////////////////////////////////////////////////////////////////////////////// //

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

	services, err := parseV2Commands(yaml, commands, config)

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
func parseV2Commands(yaml *simpleyaml.Yaml, commands map[interface{}]interface{}, config *Config) ([]*Service, error) {
	var services []*Service

	commonOptions, err := parseV2Options(yaml)

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

		serviceOptions, err := parseV2Options(commandYaml)

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

// parseV2Options parse service options in yaml based procfile
func parseV2Options(yaml *simpleyaml.Yaml) (*ServiceOptions, error) {
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

	if yaml.IsExist("kill_signal") {
		options.KillSignal, err = yaml.Get("kill_signal").String()

		if err != nil {
			return nil, fmt.Errorf("Can't parse kill_signal value: %v", err)
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

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

	services, err := parseV2Services(yaml, commands, config)

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

// parseV2Services parse services sections in yaml based procfile
func parseV2Services(yaml *simpleyaml.Yaml, commands map[interface{}]interface{}, config *Config) ([]*Service, error) {
	var services []*Service

	commonOptions := &ServiceOptions{}
	err := parseV2Options(commonOptions, yaml)

	if err != nil {
		return nil, err
	}

	for key := range commands {
		service := &Service{
			Name:    fmt.Sprint(key),
			Options: &ServiceOptions{},
		}

		serviceYaml := yaml.GetPath("commands", service.Name)

		err := parseV2Commands(service, serviceYaml)

		if err != nil {
			return nil, err
		}

		err = parseV2Options(service.Options, serviceYaml)

		if err != nil {
			return nil, err
		}

		mergeServiceOptions(service.Options, commonOptions)
		configureDefaults(service.Options, config)

		services = append(services, service)
	}

	return services, nil
}

// parseV2Commands parse service commands
func parseV2Commands(service *Service, yaml *simpleyaml.Yaml) error {
	var err error
	var cmd, log string

	cmd, err = yaml.Get("command").String()

	if err != nil {
		return fmt.Errorf("Can't parse \"command\" value: %v", err)
	}

	cmd, log, _ = parseCommand(cmd)

	if log != "" {
		service.Options.LogFile = log
	}

	service.Cmd = cmd

	if yaml.IsExist("pre") {
		service.PreCmd, err = yaml.Get("pre").String()

		if err != nil {
			return fmt.Errorf("Can't parse \"pre\" value: %v", err)
		}
	}

	if yaml.IsExist("post") {
		service.PostCmd, err = yaml.Get("post").String()

		if err != nil {
			return fmt.Errorf("Can't parse \"post\" value: %v", err)
		}
	}

	return nil
}

// parseV2Options parse service options in yaml based procfile
func parseV2Options(options *ServiceOptions, yaml *simpleyaml.Yaml) error {
	var err error

	options.Env = make(map[string]string)
	options.IsRespawnEnabled = true

	if yaml.IsExist("working_directory") {
		options.WorkingDir, err = yaml.Get("working_directory").String()

		if err != nil {
			return fmt.Errorf("Can't parse \"working_directory\" value: %v", err)
		}
	}

	if yaml.IsExist("log") {
		options.LogFile, err = yaml.Get("log").String()

		if err != nil {
			return fmt.Errorf("Can't parse \"log\" value: %v", err)
		}
	}

	if yaml.IsExist("kill_timeout") {
		options.KillTimeout, err = yaml.Get("kill_timeout").Int()

		if err != nil {
			return fmt.Errorf("Can't parse \"kill_timeout\" value: %v", err)
		}
	}

	if yaml.IsExist("kill_signal") {
		options.KillSignal, err = yaml.Get("kill_signal").String()

		if err != nil {
			return fmt.Errorf("Can't parse \"kill_signal\" value: %v", err)
		}
	}

	if yaml.IsExist("reload_signal") {
		options.ReloadSignal, err = yaml.Get("reload_signal").String()

		if err != nil {
			return fmt.Errorf("Can't parse \"reload_signal\" value: %v", err)
		}
	}

	if yaml.IsExist("count") {
		options.Count, err = yaml.Get("count").Int()

		if err != nil {
			return fmt.Errorf("Can't parse \"count\" value: %v", err)
		}
	}

	if yaml.IsExist("env") {
		env, err := yaml.Get("env").Map()

		if err != nil {
			return fmt.Errorf("Can't parse \"env\" value: %v", err)
		}

		options.Env = convertMapType(env)
	}

	if yaml.IsExist("env_file") {
		options.EnvFile, err = yaml.Get("env_file").String()

		if err != nil {
			return fmt.Errorf("Can't parse \"env_file\" value: %v", err)
		}
	}

	if yaml.IsPathExist("respawn", "count") || yaml.IsPathExist("respawn", "interval") {
		if yaml.IsPathExist("respawn", "count") {
			options.RespawnCount, err = yaml.Get("respawn").Get("count").Int()

			if err != nil {
				return fmt.Errorf("Can't parse \"respawn.count\" value: %v", err)
			}
		}

		if yaml.IsPathExist("respawn", "interval") {
			options.RespawnInterval, err = yaml.Get("respawn").Get("interval").Int()

			if err != nil {
				return fmt.Errorf("Can't parse \"respawn.interval\" value: %v", err)
			}
		}

	} else if yaml.IsExist("respawn") {
		options.IsRespawnEnabled, err = yaml.Get("respawn").Bool()

		if err != nil {
			return fmt.Errorf("Can't parse \"respawn\" value: %v", err)
		}
	}

	if yaml.IsPathExist("limits", "nproc") || yaml.IsPathExist("limits", "nofile") {
		if yaml.IsPathExist("limits", "nofile") {
			options.LimitFile, err = yaml.Get("limits").Get("nofile").Int()

			if err != nil {
				return fmt.Errorf("Can't parse \"limits.nofile\" value: %v", err)
			}
		}

		if yaml.IsPathExist("limits", "nproc") {
			options.LimitProc, err = yaml.Get("limits").Get("nproc").Int()

			if err != nil {
				return fmt.Errorf("Can't parse \"limits.nproc\" value: %v", err)
			}
		}
	}

	return nil
}

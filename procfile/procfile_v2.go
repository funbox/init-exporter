package procfile

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                       Copyright (c) 2006-2018 FB GROUP LLC                         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"

	"pkg.re/essentialkaos/ek.v10/log"

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
			return formatPropError("working_directory", err)
		}
	}

	if yaml.IsExist("log") {
		options.LogFile, err = yaml.Get("log").String()

		if err != nil {
			return formatPropError("log", err)
		}
	}

	if yaml.IsExist("kill_timeout") {
		options.KillTimeout, err = yaml.Get("kill_timeout").Int()

		if err != nil {
			return formatPropError("kill_timeout", err)
		}
	}

	if yaml.IsExist("kill_signal") {
		options.KillSignal, err = yaml.Get("kill_signal").String()

		if err != nil {
			return formatPropError("kill_signal", err)
		}
	}

	if yaml.IsExist("kill_mode") {
		options.KillMode, err = yaml.Get("kill_mode").String()

		if err != nil {
			return formatPropError("kill_mode", err)
		}
	}

	if yaml.IsExist("reload_signal") {
		options.ReloadSignal, err = yaml.Get("reload_signal").String()

		if err != nil {
			return formatPropError("reload_signal", err)
		}
	}

	if yaml.IsExist("count") {
		options.Count, err = yaml.Get("count").Int()

		if err != nil {
			return formatPropError("count", err)
		}
	}

	if yaml.IsExist("env") {
		env, err := yaml.Get("env").Map()

		if err != nil {
			return formatPropError("env", err)
		}

		options.Env = convertMapType(env)
	}

	if yaml.IsExist("env_file") {
		options.EnvFile, err = yaml.Get("env_file").String()

		if err != nil {
			return formatPropError("env_file", err)
		}
	}

	if yaml.IsPathExist("respawn", "count") || yaml.IsPathExist("respawn", "interval") {
		if yaml.IsPathExist("respawn", "count") {
			options.RespawnCount, err = yaml.Get("respawn").Get("count").Int()

			if err != nil {
				return formatPropError("respawn:count", err)
			}
		}

		if yaml.IsPathExist("respawn", "interval") {
			options.RespawnInterval, err = yaml.Get("respawn").Get("interval").Int()

			if err != nil {
				return formatPropError("respawn:interval", err)
			}
		}

	} else if yaml.IsExist("respawn") {
		options.IsRespawnEnabled, err = yaml.Get("respawn").Bool()

		if err != nil {
			return formatPropError("respawn", err)
		}
	}

	if yaml.IsExist("limits") {
		if yaml.IsPathExist("limits", "nofile") {
			options.LimitFile, err = yaml.Get("limits").Get("nofile").Int()

			if err != nil {
				return formatPropError("limits:nofile", err)
			}
		}

		if yaml.IsPathExist("limits", "nproc") {
			options.LimitProc, err = yaml.Get("limits").Get("nproc").Int()

			if err != nil {
				return formatPropError("limits:nproc", err)
			}
		}
	}

	if yaml.IsExist("resources") {
		options.Resources, err = parseV2Resources(yaml.Get("resources"))

		if err != nil {
			return err
		}
	}

	return nil
}

// parseV2Resources parse service resources options in yaml based procfile
func parseV2Resources(yaml *simpleyaml.Yaml) (*Resources, error) {
	var err error

	resources := &Resources{}

	if yaml.IsExist("cpu_weight") {
		resources.CPUWeight, err = yaml.Get("cpu_weight").Int()

		if err != nil {
			return nil, formatPropError("resources:cpu_weight", err)
		}
	}

	if yaml.IsExist("startup_cpu_weight") {
		resources.StartupCPUWeight, err = yaml.Get("startup_cpu_weight").Int()

		if err != nil {
			return nil, formatPropError("resources:startup_cpu_weight", err)
		}
	}

	if yaml.IsExist("cpu_quota") {
		resources.CPUQuota, err = yaml.Get("cpu_quota").Int()

		if err != nil {
			return nil, formatPropError("resources:cpu_quota", err)
		}
	}

	if yaml.IsExist("memory_low") {
		resources.MemoryLow, err = yaml.Get("memory_low").String()

		if err != nil {
			return nil, formatPropError("resources:memory_low", err)
		}
	}

	if yaml.IsExist("memory_high") {
		resources.MemoryHigh, err = yaml.Get("memory_high").String()

		if err != nil {
			return nil, formatPropError("resources:memory_high", err)
		}
	}

	if yaml.IsExist("memory_max") {
		resources.MemoryMax, err = yaml.Get("memory_max").String()

		if err != nil {
			return nil, formatPropError("resources:memory_max", err)
		}
	}

	if yaml.IsExist("memory_swap_max") {
		resources.MemorySwapMax, err = yaml.Get("memory_swap_max").String()

		if err != nil {
			return nil, formatPropError("resources:memory_swap_max", err)
		}
	}

	if yaml.IsExist("task_max") {
		resources.TasksMax, err = yaml.Get("task_max").Int()

		if err != nil {
			return nil, formatPropError("resources:task_max", err)
		}
	}

	if yaml.IsExist("io_weight") {
		resources.IOWeight, err = yaml.Get("io_weight").Int()

		if err != nil {
			return nil, formatPropError("resources:io_weight", err)
		}
	}

	if yaml.IsExist("startup_io_weight") {
		resources.StartupIOWeight, err = yaml.Get("startup_io_weight").Int()

		if err != nil {
			return nil, formatPropError("resources:startup_io_weight", err)
		}
	}

	if yaml.IsExist("io_device_weight") {
		resources.IODeviceWeight, err = yaml.Get("io_device_weight").String()

		if err != nil {
			return nil, formatPropError("resources:io_device_weight", err)
		}
	}

	if yaml.IsExist("io_read_bandwidth_max") {
		resources.IOReadBandwidthMax, err = yaml.Get("io_read_bandwidth_max").String()

		if err != nil {
			return nil, formatPropError("resources:io_read_bandwidth_max", err)
		}
	}

	if yaml.IsExist("io_write_bandwidth_max") {
		resources.IOWriteBandwidthMax, err = yaml.Get("io_write_bandwidth_max").String()

		if err != nil {
			return nil, formatPropError("resources:io_write_bandwidth_max", err)
		}
	}

	if yaml.IsExist("io_read_iops_max") {
		resources.IOReadIOPSMax, err = yaml.Get("io_read_iops_max").String()

		if err != nil {
			return nil, formatPropError("resources:io_read_iops_max", err)
		}
	}

	if yaml.IsExist("io_write_iops_max") {
		resources.IOWriteIOPSMax, err = yaml.Get("io_write_iops_max").String()

		if err != nil {
			return nil, formatPropError("resources:io_write_iops_max", err)
		}
	}

	if yaml.IsExist("ip_address_allow") {
		resources.IPAddressAllow, err = yaml.Get("ip_address_allow").String()

		if err != nil {
			return nil, formatPropError("resources:ip_address_allow", err)
		}
	}

	if yaml.IsExist("ip_address_deny") {
		resources.IPAddressDeny, err = yaml.Get("ip_address_deny").String()

		if err != nil {
			return nil, formatPropError("resources:ip_address_deny", err)
		}
	}

	return resources, nil
}

// formatPropError format property parsing error
func formatPropError(prop string, err error) error {
	return fmt.Errorf("Can't parse \"%s\" value: %v", prop, err)
}

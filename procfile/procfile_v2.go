package procfile

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                           Copyright (c) 2006-2021 FUNBOX                           //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"strings"

	"pkg.re/essentialkaos/ek.v12/log"
	"pkg.re/essentialkaos/ek.v12/strutil"

	"pkg.re/essentialkaos/go-simpleyaml.v2"
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
		app.WorkingDir = yamlGetSafe(yaml, "working_directory")
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

	if yaml.IsExist("start_on_device") {
		app.StartDevice = yamlGetSafe(yaml, "start_on_device")
	}

	if yaml.IsExist("strong_dependencies") {
		app.StrongDependencies, err = yaml.Get("strong_dependencies").Bool()

		if err != nil {
			return nil, fmt.Errorf("Can't parse strong_dependencies value: %v", err)
		}
	}

	if yaml.IsExist("depends") {
		deps := yamlGetSafe(yaml, "depends")
		app.Depends = strutil.Fields(deps)
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
	var cmd, log string

	cmd = yamlGetSafe(yaml, "command")
	cmd, log, _ = parseCommand(cmd)

	if log != "" {
		service.Options.LogFile = log
	}

	service.Cmd = cmd

	if yaml.IsExist("pre") {
		service.PreCmd = yamlGetSafe(yaml, "pre")
	}

	if yaml.IsExist("post") {
		service.PostCmd = yamlGetSafe(yaml, "post")
	}

	return nil
}

// parseV2Options parse service options in yaml based procfile
func parseV2Options(options *ServiceOptions, yaml *simpleyaml.Yaml) error {
	var err error

	options.Env = make(map[string]string)
	options.IsRespawnEnabled = true

	if yaml.IsExist("working_directory") {
		options.WorkingDir = yamlGetSafe(yaml, "working_directory")
	}

	if yaml.IsExist("log") {
		options.LogFile = yamlGetSafe(yaml, "log")
	}

	if yaml.IsExist("kill_timeout") {
		options.KillTimeout, err = yaml.Get("kill_timeout").Int()

		if err != nil {
			return formatPropError("kill_timeout", err)
		}
	}

	if yaml.IsExist("kill_signal") {
		options.KillSignal = yamlGetSafe(yaml, "kill_signal")
	}

	if yaml.IsExist("kill_mode") {
		options.KillMode = yamlGetSafe(yaml, "kill_mode")
	}

	if yaml.IsExist("reload_signal") {
		options.ReloadSignal = yamlGetSafe(yaml, "reload_signal")
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
		options.EnvFile = yamlGetSafe(yaml, "env_file")
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

		if yaml.IsPathExist("respawn", "delay") {
			options.RespawnDelay, err = yaml.Get("respawn").Get("delay").Int()

			if err != nil {
				return formatPropError("respawn:delay", err)
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

		if yaml.IsPathExist("limits", "memlock") {
			options.LimitMemlock, err = yaml.Get("limits").Get("memlock").Int()

			if err != nil {
				return formatPropError("limits:memlock", err)
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

	if yaml.IsExist("cpu_affinity") {
		resources.CPUAffinity = yamlGetSafe(yaml, "cpu_affinity")
	}

	if yaml.IsExist("memory_low") {
		resources.MemoryLow = yamlGetSafe(yaml, "memory_low")
	}

	if yaml.IsExist("memory_high") {
		resources.MemoryHigh = yamlGetSafe(yaml, "memory_high")
	}

	if yaml.IsExist("memory_max") {
		resources.MemoryMax = yamlGetSafe(yaml, "memory_max")
	}

	if yaml.IsExist("memory_swap_max") {
		resources.MemorySwapMax = yamlGetSafe(yaml, "memory_swap_max")
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
		resources.IODeviceWeight = yamlGetSafe(yaml, "io_device_weight")
	}

	if yaml.IsExist("io_read_bandwidth_max") {
		resources.IOReadBandwidthMax = yamlGetSafe(yaml, "io_read_bandwidth_max")
	}

	if yaml.IsExist("io_write_bandwidth_max") {
		resources.IOWriteBandwidthMax = yamlGetSafe(yaml, "io_write_bandwidth_max")
	}

	if yaml.IsExist("io_read_iops_max") {
		resources.IOReadIOPSMax = yamlGetSafe(yaml, "io_read_iops_max")
	}

	if yaml.IsExist("io_write_iops_max") {
		resources.IOWriteIOPSMax = yamlGetSafe(yaml, "io_write_iops_max")
	}

	if yaml.IsExist("ip_address_allow") {
		resources.IPAddressAllow = yamlGetSafe(yaml, "ip_address_allow")
	}

	if yaml.IsExist("ip_address_deny") {
		resources.IPAddressDeny = yamlGetSafe(yaml, "ip_address_deny")
	}

	return resources, nil
}

// yamlGetSafe returns string from YAML without potentially unsafe symbols
func yamlGetSafe(yaml *simpleyaml.Yaml, propName string) string {
	return strings.Trim(yaml.Get(propName).Dump(), "\n\r")
}

// formatPropError format property parsing error
func formatPropError(prop string, err error) error {
	return fmt.Errorf("Can't parse \"%s\" value: %v", prop, err)
}

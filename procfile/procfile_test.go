package procfile

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                       Copyright (c) 2006-2017 FB GROUP LLC                         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"testing"

	. "pkg.re/check.v1"
)

// ////////////////////////////////////////////////////////////////////////////////// //

func Test(t *testing.T) { TestingT(t) }

type ProcfileSuite struct {
	Config *Config
}

// ////////////////////////////////////////////////////////////////////////////////// //

var _ = Suite(&ProcfileSuite{&Config{Name: "test-app", WorkingDir: "/tmp"}})

// ////////////////////////////////////////////////////////////////////////////////// //

func (s *ProcfileSuite) TestProcV1Parsing(c *C) {
	app, err := Read("../testdata/procfile_v1", s.Config)

	c.Assert(err, IsNil)
	c.Assert(app, NotNil)

	c.Assert(app.ProcVersion, Equals, 1)
	c.Assert(app.Services, HasLen, 3)

	errs := app.Validate()

	if len(errs) != 0 {
		c.Fatalf("Validation errors: %v", errs)
	}

	c.Assert(app.Services[0].Name, Equals, "my_tail_cmd")
	c.Assert(app.Services[0].Cmd, Equals, "/usr/bin/tail -F /var/log/messages")
	c.Assert(app.Services[0].Options, NotNil)
	c.Assert(app.Services[0].Options.LogFile, Equals, "log/my_tail_cmd.log")

	c.Assert(app.Services[1].Name, Equals, "my_another_tail_cmd")
	c.Assert(app.Services[1].Cmd, Equals, "/usr/bin/tailf /var/log/messages")
	c.Assert(app.Services[1].PreCmd, Equals, "echo my_another_tail_cmd")
	c.Assert(app.Services[1].Options, NotNil)
	c.Assert(app.Services[1].Options.LogFile, Equals, "log/my_another_tail_cmd.log")

	c.Assert(app.Services[2].Name, Equals, "cmd_with_cd")
	c.Assert(app.Services[2].Cmd, Equals, "/usr/bin/tail -F /var/log/messages")
	c.Assert(app.Services[2].PreCmd, Equals, "echo cmd_with_cd_pre")
	c.Assert(app.Services[2].PostCmd, Equals, "echo cmd_with_cd_post")
	c.Assert(app.Services[2].Options, NotNil)
	c.Assert(app.Services[2].Options.Env, HasLen, 2)
	c.Assert(app.Services[2].Options.Env["ENV_TEST"], Equals, "100")
	c.Assert(app.Services[2].Options.Env["SOME_ENV"], Equals, "test")
	c.Assert(app.Services[2].Options.WorkingDir, Equals, "/srv/service")

	c.Assert(app.Validate(), HasLen, 0)
}

func (s *ProcfileSuite) TestProcV2Parsing(c *C) {
	app, err := Read("../testdata/procfile_v2", s.Config)

	c.Assert(err, IsNil)
	c.Assert(app, NotNil)

	c.Assert(app.ProcVersion, Equals, 2)
	c.Assert(app.Services, HasLen, 4)

	c.Assert(app.StartLevel, Equals, 2)
	c.Assert(app.StopLevel, Equals, 5)

	errs := app.Validate()

	if len(errs) != 0 {
		c.Fatalf("Validation errors: %v", errs)
	}

	for _, service := range app.Services {
		switch service.Name {
		case "my_tail_cmd":
			c.Assert(service.Cmd, Equals, "/usr/bin/tail -F /var/log/messages")
			c.Assert(service.Options, NotNil)
			c.Assert(service.Options.WorkingDir, Equals, "/var/...")
			c.Assert(service.Options.LogFile, Equals, "log/my_tail_cmd.log")
			c.Assert(service.Options.IsCustomLogEnabled(), Equals, true)
			c.Assert(service.Options.RespawnCount, Equals, 5)
			c.Assert(service.Options.RespawnInterval, Equals, 10)
			c.Assert(service.Options.IsRespawnEnabled, Equals, true)
			c.Assert(service.Options.Env, NotNil)
			c.Assert(service.Options.Env["RAILS_ENV"], Equals, "staging")
			c.Assert(service.Options.Env["TEST"], Equals, "true")
			c.Assert(service.Options.Env["JAVA_OPTS"], Equals, "\"${JAVA_OPTS} -Xms512m -Xmx1g -XX:+HeapDumpOnIutOfMemoryError -Djava.net.preferIPv4Stack=true\"")
			c.Assert(service.Options.Env["AUX_OPTS"], Equals, "'--debug --native'")
			c.Assert(service.Options.Env["QUEUE"], Equals, "log_syncronizer,file_downloader,log_searcher")
			c.Assert(service.Options.Env["LC_ALL"], Equals, "en_US.UTF-8")
			c.Assert(service.Options.EnvString(), Equals, "AUX_OPTS='--debug --native' HEX_HOME=/srv/projects/ploy/shared/tmp JAVA_OPTS=\"${JAVA_OPTS} -Xms512m -Xmx1g -XX:+HeapDumpOnIutOfMemoryError -Djava.net.preferIPv4Stack=true\" LC_ALL=en_US.UTF-8 QUEUE=log_syncronizer,file_downloader,log_searcher RAILS_ENV=staging TEST=true")
			c.Assert(service.Options.LimitFile, Equals, 4096)
			c.Assert(service.Options.LimitProc, Equals, 4096)
			c.Assert(service.Application, NotNil)
			c.Assert(service.Application.Name, Equals, "test-app")

		case "my_another_tail_cmd":
			c.Assert(service.Cmd, Equals, "/usr/bin/tail -F /var/log/messages")
			c.Assert(service.PreCmd, Equals, "/usr/bin/echo pre_command")
			c.Assert(service.PostCmd, Equals, "/usr/bin/echo post_command")
			c.Assert(service.Options, NotNil)
			c.Assert(service.Options.WorkingDir, Equals, "/srv/projects/my_website/current")
			c.Assert(service.Options.IsCustomLogEnabled(), Equals, false)
			c.Assert(service.Options.KillTimeout, Equals, 60)
			c.Assert(service.Options.KillSignal, Equals, "SIGQUIT")
			c.Assert(service.Options.ReloadSignal, Equals, "SIGUSR2")
			c.Assert(service.Options.RespawnCount, Equals, 7)
			c.Assert(service.Options.RespawnInterval, Equals, 22)
			c.Assert(service.Options.IsRespawnEnabled, Equals, false)
			c.Assert(service.Options.EnvFile, Equals, "shared/env.file")
			c.Assert(service.Options.EnvString(), Equals, "RAILS_ENV=production TEST=true")
			c.Assert(service.Options.LimitFile, Equals, 8192)
			c.Assert(service.Options.LimitProc, Equals, 8192)
			c.Assert(service.Application, NotNil)
			c.Assert(service.Application.Name, Equals, "test-app")

		case "my_one_another_tail_cmd":
			c.Assert(service.Cmd, Equals, "/usr/bin/tail -F /var/log/messages")
			c.Assert(service.Options, NotNil)
			c.Assert(service.Options.WorkingDir, Equals, "/srv/projects/my_website/current")
			c.Assert(service.Options.LogFile, Equals, "log/my_one_another_tail_cmd.log")
			c.Assert(service.Options.FullLogPath(), Equals, "/srv/projects/my_website/current/log/my_one_another_tail_cmd.log")
			c.Assert(service.Options.IsCustomLogEnabled(), Equals, true)
			c.Assert(service.Options.RespawnCount, Equals, 7)
			c.Assert(service.Options.RespawnInterval, Equals, 22)
			c.Assert(service.Options.IsRespawnEnabled, Equals, true)
			c.Assert(service.Options.Env, NotNil)
			c.Assert(service.Options.Env["RAILS_ENV"], Equals, "production")
			c.Assert(service.Options.Env["TEST"], Equals, "true")
			c.Assert(service.Options.EnvString(), Equals, "RAILS_ENV=production TEST=true")
			c.Assert(service.Options.LimitFile, Equals, 4096)
			c.Assert(service.Options.LimitProc, Equals, 4096)
			c.Assert(service.Application, NotNil)
			c.Assert(service.Application.Name, Equals, "test-app")

		case "my_multi_tail_cmd":
			c.Assert(service.Cmd, Equals, "/usr/bin/tail -F /var/log/messages")
			c.Assert(service.Options, NotNil)
			c.Assert(service.Options.Count, Equals, 2)
			c.Assert(service.Options.WorkingDir, Equals, "/srv/projects/my_website/current")
			c.Assert(service.Options.IsCustomLogEnabled(), Equals, false)
			c.Assert(service.Options.RespawnCount, Equals, 7)
			c.Assert(service.Options.RespawnInterval, Equals, 22)
			c.Assert(service.Options.IsRespawnEnabled, Equals, true)
			c.Assert(service.Options.Env, NotNil)
			c.Assert(service.Options.Env["RAILS_ENV"], Equals, "production")
			c.Assert(service.Options.Env["TEST"], Equals, "true")
			c.Assert(service.Options.EnvString(), Equals, "RAILS_ENV=production TEST=true")
			c.Assert(service.Options.LimitFile, Equals, 1024)
			c.Assert(service.Options.LimitProc, Equals, 4096)
			c.Assert(service.Application, NotNil)
			c.Assert(service.Application.Name, Equals, "test-app")

		default:
			c.Fatalf("Unknown service %s", service.Name)
		}
	}

	c.Assert(app.Validate(), HasLen, 0)
}

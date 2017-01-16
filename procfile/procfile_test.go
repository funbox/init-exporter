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

type ProcfileSuite struct{}

// ////////////////////////////////////////////////////////////////////////////////// //

var _ = Suite(&ProcfileSuite{})

// ////////////////////////////////////////////////////////////////////////////////// //

func (s *ProcfileSuite) TestProcV1Parsing(c *C) {
	app, err := Read("../testdata/procfile_v1", &Config{Name: "test-app"})

	c.Assert(err, IsNil)
	c.Assert(app, NotNil)

	c.Assert(app.ProcVersion, Equals, 1)
	c.Assert(app.Services, HasLen, 3)

	c.Assert(app.Services[0].Name, Equals, "my_tail_cmd")
	c.Assert(app.Services[0].Cmd, Equals, "/usr/bin/tail -F /var/log/messages")

	c.Assert(app.Services[1].Name, Equals, "my_another_tail_cmd")
	c.Assert(app.Services[1].Cmd, Equals, "/usr/bin/tailf /var/log/messages")

	c.Assert(app.Services[2].Name, Equals, "cmd_with_cd")
	c.Assert(app.Services[2].Cmd, Equals, "/usr/bin/tail -F /var/log/messages")
	c.Assert(app.Services[2].Options, NotNil)
	c.Assert(app.Services[2].Options.Env, HasLen, 2)
	c.Assert(app.Services[2].Options.Env["ENV_TEST"], Equals, "100")
	c.Assert(app.Services[2].Options.Env["SOME_ENV"], Equals, "test")
	c.Assert(app.Services[2].Options.WorkingDir, Equals, "/srv/service")

	c.Assert(app.Validate(), IsNil)
}

func (s *ProcfileSuite) TestProcV2Parsing(c *C) {
	app, err := Read("../testdata/procfile_v2", &Config{Name: "test-app"})

	c.Assert(err, IsNil)
	c.Assert(app, NotNil)

	c.Assert(app.ProcVersion, Equals, 2)
	c.Assert(app.Services, HasLen, 4)

	c.Assert(app.StartLevel, Equals, 2)
	c.Assert(app.StopLevel, Equals, 5)

	for _, service := range app.Services {
		switch service.Name {
		case "my_tail_cmd":
			c.Assert(service.Cmd, Equals, "/usr/bin/tail -F /var/log/messages")
			c.Assert(service.Options, NotNil)
			c.Assert(service.Options.WorkingDir, Equals, "/var/...")
			c.Assert(service.Options.IsCustomLogEnabled(), Equals, false)
			c.Assert(service.Options.RespawnCount, Equals, 5)
			c.Assert(service.Options.RespawnInterval, Equals, 10)
			c.Assert(service.Options.IsRespawnEnabled, Equals, true)
			c.Assert(service.Options.Env, NotNil)
			c.Assert(service.Options.Env["RAILS_ENV"], Equals, "staging")
			c.Assert(service.Options.Env["TEST"], Equals, "true")
			c.Assert(service.Options.EnvString(), Equals, "RAILS_ENV=staging TEST=true")
			c.Assert(service.Options.LimitFile, Equals, 4096)
			c.Assert(service.Options.LimitProc, Equals, 4096)
			c.Assert(service.Application, NotNil)
			c.Assert(service.Application.Name, Equals, "test-app")

		case "my_another_tail_cmd":
			c.Assert(service.Cmd, Equals, "/usr/bin/tail -F /var/log/messages")
			c.Assert(service.Options, NotNil)
			c.Assert(service.Options.WorkingDir, Equals, "/srv/projects/my_website/current")
			c.Assert(service.Options.IsCustomLogEnabled(), Equals, false)
			c.Assert(service.Options.KillTimeout, Equals, 60)
			c.Assert(service.Options.RespawnCount, Equals, 7)
			c.Assert(service.Options.RespawnInterval, Equals, 22)
			c.Assert(service.Options.IsRespawnEnabled, Equals, false)
			c.Assert(service.Options.Env, NotNil)
			c.Assert(service.Options.Env["RAILS_ENV"], Equals, "production")
			c.Assert(service.Options.Env["TEST"], Equals, "true")
			c.Assert(service.Options.EnvString(), Equals, "RAILS_ENV=production TEST=true")
			c.Assert(service.Options.LimitFile, Equals, 8192)
			c.Assert(service.Options.LimitProc, Equals, 8192)
			c.Assert(service.Application, NotNil)
			c.Assert(service.Application.Name, Equals, "test-app")

		case "my_one_another_tail_cmd":
			c.Assert(service.Cmd, Equals, "/usr/bin/tail -F /var/log/messages")
			c.Assert(service.Options, NotNil)
			c.Assert(service.Options.WorkingDir, Equals, "/srv/projects/my_website/current")
			c.Assert(service.Options.LogPath, Equals, "/var/log/messages_copy")
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

	c.Assert(app.Validate(), IsNil)
}

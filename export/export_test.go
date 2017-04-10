package export

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                       Copyright (c) 2006-2017 FB GROUP LLC                         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/funbox/init-exporter/procfile"

	"pkg.re/essentialkaos/ek.v7/fsutil"
	"pkg.re/essentialkaos/ek.v7/log"

	. "pkg.re/check.v1"
)

// ////////////////////////////////////////////////////////////////////////////////// //

func Test(t *testing.T) { TestingT(t) }

type ExportSuite struct {
	HelperDir string
	TargetDir string
}

// ////////////////////////////////////////////////////////////////////////////////// //

var _ = Suite(&ExportSuite{})

// ////////////////////////////////////////////////////////////////////////////////// //

func (s *ExportSuite) SetUpSuite(c *C) {
	// Disable logging
	log.Set(os.DevNull, 0)
}

func (s *ExportSuite) TestUpstartExport(c *C) {
	helperDir := c.MkDir()
	targetDir := c.MkDir()

	config := &Config{
		HelperDir:        helperDir,
		TargetDir:        targetDir,
		DisableAutoStart: true,
	}

	exporter := NewExporter(config, NewUpstart())

	c.Assert(exporter, NotNil)

	app := createTestApp(targetDir, helperDir)

	err := exporter.Install(app)

	c.Assert(err, IsNil)

	c.Assert(fsutil.IsExist(targetDir+"/test_application.conf"), Equals, true)
	c.Assert(fsutil.IsRegular(targetDir+"/test_application.conf"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(targetDir+"/test_application.conf"), Equals, true)

	c.Assert(fsutil.IsExist(targetDir+"/test_application-service1.conf"), Equals, true)
	c.Assert(fsutil.IsRegular(targetDir+"/test_application-service1.conf"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(targetDir+"/test_application-service1.conf"), Equals, true)

	c.Assert(fsutil.IsExist(targetDir+"/test_application-service2.conf"), Equals, true)
	c.Assert(fsutil.IsRegular(targetDir+"/test_application-service2.conf"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(targetDir+"/test_application-service2.conf"), Equals, true)

	c.Assert(fsutil.IsExist(helperDir+"/test_application-service1.sh"), Equals, true)
	c.Assert(fsutil.IsRegular(helperDir+"/test_application-service1.sh"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(helperDir+"/test_application-service1.sh"), Equals, true)

	c.Assert(fsutil.IsExist(helperDir+"/test_application-service2.sh"), Equals, true)
	c.Assert(fsutil.IsRegular(helperDir+"/test_application-service2.sh"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(helperDir+"/test_application-service2.sh"), Equals, true)

	appUnitData, err := ioutil.ReadFile(targetDir + "/test_application.conf")

	c.Assert(err, IsNil)
	c.Assert(appUnitData, NotNil)

	service1UnitData, err := ioutil.ReadFile(targetDir + "/test_application-service1.conf")

	c.Assert(err, IsNil)
	c.Assert(service1UnitData, NotNil)

	service2UnitData, err := ioutil.ReadFile(targetDir + "/test_application-service2.conf")

	c.Assert(err, IsNil)
	c.Assert(service2UnitData, NotNil)

	service1HelperData, err := ioutil.ReadFile(helperDir + "/test_application-service1.sh")

	c.Assert(err, IsNil)
	c.Assert(service1HelperData, NotNil)

	service2HelperData, err := ioutil.ReadFile(helperDir + "/test_application-service2.sh")

	c.Assert(err, IsNil)
	c.Assert(service2HelperData, NotNil)

	appUnit := strings.Split(string(appUnitData), "\n")
	service1Unit := strings.Split(string(service1UnitData), "\n")
	service2Unit := strings.Split(string(service2UnitData), "\n")
	service1Helper := strings.Split(string(service1HelperData), "\n")
	service2Helper := strings.Split(string(service2HelperData), "\n")

	c.Assert(appUnit[2:], DeepEquals,
		[]string{
			"start on runlevel [3]",
			"stop on runlevel [3]",
			"",
			"pre-start script",
			"",
			"bash << \"EOF\"",
			"  mkdir -p /var/log/test_application",
			"  chown -R service /var/log/test_application",
			"  chgrp -R service /var/log/test_application",
			"  chmod -R g+w /var/log/test_application",
			"EOF",
			"",
			"end script", ""},
	)

	c.Assert(service1Unit[2:], DeepEquals,
		[]string{
			"start on starting test_application",
			"stop on stopping test_application",
			"",
			"respawn",
			"respawn limit 15 25",
			"",
			"kill timeout 10",
			"kill signal SIGQUIT",
			"",
			"",
			"limit nofile 1024 1024",
			"",
			"",
			"script",
			"  touch /var/log/test_application/service1.log",
			"  chown service /var/log/test_application/service1.log",
			"  chgrp service /var/log/test_application/service1.log",
			"  chmod g+w /var/log/test_application/service1.log",
			fmt.Sprintf("  exec sudo -u service /bin/bash %s/test_application-service1.sh &>>/var/log/test_application/service1.log", helperDir),
			"end script", ""},
	)

	c.Assert(service2Unit[2:], DeepEquals,
		[]string{
			"start on starting test_application",
			"stop on stopping test_application",
			"",
			"respawn",
			"",
			"",
			"kill timeout 0",
			"",
			"reload signal SIGUSR2",
			"",
			"limit nofile 4096 4096",
			"limit nproc 4096 4096",
			"",
			"script",
			"  touch /var/log/test_application/service2.log",
			"  chown service /var/log/test_application/service2.log",
			"  chgrp service /var/log/test_application/service2.log",
			"  chmod g+w /var/log/test_application/service2.log",
			fmt.Sprintf("  exec sudo -u service /bin/bash %s/test_application-service2.sh &>>/var/log/test_application/service2.log", helperDir),
			"end script", ""},
	)

	c.Assert(service1Helper[4:], DeepEquals,
		[]string{
			"[[ -r /etc/profile.d/rbenv.sh ]] && source /etc/profile.d/rbenv.sh", "",
			"cd /srv/service/service1-dir && exec env STAGING=true /bin/echo 'service1:pre' &>>/srv/service/service1-dir/log/service1.log && exec env STAGING=true /bin/echo 'service1' &>>/srv/service/service1-dir/log/service1.log && exec env STAGING=true /bin/echo 'service1:post' &>>/srv/service/service1-dir/log/service1.log",
			""},
	)

	c.Assert(service2Helper[4:], DeepEquals,
		[]string{
			"[[ -r /etc/profile.d/rbenv.sh ]] && source /etc/profile.d/rbenv.sh", "",
			"cd /srv/service/working-dir && exec env $(cat /srv/service/working-dir/shared/env.vars | xargs) /bin/echo 'service2'",
			""},
	)

	err = exporter.Uninstall(app)

	c.Assert(err, IsNil)

	c.Assert(fsutil.IsExist(targetDir+"/test_application.conf"), Equals, false)
	c.Assert(fsutil.IsExist(targetDir+"/test_application-service1.conf"), Equals, false)
	c.Assert(fsutil.IsExist(targetDir+"/test_application-service2.conf"), Equals, false)
	c.Assert(fsutil.IsExist(helperDir+"/test_application-service1.sh"), Equals, false)
	c.Assert(fsutil.IsExist(helperDir+"/test_application-service2.sh"), Equals, false)
}

func (s *ExportSuite) TestSystemdExport(c *C) {
	helperDir := c.MkDir()
	targetDir := c.MkDir()

	config := &Config{
		HelperDir:        helperDir,
		TargetDir:        targetDir,
		DisableAutoStart: true,
	}

	exporter := NewExporter(config, NewSystemd())

	c.Assert(exporter, NotNil)

	app := createTestApp(targetDir, helperDir)

	err := exporter.Install(app)

	c.Assert(err, IsNil)

	c.Assert(fsutil.IsExist(targetDir+"/test_application.service"), Equals, true)
	c.Assert(fsutil.IsRegular(targetDir+"/test_application.service"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(targetDir+"/test_application.service"), Equals, true)

	c.Assert(fsutil.IsExist(targetDir+"/test_application-service1.service"), Equals, true)
	c.Assert(fsutil.IsRegular(targetDir+"/test_application-service1.service"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(targetDir+"/test_application-service1.service"), Equals, true)

	c.Assert(fsutil.IsExist(targetDir+"/test_application-service2.service"), Equals, true)
	c.Assert(fsutil.IsRegular(targetDir+"/test_application-service2.service"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(targetDir+"/test_application-service2.service"), Equals, true)

	c.Assert(fsutil.IsExist(helperDir+"/test_application-service1.sh"), Equals, true)
	c.Assert(fsutil.IsRegular(helperDir+"/test_application-service1.sh"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(helperDir+"/test_application-service1.sh"), Equals, true)

	c.Assert(fsutil.IsExist(helperDir+"/test_application-service2.sh"), Equals, true)
	c.Assert(fsutil.IsRegular(helperDir+"/test_application-service2.sh"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(helperDir+"/test_application-service2.sh"), Equals, true)

	appUnitData, err := ioutil.ReadFile(targetDir + "/test_application.service")

	c.Assert(err, IsNil)
	c.Assert(appUnitData, NotNil)

	service1UnitData, err := ioutil.ReadFile(targetDir + "/test_application-service1.service")

	c.Assert(err, IsNil)
	c.Assert(service1UnitData, NotNil)

	service2UnitData, err := ioutil.ReadFile(targetDir + "/test_application-service2.service")

	c.Assert(err, IsNil)
	c.Assert(service2UnitData, NotNil)

	service1HelperData, err := ioutil.ReadFile(helperDir + "/test_application-service1.sh")

	c.Assert(err, IsNil)
	c.Assert(service1HelperData, NotNil)

	service2HelperData, err := ioutil.ReadFile(helperDir + "/test_application-service2.sh")

	c.Assert(err, IsNil)
	c.Assert(service2HelperData, NotNil)

	appUnit := strings.Split(string(appUnitData), "\n")
	service1Unit := strings.Split(string(service1UnitData), "\n")
	service2Unit := strings.Split(string(service2UnitData), "\n")
	service1Helper := strings.Split(string(service1HelperData), "\n")
	service2Helper := strings.Split(string(service2HelperData), "\n")

	c.Assert(appUnit[2:], DeepEquals,
		[]string{
			"[Unit]",
			"",
			"Description=Unit for test_application application",
			"After=multi-user.target",
			"Wants=test_application-service1.service test_application-service2.service",
			"",
			"[Service]",
			"Type=oneshot",
			"RemainAfterExit=true",
			"",
			"ExecStartPre=/bin/mkdir -p /var/log/test_application",
			"ExecStartPre=/bin/chown -R service /var/log/test_application",
			"ExecStartPre=/bin/chgrp -R service /var/log/test_application",
			"ExecStartPre=/bin/chmod -R g+w /var/log/test_application",
			"ExecStart=/bin/echo \"test_application started\"",
			"ExecStop=/bin/echo \"test_application stopped\"",
			"",
			"[Install]",
			"WantedBy=multi-user.target", ""},
	)

	c.Assert(service1Unit[2:], DeepEquals,
		[]string{
			"[Unit]",
			"",
			"Description=Unit for service1 service (part of test_application application)",
			"PartOf=test_application.service",
			"",
			"[Service]",
			"Type=simple",
			"",
			"KillSignal=SIGQUIT",
			"TimeoutStopSec=10",
			"Restart=on-failure",
			"StartLimitInterval=25",
			"StartLimitBurst=15",
			"",
			"LimitNOFILE=1024",
			"",
			"",
			"ExecStartPre=/bin/touch /var/log/test_application/service1.log",
			"ExecStartPre=/bin/chown service /var/log/test_application/service1.log",
			"ExecStartPre=/bin/chgrp service /var/log/test_application/service1.log",
			"ExecStartPre=/bin/chmod g+w /var/log/test_application/service1.log",
			"",
			"User=service",
			"Group=service",
			"WorkingDirectory=/srv/service/service1-dir",
			fmt.Sprintf("ExecStart=/bin/bash %s/test_application-service1.sh &>>/var/log/test_application/service1.log", helperDir),
			"",
			""},
	)

	c.Assert(service2Unit[2:], DeepEquals,
		[]string{
			"[Unit]",
			"",
			"Description=Unit for service2 service (part of test_application application)",
			"PartOf=test_application.service",
			"",
			"[Service]",
			"Type=simple",
			"",
			"",
			"TimeoutStopSec=0",
			"Restart=on-failure",
			"",
			"",
			"",
			"LimitNOFILE=4096",
			"LimitNPROC=4096",
			"",
			"ExecStartPre=/bin/touch /var/log/test_application/service2.log",
			"ExecStartPre=/bin/chown service /var/log/test_application/service2.log",
			"ExecStartPre=/bin/chgrp service /var/log/test_application/service2.log",
			"ExecStartPre=/bin/chmod g+w /var/log/test_application/service2.log",
			"",
			"User=service",
			"Group=service",
			"WorkingDirectory=/srv/service/working-dir",
			fmt.Sprintf("ExecStart=/bin/bash %s/test_application-service2.sh &>>/var/log/test_application/service2.log", helperDir),
			"ExecReload=/bin/kill -SIGUSR2 $MAINPID",
			""},
	)

	c.Assert(service1Helper[4:], DeepEquals,
		[]string{
			"[[ -r /etc/profile.d/rbenv.sh ]] && source /etc/profile.d/rbenv.sh", "",
			"exec env STAGING=true /bin/echo 'service1:pre' &>>/srv/service/service1-dir/log/service1.log && exec env STAGING=true /bin/echo 'service1' &>>/srv/service/service1-dir/log/service1.log && exec env STAGING=true /bin/echo 'service1:post' &>>/srv/service/service1-dir/log/service1.log",
			""},
	)

	c.Assert(service2Helper[4:], DeepEquals,
		[]string{
			"[[ -r /etc/profile.d/rbenv.sh ]] && source /etc/profile.d/rbenv.sh", "",
			"exec env $(cat /srv/service/working-dir/shared/env.vars | xargs) /bin/echo 'service2'",
			""},
	)

	err = exporter.Uninstall(app)

	c.Assert(err, IsNil)

	c.Assert(fsutil.IsExist(targetDir+"/test_application.service"), Equals, false)
	c.Assert(fsutil.IsExist(targetDir+"/test_application-service1.service"), Equals, false)
	c.Assert(fsutil.IsExist(targetDir+"/test_application-service2.service"), Equals, false)
	c.Assert(fsutil.IsExist(helperDir+"/test_application-service1.sh"), Equals, false)
	c.Assert(fsutil.IsExist(helperDir+"/test_application-service2.sh"), Equals, false)
}

// ////////////////////////////////////////////////////////////////////////////////// //

func createTestApp(helperDir, targetDir string) *procfile.Application {
	app := &procfile.Application{
		Name:        "test_application",
		User:        "service",
		Group:       "service",
		StartLevel:  3,
		StopLevel:   3,
		WorkingDir:  "/srv/service/working-dir",
		ProcVersion: 2,
		Services:    []*procfile.Service{},
	}

	service1 := &procfile.Service{
		Name:        "service1",
		Cmd:         "/bin/echo 'service1'",
		PreCmd:      "/bin/echo 'service1:pre'",
		PostCmd:     "/bin/echo 'service1:post'",
		Application: app,
		Options: &procfile.ServiceOptions{
			Env:              map[string]string{"STAGING": "true"},
			WorkingDir:       "/srv/service/service1-dir",
			LogFile:          "log/service1.log",
			KillTimeout:      10,
			KillSignal:       "SIGQUIT",
			Count:            2,
			RespawnInterval:  25,
			RespawnCount:     15,
			IsRespawnEnabled: true,
			LimitFile:        1024,
		},
	}

	service2 := &procfile.Service{
		Name:        "service2",
		Cmd:         "/bin/echo 'service2'",
		Application: app,
		Options: &procfile.ServiceOptions{
			EnvFile:          "shared/env.vars",
			WorkingDir:       "/srv/service/working-dir",
			ReloadSignal:     "SIGUSR2",
			IsRespawnEnabled: true,
			LimitFile:        4096,
			LimitProc:        4096,
		},
	}

	app.Services = append(app.Services, service1, service2)

	return app
}

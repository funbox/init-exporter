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

	"pkg.re/essentialkaos/ek.v9/fsutil"
	"pkg.re/essentialkaos/ek.v9/log"

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

	c.Assert(fsutil.IsExist(targetDir+"/test_application-serviceA1.conf"), Equals, true)
	c.Assert(fsutil.IsRegular(targetDir+"/test_application-serviceA1.conf"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(targetDir+"/test_application-serviceA1.conf"), Equals, true)
	c.Assert(fsutil.IsExist(targetDir+"/test_application-serviceA2.conf"), Equals, true)
	c.Assert(fsutil.IsRegular(targetDir+"/test_application-serviceA2.conf"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(targetDir+"/test_application-serviceA2.conf"), Equals, true)

	c.Assert(fsutil.IsExist(targetDir+"/test_application-serviceB.conf"), Equals, true)
	c.Assert(fsutil.IsRegular(targetDir+"/test_application-serviceB.conf"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(targetDir+"/test_application-serviceB.conf"), Equals, true)

	c.Assert(fsutil.IsExist(helperDir+"/test_application-serviceA1.sh"), Equals, true)
	c.Assert(fsutil.IsRegular(helperDir+"/test_application-serviceA1.sh"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(helperDir+"/test_application-serviceA1.sh"), Equals, true)
	c.Assert(fsutil.IsExist(helperDir+"/test_application-serviceA2.sh"), Equals, true)
	c.Assert(fsutil.IsRegular(helperDir+"/test_application-serviceA2.sh"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(helperDir+"/test_application-serviceA2.sh"), Equals, true)

	c.Assert(fsutil.IsExist(helperDir+"/test_application-serviceB.sh"), Equals, true)
	c.Assert(fsutil.IsRegular(helperDir+"/test_application-serviceB.sh"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(helperDir+"/test_application-serviceB.sh"), Equals, true)

	appUnitData, err := ioutil.ReadFile(targetDir + "/test_application.conf")

	c.Assert(err, IsNil)
	c.Assert(appUnitData, NotNil)

	serviceA1UnitData, err := ioutil.ReadFile(targetDir + "/test_application-serviceA1.conf")

	c.Assert(err, IsNil)
	c.Assert(serviceA1UnitData, NotNil)

	serviceA2UnitData, err := ioutil.ReadFile(targetDir + "/test_application-serviceA2.conf")

	c.Assert(err, IsNil)
	c.Assert(serviceA2UnitData, NotNil)

	serviceBUnitData, err := ioutil.ReadFile(targetDir + "/test_application-serviceB.conf")

	c.Assert(err, IsNil)
	c.Assert(serviceBUnitData, NotNil)

	serviceAHelperData, err := ioutil.ReadFile(helperDir + "/test_application-serviceA1.sh")

	c.Assert(err, IsNil)
	c.Assert(serviceAHelperData, NotNil)

	serviceBHelperData, err := ioutil.ReadFile(helperDir + "/test_application-serviceB.sh")

	c.Assert(err, IsNil)
	c.Assert(serviceBHelperData, NotNil)

	appUnit := strings.Split(string(appUnitData), "\n")
	serviceA1Unit := strings.Split(string(serviceA1UnitData), "\n")
	serviceA2Unit := strings.Split(string(serviceA2UnitData), "\n")
	serviceBUnit := strings.Split(string(serviceBUnitData), "\n")
	serviceAHelper := strings.Split(string(serviceAHelperData), "\n")
	serviceBHelper := strings.Split(string(serviceBHelperData), "\n")

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

	c.Assert(serviceA1Unit[2:], DeepEquals,
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
			"  touch /var/log/test_application/serviceA.log",
			"  chown service /var/log/test_application/serviceA.log",
			"  chgrp service /var/log/test_application/serviceA.log",
			"  chmod g+w /var/log/test_application/serviceA.log",
			fmt.Sprintf("  exec sudo -u service /bin/bash %s/test_application-serviceA1.sh &>>/var/log/test_application/serviceA.log", helperDir),
			"end script", ""},
	)

	c.Assert(serviceA2Unit[2:], DeepEquals,
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
			"  touch /var/log/test_application/serviceA.log",
			"  chown service /var/log/test_application/serviceA.log",
			"  chgrp service /var/log/test_application/serviceA.log",
			"  chmod g+w /var/log/test_application/serviceA.log",
			fmt.Sprintf("  exec sudo -u service /bin/bash %s/test_application-serviceA2.sh &>>/var/log/test_application/serviceA.log", helperDir),
			"end script", ""},
	)

	c.Assert(serviceBUnit[2:], DeepEquals,
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
			"  touch /var/log/test_application/serviceB.log",
			"  chown service /var/log/test_application/serviceB.log",
			"  chgrp service /var/log/test_application/serviceB.log",
			"  chmod g+w /var/log/test_application/serviceB.log",
			fmt.Sprintf("  exec sudo -u service /bin/bash %s/test_application-serviceB.sh &>>/var/log/test_application/serviceB.log", helperDir),
			"end script", ""},
	)

	c.Assert(serviceAHelper[4:], DeepEquals,
		[]string{
			"[[ -r /etc/profile.d/rbenv.sh ]] && source /etc/profile.d/rbenv.sh", "",
			"cd /srv/service/serviceA-dir && exec env STAGING=true /bin/echo 'serviceA:pre' &>>/srv/service/serviceA-dir/log/serviceA.log && exec env STAGING=true /bin/echo 'serviceA' &>>/srv/service/serviceA-dir/log/serviceA.log && exec env STAGING=true /bin/echo 'serviceA:post' &>>/srv/service/serviceA-dir/log/serviceA.log",
			""},
	)

	c.Assert(serviceBHelper[4:], DeepEquals,
		[]string{
			"[[ -r /etc/profile.d/rbenv.sh ]] && source /etc/profile.d/rbenv.sh", "",
			"cd /srv/service/working-dir && exec env $(cat /srv/service/working-dir/shared/env.vars 2>/dev/null | xargs) STAGING=true /bin/echo 'serviceB'",
			""},
	)

	err = exporter.Uninstall(app)

	c.Assert(err, IsNil)

	c.Assert(fsutil.IsExist(targetDir+"/test_application.conf"), Equals, false)
	c.Assert(fsutil.IsExist(targetDir+"/test_application-serviceA.conf"), Equals, false)
	c.Assert(fsutil.IsExist(targetDir+"/test_application-serviceB.conf"), Equals, false)
	c.Assert(fsutil.IsExist(helperDir+"/test_application-serviceA.sh"), Equals, false)
	c.Assert(fsutil.IsExist(helperDir+"/test_application-serviceB.sh"), Equals, false)
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

	c.Assert(fsutil.IsExist(targetDir+"/test_application-serviceA1.service"), Equals, true)
	c.Assert(fsutil.IsRegular(targetDir+"/test_application-serviceA1.service"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(targetDir+"/test_application-serviceA1.service"), Equals, true)
	c.Assert(fsutil.IsExist(targetDir+"/test_application-serviceA2.service"), Equals, true)
	c.Assert(fsutil.IsRegular(targetDir+"/test_application-serviceA2.service"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(targetDir+"/test_application-serviceA2.service"), Equals, true)

	c.Assert(fsutil.IsExist(targetDir+"/test_application-serviceB.service"), Equals, true)
	c.Assert(fsutil.IsRegular(targetDir+"/test_application-serviceB.service"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(targetDir+"/test_application-serviceB.service"), Equals, true)

	c.Assert(fsutil.IsExist(helperDir+"/test_application-serviceA1.sh"), Equals, true)
	c.Assert(fsutil.IsRegular(helperDir+"/test_application-serviceA1.sh"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(helperDir+"/test_application-serviceA1.sh"), Equals, true)
	c.Assert(fsutil.IsExist(helperDir+"/test_application-serviceA2.sh"), Equals, true)
	c.Assert(fsutil.IsRegular(helperDir+"/test_application-serviceA2.sh"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(helperDir+"/test_application-serviceA2.sh"), Equals, true)

	c.Assert(fsutil.IsExist(helperDir+"/test_application-serviceB.sh"), Equals, true)
	c.Assert(fsutil.IsRegular(helperDir+"/test_application-serviceB.sh"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(helperDir+"/test_application-serviceB.sh"), Equals, true)

	appUnitData, err := ioutil.ReadFile(targetDir + "/test_application.service")

	c.Assert(err, IsNil)
	c.Assert(appUnitData, NotNil)

	serviceA1UnitData, err := ioutil.ReadFile(targetDir + "/test_application-serviceA1.service")

	c.Assert(err, IsNil)
	c.Assert(serviceA1UnitData, NotNil)

	serviceA2UnitData, err := ioutil.ReadFile(targetDir + "/test_application-serviceA2.service")

	c.Assert(err, IsNil)
	c.Assert(serviceA2UnitData, NotNil)

	serviceBUnitData, err := ioutil.ReadFile(targetDir + "/test_application-serviceB.service")

	c.Assert(err, IsNil)
	c.Assert(serviceBUnitData, NotNil)

	serviceAHelperData, err := ioutil.ReadFile(helperDir + "/test_application-serviceA1.sh")

	c.Assert(err, IsNil)
	c.Assert(serviceAHelperData, NotNil)

	serviceBHelperData, err := ioutil.ReadFile(helperDir + "/test_application-serviceB.sh")

	c.Assert(err, IsNil)
	c.Assert(serviceBHelperData, NotNil)

	appUnit := strings.Split(string(appUnitData), "\n")
	serviceA1Unit := strings.Split(string(serviceA1UnitData), "\n")
	serviceA2Unit := strings.Split(string(serviceA2UnitData), "\n")
	serviceBUnit := strings.Split(string(serviceBUnitData), "\n")
	serviceAHelper := strings.Split(string(serviceAHelperData), "\n")
	serviceBHelper := strings.Split(string(serviceBHelperData), "\n")

	c.Assert(appUnit[2:], DeepEquals,
		[]string{
			"[Unit]",
			"",
			"Description=Unit for test_application application",
			"After=multi-user.target",
			"Wants=test_application-serviceA1.service test_application-serviceA2.service test_application-serviceB.service",
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

	c.Assert(serviceA1Unit[2:], DeepEquals,
		[]string{
			"[Unit]",
			"",
			"Description=Unit for serviceA service (part of test_application application)",
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
			"ExecStartPre=/bin/touch /var/log/test_application/serviceA.log",
			"ExecStartPre=/bin/chown service /var/log/test_application/serviceA.log",
			"ExecStartPre=/bin/chgrp service /var/log/test_application/serviceA.log",
			"ExecStartPre=/bin/chmod g+w /var/log/test_application/serviceA.log",
			"",
			"User=service",
			"Group=service",
			"WorkingDirectory=/srv/service/serviceA-dir",
			fmt.Sprintf("ExecStart=/bin/sh -c '/bin/bash %s/test_application-serviceA1.sh &>>/var/log/test_application/serviceA.log'", helperDir),
			"",
			""},
	)

	c.Assert(serviceA2Unit[2:], DeepEquals,
		[]string{
			"[Unit]",
			"",
			"Description=Unit for serviceA service (part of test_application application)",
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
			"ExecStartPre=/bin/touch /var/log/test_application/serviceA.log",
			"ExecStartPre=/bin/chown service /var/log/test_application/serviceA.log",
			"ExecStartPre=/bin/chgrp service /var/log/test_application/serviceA.log",
			"ExecStartPre=/bin/chmod g+w /var/log/test_application/serviceA.log",
			"",
			"User=service",
			"Group=service",
			"WorkingDirectory=/srv/service/serviceA-dir",
			fmt.Sprintf("ExecStart=/bin/sh -c '/bin/bash %s/test_application-serviceA2.sh &>>/var/log/test_application/serviceA.log'", helperDir),
			"",
			""},
	)

	c.Assert(serviceBUnit[2:], DeepEquals,
		[]string{
			"[Unit]",
			"",
			"Description=Unit for serviceB service (part of test_application application)",
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
			"ExecStartPre=/bin/touch /var/log/test_application/serviceB.log",
			"ExecStartPre=/bin/chown service /var/log/test_application/serviceB.log",
			"ExecStartPre=/bin/chgrp service /var/log/test_application/serviceB.log",
			"ExecStartPre=/bin/chmod g+w /var/log/test_application/serviceB.log",
			"",
			"User=service",
			"Group=service",
			"WorkingDirectory=/srv/service/working-dir",
			fmt.Sprintf("ExecStart=/bin/sh -c '/bin/bash %s/test_application-serviceB.sh &>>/var/log/test_application/serviceB.log'", helperDir),
			"ExecReload=/bin/kill -SIGUSR2 $MAINPID",
			""},
	)

	c.Assert(serviceAHelper[4:], DeepEquals,
		[]string{
			"[[ -r /etc/profile.d/rbenv.sh ]] && source /etc/profile.d/rbenv.sh", "",
			"exec env STAGING=true /bin/echo 'serviceA:pre' &>>/srv/service/serviceA-dir/log/serviceA.log && exec env STAGING=true /bin/echo 'serviceA' &>>/srv/service/serviceA-dir/log/serviceA.log && exec env STAGING=true /bin/echo 'serviceA:post' &>>/srv/service/serviceA-dir/log/serviceA.log",
			""},
	)

	c.Assert(serviceBHelper[4:], DeepEquals,
		[]string{
			"[[ -r /etc/profile.d/rbenv.sh ]] && source /etc/profile.d/rbenv.sh", "",
			"exec env $(cat /srv/service/working-dir/shared/env.vars 2>/dev/null | xargs) STAGING=true /bin/echo 'serviceB'",
			""},
	)

	err = exporter.Uninstall(app)

	c.Assert(err, IsNil)

	c.Assert(fsutil.IsExist(targetDir+"/test_application.service"), Equals, false)
	c.Assert(fsutil.IsExist(targetDir+"/test_application-serviceA.service"), Equals, false)
	c.Assert(fsutil.IsExist(targetDir+"/test_application-serviceB.service"), Equals, false)
	c.Assert(fsutil.IsExist(helperDir+"/test_application-serviceA.sh"), Equals, false)
	c.Assert(fsutil.IsExist(helperDir+"/test_application-serviceB.sh"), Equals, false)
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

	serviceA := &procfile.Service{
		Name:        "serviceA",
		Cmd:         "/bin/echo 'serviceA'",
		PreCmd:      "/bin/echo 'serviceA:pre'",
		PostCmd:     "/bin/echo 'serviceA:post'",
		Application: app,
		Options: &procfile.ServiceOptions{
			Env:              map[string]string{"STAGING": "true"},
			WorkingDir:       "/srv/service/serviceA-dir",
			LogFile:          "log/serviceA.log",
			KillTimeout:      10,
			KillSignal:       "SIGQUIT",
			Count:            2,
			RespawnInterval:  25,
			RespawnCount:     15,
			IsRespawnEnabled: true,
			LimitFile:        1024,
		},
	}

	serviceB := &procfile.Service{
		Name:        "serviceB",
		Cmd:         "/bin/echo 'serviceB'",
		Application: app,
		Options: &procfile.ServiceOptions{
			EnvFile:          "shared/env.vars",
			Env:              map[string]string{"STAGING": "true"},
			WorkingDir:       "/srv/service/working-dir",
			ReloadSignal:     "SIGUSR2",
			IsRespawnEnabled: true,
			LimitFile:        4096,
			LimitProc:        4096,
		},
	}

	app.Services = append(app.Services, serviceA, serviceB)

	return app
}

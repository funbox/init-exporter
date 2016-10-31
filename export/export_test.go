package export

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                       Copyright (c) 2006-2016 FB GROUP LLC                         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/funbox/init-exporter/procfile"

	"pkg.re/essentialkaos/ek.v5/fsutil"
	"pkg.re/essentialkaos/ek.v5/log"

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

	c.Assert(fsutil.IsExist(targetDir+"/test-application.conf"), Equals, true)
	c.Assert(fsutil.IsRegular(targetDir+"/test-application.conf"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(targetDir+"/test-application.conf"), Equals, true)

	c.Assert(fsutil.IsExist(targetDir+"/test-application_service1.conf"), Equals, true)
	c.Assert(fsutil.IsRegular(targetDir+"/test-application_service1.conf"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(targetDir+"/test-application_service1.conf"), Equals, true)

	c.Assert(fsutil.IsExist(targetDir+"/test-application_service2.conf"), Equals, true)
	c.Assert(fsutil.IsRegular(targetDir+"/test-application_service2.conf"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(targetDir+"/test-application_service2.conf"), Equals, true)

	c.Assert(fsutil.IsExist(helperDir+"/test-application_service1.sh"), Equals, true)
	c.Assert(fsutil.IsRegular(helperDir+"/test-application_service1.sh"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(helperDir+"/test-application_service1.sh"), Equals, true)

	c.Assert(fsutil.IsExist(helperDir+"/test-application_service2.sh"), Equals, true)
	c.Assert(fsutil.IsRegular(helperDir+"/test-application_service2.sh"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(helperDir+"/test-application_service2.sh"), Equals, true)

	appUnitData, err := ioutil.ReadFile(targetDir + "/test-application.conf")

	c.Assert(err, IsNil)
	c.Assert(appUnitData, NotNil)

	service1UnitData, err := ioutil.ReadFile(targetDir + "/test-application_service1.conf")

	c.Assert(err, IsNil)
	c.Assert(service1UnitData, NotNil)

	service2UnitData, err := ioutil.ReadFile(targetDir + "/test-application_service2.conf")

	c.Assert(err, IsNil)
	c.Assert(service2UnitData, NotNil)

	service1HelperData, err := ioutil.ReadFile(helperDir + "/test-application_service1.sh")

	c.Assert(err, IsNil)
	c.Assert(service1HelperData, NotNil)

	service2HelperData, err := ioutil.ReadFile(helperDir + "/test-application_service2.sh")

	c.Assert(err, IsNil)
	c.Assert(service2HelperData, NotNil)

	appUnit := strings.Split(string(appUnitData), "\n")
	service1Unit := strings.Split(string(service1UnitData), "\n")
	service2Unit := strings.Split(string(service2UnitData), "\n")
	service1Helper := strings.Split(string(service1HelperData), "\n")
	service2Helper := strings.Split(string(service2HelperData), "\n")

	c.Assert(appUnit, HasLen, 16)
	c.Assert(service1Unit, HasLen, 18)
	c.Assert(service2Unit, HasLen, 18)
	c.Assert(service1Helper, HasLen, 7)
	c.Assert(service2Helper, HasLen, 7)

	c.Assert(appUnit[2:], DeepEquals,
		[]string{
			"start on [3]",
			"stop on [3]",
			"",
			"pre-start script",
			"",
			"bash << \"EOF\"",
			"  mkdir -p /var/log/test-application",
			"  chown -R service /var/log/test-application",
			"  chgrp -R service /var/log/test-application",
			"  chmod -R g+w /var/log/test-application",
			"EOF",
			"",
			"end script", ""},
	)

	c.Assert(service1Unit[2:], DeepEquals,
		[]string{
			"start on [3]",
			"stop on [3]",
			"",
			"respawn",
			"respawn limit 15 25",
			"",
			"kill timeout 10",
			"",
			"script",
			"  touch /var/log/test-application/service1.log",
			"  chown service /var/log/test-application/service1.log",
			"  chgrp service /var/log/test-application/service1.log",
			"  chmod g+w /var/log/test-application/service1.log",
			fmt.Sprintf("  exec sudo -u service /bin/bash %s/test-application_service1.sh >> /srv/service/service1-dir/custom.log >> /var/log/test-application/service1.log 2>&1", helperDir),
			"end script", ""},
	)

	c.Assert(service2Unit[2:], DeepEquals,
		[]string{
			"start on [3]",
			"stop on [3]",
			"",
			"respawn",
			"",
			"",
			"kill timeout 0",
			"",
			"script",
			"  touch /var/log/test-application/service2.log",
			"  chown service /var/log/test-application/service2.log",
			"  chgrp service /var/log/test-application/service2.log",
			"  chmod g+w /var/log/test-application/service2.log",
			fmt.Sprintf("  exec sudo -u service /bin/bash %s/test-application_service2.sh >> /var/log/test-application/service2.log 2>&1", helperDir),
			"end script", ""},
	)

	c.Assert(service1Helper[4:], DeepEquals,
		[]string{
			"[[ -r /etc/profile.d/rbenv.sh ]] && source /etc/profile.d/rbenv.sh",
			"cd /srv/service/service1-dir && exec STAGING=true /bin/echo service1",
			""},
	)

	c.Assert(service2Helper[4:], DeepEquals,
		[]string{
			"[[ -r /etc/profile.d/rbenv.sh ]] && source /etc/profile.d/rbenv.sh",
			"cd /srv/service/working-dir && exec /bin/echo service2",
			""},
	)

	err = exporter.Uninstall(app)

	c.Assert(err, IsNil)

	c.Assert(fsutil.IsExist(targetDir+"/test-application.conf"), Equals, false)
	c.Assert(fsutil.IsExist(targetDir+"/test-application_service1.conf"), Equals, false)
	c.Assert(fsutil.IsExist(targetDir+"/test-application_service2.conf"), Equals, false)
	c.Assert(fsutil.IsExist(helperDir+"/test-application_service1.sh"), Equals, false)
	c.Assert(fsutil.IsExist(helperDir+"/test-application_service2.sh"), Equals, false)
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

	c.Assert(fsutil.IsExist(targetDir+"/test-application.service"), Equals, true)
	c.Assert(fsutil.IsRegular(targetDir+"/test-application.service"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(targetDir+"/test-application.service"), Equals, true)

	c.Assert(fsutil.IsExist(targetDir+"/test-application_service1.service"), Equals, true)
	c.Assert(fsutil.IsRegular(targetDir+"/test-application_service1.service"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(targetDir+"/test-application_service1.service"), Equals, true)

	c.Assert(fsutil.IsExist(targetDir+"/test-application_service2.service"), Equals, true)
	c.Assert(fsutil.IsRegular(targetDir+"/test-application_service2.service"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(targetDir+"/test-application_service2.service"), Equals, true)

	c.Assert(fsutil.IsExist(helperDir+"/test-application_service1.sh"), Equals, true)
	c.Assert(fsutil.IsRegular(helperDir+"/test-application_service1.sh"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(helperDir+"/test-application_service1.sh"), Equals, true)

	c.Assert(fsutil.IsExist(helperDir+"/test-application_service2.sh"), Equals, true)
	c.Assert(fsutil.IsRegular(helperDir+"/test-application_service2.sh"), Equals, true)
	c.Assert(fsutil.IsNonEmpty(helperDir+"/test-application_service2.sh"), Equals, true)

	appUnitData, err := ioutil.ReadFile(targetDir + "/test-application.service")

	c.Assert(err, IsNil)
	c.Assert(appUnitData, NotNil)

	service1UnitData, err := ioutil.ReadFile(targetDir + "/test-application_service1.service")

	c.Assert(err, IsNil)
	c.Assert(service1UnitData, NotNil)

	service2UnitData, err := ioutil.ReadFile(targetDir + "/test-application_service2.service")

	c.Assert(err, IsNil)
	c.Assert(service2UnitData, NotNil)

	service1HelperData, err := ioutil.ReadFile(helperDir + "/test-application_service1.sh")

	c.Assert(err, IsNil)
	c.Assert(service1HelperData, NotNil)

	service2HelperData, err := ioutil.ReadFile(helperDir + "/test-application_service2.sh")

	c.Assert(err, IsNil)
	c.Assert(service2HelperData, NotNil)

	appUnit := strings.Split(string(appUnitData), "\n")
	service1Unit := strings.Split(string(service1UnitData), "\n")
	service2Unit := strings.Split(string(service2UnitData), "\n")
	service1Helper := strings.Split(string(service1HelperData), "\n")
	service2Helper := strings.Split(string(service2HelperData), "\n")

	c.Assert(appUnit, HasLen, 22)
	c.Assert(service1Unit, HasLen, 26)
	c.Assert(service2Unit, HasLen, 26)
	c.Assert(service1Helper, HasLen, 7)
	c.Assert(service2Helper, HasLen, 7)

	c.Assert(appUnit[2:], DeepEquals,
		[]string{
			"[Unit]",
			"",
			"Description=Unit for test-application application",
			"After=multi-user.target",
			"Wants=test-application_service1.service test-application_service2.service",
			"",
			"[Service]",
			"Type=oneshot",
			"RemainAfterExit=true",
			"",
			"ExecStartPre=/bin/mkdir -p /var/log/test-application",
			"ExecStartPre=/bin/chown -R service /var/log/test-application",
			"ExecStartPre=/bin/chgrp -R service /var/log/test-application",
			"ExecStartPre=/bin/chmod -R g+w /var/log/test-application",
			"ExecStart=/bin/echo \"test-application started\"",
			"ExecStop=/bin/echo \"test-application stopped\"",
			"",
			"[Install]",
			"WantedBy=multi-user.target", ""},
	)

	c.Assert(service1Unit[2:], DeepEquals,
		[]string{
			"[Unit]",
			"",
			"Description=Unit for service1 service (part of test-application application)",
			"PartOf=test-application.service",
			"",
			"[Service]",
			"Type=simple",
			"",
			"TimeoutStopSec=10",
			"Restart=on-failure",
			"StartLimitInterval=25",
			"StartLimitBurst=15",
			"",
			"ExecStartPre=/bin/touch /var/log/test-application/service1.log",
			"ExecStartPre=/bin/chown service /var/log/test-application/service1.log",
			"ExecStartPre=/bin/chgrp service /var/log/test-application/service1.log",
			"ExecStartPre=/bin/chmod g+w /var/log/test-application/service1.log",
			"",
			"User=service",
			"Group=service",
			"WorkingDirectory=/srv/service/service1-dir",
			"Environment=STAGING=true",
			fmt.Sprintf("ExecStart=/bin/bash %s/test-application_service1.sh >> /srv/service/service1-dir/custom.log >> /var/log/test-application/service1.log 2>&1", helperDir),
			""},
	)

	c.Assert(service2Unit[2:], DeepEquals,
		[]string{
			"[Unit]",
			"",
			"Description=Unit for service2 service (part of test-application application)",
			"PartOf=test-application.service",
			"",
			"[Service]",
			"Type=simple",
			"",
			"TimeoutStopSec=0",
			"Restart=on-failure",
			"",
			"",
			"",
			"ExecStartPre=/bin/touch /var/log/test-application/service2.log",
			"ExecStartPre=/bin/chown service /var/log/test-application/service2.log",
			"ExecStartPre=/bin/chgrp service /var/log/test-application/service2.log",
			"ExecStartPre=/bin/chmod g+w /var/log/test-application/service2.log",
			"",
			"User=service",
			"Group=service",
			"WorkingDirectory=/srv/service/working-dir",
			"",
			fmt.Sprintf("ExecStart=/bin/bash %s/test-application_service2.sh >> /var/log/test-application/service2.log 2>&1", helperDir),
			""},
	)

	c.Assert(service1Helper[4:], DeepEquals,
		[]string{
			"[[ -r /etc/profile.d/rbenv.sh ]] && source /etc/profile.d/rbenv.sh",
			"exec /bin/echo service1",
			""},
	)

	c.Assert(service2Helper[4:], DeepEquals,
		[]string{
			"[[ -r /etc/profile.d/rbenv.sh ]] && source /etc/profile.d/rbenv.sh",
			"exec /bin/echo service2",
			""},
	)

	err = exporter.Uninstall(app)

	c.Assert(err, IsNil)

	c.Assert(fsutil.IsExist(targetDir+"/test-application.service"), Equals, false)
	c.Assert(fsutil.IsExist(targetDir+"/test-application_service1.service"), Equals, false)
	c.Assert(fsutil.IsExist(targetDir+"/test-application_service2.service"), Equals, false)
	c.Assert(fsutil.IsExist(helperDir+"/test-application_service1.sh"), Equals, false)
	c.Assert(fsutil.IsExist(helperDir+"/test-application_service2.sh"), Equals, false)
}

// ////////////////////////////////////////////////////////////////////////////////// //

func createTestApp(helperDir, targetDir string) *procfile.Application {
	app := &procfile.Application{
		Name:        "test-application",
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
		Cmd:         "/bin/echo service1",
		Application: app,
		Options: &procfile.ServiceOptions{
			Env:             map[string]string{"STAGING": "true"},
			WorkingDir:      "/srv/service/service1-dir",
			LogPath:         "/srv/service/service1-dir/custom.log",
			KillTimeout:     10,
			Count:           2,
			RespawnInterval: 25,
			RespawnCount:    15,
			RespawnEnabled:  true,
		},
	}

	service2 := &procfile.Service{
		Name:        "service2",
		Cmd:         "/bin/echo service2",
		Application: app,
		Options: &procfile.ServiceOptions{
			WorkingDir:     "/srv/service/working-dir",
			RespawnEnabled: true,
		},
	}

	app.Services = append(app.Services, service1, service2)

	return app
}

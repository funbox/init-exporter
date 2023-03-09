package support

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"os/exec"
	"strings"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/fsutil"
	"github.com/essentialkaos/ek/v12/system"
	"github.com/essentialkaos/ek/v12/system/container"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// showOSInfo shows verbose information about system
func showOSInfo() {
	osInfo, err := system.GetOSInfo()

	if err == nil {
		fmtutil.Separator(false, "OS INFO")

		printInfo(12, "Name", osInfo.Name)
		printInfo(12, "Pretty Name", osInfo.PrettyName)
		printInfo(12, "Version", osInfo.VersionID)
		printInfo(12, "ID", osInfo.ID)
		printInfo(12, "ID Like", osInfo.IDLike)
		printInfo(12, "Version ID", osInfo.VersionID)
		printInfo(12, "Version Code", osInfo.VersionCodename)
		printInfo(12, "CPE", osInfo.CPEName)
	}

	systemInfo, err := system.GetSystemInfo()

	if err != nil {
		return
	} else {
		if osInfo == nil {
			fmtutil.Separator(false, "SYSTEM INFO")
			printInfo(12, "Name", systemInfo.OS)
		}
	}

	printInfo(12, "Arch", systemInfo.Arch)
	printInfo(12, "Kernel", systemInfo.Kernel)

	containerEngine := "No"

	switch container.GetEngine() {
	case container.DOCKER:
		containerEngine = "Yes (Docker)"
	case container.PODMAN:
		containerEngine = "Yes (Podman)"
	case container.LXC:
		containerEngine = "Yes (LXC)"
	}

	fmtc.NewLine()

	printInfo(12, "Container", containerEngine)
}

// showEnvInfo shows info about environment
func showEnvInfo(pkgs Pkgs) {
	fmtutil.Separator(false, "ENVIRONMENT")

	size := pkgs.getMaxSize()

	for _, pkg := range pkgs {
		printInfo(size, pkg.Name, pkg.Version)
	}
}

// collectEnvInfo collects info about packages
func collectEnvInfo() Pkgs {
	return Pkgs{
		getPackageInfo("systemd"),
		getPackageInfo("upstart"),
	}
}

// getPackageVersion returns package name and version from package manager database
func getPackageInfo(names ...string) Pkg {
	var info Pkg

	if len(names) == 0 {
		return Pkg{}
	}

	for _, name := range names {
		switch {
		case isDEBBased():
			info = getDEBPackageInfo(name)
		case isRPMBased():
			info = getRPMPackageInfo(name)
		}

		if info.Version != "" {
			return info
		}
	}

	return Pkg{names[0], ""}
}

// isDEBBased returns true if is DEB-based distro
func isRPMBased() bool {
	return fsutil.IsExist("/usr/bin/rpm")
}

// isDEBBased returns true if is DEB-based distro
func isDEBBased() bool {
	return fsutil.IsExist("/usr/bin/dpkg-query")
}

// getRPMPackageInfo returns info about RPM package
func getRPMPackageInfo(name string) Pkg {
	cmd := exec.Command("rpm", "-q", name)
	out, err := cmd.Output()

	if err != nil || len(out) == 0 {
		return Pkg{name, ""}
	}

	return Pkg{name, strings.TrimRight(string(out), "\n\r")}
}

// getDEBPackageInfo returns info about DEB package
func getDEBPackageInfo(name string) Pkg {
	cmd := exec.Command("dpkg-query", "--showformat=${Version}", "--show", name)
	out, err := cmd.Output()

	if err != nil || len(out) == 0 {
		return Pkg{name, ""}
	}

	return Pkg{name, string(out)}
}

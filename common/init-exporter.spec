###############################################################################

# rpmbuilder:relative-pack true

###############################################################################

%define _posixroot        /
%define _root             /root
%define _bin              /bin
%define _sbin             /sbin
%define _srv              /srv
%define _home             /home
%define _lib32            %{_posixroot}lib
%define _lib64            %{_posixroot}lib64
%define _libdir32         %{_prefix}%{_lib32}
%define _libdir64         %{_prefix}%{_lib64}
%define _logdir           %{_localstatedir}/log
%define _rundir           %{_localstatedir}/run
%define _lockdir          %{_localstatedir}/lock/subsys
%define _cachedir         %{_localstatedir}/cache
%define _spooldir         %{_localstatedir}/spool
%define _crondir          %{_sysconfdir}/cron.d
%define _loc_prefix       %{_prefix}/local
%define _loc_exec_prefix  %{_loc_prefix}
%define _loc_bindir       %{_loc_exec_prefix}/bin
%define _loc_libdir       %{_loc_exec_prefix}/%{_lib}
%define _loc_libdir32     %{_loc_exec_prefix}/%{_lib32}
%define _loc_libdir64     %{_loc_exec_prefix}/%{_lib64}
%define _loc_libexecdir   %{_loc_exec_prefix}/libexec
%define _loc_sbindir      %{_loc_exec_prefix}/sbin
%define _loc_bindir       %{_loc_exec_prefix}/bin
%define _loc_datarootdir  %{_loc_prefix}/share
%define _loc_includedir   %{_loc_prefix}/include
%define _rpmstatedir      %{_sharedstatedir}/rpm-state
%define _pkgconfigdir     %{_libdir}/pkgconfig

###############################################################################

%define  debug_package %{nil}

###############################################################################

Summary:         Utility for exporting services described by Procfile to init system
Name:            init-exporter
Version:         0.17.0
Release:         0%{?dist}
Group:           Development/Tools
License:         MIT
URL:             https://github.com/funbox/init-exporter

Source0:         %{name}-%{version}.tar.gz

BuildRoot:       %{_tmppath}/%{name}-%{version}-%{release}-root-%(%{__id_u} -n)

BuildRequires:   golang >= 1.8

Provides:        upstart-exporter = %{version}-%{release}
Provides:        systemd-exporter = %{version}-%{release}

Provides:        %{name} = %{version}-%{release}

###############################################################################

%description
Utility for exporting services described by Procfile to init system.

###############################################################################

%prep
%setup -q

%build
export GOPATH=$(pwd)

pushd src/github.com/funbox/%{name}
  %{__make} %{?_smp_mflags} all
popd

%install
rm -rf %{buildroot}

install -dm 755 %{buildroot}%{_bindir}
install -dm 755 %{buildroot}%{_sysconfdir}
install -dm 755 %{buildroot}%{_logdir}/%{name}
install -dm 755 %{buildroot}%{_loc_prefix}/%{name}
install -dm 755 %{buildroot}%{_localstatedir}/local/%{name}/helpers

install -pm 755 src/github.com/funbox/%{name}/%{name} \
                %{buildroot}%{_bindir}/

ln -sf %{_bindir}/%{name} %{buildroot}%{_bindir}/upstart-export
ln -sf %{_bindir}/%{name} %{buildroot}%{_bindir}/systemd-export

install -pm 755 src/github.com/funbox/%{name}/common/%{name}.conf \
                %{buildroot}%{_sysconfdir}/

%clean
rm -rf %{buildroot}

###############################################################################

%files
%defattr(-,root,root,-)
%config(noreplace) %{_sysconfdir}/%{name}.conf
%dir %{_logdir}/%{name}
%dir %{_localstatedir}/local/%{name}/helpers
%{_bindir}/init-exporter
%{_bindir}/upstart-export
%{_bindir}/systemd-export

###############################################################################

%changelog
* Wed Nov 08 2017 Anton Novojilov <andyone@fun-box.ru> - 0.17.0-0
- Reloading units after application installation and uninstallation

* Fri Oct 13 2017 Anton Novojilov <andyone@fun-box.ru> - 0.16.1-0
- Fixed bug with exporting multiple systemd units for command
- Fixed bug with generating Wants clause for systemd unit exceeding LINE_MAX

* Thu Sep 14 2017 Anton Novojilov <andyone@fun-box.ru> - 0.16.0-0
- Improved environment variables parsing in v1

* Tue Aug 29 2017 Anton Novojilov <andyone@fun-box.ru> - 0.15.2-0
- Fixed output redirect in systemd units

* Thu Jun 01 2017 Anton Novojilov <andyone@fun-box.ru> - 0.15.1-0
- Added support of asterisk symbol in environment variables
- Improved paths validation
- Improved procfile path validation

* Thu May 18 2017 Anton Novojilov <andyone@fun-box.ru> - 0.15.0-0
- Migrated to ek.v9
- Fixed count property handling
- init-exporter-converter moved to separate repository
- Changed default permissions for helpers to 0644

* Wed May 03 2017 Anton Novojilov <andyone@fun-box.ru> - 0.14.0-1
- [init-exporter-converter] Added validation for result YAML data

* Wed Apr 26 2017 Anton Novojilov <andyone@fun-box.ru> - 0.14.0-0
- Environment variables and file with environment variables now can defined
  in same time

* Mon Apr 24 2017 Anton Novojilov <andyone@fun-box.ru> - 0.13.0-1
- [init-exporter-converter] Replaced text/template by basic procfile rendering

* Mon Apr 24 2017 Anton Novojilov <andyone@fun-box.ru> - 0.13.0-0
- ek package updated to v8
- Improved v2 format validation

* Mon Apr 17 2017 Anton Novojilov <andyone@fun-box.ru> - 0.12.3-0
- Fixed typo in uninstall option name

* Thu Apr 13 2017 Anton Novojilov <andyone@fun-box.ru> - 0.12.2-0
- Added stderr redirect to /dev/null for env file reading command

* Mon Apr 10 2017 Anton Novojilov <andyone@fun-box.ru> - 0.12.1-0
- Improved environment variables validation for support appending of variables

* Mon Apr 10 2017 Anton Novojilov <andyone@fun-box.ru> - 0.12.0-0
- Improved environment variables validation
- Added argument for disabling application validation

* Fri Apr 07 2017 Anton Novojilov <andyone@fun-box.ru> - 0.11.0-0
- Added application validation before installation
- [converter] Added application validation before procfile converting
- [converter] Improved converter for better support of procfiles for local run
- Code refactoring

* Tue Apr 04 2017 Anton Novojilov <andyone@fun-box.ru> - 0.10.0-0
- Added kill signal definition feature for v2 and all exporters
- Added reload signal definition feature for v2 and all exporters
- Improved parsing commands in v2 Procfile format

* Mon Apr 03 2017 Anton Novojilov <andyone@fun-box.ru> - 0.9.0-2
- [converter] Fixed bug with wrong path to working dir

* Mon Apr 03 2017 Anton Novojilov <andyone@fun-box.ru> - 0.9.0-1
- Format converter moved to separate package
- Minor fixes in format converter

* Fri Mar 31 2017 Anton Novojilov <andyone@fun-box.ru> - 0.9.0-0
- Format support configuration feature
- Pre and post commands support
- Added format converter

* Thu Mar 09 2017 Anton Novojilov <andyone@fun-box.ru> - 0.8.0-0
- ek package updated to v7

* Fri Mar 03 2017 Anton Novojilov <andyone@fun-box.ru> - 0.7.2-1
- Fixed wrong path to upsrtart-exporter binary

* Thu Mar 02 2017 Anton Novojilov <andyone@fun-box.ru> - 0.7.2-0
- [upstart] Fixed bug with setting environment variables

* Thu Feb 23 2017 Anton Novojilov <andyone@fun-box.ru> - 0.7.1-0
- [upstart|systemd] More secure helper output redirection

* Wed Feb 22 2017 Anton Novojilov <andyone@fun-box.ru> - 0.7.0-0
- Fixed bug with export to upstart
- Improved working with default values

* Thu Feb  2 2017 Anton Novojilov <andyone@fun-box.ru> - 0.6.0-0
- Initial build

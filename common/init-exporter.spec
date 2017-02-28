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
Version:         0.7.1
Release:         0%{?dist}
Group:           Development/Tools
License:         MIT
URL:             https://github.com/funbox/init-exporter

Source0:         %{name}-%{version}.tar.gz

BuildRoot:       %{_tmppath}/%{name}-%{version}-%{release}-root-%(%{__id_u} -n)

BuildRequires:   golang >= 1.7

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
go build -o %{name} src/github.com/funbox/%{name}/%{name}.go

%install
rm -rf %{buildroot}

install -dm 755 %{buildroot}%{_bindir}
install -dm 755 %{buildroot}%{_sysconfdir}
install -dm 755 %{buildroot}%{_logdir}/%{name}
install -dm 755 %{buildroot}%{_loc_prefix}/%{name}
install -dm 755 %{buildroot}%{_localstatedir}/local/%{name}/helpers

install -pm 755 %{name} %{buildroot}%{_bindir}/

ln -sf %{_bindir}/%{name} %{buildroot}%{_bindir}/upstart-exporter
ln -sf %{_bindir}/%{name} %{buildroot}%{_bindir}/systemd-exporter

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
%{_bindir}/*-exporter

###############################################################################

%changelog
* Thu Feb 23 2017 Anton Novojilov <andyone@fun-box.ru> - 0.7.1-0
- More secure helper output redirection

* Wed Feb 22 2017 Anton Novojilov <andyone@fun-box.ru> - 0.7.0-0
- Fixed bug with export to upstart
- Improved working with default values

* Thu Feb  2 2017 Anton Novojilov <andyone@fun-box.ru> - 0.6.0-0
- Initial build

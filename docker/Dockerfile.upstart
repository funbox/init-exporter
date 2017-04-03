FROM centos:centos6

ENV GOPATH /root
ENV TARGET /root/src/github.com/funbox/init-exporter

RUN yum install -y https://yum.kaos.io/6/release/x86_64/kaos-repo-8.0-0.el6.noarch.rpm
RUN yum clean all && yum -y update

RUN yum -y install make golang

COPY . $TARGET
RUN ls $TARGET -al
RUN cd $TARGET && make all && make install

RUN useradd service
RUN mkdir -p /var/local/init-exporter/helpers && mkdir -p /var/log/init-exporter

COPY ./testdata /root

RUN yum clean all && rm -rf /tmp/*

WORKDIR /root

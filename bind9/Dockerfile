FROM debian:unstable

MAINTAINER Clint Ruoho clint@wtfismyip.com

RUN apt-get -y update
RUN apt-get -y install bind9

COPY named.conf.options /etc/bind

WORKDIR /app
ADD . /app
CMD [ "/usr/sbin/named", "-f", "-u bind" ]


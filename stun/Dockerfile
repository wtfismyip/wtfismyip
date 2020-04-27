FROM ubuntu:20.04

MAINTAINER Clint Ruoho clint@wtfismyip.com

RUN apt-get update
RUN apt-get -y install coturn

COPY turnserver.conf /etc/turnserver.conf

CMD [ "/usr/bin/turnserver" ]

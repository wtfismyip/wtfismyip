FROM debian:unstable

MAINTAINER Clint Ruoho clint@wtfismyip.com

RUN apt-get -y update
RUN apt-get -y install coturn procps

COPY turnserver.conf /etc/turnserver.conf

CMD [ "/usr/bin/turnserver" ]

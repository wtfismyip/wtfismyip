FROM debian:unstable

MAINTAINER Clint Ruoho clint@wtfismyip.com

RUN apt-get -y update
RUN apt-get -y install unbound procps

COPY unbound.conf /etc/unbound/unbound.conf

WORKDIR /app
ADD . /app
CMD [ "bash", "unbound.sh" ]

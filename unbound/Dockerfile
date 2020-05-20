FROM debian:unstable

MAINTAINER Clint Ruoho clint@wtfismyip.com

RUN apt-get update
RUN apt-get -y install unbound

COPY unbound.conf /etc/unbound/unbound.conf

WORKDIR /app
ADD . /app
CMD [ "bash", "unbound.sh" ]

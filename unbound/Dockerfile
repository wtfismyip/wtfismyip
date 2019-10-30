FROM ubuntu:18.04

MAINTAINER Clint Ruoho clint@wtfismyip.com

RUN apt-get update
RUN apt-get -y install unbound

WORKDIR /app
ADD . /app
CMD [ "bash", "unbound.sh" ]

FROM debian:unstable

MAINTAINER Clint Ruoho clint@wtfismyip.com

RUN apt clean
RUN apt-get -y update
RUN apt-get -y install unbound procps util-linux

ARG USER_ID=101
ARG GROUP_ID=101

COPY unbound.conf /etc/unbound/unbound.conf

WORKDIR /app
ADD . /app
CMD [ "bash", "unbound.sh" ]

FROM debian:unstable

MAINTAINER Clint Ruoho clint@wtfismyip.com

RUN apt-get -y update
RUN apt-get -y install redis procps

WORKDIR /app
ADD . /app
USER redis
CMD [ "bash", "redis.sh" ]

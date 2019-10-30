FROM golang:latest

MAINTAINER Clint Ruoho clint@wtfismyip.com

WORKDIR /app
COPY go.mod go.sum ./
COPY static/go /usr/local/wtf/go
COPY static/GeoIP /usr/local/wtf/GeoIP
RUN go mod download
COPY wtf.go ./
RUN go build -o wtf .

RUN groupadd -g 999 appuser && \
    useradd -r -u 999 -g appuser appuser

USER appuser
CMD [ "./wtf" ]

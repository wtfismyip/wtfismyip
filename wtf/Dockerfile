FROM golang:latest as builder

MAINTAINER Clint Ruoho clint@wtfismyip.com

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY wtf.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o wtf .

FROM alpine:latest

WORKDIR /app

RUN mkdir -p /usr/local/wtf/GeoIP
COPY static/GeoIP /usr/local/wtf/GeoIP
RUN addgroup -S appuser && adduser -S -G appuser appuser
USER appuser

COPY --from=builder /app/wtf .

CMD [ "./wtf" ]

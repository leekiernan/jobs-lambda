FROM golang:latest

RUN apt-get update && apt-get install -y \
  cron \
  curl \
  libfontconfig \
  unzip \
  vim \
  xvfb

COPY ./crontab /etc/cron.d/runner-cron
RUN chmod 0644 /etc/cron.d/runner-cron
RUN crontab /etc/cron.d/runner-cron

WORKDIR /root/go/src/
COPY ./checker checker
COPY ./runner runner

ENV GOPATH /root/go

RUN go get -u "github.com/lib/pq"
# RUN go get -d -v ./...
# RUN go install -v ./...

ENV EDITOR vim

# COPY entrypoint.sh /
# RUN chmod +x /entrypoint.sh
# ENTRYPOINT  ["/entrypoint.sh"]

# CMD ["app"]
CMD ["cron", "-f"]

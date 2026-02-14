FROM golang:1.25.5

WORKDIR /opt/mao
COPY . .

RUN go mod download

RUN groupadd --gid 1000 mao_user && useradd --uid 1000 --gid mao_user -ms /bin/bash mao_user

USER mao_user

EXPOSE 9090
CMD ["go", "run", "/opt/mao/server/server.go"]

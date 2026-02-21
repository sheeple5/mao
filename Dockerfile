FROM golang:1.25.5

WORKDIR /opt/mao
COPY . .

RUN go mod download

RUN groupadd --gid 1000 mao_user && useradd --uid 1000 --gid mao_user -ms /bin/bash mao_user
RUN chown -R mao_user:mao_user /opt/mao

USER mao_user

EXPOSE 9090

WORKDIR /opt/mao/server
CMD ["go", "run", "/opt/mao/server/server.go"]

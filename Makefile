.PHONY: build build_server build_client run_server run_client stop

build:
	docker build -t mao .

build_server:
	docker run --rm -v $$(pwd):/opt/mao mao go build -o mao_server server/server.go

build_client:
	docker run --rm -v $$(pwd):/opt/mao mao go build  -o mao_client client/client.go

run_server:
	@docker run --name mao -d --rm -e OPENAI_TOKEN=$(OPENAI_TOKEN) -p 9090:9090 mao

run_client:
	@docker run --name mao_client --rm -it mao go run /opt/mao/client/client.go

stop:
	docker stop mao || true

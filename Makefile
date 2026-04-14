.PHONY: test build docker-up docker-down clean

test:
	go test -v ./...

build:
	go build -o main .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

clean:
	docker-compose down -v
	rm -f main

ci-local: test build
	@echo "✅ CI checks passed locally!"

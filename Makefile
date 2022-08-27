BINARY_NAME=myapp
DSN="host=79.175.177.109 port=5432 user=postgres password=oV982hF9 dbname=subsription_system sslmode=disable"
REDIS="79.175.177.109:6379"

build:
	@echo "Building..."
	env CGO_ENABLED=0 go build -ldflags="-s -w" -o ${BINARY_NAME} ./cmd/web
	@echo "Built!"

run: build
	@echo "Starting..."
	@env DSN=${DSN} REDIS=${REDIS} ./${BINARY_NAME} &
	@echo "Started"

clean:
	@echo "Cleaning"
	@go clean
	@rm ${BINARY_NAME}
	@echo "Cleaned"

start: run

stop:
	@echo "Stopping..."
	@-pkill -SIGTERM -f "./${BINARY_NAME}"
	@echo "Stopped..."

restart: stop start

test:
	go test -v ./...
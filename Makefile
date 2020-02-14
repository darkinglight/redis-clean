CC=go build
CFLAGS=-gcflags=all="-N -l"

all: build
build: main.go
	$(CC) -o bin/redis-clean main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(CC) -o bin/redis-clean.exe main.go
gdb-build: main.go
	$(CC) -o bin/redis-clean $(CFLAGS) main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(CC) -o bin/redis-clean.exe $(CFLAGS) main.go
clean:
	rm -fr bin/*

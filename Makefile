CC=go build
CFLAGS=

all: main
main: main.go
	$(CC) -o bin/redis-clean $(CFLAGS) main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(CC) -o bin/redis-clean.exe $(CFLAGS) main.go
clean:
	rm -fr bin/*

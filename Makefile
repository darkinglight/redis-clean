CC=go build
CFLAGS=-gcflags=all="-N -l"

all: bin/redis-clean bin/redis-clean.exe
bin/redis-clean: main.go
	$(CC) -o bin/redis-clean main.go
bin/redis-clean.exe: main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(CC) -o bin/redis-clean.exe main.go

gdb-build: main.go
	$(CC) -o bin/redis-clean $(CFLAGS) main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(CC) -o bin/redis-clean.exe $(CFLAGS) main.go

tar: bin/redis-clean-linux.tar.gz bin/redis-clean-windows.tar.gz
bin/redis-clean-linux.tar.gz: bin/redis-clean
	tar -zcvf bin/redis-clean-linux.tar.gz config.yaml bin/redis-clean README.md
bin/redis-clean-windows.tar.gz: bin/redis-clean.exe
	tar -zcvf bin/redis-clean-windows.tar.gz config.yaml bin/redis-clean.exe README.md

clean:
	rm -fr bin/*

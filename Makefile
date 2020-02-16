CC=go build
CFLAGS=-gcflags=all="-N -l"
FILE=main.go config.go process.go redis.go

.PHONY:all
all: bin/redis-clean bin/redis-clean.exe
bin/redis-clean: $(FILE)
	$(CC) -o $@ $^
bin/redis-clean.exe: $(FILE)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(CC) -o $@ $^

.PHONY:gdb-build
gdb-build: $(FILE)
	$(CC) -o bin/redis-clean $(CFLAGS) $(FILE)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(CC) -o bin/redis-clean.exe $(CFLAGS) $(FILE)

.PHONY:tar
tar: bin/redis-clean-linux.tar.gz bin/redis-clean-windows.tar.gz
bin/redis-clean-linux.tar.gz: bin/redis-clean
	tar -zcvf $@ config.yaml README.md -C bin redis-clean
bin/redis-clean-windows.tar.gz: bin/redis-clean.exe
	tar -zcvf $@ config.yaml README.md -C bin redis-clean.exe

.PHONY:clean
clean:
	rm -fr bin/*

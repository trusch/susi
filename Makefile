OS=linux
ARCH=amd64

bin/susi-server: Makefile src/susi-server/*.go src/susi-server/*/*.go src/susi-server/*/*/*.go bin
	GOOS=$(OS) GOARCH=$(ARCH) go build -o bin/susi-server src/susi-server/susi-server.go

test: dev/cert.pem bin/susi-server
	-killall susi-server
	./bin/susi-server

dev/cert.pem:
	cd dev && go run generate_cert.go --host 127.0.0.1

clean:
	-rm -rf bin
	-rm -rf ./dev/cert.pem ./dev/key.pem
	-rm -rf ./build/susi-server.deb
	-rm -rf ./build/susi-server/usr/bin/susi-server

del-binary:
	-rm -f bin/susi-server

fmt:
	for d in src/susi-server/*/; do go fmt $$d/*; done

bin:
	mkdir bin

server-package: del-binary bin/susi-server
	mkdir -p build/susi-server/usr/bin build/susi-server/usr/share/susi build/susi-server/etc
	cp bin/susi-server build/susi-server/usr/bin/susi-server
	cd build && dpkg -b susi-server
bin/susi-server: src/susi-server/*.go src/susi-server/*/*.go bin
	go build -o bin/susi-server src/susi-server/susi-server.go

test: dev/cert.pem bin/susi-server
	-killall susi-server
	./bin/susi-server

dev/cert.pem:
	cd dev && go run generate_cert.go --host 127.0.0.1

clean:
	-rm -rf bin
	-rm -rf ./dev/cert.pem ./dev/key.pem

fmt:
	for d in src/susi-server/*/; do go fmt $$d/*; done

bin:
	mkdir bin

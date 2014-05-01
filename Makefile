bin/susi-server: src/susi-server/*.go src/susi-server/*/*.go bin
	go build -o bin/susi-server src/susi-server/susi-server.go

test: bin/susi-server
	-killall susi-server
	./bin/susi-server

clean:
	-rm -rf bin
	-rm -rf ./dev/cert.pem ./dev/key.pem

bin:
	mkdir bin

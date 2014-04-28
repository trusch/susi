bin/susi-server: src/susi-server/*.go src/susi-server/*/*.go bin
	go build -o bin/susi-server src/susi-server/susi-server.go

clean:
	-rm -rf bin

bin:
	mkdir bin

test:
	cd ircfmt && go test . && go vet .
	cd ircmap && go test . && go vet .
	cd ircmatch && go test . && go vet .
	cd ircmsg && go test . && go vet .
	cd ircutils && go test . && go vet .

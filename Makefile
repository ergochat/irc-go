.PHONY: test ircevent gofmt

test:
	cd ircfmt && go test . && go vet .
	cd ircmsg && go test . && go vet .
	cd ircreader && go test . && go vet .
	cd ircutils && go test . && go vet .
	$(info Note: ircevent must be tested separately)
	./.check-gofmt.sh

# ircevent requires a local ircd for testing, plus some env vars:
# IRCEVENT_SASL_LOGIN and IRCEVENT_SASL_PASSWORD
ircevent:
	cd ircevent && go test . && go vet .
	./.check-gofmt.sh

gofmt:
	./.check-gofmt.sh --fix

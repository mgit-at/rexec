bin:
	go build -tags netgo

all: bin
	cd rexec_server && go build -tags netgo

update:
	glide update -u -s -v --cache

.PHONY: bin all update

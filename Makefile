bin:
	go build

all: bin
	cd rexec_server && go build

update:
	glide update -u -s -v --cache

.PHONY: bin all update

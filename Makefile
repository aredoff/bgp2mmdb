.PHONY: build test clean

build:
	go build -o bgp2mmdb ./cmd/bgp2mmdb

test: build
	./bgp2mmdb -lookup 8.8.8.8 -mmdb asn.mmdb

convert: build
	./bgp2mmdb -input bview.20250711.0800.gz -output asn.mmdb

clean:
	rm -f bgp2mmdb *.mmdb bview.*.gz

install-deps:
	go mod download

all: install-deps build

# Download and convert with default settings
download: build
	./bgp2mmdb -output asn.mmdb 
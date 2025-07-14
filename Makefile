.PHONY: build test clean download

build:
	go build -o bgp2mmdb ./cmd/bgp2mmdb

test-build:
	cd cmd/test && go build -o ../../test-mmdb .

download: build
	./bgp2mmdb -download -output asn.mmdb

convert: build
	./bgp2mmdb -input bview.20250711.0800.gz -output asn.mmdb

test: test-build
	./test-mmdb

clean:
	rm -f bgp2mmdb test-mmdb *.mmdb bview.*.gz

install-deps:
	go mod download

all: install-deps build test-build

# Download and convert with custom RRC
convert-rrc01: build
	./bgp2mmdb -download -rrc rrc01 -output asn.mmdb

# Keep downloaded file
download-keep: build
	./bgp2mmdb -download -keep -output asn.mmdb 
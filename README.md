# BGP to MMDB Converter

High-performance utility for converting BGP RIB files in MRT format to MMDB databases for fast ASN lookup by IP address.

## Features

- 🚀 **Auto-download** - downloads latest BGP data from RIPE
- 📦 **Compressed files support** (.gz)
- 💾 **Efficient memory usage** (< 2GB)
- ⚡ **Fast conversion** of large files
- 🔄 **Streaming parser** to minimize RAM usage
- 🎯 **Optimized MMDB** for fast lookups

## Installation

```bash
git clone https://github.com/aredoff/bgp2mmdb.git
cd bgp2mmdb
make install-deps
make build
```

## Usage

### Auto-download and convert (recommended)

```bash
# Download latest BGP data from RIPE and convert
./bgp2mmdb -download -output asn.mmdb

# Use multiple RRC collectors for better coverage (recommended)
./bgp2mmdb -download -multi -output asn.mmdb

# Use custom list of RRC collectors
./bgp2mmdb -download -multi -rrcs "rrc00,rrc01,rrc05,rrc10" -output asn.mmdb

# Use different RRC collector
./bgp2mmdb -download -rrc rrc01 -output asn.mmdb

# Keep downloaded file
./bgp2mmdb -download -keep -output asn.mmdb
```

### Convert local file

```bash
./bgp2mmdb -input bview.20250711.0800.gz -output asn.mmdb
```

### Parameters

- `-download` - download latest BGP data from RIPE
- `-input` - path to local MRT file (.gz supported)
- `-output` - path for creating MMDB file
- `-rrc` - RRC collector (rrc00, rrc01, ..., rrc25, default rrc00)
- `-multi` - use multiple RRC collectors (rrc00-rrc05) for better coverage
- `-keep` - keep downloaded file after conversion
- `-mem` - memory limit in MB (default 2048)

## Makefile commands

```bash
# Download and convert (single RRC)
make download

# Download from multiple RRC (best coverage)
make download-multi

# Convert local file
make convert

# Use RRC01
make convert-rrc01

# Download with file keeping
make download-keep

# Test MMDB
make test

# Clean up
make clean
```

## Data sources

Automatically downloads BGP RIB data from:
- **RIPE NCC**: https://data.ris.ripe.net/
- **RRC collectors**: rrc00 (Amsterdam) by default, or multiple (rrc00-rrc05) with `-multi`
- **Format**: bview.YYYYMMDD.HHMM.gz
- **Times**: 16:00, 08:00, 12:00, 00:00 (in priority order)
- **Multi-RRC**: Better prefix coverage by combining data from multiple collectors

## Output format

MMDB contains for each IP:
- `asn` - autonomous system number
- `organization` - organization name (AS{number})
- `network` - network prefix

## Performance

- **Memory**: < 2GB during conversion
- **Speed**: ~500K records/sec
- **MMDB size**: ~30MB for full BGP table
- **Time**: ~2 minutes for full conversion

## Usage example

```bash
# Download and convert from multiple RRC (best coverage)
./bgp2mmdb -download -multi -output asn.mmdb

# Download and convert from single RRC
./bgp2mmdb -download -output asn.mmdb

# Test result
go run cmd/test/main.go -mmdb asn.mmdb 8.8.8.8
```

Result:
```
IP: 8.8.8.8 | ASN: 3356 | Network: 8.0.0.0/12
```

## Usage as library

```go
package main

import (
    "github.com/aredoff/bgp2mmdb"
)

func main() {
    converter := bgp2mmdb.NewConverter(2048) // 2GB limit
    err := converter.Convert("input.gz", "output.mmdb")
    if err != nil {
        panic(err)
    }
}
```

## License

MIT License 
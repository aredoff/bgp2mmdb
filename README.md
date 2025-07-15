# BGP to MMDB Converter

High-performance utility for converting BGP RIB files in MRT format to MMDB databases for fast ASN lookup by IPv4 and IPv6 addresses.

## Features

- 🚀 **Auto-download** - automatic BGP data download from all RIPE RRC collectors
- 🌐 **Universal input** - files and URLs in one list
- 📦 **Compressed files support** (.gz)
- 🌍 **IPv4 and IPv6 support** - full dual-stack coverage
- 💾 **Efficient memory usage** (< 2GB)
- ⚡ **Fast conversion** of large files
- 🔄 **Streaming parser** to minimize RAM usage
- 🎯 **Optimized MMDB** for fast lookups

## Installation

### Download from releases (recommended)

```bash
# Linux x86-64
wget https://github.com/aredoff/bgp2mmdb/releases/latest/download/bgp2mmdb-linux-amd64
chmod +x bgp2mmdb-linux-amd64
./bgp2mmdb-linux-amd64 -output asn.mmdb

# Linux ARM64
wget https://github.com/aredoff/bgp2mmdb/releases/latest/download/bgp2mmdb-linux-arm64
chmod +x bgp2mmdb-linux-arm64
./bgp2mmdb-linux-arm64 -output asn.mmdb

# Windows x86-64
# Download from: https://github.com/aredoff/bgp2mmdb/releases/latest/download/bgp2mmdb-windows-amd64.exe
# Then run: bgp2mmdb-windows-amd64.exe -output asn.mmdb
```

### Build from source

```bash
git clone https://github.com/aredoff/bgp2mmdb.git
cd bgp2mmdb
make install-deps
make build
```

## Usage

### Auto-download from RIPE (recommended)

```bash
# Download latest BGP views from all RIPE RRC collectors
./bgp2mmdb -input ripe -output asn.mmdb

# Same as above (default behavior)
./bgp2mmdb -output asn.mmdb
```

When using `-input ripe` or running without `-input`, the tool automatically downloads from these URLs:
```
http://data.ris.ripe.net/rrc00/latest-bview.gz  # Amsterdam, NL
http://data.ris.ripe.net/rrc01/latest-bview.gz  # London, UK
http://data.ris.ripe.net/rrc03/latest-bview.gz  # Amsterdam, NL
http://data.ris.ripe.net/rrc04/latest-bview.gz  # Geneva, CH
http://data.ris.ripe.net/rrc05/latest-bview.gz  # Vienna, AT
http://data.ris.ripe.net/rrc06/latest-bview.gz  # Otemachi, JP
http://data.ris.ripe.net/rrc07/latest-bview.gz  # Stockholm, SE
http://data.ris.ripe.net/rrc10/latest-bview.gz  # Milan, IT
http://data.ris.ripe.net/rrc11/latest-bview.gz  # New York, US
http://data.ris.ripe.net/rrc12/latest-bview.gz  # Frankfurt, DE
http://data.ris.ripe.net/rrc13/latest-bview.gz  # Moscow, RU
http://data.ris.ripe.net/rrc14/latest-bview.gz  # Palo Alto, US
http://data.ris.ripe.net/rrc15/latest-bview.gz  # São Paulo, BR
http://data.ris.ripe.net/rrc16/latest-bview.gz  # Miami, US
http://data.ris.ripe.net/rrc18/latest-bview.gz  # Barcelona, ES
http://data.ris.ripe.net/rrc19/latest-bview.gz  # Johannesburg, ZA
http://data.ris.ripe.net/rrc20/latest-bview.gz  # Zürich, CH
http://data.ris.ripe.net/rrc21/latest-bview.gz  # Paris, FR
http://data.ris.ripe.net/rrc22/latest-bview.gz  # Bucharest, RO
http://data.ris.ripe.net/rrc23/latest-bview.gz  # Singapore, SG
http://data.ris.ripe.net/rrc24/latest-bview.gz  # Montreal, CA
http://data.ris.ripe.net/rrc26/latest-bview.gz  # Dubai, AE
```

This provides comprehensive global BGP view coverage from 22 RIPE RRC collectors worldwide.

### Local files

```bash
# Single file
./bgp2mmdb -input bview.20250714.0800.gz -output asn.mmdb

# Multiple files
./bgp2mmdb -input file1.gz,file2.gz,file3.gz -output asn.mmdb
```

### Download URLs

```bash
# Single URL
./bgp2mmdb -input https://data.ris.ripe.net/rrc00/2025.01/bview.20250114.0800.gz -output asn.mmdb

# Multiple URLs
./bgp2mmdb -input "https://data.ris.ripe.net/rrc00/2025.01/bview.20250114.0800.gz,https://data.ris.ripe.net/rrc01/2025.01/bview.20250114.0800.gz" -output asn.mmdb
```

### Mixed mode

```bash
# Files and URLs together
./bgp2mmdb -input "local_file.gz,https://example.com/remote_file.gz,another_local.gz" -output asn.mmdb
```

### IP Lookup

```bash
# Lookup IPv4 in existing MMDB
./bgp2mmdb -lookup 8.8.8.8 -mmdb asn.mmdb

# Lookup IPv6 in existing MMDB  
./bgp2mmdb -lookup 2001:4860:4860::8888 -mmdb asn.mmdb

# Use default MMDB file
./bgp2mmdb -lookup 1.1.1.1
```

### Parameters

**Conversion mode:**
- `-input` - comma-separated list of files and/or URLs, or "ripe" to auto-download from all RIPE RRC collectors (default: ripe)
- `-output` - path for creating MMDB file (default: asn.mmdb)

**Lookup mode:**
- `-lookup` - IP address to lookup in existing MMDB file
- `-mmdb` - MMDB file path for lookup mode (default: asn.mmdb)

## Makefile commands

```bash
# Build
make build

# Convert local file
make convert

# Test MMDB
make test

# Clean up
make clean
```

## Input format

Supports BGP routing table data in the following formats:

**MRT (Multi-threaded Routing Toolkit) format:**
- BGP RIB (Routing Information Base) dumps
- TABLE_DUMP_V2 message types
- Peer Index Table with BGP speaker information
- RIB entries with IPv4 and IPv6 prefix announcements and AS_PATH attributes
- Both compressed (.gz) and uncompressed files

**Supported MRT subtypes:**
- `PEER_INDEX_TABLE` - BGP peer information
- `RIB_IPV4_UNICAST` - IPv4 unicast routing entries
- `RIB_IPV6_UNICAST` - IPv6 unicast routing entries
- BGP attributes: ORIGIN, AS_PATH, NEXT_HOP

## Output format

MMDB contains for each IP:
- `asn` - autonomous system number
- `network` - network prefix (e.g., 8.8.8.0/24 or 2001:db8::/32)

## Performance

- **Memory**: < 2GB during conversion
- **Speed**: ~500K records/sec
- **MMDB size**: ~30-50MB for full BGP table
- **Time**: ~2-3 minutes for full conversion

## Usage example

```bash
# Auto-download from all RIPE RRC collectors (recommended)
./bgp2mmdb -input ripe -output asn.mmdb

# Or simply (same as above)
./bgp2mmdb -output asn.mmdb

# Test result
./bgp2mmdb -lookup 8.8.8.8 -mmdb asn.mmdb
```

Result:
```
IP: 8.8.8.8 | ASN: 15169 | Network: 8.8.8.0/24
```

## Usage as library

```go
package main

import (
    "github.com/aredoff/bgp2mmdb"
)

func main() {
    converter := bgp2mmdb.NewConverter()
    
    // Process multiple files
    converter.ProcessFile("file1.gz")
    converter.ProcessFile("file2.gz")
    
    // Write MMDB
    err := converter.WriteMMDB("output.mmdb")
    if err != nil {
        panic(err)
    }
}
```

## License

MIT License 
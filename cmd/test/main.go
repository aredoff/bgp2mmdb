package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/aredoff/bgp2mmdb"
	"github.com/oschwald/maxminddb-golang"
)

func main() {
	var mmdbPath = flag.String("mmdb", "", "Path to MMDB file")
	flag.Parse()

	args := flag.Args()
	if *mmdbPath == "" || len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s -mmdb <path-to-mmdb> <ip-address>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	ipAddr := args[0]

	db, err := maxminddb.Open(*mmdbPath)
	if err != nil {
		fmt.Printf("Error opening MMDB: %v\n", err)
		return
	}
	defer db.Close()

	ip := net.ParseIP(ipAddr)
	if ip == nil {
		fmt.Printf("Invalid IP: %s\n", ipAddr)
		return
	}

	var record bgp2mmdb.ASNRecord
	err = db.Lookup(ip, &record)
	if err != nil {
		fmt.Printf("Lookup failed for %s: %v\n", ipAddr, err)
		return
	}

	fmt.Printf("IP: %s | ASN: %d | Network: %s\n", ipAddr, record.ASN, record.Network)
}

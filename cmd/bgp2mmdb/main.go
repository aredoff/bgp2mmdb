package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/aredoff/bgp2mmdb"
	"github.com/oschwald/maxminddb-golang"
)

var (
	defaultRRC = []string{
		"rrc00",
		"rrc01",
		"rrc03",
		"rrc04",
		"rrc05",
		"rrc06",
		"rrc07",
		"rrc10",
		"rrc11",
		"rrc12",
		"rrc13",
		"rrc14",
		"rrc15",
		"rrc16",
		"rrc18",
		"rrc19",
		"rrc20",
		"rrc21",
		"rrc22",
		"rrc23",
	}
)

func main() {
	var (
		inputList  = flag.String("input", "ripe", "Comma-separated list of input files or URLs, if 'ripe', will download the latest bview's from RIPE")
		outputFile = flag.String("output", "asn.mmdb", "Output MMDB file")
		memLimit   = flag.Int("mem", 2048, "Memory limit in MB")
		lookupIP   = flag.String("lookup", "", "IP address to lookup in existing MMDB file")
		mmdbFile   = flag.String("mmdb", "asn.mmdb", "MMDB file path for lookup mode")
	)
	flag.Parse()

	// Lookup mode
	if *lookupIP != "" {
		lookupMode(*mmdbFile, *lookupIP)
		return
	}

	// Convert mode
	if *inputList == "" {
		flag.Usage()
		os.Exit(1)
	}

	var inputs []string
	if *inputList == "ripe" {
		for _, rrc := range defaultRRC {
			inputs = append(inputs, fmt.Sprintf("http://data.ris.ripe.net/%s/latest-bview.gz", rrc))
		}
	} else {
		inputs = strings.Split(*inputList, ",")
		for i := range inputs {
			inputs[i] = strings.TrimSpace(inputs[i])
		}
	}

	if len(inputs) == 0 {
		log.Fatal("No input files or URLs provided")
	}

	start := time.Now()
	conv := bgp2mmdb.NewConverter(*memLimit)

	// Process each input
	for i, input := range inputs {
		if input == "" {
			continue
		}

		if isURL(input) {
			fmt.Printf("Downloading: %s\n", input)
			fileName, err := downloadFile(input, i)
			if err != nil {
				log.Printf("Warning: Failed to download %s: %v", input, err)
				continue
			}
			fmt.Printf("Processing: %s\n", fileName)
			err = conv.ProcessFile(fileName)
			if err != nil {
				log.Printf("Warning: Failed to process %s: %v", fileName, err)
			}
			// Cleanup downloaded file immediately after processing
			os.Remove(fileName)
			fmt.Printf("Cleaned up: %s\n", fileName)
		} else {
			fmt.Printf("Processing: %s\n", input)
			err := conv.ProcessFile(input)
			if err != nil {
				log.Printf("Warning: Failed to process %s: %v", input, err)
			}
		}
	}

	// Write MMDB
	err := conv.WriteMMDB(*outputFile)
	if err != nil {
		log.Fatalf("Failed to write MMDB: %v", err)
	}

	elapsed := time.Since(start)
	fmt.Printf("Conversion completed in %v\n", elapsed)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Memory used: %.2f MB\n", float64(m.Alloc)/1024/1024)

	if fileInfo, err := os.Stat(*outputFile); err == nil {
		fmt.Printf("Output file size: %.2f MB\n", float64(fileInfo.Size())/1024/1024)
	}
}

func lookupMode(mmdbPath, ipAddr string) {
	db, err := maxminddb.Open(mmdbPath)
	if err != nil {
		log.Fatalf("Error opening MMDB: %v", err)
	}
	defer db.Close()

	ip := net.ParseIP(ipAddr)
	if ip == nil {
		log.Fatalf("Invalid IP: %s", ipAddr)
	}

	var record bgp2mmdb.ASNRecord
	err = db.Lookup(ip, &record)
	if err != nil {
		log.Fatalf("Lookup failed for %s: %v", ipAddr, err)
	}

	if record.ASN == 0 {
		fmt.Printf("IP: %s | No data found\n", ipAddr)
	} else {
		fmt.Printf("IP: %s | ASN: %d | Organization: %s | Network: %s\n",
			ipAddr, record.ASN, record.Organization, record.Network)
	}
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func downloadFile(url string, index int) (string, error) {
	// Extract filename from URL and make it unique
	parts := strings.Split(url, "/")
	baseName := parts[len(parts)-1]
	if baseName == "" {
		baseName = "downloaded.gz"
	}

	// Add index to make filename unique
	fileName := fmt.Sprintf("%d_%s", index, baseName)

	// Check if file already exists
	if _, err := os.Stat(fileName); err == nil {
		fmt.Printf("File %s already exists, skipping download\n", fileName)
		return fileName, nil
	}

	// Create HTTP request
	client := &http.Client{
		Timeout: 10 * time.Minute,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Create file
	out, err := os.Create(fileName)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Download with progress
	size := resp.ContentLength
	if size > 0 {
		fmt.Printf("Downloading %s (%.1f MB)...\n", fileName, float64(size)/1024/1024)
	} else {
		fmt.Printf("Downloading %s...\n", fileName)
	}

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(fileName)
		return "", err
	}

	fmt.Printf("Downloaded: %.1f MB\n", float64(written)/1024/1024)
	return fileName, nil
}

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/aredoff/bgp2mmdb"
)

func main() {
	var (
		inputFile    = flag.String("input", "", "Input MRT file (.gz)")
		outputFile   = flag.String("output", "", "Output MMDB file")
		memLimit     = flag.Int("mem", 2048, "Memory limit in MB")
		downloadMode = flag.Bool("download", false, "Download latest BGP view from RIPE")
		ripeRRC      = flag.String("rrc", "rrc00", "RIPE RRC collector (rrc00, rrc01, etc.)")
		multiRRC     = flag.Bool("multi", false, "Use multiple RRC collectors (rrc00-rrc04)")
		rrcList      = flag.String("rrcs", "rrc00,rrc03,rrc25", "Comma-separated list of RRC collectors for multi mode")
		keepFile     = flag.Bool("keep", false, "Keep downloaded file after conversion")
	)
	flag.Parse()

	if *downloadMode {
		fmt.Println("Downloading latest BGP view from RIPE...")

		if *multiRRC {
			rrcs := strings.Split(*rrcList, ",")
			for i := range rrcs {
				rrcs[i] = strings.TrimSpace(rrcs[i])
			}

			downloadedFiles, err := downloadMultipleRRC(rrcs)
			if err != nil {
				log.Fatalf("Failed to download BGP views: %v", err)
			}

			// Process multiple files
			conv := bgp2mmdb.NewConverter(*memLimit)
			for _, file := range downloadedFiles {
				fmt.Printf("Processing: %s\n", file)
				err := conv.ProcessFile(file)
				if err != nil {
					log.Printf("Warning: Failed to process %s: %v", file, err)
				}

				if !*keepFile {
					os.Remove(file)
					fmt.Printf("Cleaned up: %s\n", file)
				}
			}

			start := time.Now()
			err = conv.WriteMMDB(*outputFile)
			if err != nil {
				log.Fatalf("Failed to write MMDB: %v", err)
			}

			elapsed := time.Since(start)
			fmt.Printf("Multi-RRC conversion completed in %v\n", elapsed)

			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			fmt.Printf("Memory used: %.2f MB\n", float64(m.Alloc)/1024/1024)

			if fileInfo, err := os.Stat(*outputFile); err == nil {
				fmt.Printf("Output file size: %.2f MB\n", float64(fileInfo.Size())/1024/1024)
			}
			return
		} else {
			downloadedFile, err := downloadLatestBGPView(*ripeRRC)
			if err != nil {
				log.Fatalf("Failed to download BGP view: %v", err)
			}
			*inputFile = downloadedFile
			fmt.Printf("Downloaded: %s\n", downloadedFile)

			if !*keepFile {
				defer func() {
					os.Remove(downloadedFile)
					fmt.Printf("Cleaned up: %s\n", downloadedFile)
				}()
			}
		}
	}

	if *inputFile == "" || *outputFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	start := time.Now()

	conv := bgp2mmdb.NewConverter(*memLimit)
	err := conv.Convert(*inputFile, *outputFile)
	if err != nil {
		log.Fatalf("Conversion failed: %v", err)
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

func downloadLatestBGPView(rrc string) (string, error) {
	now := time.Now()

	// Try the last few days as data may not be available every day
	for days := 0; days < 7; days++ {
		date := now.AddDate(0, 0, -days)

		// Try different times (16:00, 08:00, 12:00, 00:00)
		times := []string{"1600", "0800", "1200", "0000"}

		for _, timeStr := range times {
			url := fmt.Sprintf("https://data.ris.ripe.net/%s/%04d.%02d/bview.%04d%02d%02d.%s.gz",
				rrc, date.Year(), date.Month(), date.Year(), date.Month(), date.Day(), timeStr)

			fileName := fmt.Sprintf("%s_bview.%04d%02d%02d.%s.gz",
				rrc, date.Year(), date.Month(), date.Day(), timeStr)

			fmt.Printf("Trying: %s\n", url)

			err := downloadFile(url, fileName)
			if err == nil {
				return fileName, nil
			}

			fmt.Printf("Failed: %v\n", err)
		}
	}

	return "", fmt.Errorf("no BGP view files found for the last 7 days")
}

func downloadMultipleRRC(rrcs []string) ([]string, error) {
	var downloadedFiles []string

	for _, rrc := range rrcs {
		fmt.Printf("Trying RRC: %s\n", rrc)
		file, err := downloadLatestBGPView(rrc)
		if err != nil {
			fmt.Printf("Failed to download from %s: %v\n", rrc, err)
			continue
		}
		downloadedFiles = append(downloadedFiles, file)
		fmt.Printf("Successfully downloaded from %s\n", rrc)
	}

	if len(downloadedFiles) == 0 {
		return nil, fmt.Errorf("failed to download from any RRC collectors")
	}

	fmt.Printf("Downloaded %d files from %d RRC collectors\n", len(downloadedFiles), len(rrcs))
	return downloadedFiles, nil
}

func downloadFile(url, fileName string) error {
	// Check if file already exists
	if _, err := os.Stat(fileName); err == nil {
		fmt.Printf("File %s already exists, skipping download\n", fileName)
		return nil
	}

	// Create HTTP request
	client := &http.Client{
		Timeout: 10 * time.Minute,
	}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Create file
	out, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer out.Close()

	// Download with progress
	size := resp.ContentLength
	fmt.Printf("Downloading %s (%.1f MB)...\n", fileName, float64(size)/1024/1024)

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(fileName)
		return err
	}

	fmt.Printf("Downloaded: %.1f MB\n", float64(written)/1024/1024)
	return nil
}

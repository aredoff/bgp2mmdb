package bgp2mmdb

import (
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type Converter struct {
	memLimitBytes int64
	prefixMap     map[string]*PrefixInfo
	batchSize     int
}

type PrefixInfo struct {
	ASN    uint32
	Prefix string
	ASPath []uint32
}

func NewConverter(memLimitMB int) *Converter {
	return &Converter{
		memLimitBytes: int64(memLimitMB) * 1024 * 1024,
		prefixMap:     make(map[string]*PrefixInfo),
		batchSize:     10000,
	}
}

func (c *Converter) Convert(inputFile, outputFile string) error {
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	var reader *gzip.Reader
	if filepath.Ext(inputFile) == ".gz" {
		reader, err = gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer reader.Close()
	}

	parser := NewMRTParser()
	if reader != nil {
		err = parser.Parse(reader, c.processMRTEntry)
	} else {
		err = parser.Parse(file, c.processMRTEntry)
	}

	if err != nil {
		return fmt.Errorf("failed to parse MRT file: %w", err)
	}

	return CreateMMDB(c.prefixMap, outputFile)
}

func (c *Converter) ProcessFile(inputFile string) error {
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	var reader *gzip.Reader
	if filepath.Ext(inputFile) == ".gz" {
		reader, err = gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer reader.Close()
	}

	parser := NewMRTParser()
	if reader != nil {
		err = parser.Parse(reader, c.processMRTEntry)
	} else {
		err = parser.Parse(file, c.processMRTEntry)
	}

	if err != nil {
		return fmt.Errorf("failed to parse MRT file: %w", err)
	}

	return nil
}

func (c *Converter) WriteMMDB(outputFile string) error {
	return CreateMMDB(c.prefixMap, outputFile)
}

func (c *Converter) processMRTEntry(entry interface{}) error {
	switch e := entry.(type) {
	case *PeerIndexTable:
		fmt.Printf("Processed Peer Index Table with %d peers\n", len(e.Peers))
	case *RIBEntry:
		for _, ribEntry := range e.Entries {
			if ribEntry.Prefix != "" && ribEntry.ASN != 0 {
				// Skip default route - it interferes with proper lookup
				if ribEntry.Prefix == "0.0.0.0/0" {
					continue
				}

				// Select best route by AS_PATH length (shorter = better)
				if existing, exists := c.prefixMap[ribEntry.Prefix]; exists {
					// If new route has shorter AS_PATH, replace it
					if len(ribEntry.ASPath) < len(existing.ASPath) {
						c.prefixMap[ribEntry.Prefix] = &PrefixInfo{
							ASN:    ribEntry.ASN,
							Prefix: ribEntry.Prefix,
							ASPath: ribEntry.ASPath,
						}
					}
				} else {
					c.prefixMap[ribEntry.Prefix] = &PrefixInfo{
						ASN:    ribEntry.ASN,
						Prefix: ribEntry.Prefix,
						ASPath: ribEntry.ASPath,
					}
				}
			}
		}

		if len(c.prefixMap)%10000 == 0 && len(c.prefixMap) > 0 {
			fmt.Printf("Processed %d prefixes...\n", len(c.prefixMap))
		}

		if len(c.prefixMap) >= c.batchSize {
			return c.flushBatch()
		}
	}
	return nil
}

func (c *Converter) flushBatch() error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	if int64(m.Alloc) > c.memLimitBytes*8/10 {
		runtime.GC()
		runtime.ReadMemStats(&m)
		if int64(m.Alloc) > c.memLimitBytes*9/10 {
			return fmt.Errorf("memory limit exceeded")
		}
	}
	return nil
}

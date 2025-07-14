package bgp2mmdb

import (
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultPrefixStartSize = 2000000
)

type Converter struct {
	prefixMap map[string]*PrefixInfo
}

type PrefixInfo struct {
	ASN    uint32
	Prefix string
	ASPath []uint32
}

func NewConverter() *Converter {
	return &Converter{
		prefixMap: make(map[string]*PrefixInfo, defaultPrefixStartSize),
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
				// Skip default routes - they interfere with proper make
				if ribEntry.Prefix == "0.0.0.0/0" || ribEntry.Prefix == "::/0" {
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
	}
	return nil
}

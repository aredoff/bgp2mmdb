package bgp2mmdb

import (
	"fmt"
	"net"
	"os"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

type ASNRecord struct {
	ASN          uint32 `maxminddb:"asn"`
	Organization string `maxminddb:"organization"`
	Network      string `maxminddb:"network"`
}

func CreateMMDB(prefixMap map[string]*PrefixInfo, outputFile string) error {
	writer, err := mmdbwriter.New(
		mmdbwriter.Options{
			DatabaseType: "ASN-DB",
			Description: map[string]string{
				"en": "ASN database generated from BGP RIB data",
			},
			RecordSize: 28,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create MMDB writer: %w", err)
	}

	recordCount := 0
	for prefix, info := range prefixMap {
		_, network, err := net.ParseCIDR(prefix)
		if err != nil {
			continue
		}

		record := mmdbtype.Map{
			"asn":          mmdbtype.Uint32(info.ASN),
			"organization": mmdbtype.String(fmt.Sprintf("AS%d", info.ASN)),
			"network":      mmdbtype.String(prefix),
		}

		err = writer.Insert(network, record)
		if err != nil {
			continue
		}

		recordCount++
		if recordCount%10000 == 0 {
			fmt.Printf("Processed %d records\n", recordCount)
		}
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	_, err = writer.WriteTo(file)
	if err != nil {
		return fmt.Errorf("failed to write MMDB: %w", err)
	}

	fmt.Printf("Successfully created MMDB with %d records\n", recordCount)
	return nil
}

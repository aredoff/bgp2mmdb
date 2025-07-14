package bgp2mmdb

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

const (
	MRTHeaderLength = 12

	MRTTypeTABLEDUMPV2       = 13
	MRTSubtypePEERINDEXTABLE = 1
	MRTSubtypeRIBIPV4UNICAST = 2
	MRTSubtypeRIBIPV6UNICAST = 4

	BGPAttrOrigin  = 1
	BGPAttrASPath  = 2
	BGPAttrNextHop = 3
)

type MRTHeader struct {
	Timestamp uint32
	Type      uint16
	Subtype   uint16
	Length    uint32
}

type PeerIndexTable struct {
	CollectorIP net.IP
	ViewName    string
	Peers       []PeerEntry
}

type PeerEntry struct {
	Type  uint8
	BGPID net.IP
	IP    net.IP
	ASN   uint32
}

type RIBEntry struct {
	SeqNo   uint32
	Prefix  net.IPNet
	Entries []RIBSubEntry
}

type RIBSubEntry struct {
	PeerIndex uint16
	Timestamp uint32
	ASPath    []uint32
	ASN       uint32
	Prefix    string
}

type MRTParser struct {
	peerTable *PeerIndexTable
}

func NewMRTParser() *MRTParser {
	return &MRTParser{}
}

func (p *MRTParser) Parse(reader io.Reader, callback func(interface{}) error) error {
	for {
		header, err := p.readMRTHeader(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read MRT header: %w", err)
		}

		data := make([]byte, header.Length)
		_, err = io.ReadFull(reader, data)
		if err != nil {
			return fmt.Errorf("failed to read MRT data: %w", err)
		}

		entry, err := p.parseMRTEntry(header, data)
		if err != nil {
			continue
		}

		if entry != nil {
			err = callback(entry)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *MRTParser) readMRTHeader(reader io.Reader) (*MRTHeader, error) {
	var header MRTHeader
	err := binary.Read(reader, binary.BigEndian, &header)
	return &header, err
}

func (p *MRTParser) parseMRTEntry(header *MRTHeader, data []byte) (interface{}, error) {
	if header.Type != MRTTypeTABLEDUMPV2 {
		return nil, nil
	}

	switch header.Subtype {
	case MRTSubtypePEERINDEXTABLE:
		entry, err := p.parsePeerIndexTable(data)
		if err == nil {
			p.peerTable = entry
		}
		return entry, err
	case MRTSubtypeRIBIPV4UNICAST:
		return p.parseRIBEntry(data, false) // IPv4
	case MRTSubtypeRIBIPV6UNICAST:
		return p.parseRIBEntry(data, true) // IPv6
	}

	return nil, nil
}

func (p *MRTParser) parsePeerIndexTable(data []byte) (*PeerIndexTable, error) {
	if len(data) < 6 {
		return nil, fmt.Errorf("insufficient data for peer index table")
	}

	table := &PeerIndexTable{}
	offset := 0

	table.CollectorIP = net.IP(data[offset : offset+4])
	offset += 4

	viewNameLen := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2

	if len(data) < offset+int(viewNameLen)+2 {
		return nil, fmt.Errorf("insufficient data for view name")
	}

	if viewNameLen > 0 {
		table.ViewName = string(data[offset : offset+int(viewNameLen)])
		offset += int(viewNameLen)
	}

	peerCount := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2

	table.Peers = make([]PeerEntry, peerCount)

	for i := uint16(0); i < peerCount; i++ {
		if len(data) < offset+1 {
			return nil, fmt.Errorf("insufficient data for peer entry")
		}

		peer := &table.Peers[i]
		peer.Type = data[offset]
		offset++

		ipLen := 4
		asnLen := 2
		if peer.Type&0x01 != 0 {
			ipLen = 16
		}
		if peer.Type&0x02 != 0 {
			asnLen = 4
		}

		if len(data) < offset+4+ipLen+asnLen {
			return nil, fmt.Errorf("insufficient data for peer entry")
		}

		peer.BGPID = net.IP(data[offset : offset+4])
		offset += 4

		peer.IP = net.IP(data[offset : offset+ipLen])
		offset += ipLen

		if asnLen == 4 {
			peer.ASN = binary.BigEndian.Uint32(data[offset : offset+4])
		} else {
			peer.ASN = uint32(binary.BigEndian.Uint16(data[offset : offset+2]))
		}
		offset += asnLen
	}

	return table, nil
}

func (p *MRTParser) parseRIBEntry(data []byte, isIPv6 bool) (*RIBEntry, error) {
	if len(data) < 7 {
		return nil, fmt.Errorf("insufficient data for RIB entry")
	}

	entry := &RIBEntry{}
	offset := 0

	entry.SeqNo = binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4

	prefixLen := data[offset]
	offset++

	var prefixBytes int
	var prefixIP net.IP
	var maskBits int
	var prefixStr string

	if isIPv6 {
		// IPv6
		maskBits = 128
		prefixBytes = int((prefixLen + 7) / 8)
		if prefixBytes > 16 {
			prefixBytes = 16
		}

		if len(data) < offset+prefixBytes+2 {
			return nil, fmt.Errorf("insufficient data for IPv6 prefix")
		}

		prefixData := make([]byte, 16)
		if prefixBytes > 0 {
			copy(prefixData, data[offset:offset+prefixBytes])
		}
		prefixIP = net.IP(prefixData)

		if prefixLen > 0 {
			prefixStr = fmt.Sprintf("%s/%d", prefixIP.String(), prefixLen)
		} else {
			prefixStr = "::/0"
		}
	} else {
		// IPv4
		maskBits = 32
		prefixBytes = int((prefixLen + 7) / 8)
		if prefixBytes > 4 {
			prefixBytes = 4
		}

		if len(data) < offset+prefixBytes+2 {
			return nil, fmt.Errorf("insufficient data for IPv4 prefix")
		}

		prefixData := make([]byte, 4)
		if prefixBytes > 0 {
			copy(prefixData, data[offset:offset+prefixBytes])
		}
		prefixIP = net.IPv4(prefixData[0], prefixData[1], prefixData[2], prefixData[3])

		if prefixLen > 0 {
			prefixStr = fmt.Sprintf("%s/%d", prefixIP.String(), prefixLen)
		} else {
			prefixStr = "0.0.0.0/0"
		}
	}

	entry.Prefix = net.IPNet{
		IP:   prefixIP,
		Mask: net.CIDRMask(int(prefixLen), maskBits),
	}

	offset += prefixBytes

	entryCount := binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2

	entry.Entries = make([]RIBSubEntry, entryCount)

	for i := uint16(0); i < entryCount; i++ {
		if len(data) < offset+8 {
			return nil, fmt.Errorf("insufficient data for RIB sub entry")
		}

		subEntry := &entry.Entries[i]
		subEntry.PeerIndex = binary.BigEndian.Uint16(data[offset : offset+2])
		offset += 2

		subEntry.Timestamp = binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4

		attrLen := binary.BigEndian.Uint16(data[offset : offset+2])
		offset += 2

		if len(data) < offset+int(attrLen) {
			return nil, fmt.Errorf("insufficient data for BGP attributes")
		}

		attrs, err := p.parseBGPAttributes(data[offset : offset+int(attrLen)])
		if err == nil {
			subEntry.ASPath = attrs.ASPath
			if len(attrs.ASPath) > 0 {
				subEntry.ASN = attrs.ASPath[len(attrs.ASPath)-1]
			}
			subEntry.Prefix = prefixStr
		}
		offset += int(attrLen)
	}

	return entry, nil
}

type BGPAttributes struct {
	Origin  uint8
	ASPath  []uint32
	NextHop net.IP
}

func (p *MRTParser) parseBGPAttributes(data []byte) (*BGPAttributes, error) {
	attrs := &BGPAttributes{}
	offset := 0

	for offset < len(data) {
		if len(data) < offset+3 {
			break
		}

		flags := data[offset]
		attrType := data[offset+1]
		offset += 2

		var attrLen int
		if flags&0x10 != 0 {
			if len(data) < offset+2 {
				break
			}
			attrLen = int(binary.BigEndian.Uint16(data[offset : offset+2]))
			offset += 2
		} else {
			if len(data) < offset+1 {
				break
			}
			attrLen = int(data[offset])
			offset++
		}

		if len(data) < offset+attrLen {
			break
		}

		attrData := data[offset : offset+attrLen]
		offset += attrLen

		switch attrType {
		case BGPAttrASPath:
			attrs.ASPath = p.parseASPath(attrData)
		case BGPAttrOrigin:
			if len(attrData) >= 1 {
				attrs.Origin = attrData[0]
			}
		case BGPAttrNextHop:
			if len(attrData) >= 4 {
				attrs.NextHop = net.IP(attrData[:4])
			}
		}
	}

	return attrs, nil
}

func (p *MRTParser) parseASPath(data []byte) []uint32 {
	var asPath []uint32
	offset := 0

	for offset < len(data) {
		if len(data) < offset+2 {
			break
		}

		_ = data[offset] // segType
		segLen := data[offset+1]
		offset += 2

		for i := uint8(0); i < segLen; i++ {
			if len(data) < offset+4 {
				break
			}
			asn := binary.BigEndian.Uint32(data[offset : offset+4])
			asPath = append(asPath, asn)
			offset += 4
		}
	}

	return asPath
}

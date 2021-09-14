// Package rdr implements Realtime Data Retrieval (RDR) protocol.
// https://redmine.named-data.net/projects/ndn-tlv/wiki/RDR
package rdr

import (
	"encoding"
	"fmt"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// KeywordMetadata is the 32=metadata component.
var KeywordMetadata = ndn.MakeNameComponent(an.TtKeywordNameComponent, []byte("metadata"))

// MakeDiscoveryInterest creates an RDR discovery Interest.
// KeywordMetadata is appended automatically if it does not exist.
func MakeDiscoveryInterest(prefix ndn.Name) ndn.Interest {
	if !prefix[len(prefix)-1].Equal(KeywordMetadata) {
		prefix = prefix.Append(KeywordMetadata)
	}
	return ndn.Interest{
		Name:        prefix,
		CanBePrefix: true,
		MustBeFresh: true,
	}
}

// IsDiscoveryInterest determines whether an Interest is an RDR discovery Interest.
func IsDiscoveryInterest(interest ndn.Interest) bool {
	return len(interest.Name) > 1 && interest.Name[len(interest.Name)-1].Equal(KeywordMetadata) &&
		interest.CanBePrefix && interest.MustBeFresh
}

// Metadata contains RDR metadata packet content.
type Metadata struct {
	Name ndn.Name
}

var (
	_ encoding.BinaryMarshaler   = Metadata{}
	_ encoding.BinaryUnmarshaler = (*Metadata)(nil)
)

// MarshalBinary encodes to TLV-VALUE.
func (m Metadata) MarshalBinary() (value []byte, e error) {
	return m.Encode()
}

// Encode encodes to TLV-VALUE with extensions.
func (m Metadata) Encode(extensions ...tlv.Fielder) (value []byte, e error) {
	return tlv.EncodeFrom(append([]tlv.Fielder{m.Name}, extensions...)...)
}

// UnmarshalBinary decodes from TLV-VALUE.
func (m *Metadata) UnmarshalBinary(value []byte) (e error) {
	return m.Decode(value, nil)
}

// Decode decodes from TLV-VALUE with extensions.
func (m *Metadata) Decode(value []byte, extensions MetadataDecoderMap) error {
	*m = Metadata{}
	d := tlv.DecodingBuffer(value)
	hasName := false
	for _, de := range d.Elements() {
		var f MetadataFieldDecoder
		if de.Type == an.TtName && !hasName {
			f = m.decodeName
			hasName = true
		} else if f = extensions[de.Type]; f != nil {
			// use extension decoder for TLV-TYPE
		} else if f = extensions[0]; f != nil {
			// use general extension decoder
		} else {
			// ignore unknown field
			continue
		}

		if e := f(de); e != nil {
			return fmt.Errorf("TLV-TYPE 0x%02x: %w", de.Type, e)
		}
	}
	return d.ErrUnlessEOF()
}

func (m *Metadata) decodeName(de tlv.DecodingElement) error {
	return de.UnmarshalValue(&m.Name)
}

// Metadata extension decoder.
type (
	MetadataFieldDecoder func(de tlv.DecodingElement) error
	MetadataDecoderMap   map[uint32]MetadataFieldDecoder
)
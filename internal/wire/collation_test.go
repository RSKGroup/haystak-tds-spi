package wire

import (
	"bytes"
	"testing"

	"github.com/RSKGroup/haystak-tds-spi/tds/catalog"
	"github.com/RSKGroup/haystak-tds-spi/tds/types"
)

func TestColMetadataStringCollationValid(t *testing.T) {
	md := colMetadata([]catalog.Column{{Name: "name", Type: types.Type{Kind: types.String, MaxLen: 128}}})
	if !bytes.Contains(md, []byte{0x09, 0x04, 0xD0, 0x00, 0x34}) {
		t.Fatalf("colmetadata missing SQL_Latin1_General_CP1_CI_AS collation; got % x", md)
	}
}

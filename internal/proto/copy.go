// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package proto

import (
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

// Copy takes a proto message and destination and makes a safe copy of it
// by marshaling the message to the text format and unmarshaling it on a new
// struct. This allows us to safely send proto through channels without
// copying the mutexes that are present in the proto genereated structs.
func Copy(in proto.Message, out proto.Message) error {
	bytes, err := prototext.Marshal(in)
	if err != nil {
		return err
	}

	return prototext.Unmarshal(bytes, out)
}

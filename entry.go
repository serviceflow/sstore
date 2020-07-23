// Copyright 2020-2026 The sstore Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sstore

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"sync"
)

type entry struct {
	ID   int64
	name string
	data []byte
	cb   func(err error)
}

var entriesPool = sync.Pool{New: func() interface{} {
	return make([]*entry, 0, 64)
}}

func (e *entry) size() int {
	return len(e.data) + len(e.name) + 4 /*len(e.data)*/ + 4 /*len(e.name)*/ + 8 /*ID*/
}

func (e *entry) encode() []byte {
	var buf = make([]byte, e.size()+4)
	writer := bytes.NewBuffer(buf)
	_ = e.write(writer)
	return buf
}

func (e *entry) write(writer io.Writer) error {
	if err := binary.Write(writer, binary.BigEndian, uint32(e.size())); err != nil {
		return errors.WithStack(err)
	}
	if err := binary.Write(writer, binary.BigEndian, uint32(len(e.name))); err != nil {
		return errors.WithStack(err)
	}
	if err := binary.Write(writer, binary.BigEndian, uint32(len(e.data))); err != nil {
		return errors.WithStack(err)
	}
	if err := binary.Write(writer, binary.BigEndian, e.ID); err != nil {
		return errors.WithStack(err)
	}
	if _, err := writer.Write([]byte(e.name)); err != nil {
		return errors.WithStack(err)
	}
	if _, err := writer.Write(e.data); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func decodeEntry(reader io.Reader) (*entry, error) {
	var nameLen uint32
	var dataLen uint32
	var size uint32
	var e = new(entry)
	if err := binary.Read(reader, binary.BigEndian, &size); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.BigEndian, &nameLen); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.BigEndian, &dataLen); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.BigEndian, &e.ID); err != nil {
		return nil, err
	}
	if size != nameLen+dataLen+16 {
		return nil, errors.WithStack(io.ErrUnexpectedEOF)
	}
	var name = make([]byte, nameLen)
	e.data = make([]byte, dataLen)

	n, err := io.ReadFull(reader, name)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	e.name = string(name)
	if n != int(nameLen) {
		return nil, errors.WithStack(io.ErrUnexpectedEOF)
	}
	n, err = io.ReadFull(reader, e.data)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if n != int(dataLen) {
		return nil, errors.WithMessage(io.ErrUnexpectedEOF,
			fmt.Sprintf("n[%d] datalen[%d]", n, dataLen))
	}
	return e, nil
}

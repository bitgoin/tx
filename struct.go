/*
 * Copyright (c) 2016, Shinya Yagyu
 * All rights reserved.
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 * 1. Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 * 3. Neither the name of the copyright holder nor the names of its
 *    contributors may be used to endorse or promote products derived from this
 *    software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
 * LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
 * CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
 * SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
 * INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
 * CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 * POSSIBILITY OF SUCH DAMAGE.
 */

package tx

import (
	"bytes"
	"crypto/sha256"
	"log"

	"github.com/bitgoin/packer"
)

//TxIn is the info of input transaction.
type TxIn struct {
	Hash   []byte `len:"32"`
	Index  uint32
	Script []byte `len:"prefix"`
	Seq    uint32
}

//TxOut is the info of output transaction.
type TxOut struct {
	Value  uint64
	Script []byte `len:"prefix"`
}

//Tx describes a bitcoin transaction,
type Tx struct {
	Version  uint32
	TxIn     []*TxIn  `len:"prefix"`
	TxOut    []*TxOut `len:"prefix"`
	Locktime uint32
}

func hash(b interface{}) []byte {
	var buf bytes.Buffer
	if err := packer.Pack(&buf, b); err != nil {
		log.Fatal(err)
	}
	bs := buf.Bytes()
	h := sha256.Sum256(bs)
	h = sha256.Sum256(h[:])
	return h[:]
}

//Hash returns hash of the tx.
func (t *Tx) Hash() []byte {
	return hash(*t)
}

//Pack packs Tx struct to bin.
func (t *Tx) Pack() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := packer.Pack(buf, *t)
	return buf.Bytes(), err
}

//ParseTX parses byte array and returns Tx struct.
func ParseTX(dat []byte) (*Tx, error) {
	tx := Tx{}
	buf := bytes.NewBuffer(dat)
	err := packer.Unpack(buf, &tx)
	return &tx, err
}

//Reverse reverse bits.
func Reverse(bs []byte) []byte {
	b := make([]byte, len(bs))
	for i := 0; i < len(bs)/2; i++ {
		b[i], b[len(bs)-1-i] = bs[len(bs)-1-i], bs[i]
	}
	return b
}

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
	"errors"
	"fmt"
	"sort"

	"math"

	"github.com/bitgoin/address"
	"github.com/bitgoin/packer"
)

//UTXO represents an available transaction.
type UTXO struct {
	Key     *address.PrivateKey
	TxHash  []byte
	Value   uint64
	Script  []byte
	TxIndex uint32
}

//UTXOs is array of coins.
type UTXOs []*UTXO

func (c UTXOs) Len() int           { return len(c) }
func (c UTXOs) Less(i, j int) bool { return c[i].Value < c[j].Value }
func (c UTXOs) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

//Send is information about addrress and amount to send.
type Send struct {
	Addr   string
	Amount uint64
}

//DefaultP2PKScript returns default p2pk script.
func DefaultP2PKScript(btcadr string) ([]byte, error) {
	addr, err := address.DecodeAddress(btcadr)
	if err != nil {
		return nil, err
	}
	script := make([]byte, 0, len(addr)+5)
	script = append(script, opDUP, opHASH160, byte(len(addr)))
	script = append(script, addr...)
	return append(script, opEQUALVERIFY, opCHECKSIG), nil
}

func p2pkTtxout(send *Send) (*TxOut, error) {
	script, err := DefaultP2PKScript(send.Addr)
	if err != nil {
		return nil, err
	}
	return &TxOut{
		Value:  send.Amount,
		Script: script,
	}, nil
}

func p2pkTxouts(fee uint64, sends ...*Send) ([]*TxOut, uint64, error) {
	total := fee
	txouts := make([]*TxOut, 0, len(sends))
	for _, send := range sends {
		if send.Amount == 0 {
			continue
		}
		total += send.Amount
		txout, err := p2pkTtxout(send)
		if err != nil {
			return nil, 0, err
		}
		txouts = append(txouts, txout)
	}
	return txouts, total, nil
}

func newTxins(total uint64, coins UTXOs, refundAddress string, locktime uint32) ([]*TxIn, []*UTXO, *TxOut, error) {
	var seq uint32 = math.MaxUint32
	if locktime != 0 {
		seq = 0
	}
	var txins []*TxIn
	var amount uint64
	sort.Sort(coins)
	var used []*UTXO
	for i := 0; i < len(coins) && amount < total; i++ {
		c := coins[i]
		txins = append(txins, &TxIn{
			Hash:   c.TxHash,
			Index:  c.TxIndex,
			Script: []byte{}, //pubscript to sign.
			Seq:    seq,
		})
		used = append(used, c)
		amount += c.Value
	}
	if amount < total {
		return nil, nil, nil, fmt.Errorf("shortage of coin %d < %d %d",
			amount, total, len(coins))
	}
	remain := amount - total
	var mto *TxOut
	var err error
	if remain > 0 {
		if refundAddress == "" {
			return nil, nil, nil, errors.New("refund address is empty")
		}
		s := Send{
			Addr:   refundAddress,
			Amount: remain,
		}
		mto, err = p2pkTtxout(&s)
	}
	return txins, used, mto, err
}

func signTx(result *Tx, used []*UTXO) ([][]byte, error) {
	sign := make([][]byte, len(used))
	var err error
	for i, p := range used {
		var buf bytes.Buffer
		backup := result.TxIn[i].Script
		result.TxIn[i].Script = p.Script
		if err = packer.Pack(&buf, *result); err != nil {
			return nil, err
		}
		beforeb := buf.Bytes()
		beforeb = append(beforeb, 0x01, 0, 0, 0) //hash code type
		h := sha256.Sum256(beforeb)
		h = sha256.Sum256(h[:])
		sign[i], err = p.Key.Sign(h[:])
		if err != nil {
			return nil, err
		}
		result.TxIn[i].Script = backup
	}
	return sign, nil
}

//FillP2PKsign embeds sign script to result Tx.
func FillP2PKsign(result *Tx, used []*UTXO) error {
	signs, err := signTx(result, used)
	if err != nil {
		return err
	}
	for i, s := range signs {
		s = append(s, 0x1)
		scr := result.TxIn[i].Script[:0]
		scr = append(scr, byte(len(s)))
		scr = append(scr, s...)
		pub := used[i].Key.PublicKey.Serialize()
		scr = append(scr, byte(len(pub)))
		scr = append(scr, pub...)
		result.TxIn[i].Script = scr
	}
	return nil
}

//NewP2PK creates msg.Tx from send infos.
//last index of sends must be refund address, and its amount must be 0..
func NewP2PK(fee uint64, coins UTXOs, locktime uint32, sends ...*Send) (*Tx, error) {
	result, used, err := NewP2PKunsign(fee, coins, locktime, sends...)
	if err != nil {
		return nil, err
	}
	err = FillP2PKsign(result, used)
	return result, err
}

//NewP2PKunsign creates msg.Tx from send infos without signing tx..
//last index of sends must be refund address, and its amount must be 0..
func NewP2PKunsign(fee uint64, coins UTXOs, locktime uint32, sends ...*Send) (*Tx, []*UTXO, error) {
	txouts, total, err := p2pkTxouts(fee, sends...)
	if err != nil {
		return nil, nil, err
	}
	if sends[len(sends)-1].Amount != 0 {
		return nil, nil, errors.New("last index of sends must be refund address and amount must be 0")
	}

	txins, used, mto, err := newTxins(total, coins, sends[len(sends)-1].Addr, locktime)
	if err != nil {
		return nil, nil, err
	}
	if mto != nil {
		txouts = append(txouts, mto)
	}
	return &Tx{
		Version:  1,
		TxIn:     txins,
		TxOut:    txouts,
		Locktime: locktime,
	}, used, nil
}

//CustomTx returns OP_RETURN txout with the custome data.
func CustomTx(data []byte) *TxOut {
	//Add custom data
	script := make([]byte, 0, 1+1+len(data))
	script = append(script, opRETURN)
	script = append(script, byte(len(data)))
	script = append(script, data...)

	return &TxOut{
		Script: script,
	}
}

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

	"github.com/bitgoin/address"
	"github.com/bitgoin/address/btcec"
	"github.com/bitgoin/packer"
)

//PubInfo is infor of public key in M of N multisig.
type PubInfo struct {
	Pubs   []*address.PublicKey
	Amount uint64
	bond   *Tx
	Fee    uint64
	M      byte
}

func (p *PubInfo) redeemScript() []byte {
	scr := make([]byte, 0, 3+len(p.Pubs)*btcec.PubKeyBytesLenUncompressed)
	scr = append(scr, op1+(p.M-1))
	for _, pu := range p.Pubs {
		ser := pu.Serialize()
		scr = append(scr, byte(len(ser)))
		scr = append(scr, ser...)
	}
	scr = append(scr, op1+(byte(len(p.Pubs)-1)))
	scr = append(scr, opCHECKMULTISIG)
	return scr
}

func (p *PubInfo) redeemHash() []byte {
	redeem := p.redeemScript()
	hash160 := address.AddressBytes(redeem)
	script := make([]byte, 0, len(hash160)+3)
	script = append(script, opHASH160, byte(len(hash160)))
	script = append(script, hash160...)
	script = append(script, opEQUAL)
	return script
}

//BondTx creates a bond transaction.
func (p *PubInfo) BondTx(coins UTXOs, refund string, locktime uint32) (*Tx, error) {
	n := len(p.Pubs)
	if n == 0 || n > 7 {
		return nil, errors.New("N must be 0~7")
	}
	if p.M == 0 || p.M > byte(n) {
		return nil, errors.New("M must be 0~N")
	}
	txouts := make([]*TxOut, 1, 2)
	txouts[0] = &TxOut{
		Value:  p.Amount,
		Script: p.redeemHash(),
	}
	txins, privs, mto, err := newTxins(p.Amount+p.Fee, coins, refund, locktime)
	if err != nil {
		return nil, err
	}
	if mto != nil {
		txouts = append(txouts, mto)
	}
	result := Tx{
		Version:  1,
		TxIn:     txins,
		TxOut:    txouts,
		Locktime: 0,
	}
	err = FillP2PKsign(&result, privs)
	p.bond = &result
	return &result, err
}

func (p *PubInfo) searchTxout() (uint32, error) {
	hash := p.redeemHash()
	for i, out := range p.bond.TxOut {
		if bytes.Equal(out.Script, hash) {
			return uint32(i), nil
		}
	}
	return 0, errors.New("not found")
}

func (p *PubInfo) txForSign(locktime uint32, sends ...*Send) (*Tx, error) {
	if p.bond == nil {
		return nil, errors.New("must call MultisigOut first")
	}
	txouts, total, err := p2pkTxouts(p.Fee, sends...)
	if err != nil {
		return nil, err
	}
	if p.Amount < total {
		return nil, errors.New("total coins of output must be less than one of input")
	}
	index, err := p.searchTxout()
	if err != nil {
		return nil, err
	}
	utxos := UTXOs{
		&UTXO{
			TxHash:  p.bond.Hash(),
			TxIndex: index,
			Script:  p.redeemScript(),
			Value:   p.Amount,
		},
	}
	mtxin, _, txout, err := newTxins(total, utxos, sends[len(sends)-1].Addr, locktime)
	if err != nil {
		return nil, err
	}
	mtxin[0].Script = p.redeemScript()
	if txout != nil {
		txouts = append(txouts, txout)
	}
	mtx := Tx{
		Version:  1,
		TxIn:     mtxin,
		TxOut:    txouts,
		Locktime: locktime,
	}

	return &mtx, nil
}

func (p *PubInfo) verify(mtx *Tx, sign []byte, i int) error {
	var buf bytes.Buffer
	if err := packer.Pack(&buf, *mtx); err != nil {
		return err
	}
	beforeb := buf.Bytes()
	beforeb = append(beforeb, 0x01, 0, 0, 0) //hash code type
	h := sha256.Sum256(beforeb)
	h = sha256.Sum256(h[:])
	return p.Pubs[i].Verify(sign, h[:])
}

//SignMultisig signs multisig transaction by priv.
func (p *PubInfo) SignMultisig(priv *address.PrivateKey,
	locktime uint32, sends ...*Send) ([]byte, error) {
	mtx, err := p.txForSign(locktime, sends...)
	if err != nil {
		return nil, err
	}
	prev := &UTXO{
		Key:    priv,
		Script: p.redeemScript(),
	}
	signs, err := signTx(mtx, []*UTXO{prev})
	if err != nil {
		return nil, err
	}
	return signs[0], nil
}

func (p *PubInfo) embedSigns(mtx *Tx, sigs [][]byte) error {
	redeem := p.redeemScript()
	script2 := make([]byte, 0, 74*len(sigs)+len(redeem)+3)
	script2 = append(script2, op0)
	var nsig byte
	for i, s := range sigs {
		if s == nil {
			continue
		}
		if err := p.verify(mtx, s, i); err != nil {
			return fmt.Errorf("%s at %d", err, i)
		}
		script2 = append(script2, byte(len(s)+1))
		script2 = append(script2, s...)
		script2 = append(script2, 0x01)
		nsig++
	}
	if nsig != p.M {
		return errors.New("signatures are not enough")
	}
	if len(redeem) > 255 {
		return errors.New("len of redeem script must be less than 255")
	}
	script2 = append(script2, opPUSHDATA1, byte(len(redeem)))
	script2 = append(script2, redeem...)
	mtx.TxIn[0].Script = script2

	return nil
}

//SpendBondTx creates tx which spends bond.
//Bond field in PubInfo must be filled previously.
func (p *PubInfo) SpendBondTx(locktime uint32, sigs [][]byte, sends ...*Send) (*Tx, error) {
	if len(sigs) == 0 {
		return nil, errors.New("must fill sigs")
	}
	if p.bond == nil {
		return nil, errors.New("must fill prev in pubinfo")
	}
	mtx, err := p.txForSign(locktime, sends...)
	if err != nil {
		return nil, err
	}
	err = p.embedSigns(mtx, sigs)
	return mtx, err
}

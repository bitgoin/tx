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
	"errors"

	"bytes"

	"github.com/bitgoin/address"
)

type mpay struct {
	*PubInfo
	priv *address.PrivateKey
}

//MicroPayer is struct for payer of micropayment.
type MicroPayer mpay

//MicroPayee is struct for payee of micropayment.
type MicroPayee mpay

//NewMicroPayer returns struct for payer.
func NewMicroPayer(payer *address.PrivateKey, payee *address.PublicKey, amount uint64, fee uint64) *MicroPayer {
	pk := make([]*address.PublicKey, 2)
	pk[0] = payer.PublicKey
	pk[1] = payee
	return &MicroPayer{
		PubInfo: &PubInfo{
			Pubs:   pk,
			Amount: amount,
			M:      2,
			Fee:    fee,
		},
		priv: payer,
	}
}

//NewMicroPayee returns struct for payee.
func NewMicroPayee(payer *address.PublicKey, payee *address.PrivateKey, amount uint64, fee uint64) *MicroPayee {
	pk := make([]*address.PublicKey, 2)
	pk[0] = payer
	pk[1] = payee.PublicKey
	return &MicroPayee{
		PubInfo: &PubInfo{
			Pubs:   pk,
			Amount: amount,
			M:      2,
			Fee:    fee,
		},
		priv: payee,
	}
}

func (m *PubInfo) sendstruct(amount uint64) ([]*Send, error) {
	payer := m.Pubs[0].Address()
	payee := m.Pubs[1].Address()
	sends := make([]*Send, 0, 2)
	switch {
	case m.Amount-m.Fee-amount < 0:
		return nil, errors.New("negative amount for payer")
	case m.Amount-m.Fee-amount == 0:
	default:
		sends = append(sends, &Send{
			Addr:   payer,
			Amount: m.Amount - m.Fee - amount,
		})
	}
	switch {
	case amount < 0:
		return nil, errors.New("negative amount for payee")
	case amount == 0:
	default:
		sends = append(sends, &Send{
			Addr:   payee,
			Amount: amount,
		})
	}
	return sends, nil
}

//SignRefund sings refund tx.
func (m *MicroPayee) SignRefund(refund *Tx, locktime uint32) ([]byte, error) {
	if refund.Locktime != locktime {
		return nil, errors.New("locktime in refund tx is illegal ")
	}
	if len(refund.TxIn) != 1 {
		return nil, errors.New("illegal txin number")
	}
	if refund.TxIn[0].Index != 0 {
		return nil, errors.New("illegal txin index")
	}
	refund.TxIn[0].Script = m.PubInfo.redeemHash()
	signs, err := signTx(refund, []*address.PrivateKey{m.priv})
	if err != nil {
		return nil, err
	}
	return signs[0], nil
}

//CheckBond checks and sets bond tx.
func (m *MicroPayee) CheckBond(refund, bond *Tx) error {
	if !bytes.Equal(bond.TxOut[0].Script, m.PubInfo.redeemHash()) {
		return errors.New("illegal script in bond")
	}
	if !bytes.Equal(refund.TxIn[0].Hash, bond.Hash()) {
		return errors.New("illegal txin hash in refund")
	}
	m.PubInfo.prev = bond
	return nil
}

//CreateBond returns bond and refund tx for sign.
func (m *MicroPayer) CreateBond(locktime uint32, coins UTXOs, ref string) (*Tx, *Tx, error) {
	bond, err := m.MultisigOut(coins, ref, locktime)
	if err != nil {
		return nil, nil, err
	}
	sends, err := m.sendstruct(0)
	if err != nil {
		return nil, nil, err
	}
	refund, err := m.PubInfo.txForSign(locktime, sends...)
	return bond, refund, err
}

//SignRefund signs refund..
func (m *MicroPayer) SignRefund(refund *Tx, sign []byte) error {
	signs := make([][]byte, 2)
	mysign, err := signTx(refund, []*address.PrivateKey{m.priv})
	if err != nil {
		return err
	}
	signs[0] = mysign[0]
	signs[1] = sign
	return m.PubInfo.embedSigns(refund, signs)
}

//Filter returns redeem script and its hash, which payee should wait for..
func (m *MicroPayee) Filter() ([]byte, []byte) {
	r := m.redeemScript()
	return r, address.AddressBytes(r)
}

//SignIncremented signs incremented tx..
func (m *MicroPayer) SignIncremented(amount uint64) ([]byte, error) {
	sends, err := m.sendstruct(amount)
	if err != nil {
		return nil, err
	}
	return m.SignMultisig(m.priv, 0, sends...)
}

//IncrementedTx returns an incremented tx..
func (m *MicroPayee) IncrementedTx(amount uint64, sign []byte) (*Tx, error) {
	sends, err := m.sendstruct(amount)
	if err != nil {
		return nil, err
	}
	mysign, err := m.SignMultisig(m.priv, 0, sends...)
	if err != nil {
		return nil, err
	}
	return m.MultisigIn(0, [][]byte{sign, mysign}, sends...)
}

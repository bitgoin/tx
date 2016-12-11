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
	"encoding/hex"
	"log"
	"testing"
	"time"

	"github.com/bitgoin/address"
)

func TestMicro(t *testing.T) {
	wif := "928Qr9J5oAC6AYieWJ3fG3dZDjuC7BFVUqgu4GsvRVpoXiTaJJf"
	//n3Bp1hbgtmwDtjQTpa6BnPPCA8fTymsiZy
	txKey, err := address.FromWIF(wif, address.BitcoinTest)
	if err != nil {
		t.Error(err)
	}
	adr := txKey.PublicKey.Address()
	log.Println("address for tx=", adr)

	wif2 := "92DUfNPumHzpCkKjmeqiSEDB1PU67eWbyUgYHhK9ziM7NEbqjnK"
	//ms5repuZHtBrKRE93FdWqz8JEo6d8ikM3k
	txKey2, err := address.FromWIF(wif2, address.BitcoinTest)
	if err != nil {
		t.Error(err)
	}
	txhashes := []string{
		"12c2f61d839b2b38146715e4dfc0fd914906253920480298816f108513e53e5c",
		"12c2f61d839b2b38146715e4dfc0fd988806253920480298816f108513e53e5c",
	}
	values := []uint64{100 * Unit, 150 * Unit}
	script, err := hex.DecodeString("76a914d94987ba89c258372030bc9d610f89547757896488ac")
	if err != nil {
		t.Fatal(err)
	}

	utxos := make(UTXOs, len(txhashes))
	for i, h := range txhashes {
		var ha []byte
		ha, err = hex.DecodeString(h)
		if err != nil {
			t.Fatal(err)
		}
		ha = Reverse(ha)
		utxos[i] = &UTXO{
			Key:     txKey,
			TxHash:  ha,
			Value:   values[i],
			Script:  script,
			TxIndex: uint32(i + 1),
		}
	}

	payer := NewMicroPayer(txKey, txKey2.PublicKey, 200*Unit, 0.001*Unit)
	payee := NewMicroPayee(txKey.PublicKey, txKey2, 200*Unit, 0.001*Unit)
	locktime := uint32(time.Now().Add(time.Hour).Unix())

	bond, refund, err := payer.CreateBond(locktime, utxos, txKey.PublicKey.Address())
	if err != nil {
		t.Error(err)
	}
	sign, err := payee.SignRefund(refund, locktime)
	if err != nil {
		t.Error(err)
	}

	if err := payer.SignRefund(refund, sign); err != nil {
		t.Error(err)
	}
	if err := payee.CheckBond(refund, bond); err != nil {
		t.Error(err)
	}

	signIP, err := payer.SignIncremented(0.001 * Unit)
	if err != nil {
		t.Error(err)
	}
	log.Println(hex.EncodeToString(signIP))
	tx, err := payee.IncrementedTx(0.001*Unit, signIP)
	if err != nil {
		t.Error(err)
	}
	bbond, err := bond.Pack()
	if err != nil {
		t.Error(err)
	}
	bref, err := refund.Pack()
	if err != nil {
		t.Error(err)
	}
	btx, err := tx.Pack()
	if err != nil {
		t.Error(err)
	}
	log.Print("bond ", hex.EncodeToString(bbond))
	log.Print("refund ", hex.EncodeToString(bref))
	log.Print("incremented tx ", hex.EncodeToString(btx))
}

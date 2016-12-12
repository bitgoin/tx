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
	"encoding/hex"
	"log"
	"testing"

	"github.com/bitgoin/address"
)

func TestRedeemScript(t *testing.T) {
	//MTi4x2NtDpdyXSwEvwU3aZ1Uronz1JBNC3
	pkey, err := address.FromWIF("T81eGkQ2nrQZGvkcSKCtV1tZJ4WrsKhRsBA1jCgyfMdDjmn5TwGn", address.MonacoinMain)
	if err != nil {
		t.Fatal(err)
	}

	//MAQnZ4FJ8rXPtRTZ9zwbwBmxaz9h9DTYxg
	pkey2, err := address.FromWIF("T4MzbNi83oaNzi8Yid22ZeNqHzaFhLqQkKmkffuQ58jR4ytz9QG2", address.MonacoinMain)
	if err != nil {
		t.Fatal(err)
	}
	//MWd1DJDeuXrdYD5dPpdUvoxHKvxVvAE8cs
	pkey3, err := address.FromWIF("T9QEmRobyTDTJe4qzSEu2mD1SMu6Wtzun6xkawnwRpBX5brimeCN", address.MonacoinMain)
	if err != nil {
		t.Fatal(err)
	}

	txhashes := []string{
		"12c2f61d839b2b38146715e4dfc0fd914906253920480298816f108513e53e5c",
		"12c2f61d839b2b38146715e4dfc0fd988806253920480298816f108513e53e5c",
	}

	script, err := hex.DecodeString("76a914d94987ba89c258372030bc9d610f89547757896488ac")
	if err != nil {
		t.Fatal(err)
	}
	redeem, err := hex.DecodeString("52210235dad6f5b0655e5ec633e71c3d8e0acee49a314c76a2650f6d60bc291d631c9d2103bd9b94f58dd51233a1380accd944aa44d9846fab673497ca4de794f79ecdbccd210373f0f5d4488616b20537810f5281ea27dd65213fa40be696086c6d2c3319419e53ae")
	if err != nil {
		t.Fatal(err)
	}
	hashout := make([][]byte, 2)
	hashout[0], err = hex.DecodeString("e0f6208a5718f126aa592c432a246761e6e4f1ac428e703f32e02f4828fab266")
	if err != nil {
		t.Fatal(err)
	}
	hashout[1], err = hex.DecodeString("4738c127eef819608dafc005f83a9ec9bf8b98d97ec9adf254f7fe8954ec10c2")
	if err != nil {
		t.Fatal(err)
	}
	values := []uint64{100 * Unit, 150 * Unit}

	utxos := make(UTXOs, len(txhashes))
	for i, h := range txhashes {
		var ha []byte
		ha, err = hex.DecodeString(h)
		if err != nil {
			t.Fatal(err)
		}
		ha = Reverse(ha)
		utxos[i] = &UTXO{
			Key:     pkey,
			TxHash:  ha,
			Value:   values[i],
			Script:  script,
			TxIndex: uint32(i + 1),
		}
	}
	pi := &PubInfo{
		Pubs:   []*address.PublicKey{pkey2.PublicKey, pkey3.PublicKey, pkey.PublicKey},
		Amount: 200 * Unit,
		M:      2,
		Fee:    0.001 * Unit,
	}

	txout, err := pi.BondTx(utxos, pkey.PublicKey.Address(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(redeem, pi.redeemScript()) {
		t.Fatal("redeem script is illegal")
	}
	log.Println(hex.EncodeToString(txout.Hash()))
	log.Println(hex.EncodeToString(txout.TxOut[0].Script))
	log.Println(hex.EncodeToString(pi.redeemScript()))

	byt, err := txout.Pack()
	if err != nil {
		t.Fatal(err)
	}
	log.Println(hex.EncodeToString(byt))
	for i, in := range txout.TxIn {
		slen := in.Script[0]
		script = in.Script[1:slen]
		//in.Script[slen]=0x01,in.Script[slen+1]=length of pubkey
		var pubk *address.PublicKey
		pubk, err = address.NewPublicKey(in.Script[slen+2:], address.MonacoinMain)
		if err != nil {
			t.Fatal(err)
		}
		if err = pubk.Verify(script, hashout[i]); err != nil {
			t.Error("illegal tx")
		}
	}

	//for test
	scr := "483045022100902c0effe741979fd353a038897ab7eee17e1bea3ea8987298e52539de9a70f20220458310b9129b1123a72b22f0206857bec67b71d1e3df3502c8adef93f37818e801210373f0f5d4488616b20537810f5281ea27dd65213fa40be696086c6d2c3319419e"
	txhash := "1eb8d0cfd1963d6295fcb5a76800fb8ae0a0c5332c349131d9bdf3d340f57eed"
	scrb, err := hex.DecodeString(scr)
	if err != nil {
		t.Fatal(err)
	}
	pi.bond.TxIn[0].Script = scrb
	pi.bond.TxIn[1].Script = scrb
	txhashb, err := hex.DecodeString(txhash)
	if err != nil {
		t.Fatal(err)
	}
	txhashb = Reverse(txhashb)
	if !bytes.Equal(pi.bond.Hash(), txhashb) {
		t.Fatal("tx unamtches")
	}
	hashin, err := hex.DecodeString("933ce8591ea3a3c1267b08c9a59ee72e25ef0371bb7fce39fd739426dc260790")
	if err != nil {
		t.Fatal(err)
	}

	send := []*Send{
		&Send{
			Addr:   "MTi4x2NtDpdyXSwEvwU3aZ1Uronz1JBNC3",
			Amount: 200*Unit - 0.001*Unit,
		},
		&Send{
			Addr:   "",
			Amount: 0,
		},
	}
	sig, err := pi.SignMultisig(pkey, 0, send...)
	if err != nil {
		t.Fatal(err)
	}
	sig2, err := pi.SignMultisig(pkey2, 0, send...)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := pi.SpendBondTx(0, [][]byte{sig2, nil, sig}, send...)
	if err != nil {
		t.Fatal(err)
	}

	byt, err = tx.Pack()
	if err != nil {
		t.Fatal(err)
	}
	log.Println(hex.EncodeToString(byt))
	slen := tx.TxIn[0].Script[1]
	script = tx.TxIn[0].Script[2 : slen+2-1]
	if err = pkey2.PublicKey.Verify(script, hashin); err != nil {
		t.Error("illegal tx")
	}
	slen2 := tx.TxIn[0].Script[slen+2]
	script = tx.TxIn[0].Script[slen+2+1 : slen+slen2+3-1]
	if err = pkey.PublicKey.Verify(script, hashin); err != nil {
		t.Error("illegal tx")
	}
}

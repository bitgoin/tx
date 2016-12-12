
[![Build Status](https://travis-ci.org/bitgoin/tx.svg?branch=master)](https://travis-ci.org/bitgoin/tx)
[![GoDoc](https://godoc.org/github.com/bitgoin/address?status.svg)](https://godoc.org/github.com/bitgoin/tx)
[![GitHub license](https://img.shields.io/badge/license-BSD-blue.svg)](https://raw.githubusercontent.com/bitgoin/tx/LICENSE)


# tx 

## Overview

This  library is for handling bitcoin transactions(tx), including P2PK(publish to public keey), P2SH(publish to script hash) and
Micropayment Channel.

## Requirements

This requires

* git
* go 1.3+


## Installation

     $ go get github.com/bitgoin/tx


## Example
(This example omits error handlings for simplicity.)

### P2PK
```go

import "github.com/bitgoin/tx"

func main(){
    fee:=0.001*Unit
	locktime:=0

	//prepare private key.
	txKey, err := address.FromWIF("some wif", address.BitcoinTest)

	//set UTXOs to be used in P2PK with script.
	//Assume script in UTXOs are default P2PK style.
	//(OP_DUP OP_HASH160 <pubKeyHash> OP_EQUALVERIFY OP_CHECKSIG)
	script, err := tx.DefaultP2PKScript(txKey.PublicKey.Address())

	//prepare unspent transaction outputs with its privatekey.
	coins := tx.UTXOs{
		&tx.UTXO{
			Key:     txKey,
			TxHash:  hash,
			TxIndex: 1,
			Script:  script,
			Value:   68000000 + fee,
		}}

	//prepare send addresses and its amount.
	//last address must be refund address and its amount must be 0.
	send := []*tx.Send{
		&tx.Send{
			Addr:   "n2eMqTT929pb1RDNuqEnxdaLau1rxy3efi",
			Amount: 68000000,
		},
		&Send{
			Addr:   "",
			Amount: 0,
		},
	}

	//get TX.
	tx, err := tx.NewP2PK(fee, coins, locktime, send...)

	//get binary form of tx.
	rawtx, err := tx.Pack()


    //if you want to add custom data to tx  with OP_RETURN.

	//get unsigned TX.
	ntx, used, err := tx.NewP2PKunsign(0.0001*Unit, coins, 0, send...)

	//add custom txout and add it to the tx.
	txout := tx.CustomTx([]byte("some public data"))
	ntx.TxOut = append(ntx.TxOut, txout)

	//sign tx.
	err := tx.FillP2PKsign(ntx, used);
}
```

### P2SH
```go

import "github.com/bitgoin/tx"

func main(){
    fee:=0.001*Unit
	locktime:=0

	//prepare private key.
	txKey1, err := address.FromWIF("some wif1", address.BitcoinTest)
	txKey2, err := address.FromWIF("some wif2", address.BitcoinTest)
	txKey3, err := address.FromWIF("some wif3", address.BitcoinTest)

	//set UTXOs to be used in P2SH with script.
	//Assume script in UTXOs are default P2PK style.
	//(OP_DUP OP_HASH160 <pubKeyHash> OP_EQUALVERIFY OP_CHECKSIG)
	script, err := tx.DefaultP2PKScript(txKey.PublicKey.Address())
	coins := tx.UTXOs{
		&tx.UTXO{
			Key:     txKey,
			TxHash:  hash,
			TxIndex: 1,
			Script:  script,
			Value:   68000000 + fee,
		}}

    //prepare M of N contract info.
	//set publickeys, amount, M, and fee. 
	pi := &PubInfo{
		Pubs:   []*address.PublicKey{pkey2.PublicKey, pkey3.PublicKey, pkey.PublicKey},
		Amount: 200 * Unit,
		M:      2,
		Fee:    fee,
	}

	//make bond transaction from coins.
	txout, err := pi.BondTx(utxos, pkey.PublicKey.Address(), locktime)

	//prepare send addresses and its amount.
	//last address must be refund address and its amount must be 0.
	send := []*tx.Send{
		&tx.Send{
			Addr:   "n2eMqTT929pb1RDNuqEnxdaLau1rxy3efi",
			Amount: 68000000,
		},
		&Send{
			Addr:   "",
			Amount: 0,
		},
	}

    //get 2(=M) signs 
	sig2, err := pi.SignMultisig(pkey2, locktime, send...)
	sig, err := pi.SignMultisig(pkey, locktime, send...)

    //make transaction which spends bond.
	//signs must be filled in same order as Pubinfo.Pubs.
	tx, err := pi.SpendBondTx(0, [][]byte{sig2, nil, sig}, send...)
}
```

### Micropayment
```go

import "github.com/bitgoin/tx"

func main(){
    fee:=0.001*Unit

	//prepare private key.
	txKey1, err := address.FromWIF("some wif1", address.BitcoinTest)
	txKey2, err := address.FromWIF("some wif2", address.BitcoinTest)

	//set UTXOs with script.
	//Assume script in UTXOs are default P2PK style.
	//(OP_DUP OP_HASH160 <pubKeyHash> OP_EQUALVERIFY OP_CHECKSIG)
	script, err := tx.DefaultP2PKScript(txKey.PublicKey.Address())
	coins := tx.UTXOs{
		&tx.UTXO{
			Key:     txKey,
			TxHash:  hash,
			TxIndex: 1,
			Script:  script,
			Value:   68000000 + fee,
		}}

    //prepare micropayer or micropayee.
	bondAmount:=200*Unit
	payer := NewMicroPayer(txKey, txKey2.PublicKey, bondAmount, fee)
	payee := NewMicroPayee(txKey.PublicKey, txKey2, bondAmount, fee)

    //payer creates bond and refund tx without sign.
	//refund tx will be validated after one hour.
	locktime:=uint32(time.Now().Add(time.Hour).Unix())
	bond, refund, err := payer.CreateBond(locktime, utxos, txKey.PublicKey.Address())

    //payee gets payer's refund tx by some way and signs it.
	sign, err := payee.SignRefund(refund, locktime)

    //payer gets and checks payee's sign and sings the refund.
    err := payer.SignRefund(refund, sign)

    //payee gets payer's bond and checks it.
	err := payee.CheckBond(refund, bond)

    for {
        //payee starts to work and after a while requests to increment his amount.
		time.Sleep(time.Hour)
	    //payer calculates sign for incremented tx.
	    signIP, err := payer.SignIncremented(0.001 * Unit)

    	//payee get payer's sign ,checks it, and get incremented tx.
	    tx, err := payee.IncrementedTx(0.001*Unit, signIP)
    }

}
```

* Note

Payer must send refund tx after locktime.

http://chimera.labs.oreilly.com/books/1234000001802/ch05.html#tx_propagation

>Transactions with locktime specifying a future block or time must be held by the originating system
>and transmitted to the bitcoin network only after they become valid.


# Contribution
Improvements to the codebase and pull requests are encouraged.



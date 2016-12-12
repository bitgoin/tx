package tx

import (
	"encoding/hex"
	"log"
	"testing"

	"github.com/bitgoin/address"
)

func TestTX(t *testing.T) {
	wif := "928Qr9J5oAC6AYieWJ3fG3dZDjuC7BFVUqgu4GsvRVpoXiTaJJf"
	txKey, err := address.FromWIF(wif, address.BitcoinTest)
	if err != nil {
		t.Error(err)
	}

	hash, err := hex.DecodeString("1a103718e2e0462c50cb057a0f39d7c6cbf960276452d07dc4a50ddca725949c")
	if err != nil {
		t.Error(err)
	}
	for i := 0; i < len(hash)/2; i++ {
		hash[i], hash[len(hash)-1-i] = hash[len(hash)-1-i], hash[i]
	}

	script, err := DefaultP2PKScript(txKey.PublicKey.Address())
	if err != nil {
		t.Error(err)
	}
	coins := UTXOs{
		&UTXO{
			Key:     txKey,
			TxHash:  hash,
			TxIndex: 1,
			Script:  script,
			Value:   68000000 + 0.0001*Unit,
		}}

	send := []*Send{
		&Send{
			Addr:   "n2eMqTT929pb1RDNuqEnxdaLau1rxy3efi",
			Amount: 68000000,
		},
		&Send{
			Addr:   "",
			Amount: 0,
		},
	}

	tx, err := NewP2PK(0.0001*Unit, coins, 0, send...)
	if err != nil {
		t.Error(err)
	}
	rawtx, err := tx.Pack()
	if err != nil {
		t.Error(err)
	}

	ok := "01000000019c9425a7dc0da5c47dd052642760f9cbc6d7390f7a05cb502c46e0e21837101a010000008a473044022030ebb89d54e76b9e14b8eb21aa30055eb54289dcd3aad9b415ebcc153b211eee0220720fa77cfc2c25da52899f3bf9a947869bc89d26066c02a1c428e9530a3f49b10141049f160b18fa4acedccdc063961d63b3a23385b1e67159d07521cb46d4e7209ecd443e473796e7ace130164c660fbcfb7dcac8437cc55f3ceafb546054c8d8cbdfffffffff0100990d04000000001976a914e7c1345fc8f87c68170b3aa798a956c2fe6a9eff88ac00000000"
	if hex.EncodeToString(rawtx) != ok {
		t.Error("invalid tx", hex.EncodeToString(rawtx))
	}

	//get unsigned TX.
	tx, used, err := NewP2PKunsign(0.0001*Unit, coins, 0, send...)

	//add custom txout and add it to tx.
	txout := CustomTx([]byte("some public data"))
	tx.TxOut = append(tx.TxOut, txout)
	if err != nil {
		t.Error(err)
	}
	//sign tx.
	if err = FillP2PKsign(tx, used); err != nil {
		t.Error(err)
	}
	rawtx, err = tx.Pack()
	if err != nil {
		t.Error(err)
	}
	log.Print(hex.EncodeToString(rawtx))
}

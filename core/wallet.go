// Copyright 2021 Hailiang Zhao <hliangzhao@zju.edu.cn>
// This file is part of the lightChain.
//
// The lightChain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The lightChain is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the lightChain. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	`bytes`
	`crypto/ecdsa`
	`crypto/elliptic`
	`crypto/rand`
	`crypto/sha256`
	`encoding/gob`
	`errors`
	`fmt`
	`golang.org/x/crypto/ripemd160`
	`io/ioutil`
	`lightChain/utils`
	`log`
	`os`
)

const version = byte(0x00)
const walletFile = "wallets"
const addrCheckSumLen = 4

// Wallet consists of a private key (generated by the ecdsa) and a public key.
type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PubKey     []byte
}

// NewWallet creates a new Wallet instance and returns the pointer to it.
func NewWallet() *Wallet {
	private, public := newKeyPair()
	return &Wallet{private, public}
}

// GetAddr returns the address of a wallet based on the wallet's public key and hashing algorithms.
func (wallet *Wallet) GetAddr() []byte {
	pubKeyHash := HashingPubKey(wallet.PubKey)

	versionedPayload := append([]byte{version}, pubKeyHash...)

	checksum := getChecksum(versionedPayload)
	fullPayload := append(versionedPayload, checksum...)

	return utils.Base58Encoding(fullPayload)
}

// ValidateAddr checks whether addr is a valid address.
func ValidateAddr(addr string) bool {
	pubKeyHash := utils.Base58Decoding([]byte(addr))
	actualChecksum := pubKeyHash[len(pubKeyHash) - addrCheckSumLen:]

	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1: len(pubKeyHash) - addrCheckSumLen]
	targetChecksum := getChecksum(append([]byte{version}, pubKeyHash...))

	return bytes.Compare(actualChecksum, targetChecksum) == 0
}

// HashingPubKey hashes the public key and returns the result.
func HashingPubKey(pubKey []byte) []byte {
	sha := sha256.Sum256(pubKey)
	hasher := ripemd160.New()
	_, err := hasher.Write(sha[:])
	if err != nil {
		log.Panic(err)
	}
	return hasher.Sum(nil)
}

// newKeyPair returns a private-public key pair.
func newKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}
	pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)
	return *private, pubKey
}

// getChecksum generates the checksum (a 4-byte slice) of given payload.
func getChecksum(payload []byte) []byte {
	sha1 := sha256.Sum256(payload)
	sha2 := sha256.Sum256(sha1[:])
	return sha2[:addrCheckSumLen]
}


// Wallets is a collection of Wallet.
type Wallets struct {
	WalletsMap map[string]*Wallet       // key: address of the wallet, value: the wallet itself
}

// Save2File saves the content of wallets into a local file.
func (wallets *Wallets) Save2File() {
	var buf bytes.Buffer
	gob.Register(elliptic.P256())

	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(*wallets)
	if err != nil {
		log.Panic(err)
	}

	err = ioutil.WriteFile(walletFile, buf.Bytes(), 0644)
	if err != nil {
		log.Panic(err)
	}
}

// LoadFromFile loads the wallets into the caller wallets.
func (wallets *Wallets) LoadFromFile() error {
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}

	rawContent, err := ioutil.ReadFile(walletFile)
	if err != nil {
		log.Panic(err)
	}

	var tmpWallets Wallets
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(rawContent))
	err = decoder.Decode(&tmpWallets)
	if err != nil {
		log.Panic(err)
	}

	wallets.WalletsMap = tmpWallets.WalletsMap
	return nil
}

// GetAddrs returns a slice of all addresses from wallets.
func (wallets *Wallets) GetAddrs() []string {
	var addrs []string
	for addr := range wallets.WalletsMap {
		addrs = append(addrs, addr)
	}
	return addrs
}

// GetWallet returns the Wallet by its addr.
func (wallets *Wallets) GetWallet(addr string) (Wallet, error) {
	if _, ok := wallets.WalletsMap[addr]; !ok {
		return Wallet{}, errors.New("address not found in wallets")
	}
	return *wallets.WalletsMap[addr], nil
}

// NewWallets creates Wallets from local walletFile.
func NewWallets() (*Wallets, error) {
	wallets := Wallets{}
	wallets.WalletsMap = make(map[string]*Wallet)

	err := wallets.LoadFromFile()
	return &wallets, err
}

// AddWallet creates a new Wallet, add it (and its address) to the caller and returns the address.
func (wallets *Wallets) AddWallet() string {
	wallet := NewWallet()
	addr := fmt.Sprintf("%s", wallet.GetAddr())

	wallets.WalletsMap[addr] = wallet
	return addr
}

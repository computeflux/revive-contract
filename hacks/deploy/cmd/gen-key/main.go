package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"

	"github.com/vedhavyas/go-subkey/v2/sr25519"
	"github.com/wetee-dao/ink.go/util"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: gen-key <count> [network-id]")
		fmt.Println("  count       number of key pairs to generate")
		fmt.Println("  network-id  SS58 network id, default: 42")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  gen-key 1")
		fmt.Println("  gen-key 3 0")
		os.Exit(1)
	}

	count, err := strconv.Atoi(os.Args[1])
	if err != nil || count <= 0 {
		fmt.Fprintf(os.Stderr, "invalid count: %s\n", os.Args[1])
		os.Exit(1)
	}

	network := uint16(42)
	if len(os.Args) >= 3 {
		n, err := strconv.ParseUint(os.Args[2], 10, 16)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid network-id: %s\n", os.Args[2])
			os.Exit(1)
		}
		network = uint16(n)
	}

	for i := 0; i < count; i++ {
		kp, err := sr25519.Scheme{}.Generate()
		if err != nil {
			fmt.Fprintf(os.Stderr, "generate key pair: %v\n", err)
			os.Exit(1)
		}

		seed := kp.Seed()
		publicKey := kp.Public()
		ss58Address := kp.SS58Address(network)
		h160, _ := util.H160FromPublicKey(publicKey)

		fmt.Printf("[%d]\n", i+1)
		fmt.Printf("  Seed (hex)     : 0x%s\n", hex.EncodeToString(seed))
		fmt.Printf("  PublicKey      : 0x%s\n", hex.EncodeToString(publicKey))
		fmt.Printf("  SS58 Address   : %s\n", ss58Address)
		fmt.Printf("  H160 Address   : %s\n", h160.Hex())
		fmt.Println()
	}
}

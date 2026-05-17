package contracts

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"testing"

	"wetee/test/contracts/proxy"
	"wetee/test/contracts/subnet"
	"wetee/test/contracts/token"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	chain "github.com/wetee-dao/ink.go"
	"github.com/wetee-dao/ink.go/pallet/revive"
	"github.com/wetee-dao/ink.go/util"
)

func TestTokenUpdate(t *testing.T) {
	client, err := chain.InitClient([]string{TestChainUrl}, true)
	if err != nil {
		panic(err)
	}

	pk, err := chain.Sr25519PairFromSecret("//Alice", 42)
	if err != nil {
		util.LogWithPurple("Sr25519PairFromSecret", err)
		panic(err)
	}

	tokenData, err := os.ReadFile("../../target/token.release.polkavm")
	if err != nil {
		util.LogWithPurple("read file error", err)
		panic(err)
	}

	salt := genSalt()
	res, err := token.DeployTokenWithNew(chain.DeployParams{
		Client: client,
		Signer: &pk,
		Code:   util.InkCode{Upload: &tokenData},
		Salt:   util.NewSome(salt),
	})
	if err != nil {
		util.LogWithPurple("DeployTokenWithNew", err)
		panic(err)
	}
	fmt.Println("token address: ", res.Hex())

	tokenIns, err := proxy.InitProxyContract(client, TokenAddress)
	if err != nil {
		util.LogWithPurple("InitTokenContract", err)
		panic(err)
	}

	err = tokenIns.ExecUpgrade(*res, chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		util.LogWithPurple("ExecUpgrade", err)
		panic(err)
	}

	fmt.Println("new token address: ", res.Hex())
	fmt.Println("proxy address: ", tokenIns.ContractAddress().Hex())
}

func TestSubnetUpdate(t *testing.T) {
	client, err := chain.InitClient([]string{TestChainUrl}, true)
	if err != nil {
		panic(err)
	}

	pk, err := chain.Sr25519PairFromSecret("//Alice", 42)
	if err != nil {
		util.LogWithPurple("Sr25519PairFromSecret", err)
		panic(err)
	}

	netData, err := os.ReadFile("../../target/subnet.release.polkavm")
	if err != nil {
		util.LogWithPurple("read file error", err)
		panic(err)
	}

	salt := genSalt()
	res, err := subnet.DeploySubnetWithNew(chain.DeployParams{
		Client: client,
		Signer: &pk,
		Code:   util.InkCode{Upload: &netData},
		Salt:   util.NewSome(salt),
	})
	if err != nil {
		util.LogWithPurple("DeploySubnetWithNew", err)
		panic(err)
	}

	subnetIns, err := proxy.InitProxyContract(client, SubnetAddress)
	if err != nil {
		util.LogWithPurple("InitSubnetContract", err)
		panic(err)
	}

	err = subnetIns.ExecUpgrade(*res, chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		util.LogWithPurple("ExecUpgrade", err)
		panic(err)
	}

	fmt.Println("new subnet address: ", res.Hex())
	fmt.Println("proxy address: ", subnetIns.ContractAddress().Hex())
}

func TestMapAccount(t *testing.T) {
	client, err := chain.InitClient([]string{TestChainUrl}, true)
	if err != nil {
		panic(err)
	}

	pk, err := chain.Sr25519PairFromSecret("//Alice", 42)
	if err != nil {
		util.LogWithPurple("Sr25519PairFromSecret", err)
		panic(err)
	}

	h160 := pk.H160Address()

	_, isSome, err := revive.GetOriginalAccountLatest(client.Api().RPC.State, h160)
	if err != nil {
		util.LogWithPurple("GetOriginalAccountLatest", err)
		panic(err)
	}
	if !isSome {
		runtimeCall := revive.MakeMapAccountCall()
		call, err := (runtimeCall).AsCall()
		if err != nil {
			panic(err)
		}

		err = client.SignAndSubmit(&pk, call, true, 0)
		if err != nil {
			panic(err)
		}
	}
}

func TestSetPrice(t *testing.T) {
	client, err := chain.InitClient([]string{TestChainUrl}, true)
	if err != nil {
		panic(err)
	}

	pk, err := chain.Sr25519PairFromSecret("//Alice", 42)
	if err != nil {
		util.LogWithPurple("Sr25519PairFromSecret", err)
		panic(err)
	}

	subnetIns, err := subnet.InitSubnetContract(client, SubnetAddress)
	if err != nil {
		util.LogWithPurple("InitCloudContract", err)
		panic(err)
	}

	err = subnetIns.ExecSetLevelPrice(1, subnet.RunPrice{
		CpuPer:       1,
		CvmCpuPer:    1,
		MemoryPer:    1,
		CvmMemoryPer: 1,
		DiskPer:      1,
		GpuPer:       1,
	}, chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		util.LogWithPurple("ExecSetLevelPrice", err)
		panic(err)
	}
}

func TestSetAssetPrice(t *testing.T) {
	client, err := chain.InitClient([]string{TestChainUrl}, true)
	if err != nil {
		panic(err)
	}

	pk, err := chain.Sr25519PairFromSecret("//Alice", 42)
	if err != nil {
		util.LogWithPurple("Sr25519PairFromSecret", err)
		panic(err)
	}

	subnetIns, err := subnet.InitSubnetContract(client, SubnetAddress)
	if err != nil {
		util.LogWithPurple("InitCloudContract", err)
		panic(err)
	}

	name := []byte("T")
	err = subnetIns.ExecSetAsset(subnet.AssetInfo{
		Native: &name,
	}, types.NewU256(*big.NewInt(1000)), chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	})

	if err != nil {
		util.LogWithPurple("ExecSetAsset", err)
		panic(err)
	}
}

func genSalt() [32]byte {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	randomBytes := [32]byte{}
	copy(randomBytes[:], bytes)

	return randomBytes
}

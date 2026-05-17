package contracts

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"testing"

	"wetee/test/contracts/cloud"
	"wetee/test/contracts/proxy"
	"wetee/test/contracts/subnet"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	chain "github.com/wetee-dao/ink.go"
	"github.com/wetee-dao/ink.go/pallet/revive"
	"github.com/wetee-dao/ink.go/util"
)

func TestCloudUpdate(t *testing.T) {
	client, err := chain.InitClient([]string{TestChainUrl}, true)
	if err != nil {
		panic(err)
	}

	pk, err := chain.Sr25519PairFromSecret("//Alice", 42)
	if err != nil {
		util.LogWithPurple("Sr25519PairFromSecret", err)
		panic(err)
	}

	/// init pod
	cloudData, err := os.ReadFile("../../target/cloud.release.polkavm")
	if err != nil {
		util.LogWithPurple("read file error", err)
		panic(err)
	}

	salt := genSalt()
	res, err := cloud.DeployCloudWithNew(chain.DeployParams{
		Client: client,
		Signer: &pk,
		Code:   util.InkCode{Upload: &cloudData},
		Salt:   util.NewSome(salt),
	})
	if err != nil {
		util.LogWithPurple("DeployCloudWithNew", err)
		panic(err)
	}
	fmt.Println("cloud address: ", res.Hex())

	cloudIns, err := proxy.InitProxyContract(client, CloudAddress)
	if err != nil {
		util.LogWithPurple("InitCloudContract", err)
		panic(err)
	}

	err = cloudIns.ExecUpgrade(*res, chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		util.LogWithPurple("ExecUpgrade", err)
		panic(err)
	}

	fmt.Println("new cloud address: ", res.Hex())
	fmt.Println("proxy address: ", cloudIns.ContractAddress().Hex())
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

func TestCloudQuerySecret(t *testing.T) {
	client, err := chain.InitClient([]string{TestChainUrl}, true)
	if err != nil {
		panic(err)
	}

	pk, err := chain.Sr25519PairFromSecret("//Alice", 42)
	if err != nil {
		util.LogWithPurple("Sr25519PairFromSecret", err)
		panic(err)
	}

	cloudContract, err := cloud.InitCloudContract(client, CloudAddress)
	if err != nil {
		panic(err)
	}

	fmt.Println(pk.SS58Address(42))
	fmt.Println(pk.H160Address().Hex())

	_call := chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	}
	err = cloudContract.ExecDelDisk(0, _call)
	if err != nil {
		panic(err)
	}
	fmt.Println("del disk:", err)

	// err = cloudContract.ExecInitDisk([]byte("node0"), 10, _call)
	// if err != nil {
	// 	panic(err)
	// }

	disk, _, err := cloudContract.QueryDisk(pk.H160Address(), 0, chain.DryRunParams{
		Origin:    pk.AccountID(),
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("disk", disk)

	disks, _, err := cloudContract.QueryUserDisks(pk.H160Address(), util.NewNone[uint64](), 1000, chain.DryRunParams{
		Origin:    pk.AccountID(),
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("disks:", disks)
}

func TestPodCharge(t *testing.T) {

}

func TestCloudTransfer(t *testing.T) {
	client, err := chain.InitClient([]string{TestChainUrl}, true)
	if err != nil {
		panic(err)
	}

	pk, err := chain.Sr25519PairFromSecret("//Alice", 42)
	if err != nil {
		util.LogWithPurple("Sr25519PairFromSecret", err)
		panic(err)
	}

	cloudContract, err := cloud.InitCloudContract(client, CloudAddress)
	if err != nil {
		panic(err)
	}

	tName := []byte("T")
	err = cloudContract.ExecTransfer(cloud.AssetInfo{
		Native: &tName,
	}, pk.H160Address(), types.NewU256(*big.NewInt(10001)), chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		panic(err)
	}
}

// TestSubnetQuerySideChainKey tests querying side chain key from subnet contract
func TestCloudQuerySecrets(t *testing.T) {
	client, err := chain.InitClient([]string{TestChainUrl}, true)
	if err != nil {
		panic(err)
	}

	pk, err := chain.Sr25519PairFromSecret("//Alice", 42)
	if err != nil {
		util.LogWithPurple("Sr25519PairFromSecret", err)
		panic(err)
	}

	cloudIns, err := cloud.InitCloudContract(client, CloudAddress)
	if err != nil {
		util.LogWithPurple("InitCloudContract", err)
		panic(err)
	}

	err = cloudIns.ExecCreateSecret([]byte("test"), types.H256{}, chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		util.LogWithPurple("ExecCreateSecret", err)
		panic(err)
	}

	secrets, _, err := cloudIns.QueryUserSecrets(
		pk.H160Address(),
		util.NewNone[uint64](),
		100,
		chain.DryRunParams{
			Origin:    pk.AccountID(),
			PayAmount: types.NewU128(*big.NewInt(0)),
		},
	)
	if err != nil {
		util.LogWithPurple("QueryUserSecrets", err)
		panic(err)
	}

	fmt.Printf("Secrets: %+v\n", secrets)
}

package contracts

import (
	"fmt"
	"math/big"
	"os"
	"testing"
	"wetee/test/contracts/cloud"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	chain "github.com/wetee-dao/ink.go"
	"github.com/wetee-dao/ink.go/util"
)

func TestCloudQueryPodCode(t *testing.T) {
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

	codeHash, _, err := cloudContract.QueryPodImpl(chain.DryRunParams{
		Origin:    pk.AccountID(),
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("codeHash", codeHash.Hex())
}

func TestGetPod(t *testing.T) {
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

	pod, _, err := cloudContract.QueryPod(1, chain.DryRunParams{
		Origin:    pk.AccountID(),
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("pod", pod)
}

func TestCloudSetPodCode(t *testing.T) {
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

	/// init pod
	podData, err := os.ReadFile("../../target/pod.release.polkavm")
	if err != nil {
		util.LogWithPurple("read file error", err)
		panic(err)
	}

	podCode, err := client.UploadInkCode(podData, &pk)
	if err != nil {
		util.LogWithPurple("UploadInkCode", err)
		panic(err)
	}

	// Deploy Pod implementation from uploaded code hash
	podImplAddr, err := client.DeployContract(
		util.InkCode{Existing: podCode},
		&pk,
		types.NewU128(*big.NewInt(0)),
		util.InkContractInput{
			Selector: "0x00000000",
			Args:     []any{},
		},
		util.NewSome(genSalt()),
	)
	if err != nil {
		util.LogWithPurple("DeployContract", err)
		panic(err)
	}

	err = cloudContract.ExecSetPodImpl(*podImplAddr, chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	})

	if err != nil {
		panic(err)
	}

	fmt.Println("set pod impl success")
}

func TestCloudUpdatePodContract(t *testing.T) {
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
	podData, err := os.ReadFile("../../target/pod.release.polkavm")
	if err != nil {
		util.LogWithPurple("read file error", err)
		panic(err)
	}

	/// upload pod code
	podCode, err := client.UploadInkCode(podData, &pk)
	if err != nil {
		util.LogWithPurple("UploadInkCode", err)
		panic(err)
	}

	// Deploy Pod implementation from uploaded code hash
	podImplAddr, err := client.DeployContract(
		util.InkCode{Existing: podCode},
		&pk,
		types.NewU128(*big.NewInt(0)),
		util.InkContractInput{
			Selector: "0x00000000",
			Args:     []any{},
		},
		util.NewSome(genSalt()),
	)
	if err != nil {
		util.LogWithPurple("DeployContract", err)
		panic(err)
	}

	cloudContract, err := cloud.InitCloudContract(client, CloudAddress)
	if err != nil {
		panic(err)
	}

	err = cloudContract.ExecSetPodImpl(*podImplAddr, chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		panic(err)
	}

	err = cloudContract.ExecUpdatePodContract(0, chain.ExecParams{
		Signer:    &pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		panic(err)
	}
}

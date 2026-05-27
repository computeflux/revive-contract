package contracts

import (
	"fmt"
	"math/big"
	"testing"

	"wetee/test/contracts/subnet"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	chain "github.com/wetee-dao/ink.go"
	"github.com/wetee-dao/ink.go/util"
)

func newSubnetIns(t *testing.T) (*subnet.Subnet, chain.SignerType) {
	t.Helper()
	cfg := loadConfig(t)
	client := newClient(t, cfg)
	pk := newSigner(t, cfg)
	ins, err := subnet.InitSubnetContract(client, cfg.Contracts.Subnet)
	if err != nil {
		t.Fatal(err)
	}
	return ins, pk
}

func dryRunParam(pk chain.SignerType) chain.DryRunParams {
	return chain.DefaultParamWithOrigin(pk.AccountID())
}

func TestSubnetQueryEpochInfo(t *testing.T) {
	ins, pk := newSubnetIns(t)
	info, _, err := ins.QueryEpochInfo(dryRunParam(pk))
	if err != nil {
		t.Fatal("QueryEpochInfo:", err)
	}
	fmt.Printf("Epoch: %d  Slot: %d  LastBlock: %d  Now: %d\n",
		info.Epoch, info.EpochSlot, info.LastEpochBlock, info.Now)
}

func TestSubnetQueryTeeChainKey(t *testing.T) {
	ins, pk := newSubnetIns(t)
	key, _, err := ins.QueryTeeChainKey(dryRunParam(pk))
	if err != nil {
		t.Fatal("QueryTeeChainKey:", err)
	}
	fmt.Println("TeeChainKey:", key.Hex())
}

func TestSubnetQueryRegions(t *testing.T) {
	ins, pk := newSubnetIns(t)
	regions, _, err := ins.QueryRegions(dryRunParam(pk))
	if err != nil {
		t.Fatal("QueryRegions:", err)
	}
	for _, r := range *regions {
		fmt.Printf("Region %d: %s\n", r.F0, string(r.F1))
	}
}

func TestSubnetQueryRegion(t *testing.T) {
	ins, pk := newSubnetIns(t)
	region, _, err := ins.QueryRegion(0, dryRunParam(pk))
	if err != nil {
		t.Fatal("QueryRegion:", err)
	}
	if region.IsSome() {
		v, _ := region.UnWrap()
		fmt.Println("Region 0:", string(v))
	} else {
		fmt.Println("Region 0 not found")
	}
}

func TestSubnetQueryLevelPrice(t *testing.T) {
	ins, pk := newSubnetIns(t)
	price, _, err := ins.QueryLevelPrice(0, dryRunParam(pk))
	if err != nil {
		t.Fatal("QueryLevelPrice:", err)
	}
	if price.IsSome() {
		v, _ := price.UnWrap()
		fmt.Printf("Level0: cpu=%d cvm_cpu=%d mem=%d cvm_mem=%d disk=%d gpu=%d\n",
			v.CpuPer, v.CvmCpuPer, v.MemoryPer, v.CvmMemoryPer, v.DiskPer, v.GpuPer)
	} else {
		fmt.Println("Level 0 not set")
	}
}

func TestSubnetQueryAsset(t *testing.T) {
	ins, pk := newSubnetIns(t)
	asset, _, err := ins.QueryAsset(0, dryRunParam(pk))
	if err != nil {
		t.Fatal("QueryAsset:", err)
	}
	if asset.IsSome() {
		v, _ := asset.UnWrap()
		fmt.Printf("Asset0 price: %s\n", v.F1.String())
	} else {
		fmt.Println("Asset 0 not found")
	}
}

func TestSubnetQueryWorker(t *testing.T) {
	ins, pk := newSubnetIns(t)
	w, _, err := ins.QueryWorker(0, dryRunParam(pk))
	if err != nil {
		t.Fatal("QueryWorker:", err)
	}
	if w.IsSome() {
		v, _ := w.UnWrap()
		fmt.Printf("Worker0: name=%s level=%d port=%d status=%d\n",
			string(v.Name), v.Level, v.Port, v.Status)
	} else {
		fmt.Println("Worker 0 not found")
	}
}

func TestSubnetQueryWorkers(t *testing.T) {
	ins, pk := newSubnetIns(t)
	workers, _, err := ins.QueryWorkers(util.NewNone[uint64](), 10, dryRunParam(pk))
	if err != nil {
		t.Fatal("QueryWorkers:", err)
	}
	fmt.Printf("Workers count: %d\n", len(*workers))
}

func TestSubnetQueryUserWorker(t *testing.T) {
	ins, pk := newSubnetIns(t)
	h160, _ := util.H160FromPublicKey(pk.Public())
	uw, _, err := ins.QueryUserWorker(h160, dryRunParam(pk))
	if err != nil {
		t.Fatal("QueryUserWorker:", err)
	}
	if uw.IsSome() {
		v, _ := uw.UnWrap()
		fmt.Printf("UserWorker: id=%d name=%s\n", v.F0, string(v.F1.Name))
	} else {
		fmt.Println("No worker for user")
	}
}

func TestSubnetQueryMintWorker(t *testing.T) {
	ins, pk := newSubnetIns(t)
	mw, _, err := ins.QueryMintWorker(pk.AccountID(), dryRunParam(pk))
	if err != nil {
		t.Fatal("QueryMintWorker:", err)
	}
	if mw.IsSome() {
		v, _ := mw.UnWrap()
		fmt.Printf("MintWorker: id=%d name=%s\n", v.F0, string(v.F1.Name))
	} else {
		fmt.Println("No mint worker")
	}
}

func TestSubnetQueryBootNodes(t *testing.T) {
	ins, pk := newSubnetIns(t)
	nodes, _, err := ins.QueryBootNodes(dryRunParam(pk))
	if err != nil {
		t.Fatal("QueryBootNodes:", err)
	}
	if !nodes.IsErr {
		fmt.Printf("BootNodes count: %d\n", len(nodes.V))
	}
}

func TestSubnetQuerySecrets(t *testing.T) {
	ins, pk := newSubnetIns(t)
	secrets, _, err := ins.QuerySecrets(dryRunParam(pk))
	if err != nil {
		t.Fatal("QuerySecrets:", err)
	}
	fmt.Printf("Secrets count: %d\n", len(*secrets))
}

func TestSubnetQueryGetPendingSecrets(t *testing.T) {
	ins, pk := newSubnetIns(t)
	pending, _, err := ins.QueryGetPendingSecrets(dryRunParam(pk))
	if err != nil {
		t.Fatal("QueryGetPendingSecrets:", err)
	}
	fmt.Printf("Pending secrets count: %d\n", len(*pending))
}

func TestSubnetQueryValidators(t *testing.T) {
	ins, pk := newSubnetIns(t)
	validators, _, err := ins.QueryValidators(dryRunParam(pk))
	if err != nil {
		t.Fatal("QueryValidators:", err)
	}
	fmt.Printf("Validators count: %d\n", len(*validators))
}

func TestSubnetQueryNextEpochValidators(t *testing.T) {
	ins, pk := newSubnetIns(t)
	next, _, err := ins.QueryNextEpochValidators(dryRunParam(pk))
	if err != nil {
		t.Fatal("QueryNextEpochValidators:", err)
	}
	if !next.IsErr {
		fmt.Printf("Next epoch validators count: %d\n", len(next.V))
	}
}

func TestSetEpoch(t *testing.T) {
	ins, pk := newSubnetIns(t)
	err := ins.ExecSetEpochSlot(7000, chain.ExecParams{
		Signer:    pk,
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		t.Fatal("SetEpoch:", err)
	}
}

func TestSubnetQueryAll(t *testing.T) {
	fmt.Println("=== Subnet Queries ===")
	t.Run("EpochInfo", TestSubnetQueryEpochInfo)
	t.Run("TeeChainKey", TestSubnetQueryTeeChainKey)
	t.Run("Regions", TestSubnetQueryRegions)
	t.Run("Region", TestSubnetQueryRegion)
	t.Run("LevelPrice", TestSubnetQueryLevelPrice)
	t.Run("Asset", TestSubnetQueryAsset)
	t.Run("Worker", TestSubnetQueryWorker)
	t.Run("Workers", TestSubnetQueryWorkers)
	t.Run("UserWorker", TestSubnetQueryUserWorker)
	t.Run("MintWorker", TestSubnetQueryMintWorker)
	t.Run("BootNodes", TestSubnetQueryBootNodes)
	t.Run("Secrets", TestSubnetQuerySecrets)
	t.Run("PendingSecrets", TestSubnetQueryGetPendingSecrets)
	t.Run("Validators", TestSubnetQueryValidators)
	t.Run("NextEpochValidators", TestSubnetQueryNextEpochValidators)
}

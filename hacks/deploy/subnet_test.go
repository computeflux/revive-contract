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

// TestSubnetQueryEpochInfo tests querying epoch information from subnet contract
func TestSubnetQueryEpochInfo(t *testing.T) {
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
		util.LogWithPurple("InitSubnetContract", err)
		panic(err)
	}

	epochInfo, _, err := subnetIns.QueryEpochInfo(chain.DryRunParams{
		Origin:    pk.AccountID(),
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		util.LogWithPurple("QueryEpochInfo", err)
		panic(err)
	}

	fmt.Printf("Epoch Info:\n")
	fmt.Printf("  Epoch: %d\n", epochInfo.Epoch)
	fmt.Printf("  EpochSlot: %d\n", epochInfo.EpochSlot)
	fmt.Printf("  LastEpochBlock: %d\n", epochInfo.LastEpochBlock)
	fmt.Printf("  Now: %d\n", epochInfo.Now)
	fmt.Printf("  SideChainPub: %s\n", epochInfo.SideChainPub.Hex())
}

// TestSubnetQuerySideChainKey tests querying side chain key from subnet contract
func TestSubnetQuerySideChainKey(t *testing.T) {
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
		util.LogWithPurple("InitSubnetContract", err)
		panic(err)
	}

	sideChainKey, _, err := subnetIns.QuerySideChainKey(chain.DryRunParams{
		Origin:    pk.AccountID(),
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		util.LogWithPurple("QuerySideChainKey", err)
		panic(err)
	}

	fmt.Printf("Side Chain Key: %s\n", sideChainKey.Hex())
}

// TestSubnetQueryRegions tests querying all regions from subnet contract
func TestSubnetQueryRegions(t *testing.T) {
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
		util.LogWithPurple("InitSubnetContract", err)
		panic(err)
	}

	regions, _, err := subnetIns.QueryRegions(chain.DryRunParams{
		Origin:    pk.AccountID(),
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		util.LogWithPurple("QueryRegions", err)
		panic(err)
	}

	fmt.Printf("Regions:\n")
	for _, region := range *regions {
		fmt.Printf("  ID: %d, Name: %s\n", region.F0, string(region.F1))
	}
}

// TestSubnetQueryRegion tests querying a specific region by ID
func TestSubnetQueryRegion(t *testing.T) {
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
		util.LogWithPurple("InitSubnetContract", err)
		panic(err)
	}

	region, _, err := subnetIns.QueryRegion(0, chain.DryRunParams{
		Origin:    pk.AccountID(),
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		util.LogWithPurple("QueryRegion", err)
		panic(err)
	}

	if region.IsSome() {
		v, _ := region.UnWrap()
		fmt.Printf("Region 0: %s\n", string(v))
	} else {
		fmt.Println("Region 0 not found")
	}
}

// TestSubnetQueryLevelPrice tests querying price for a specific level
func TestSubnetQueryLevelPrice(t *testing.T) {
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
		util.LogWithPurple("InitSubnetContract", err)
		panic(err)
	}

	price, _, err := subnetIns.QueryLevelPrice(0, chain.DryRunParams{
		Origin:    pk.AccountID(),
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		util.LogWithPurple("QueryLevelPrice", err)
		panic(err)
	}

	if price.IsSome() {
		v, _ := price.UnWrap()
		fmt.Printf("Level 0 Price:\n")
		fmt.Printf("  CpuPer: %d\n", v.CpuPer)
		fmt.Printf("  CvmCpuPer: %d\n", v.CvmCpuPer)
		fmt.Printf("  MemoryPer: %d\n", v.MemoryPer)
		fmt.Printf("  CvmMemoryPer: %d\n", v.CvmMemoryPer)
		fmt.Printf("  DiskPer: %d\n", v.DiskPer)
		fmt.Printf("  GpuPer: %d\n", v.GpuPer)
	} else {
		fmt.Println("Level 0 price not set")
	}
}

// TestSubnetQueryAsset tests querying asset information by ID
func TestSubnetQueryAsset(t *testing.T) {
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
		util.LogWithPurple("InitSubnetContract", err)
		panic(err)
	}

	asset, _, err := subnetIns.QueryAsset(0, chain.DryRunParams{
		Origin:    pk.AccountID(),
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		util.LogWithPurple("QueryAsset", err)
		panic(err)
	}

	if asset.IsSome() {
		v, _ := asset.UnWrap()
		fmt.Printf("Asset 0:\n")
		if v.F0.Native != nil {
			fmt.Printf("  Type: Native, Name: %s\n", string(*v.F0.Native))
		}
		if v.F0.ERC20 != nil {
			fmt.Printf("  Type: ERC20, Contract: %s\n", v.F0.ERC20.F1.Hex())
		}
		fmt.Printf("  Price: %s\n", v.F1.String())
	} else {
		fmt.Println("Asset 0 not found")
	}
}

// TestSubnetQueryWorker tests querying worker information by ID
func TestSubnetQueryWorker(t *testing.T) {
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
		util.LogWithPurple("InitSubnetContract", err)
		panic(err)
	}

	worker, _, err := subnetIns.QueryWorker(0, chain.DryRunParams{
		Origin:    pk.AccountID(),
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		util.LogWithPurple("QueryWorker", err)
		panic(err)
	}

	if worker.IsSome() {
		v, _ := worker.UnWrap()
		fmt.Printf("Worker 0:\n")
		fmt.Printf("  Name: %s\n", string(v.Name))
		fmt.Printf("  Owner: %s\n", v.Owner.Hex())
		fmt.Printf("  Level: %d\n", v.Level)
		fmt.Printf("  RegionId: %d\n", v.RegionId)
		fmt.Printf("  StartBlock: %d\n", v.StartBlock)
		fmt.Printf("  Port: %d\n", v.Port)
		fmt.Printf("  Status: %d\n", v.Status)
	} else {
		fmt.Println("Worker 0 not found")
	}
}

// TestSubnetQueryWorkers tests querying workers list with pagination
func TestSubnetQueryWorkers(t *testing.T) {
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
		util.LogWithPurple("InitSubnetContract", err)
		panic(err)
	}

	workers, _, err := subnetIns.QueryWorkers(util.NewNone[uint64](), 10, chain.DryRunParams{
		Origin:    pk.AccountID(),
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		util.LogWithPurple("QueryWorkers", err)
		panic(err)
	}

	fmt.Printf("Workers (first 10):\n")
	for _, w := range *workers {
		fmt.Printf("  ID: %d, Name: %s, Status: %d\n", w.F0, string(w.F1.Name), w.F1.Status)
	}
}

// TestSubnetQueryUserWorker tests querying worker by user address
func TestSubnetQueryUserWorker(t *testing.T) {
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
		util.LogWithPurple("InitSubnetContract", err)
		panic(err)
	}

	userWorker, _, err := subnetIns.QueryUserWorker(pk.H160Address(), chain.DryRunParams{
		Origin:    pk.AccountID(),
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		util.LogWithPurple("QueryUserWorker", err)
		panic(err)
	}

	if userWorker.IsSome() {
		v, _ := userWorker.UnWrap()
		fmt.Printf("User Worker:\n")
		fmt.Printf("  ID: %d, Name: %s\n", v.F0, string(v.F1.Name))
	} else {
		fmt.Println("No worker found for user")
	}
}

// TestSubnetQueryMintWorker tests querying worker by mint account ID
func TestSubnetQueryMintWorker(t *testing.T) {
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
		util.LogWithPurple("InitSubnetContract", err)
		panic(err)
	}

	// Query with Alice's account ID
	mintWorker, _, err := subnetIns.QueryMintWorker(pk.AccountID(), chain.DryRunParams{
		Origin:    pk.AccountID(),
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		util.LogWithPurple("QueryMintWorker", err)
		panic(err)
	}

	if mintWorker.IsSome() {
		v, _ := mintWorker.UnWrap()
		fmt.Printf("Mint Worker:\n")
		fmt.Printf("  ID: %d, Name: %s\n", v.F0, string(v.F1.Name))
	} else {
		fmt.Println("No worker found for mint account")
	}
}

// TestSubnetQueryBootNodes tests querying boot nodes
func TestSubnetQueryBootNodes(t *testing.T) {
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
		util.LogWithPurple("InitSubnetContract", err)
		panic(err)
	}

	bootNodes, _, err := subnetIns.QueryBootNodes(chain.DryRunParams{
		Origin:    pk.AccountID(),
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		util.LogWithPurple("QueryBootNodes", err)
		panic(err)
	}

	if !bootNodes.IsErr {
		fmt.Printf("Boot Nodes:\n")
		for _, node := range bootNodes.V {
			fmt.Printf("  Name: %s, Validator: %x, P2P: %x\n",
				string(node.Name), node.ValidatorId, node.P2pId)
		}
	} else {
		fmt.Printf("Error: %s\n", bootNodes.E.Error())
	}
}

// TestSubnetQuerySecrets tests querying all secret nodes
func TestSubnetQuerySecrets(t *testing.T) {
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
		util.LogWithPurple("InitSubnetContract", err)
		panic(err)
	}

	secrets, _, err := subnetIns.QuerySecrets(chain.DryRunParams{
		Origin:    pk.AccountID(),
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		util.LogWithPurple("QuerySecrets", err)
		panic(err)
	}

	fmt.Printf("Secrets:\n")
	for _, s := range *secrets {
		fmt.Printf("  ID: %d, Name: %s, Status: %d\n", s.F0, string(s.F1.Name), s.F1.Status)
	}
}

// TestSubnetQueryGetPendingSecrets tests querying pending secrets
func TestSubnetQueryGetPendingSecrets(t *testing.T) {
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
		util.LogWithPurple("InitSubnetContract", err)
		panic(err)
	}

	pendingSecrets, _, err := subnetIns.QueryGetPendingSecrets(chain.DryRunParams{
		Origin:    pk.AccountID(),
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		util.LogWithPurple("QueryGetPendingSecrets", err)
		panic(err)
	}

	fmt.Printf("Pending Secrets:\n")
	for _, ps := range *pendingSecrets {
		fmt.Printf("  Secret ID: %d, Block: %d\n", ps.F0, ps.F1)
	}
}

// TestSubnetQueryValidators tests querying all validators
func TestSubnetQueryValidators(t *testing.T) {
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
		util.LogWithPurple("InitSubnetContract", err)
		panic(err)
	}

	validators, _, err := subnetIns.QueryValidators(chain.DryRunParams{
		Origin:    pk.AccountID(),
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		util.LogWithPurple("QueryValidators", err)
		panic(err)
	}

	fmt.Printf("Validators:\n")
	for _, v := range *validators {
		fmt.Printf("  ID: %d, Name: %s, Block: %d\n", v.F0, string(v.F1.Name), v.F2)
	}
}

// TestSubnetQueryNextEpochValidators tests querying next epoch validators
func TestSubnetQueryNextEpochValidators(t *testing.T) {
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
		util.LogWithPurple("InitSubnetContract", err)
		panic(err)
	}

	nextValidators, _, err := subnetIns.QueryNextEpochValidators(chain.DryRunParams{
		Origin:    pk.AccountID(),
		PayAmount: types.NewU128(*big.NewInt(0)),
	})
	if err != nil {
		util.LogWithPurple("QueryNextEpochValidators", err)
		panic(err)
	}

	if !nextValidators.IsErr {
		fmt.Printf("Next Epoch Validators:\n")
		for _, v := range nextValidators.V {
			fmt.Printf("  ID: %d, Name: %s, Block: %d\n", v.F0, string(v.F1.Name), v.F2)
		}
	} else {
		fmt.Printf("Error: %s\n", nextValidators.E.Error())
	}
}

// TestSubnetQueryAll runs all query tests sequentially
func TestSubnetQueryAll(t *testing.T) {
	fmt.Println("=== Testing Subnet Contract Queries ===")

	t.Run("EpochInfo", TestSubnetQueryEpochInfo)
	t.Run("SideChainKey", TestSubnetQuerySideChainKey)
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

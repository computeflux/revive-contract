// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

import "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";
import "../lib/Types.sol";
import "../lib/List2D.sol";

contract Subnet is UUPSUpgradeable {
    using List2D for List2D.List;
    address public govContract;
    uint32 public epochSlot;
    uint32 public epoch;
    uint256 public lastEpochBlock;
    address public sideChainMultiKey;
    uint32 public nextRegionId;
    uint64 public nextWorkerId;
    uint64 public nextSecretId;
    uint32 public nextAssetId;

    mapping(uint32 => bytes) public regions;
    mapping(uint64 => uint8) public workerStatus;
    mapping(address => uint64) public ownerOfWorker;
    mapping(bytes32 => uint64) public mintOfWorker;
    mapping(address => uint64) public secretOfUser;
    mapping(uint64 => uint256) public secretMortgages;
    mapping(uint8 => RunPrice) public levelPrices;
    mapping(uint32 => AssetInfo) public assetInfos;
    mapping(uint32 => uint256) public assetPrices;
    address public tokenContract;
    address public blsContract;
    uint256 public minMortgageAmount;
    mapping(uint8 => uint256) public levelMinMortgages;

    mapping(uint64 => K8sCluster) public workers;
    mapping(uint64 => SecretNode) public secrets;

    mapping(uint32 => uint64) public bootNodes;
    uint32 public bootNodesLen;

    mapping(uint64 => uint32) public runningValidators;
    uint64[] public runningValidatorIds;
    mapping(uint64 => uint32) public pendingValidators;
    uint64[] public pendingValidatorIds;

    // region => worker ids
    mapping(uint32 => uint64[]) public regionWorkers;
    // worker => mortgages
    mapping(uint64 => List2D.List) internal workerMortgages;

    // 预留存储空间（可升级合约兼容）
    uint256[50] private __gap;

    error MustCallByMainContract();
    error WorkerNotExist();
    error WorkerNotOwnedByCaller();
    error WorkerStatusNotReady();
    error InvalidBlsKey();
    error WorkerMortgageNotExist();
    error TransferFailed();
    error NodeNotExist();
    error NodeIsRunning();
    error InvalidSideChainCaller();
    error RegionNotExist();
    error DepositNotEnough();
    error MortgageNotEnough();
    error SlashAmountTooLarge();
    error ResourceNotEnough();
    error EpochNotExpired();
    error TokenContractNotSet();
    error BlsContractNotSet();
    error CallFailed();
    error InsufficientPoints();

    modifier onlyGov() {
        if (msg.sender != govContract) revert MustCallByMainContract();
        _;
    }

    modifier onlySideChain() {
        if (msg.sender != sideChainMultiKey) revert InvalidSideChainCaller();
        _;
    }

    // ========== 初始化 ==========

    /// @custom:oz-upgrades-unsafe-allow constructor
    constructor() {
        _disableInitializers();
    }

    /**
     * @notice 代理初始化（替代 constructor）
     * @param _govContract 治理合约地址
     * @param _sideChainMultiKey 侧链多签地址
     * @param _blsContract BLS gateway 合约地址
     */
    function initialize(
        address _govContract,
        address _sideChainMultiKey,
        address _blsContract
    ) public initializer {
        __UUPSUpgradeable_init();
        govContract = _govContract;
        sideChainMultiKey = _sideChainMultiKey;
        blsContract = _blsContract;
        epochSlot = 72000;
    }

    /// @dev UUPS 升级授权：仅 gov 可升级
    function _authorizeUpgrade(address) internal override onlyGov {}

    // ========== Epoch ==========

    function epochInfo() external view returns (EpochInfo memory) {
        return
            EpochInfo({
                epoch: epoch,
                epochSlot: epochSlot,
                lastEpochBlock: lastEpochBlock,
                now: uint32(block.number),
                sideChainPub: sideChainMultiKey
            });
    }

    function setEpochSlot(uint32 _epochSlot) external onlyGov {
        epochSlot = _epochSlot;
    }

    function sideChainKey() external view returns (address) {
        return sideChainMultiKey;
    }

    // ========== Region ==========

    function setRegion(bytes calldata name) external onlyGov returns (uint32) {
        uint32 id = nextRegionId;
        nextRegionId++;
        regions[id] = name;
        return id;
    }

    function region(uint32 id) external view returns (bytes memory) {
        return regions[id];
    }

    function getRegions()
        external
        view
        returns (uint32[] memory ids, bytes[] memory names)
    {
        uint256 count = nextRegionId;
        ids = new uint32[](count);
        names = new bytes[](count);
        for (uint32 i = 0; i < count; i++) {
            ids[i] = i;
            names[i] = regions[i];
        }
        return (ids, names);
    }

    // ========== 价格 & 资产 ==========

    function setLevelPrice(
        uint8 level,
        RunPrice calldata price
    ) external onlyGov {
        levelPrices[level] = price;
    }

    function getLevelPrice(
        uint8 level
    ) external view returns (RunPrice memory) {
        return levelPrices[level];
    }

    function setAsset(
        AssetInfo calldata info,
        uint256 price
    ) external onlyGov returns (uint32) {
        uint32 id = nextAssetId;
        nextAssetId++;
        assetInfos[id] = info;
        assetPrices[id] = price;
        return id;
    }

    function asset(
        uint32 id
    ) external view returns (AssetInfo memory, uint256) {
        return (assetInfos[id], assetPrices[id]);
    }

    // ========== 配置 ==========

    function setMinMortgage(uint256 amount) external onlyGov {
        minMortgageAmount = amount;
    }

    function getMinMortgage() external view returns (uint256) {
        return minMortgageAmount;
    }

    function setLevelMinMortgage(uint8 level, uint256 amount) external onlyGov {
        levelMinMortgages[level] = amount;
    }

    function getLevelMinMortgage(uint8 level) external view returns (uint256) {
        uint256 v = levelMinMortgages[level];
        if (v != 0) return v;
        return minMortgageAmount;
    }

    // ========== Worker ==========

    function workerTotalResources(
        uint64 workerId
    ) external view returns (uint32, uint32, uint32, uint32, uint32, uint32) {
        uint64[] memory k2s = workerMortgages[workerId].listAllK2sDesc();
        uint32 totalCpu = 0;
        uint32 totalMem = 0;
        uint32 totalCvmCpu = 0;
        uint32 totalCvmMem = 0;
        uint32 totalDisk = 0;
        uint32 totalGpu = 0;
        for (uint256 i = 0; i < k2s.length; i++) {
            AssetDeposit memory dep = abi.decode(
                workerMortgages[workerId].get(k2s[i]),
                (AssetDeposit)
            );
            if (dep.deleted == 0) {
                totalCpu += dep.cpu;
                totalMem += dep.mem;
                totalCvmCpu += dep.cvmCpu;
                totalCvmMem += dep.mem;
                totalDisk += dep.disk;
                totalGpu += dep.gpu;
            }
        }
        return (
            totalCpu,
            totalMem,
            totalCvmCpu,
            totalCvmMem,
            totalDisk,
            totalGpu
        );
    }

    function workerTotalMortgage(
        uint64 workerId
    ) external view returns (uint256) {
        uint64[] memory k2s = workerMortgages[workerId].listAllK2sDesc();
        uint256 total = 0;
        for (uint256 i = 0; i < k2s.length; i++) {
            AssetDeposit memory dep = abi.decode(
                workerMortgages[workerId].get(k2s[i]),
                (AssetDeposit)
            );
            if (dep.deleted == 0) {
                total += dep.amount;
            }
        }
        return total;
    }

    function slashWorkerMortgage(
        uint64 workerId,
        uint256 amount,
        address to
    ) external onlyGov {
        uint64[] memory k2s = workerMortgages[workerId].listAllK2sDesc();
        uint256 remaining = amount;
        for (uint256 i = k2s.length; i > 0 && remaining > 0; ) {
            i--;
            AssetDeposit memory dep = abi.decode(
                workerMortgages[workerId].get(k2s[i]),
                (AssetDeposit)
            );
            if (dep.deleted != 0) continue;
            if (dep.amount <= remaining) {
                remaining -= dep.amount;
                dep.amount = 0;
                dep.deleted = uint8(block.number % 256);
            } else {
                dep.amount -= remaining;
                remaining = 0;
            }
            workerMortgages[workerId].update(k2s[i], abi.encode(dep));
        }
        if (remaining > 0) revert SlashAmountTooLarge();
        (bool success, ) = payable(to).call{value: amount}("");
        if (!success) revert TransferFailed();
    }

    function getWorker(uint64 id) external view returns (K8sCluster memory) {
        K8sCluster memory w = workers[id];
        w.status = workerStatus[id];
        return w;
    }

    function getWorkers(
        uint64 start,
        uint64 size
    ) external view returns (K8sCluster[] memory, uint64[] memory) {
        uint64 total = nextWorkerId;
        if (total == 0 || size == 0)
            return (new K8sCluster[](0), new uint64[](0));
        uint64 cur = start >= total ? total - 1 : start;
        uint64 count = 0;
        uint64 temp = cur;
        while (true) {
            if (workers[temp].owner != address(0)) count++;
            if (temp == 0 || count >= size) break;
            temp--;
        }
        K8sCluster[] memory out = new K8sCluster[](count);
        uint64[] memory ids = new uint64[](count);
        uint256 idx = 0;
        while (true) {
            if (workers[cur].owner != address(0)) {
                K8sCluster memory w = workers[cur];
                w.status = workerStatus[cur];
                out[idx] = w;
                ids[idx] = cur;
                idx++;
            }
            if (cur == 0 || idx >= count) break;
            cur--;
        }
        return (out, ids);
    }

    function userWorker(
        address user
    ) external view returns (uint64, K8sCluster memory) {
        uint64 id = ownerOfWorker[user];
        K8sCluster memory w = workers[id];
        w.status = workerStatus[id];
        return (id, w);
    }

    function mintWorker(
        bytes32 id
    ) external view returns (uint64, K8sCluster memory) {
        uint64 wid = mintOfWorker[id];
        K8sCluster memory w = workers[wid];
        w.status = workerStatus[wid];
        return (wid, w);
    }

    function workerRegister(
        bytes calldata name,
        bytes32 p2pId,
        Ip calldata ip,
        uint32 port,
        uint8 level,
        uint32 regionId
    ) external returns (uint64) {
        if (regions[regionId].length == 0) revert RegionNotExist();
        if (ownerOfWorker[msg.sender] != 0) revert WorkerNotOwnedByCaller();
        if (nextWorkerId == type(uint64).max) revert WorkerNotExist();
        uint64 wid = nextWorkerId;
        nextWorkerId++;
        workers[wid] = K8sCluster({
            id: wid,
            name: name,
            owner: msg.sender,
            level: level,
            regionId: regionId,
            startBlock: block.number,
            stopBlock: 0,
            terminalBlock: 0,
            p2pId: p2pId,
            ip: ip,
            port: port,
            status: 0
        });
        ownerOfWorker[msg.sender] = wid;
        mintOfWorker[p2pId] = wid;
        regionWorkers[regionId].push(wid);
        return wid;
    }

    function workerUpdate(
        uint64 id,
        bytes calldata name,
        Ip calldata ip,
        uint32 port
    ) external {
        K8sCluster storage w = workers[id];
        if (w.owner == address(0)) revert WorkerNotExist();
        if (w.owner != msg.sender) revert WorkerNotOwnedByCaller();
        w.name = name;
        w.ip = ip;
        w.port = port;
    }

    function workerMortgage(
        uint64 id,
        uint32 cpu,
        uint32 mem,
        uint32 cvmCpu,
        uint32 cvmMem,
        uint32 disk,
        uint32 gpu,
        uint256 deposit
    ) external payable returns (uint64) {
        K8sCluster storage w = workers[id];
        if (w.owner == address(0)) revert WorkerNotExist();
        if (w.owner != msg.sender) revert WorkerNotOwnedByCaller();
        uint8 status = workerStatus[id];
        if (status != 0 && status != 2) revert WorkerStatusNotReady();
        if (msg.value < deposit) revert DepositNotEnough();
        AssetDeposit memory dep = AssetDeposit({
            amount: msg.value,
            cpu: cpu,
            cvmCpu: cvmCpu,
            mem: mem,
            cvmMem: cvmMem,
            disk: disk,
            gpu: gpu,
            deleted: 0
        });
        return workerMortgages[id].insert(abi.encode(dep));
    }

    function workerUnmortgage(
        uint64 workerId,
        uint64 mortgageId
    ) external returns (uint64) {
        K8sCluster storage w = workers[workerId];
        if (w.owner == address(0)) revert WorkerNotExist();
        if (w.owner != msg.sender) revert WorkerNotOwnedByCaller();
        uint8 status = workerStatus[workerId];
        if (status != 0 && status != 2) revert WorkerStatusNotReady();
        AssetDeposit memory dep = abi.decode(
            workerMortgages[workerId].get(mortgageId),
            (AssetDeposit)
        );
        if (dep.deleted != 0) revert WorkerMortgageNotExist();
        dep.deleted = uint8(block.number % 256);
        workerMortgages[workerId].update(mortgageId, abi.encode(dep));
        (bool success, ) = payable(w.owner).call{value: dep.amount}("");
        if (!success) revert TransferFailed();
        return mortgageId;
    }

    function workerStart(uint64 id) external onlySideChain {
        if (workers[id].owner == address(0)) revert WorkerNotExist();
        uint256 total = this.workerTotalMortgage(id);
        uint256 min = levelMinMortgages[workers[id].level];
        if (min == 0) min = minMortgageAmount;
        if (total < min) revert MortgageNotEnough();
        (
            uint32 cpu,
            uint32 mem,
            uint32 cvmCpu,
            uint32 cvmMem,
            uint32 disk,
            uint32 gpu
        ) = this.workerTotalResources(id);
        if (
            cpu == 0 &&
            mem == 0 &&
            cvmCpu == 0 &&
            cvmMem == 0 &&
            disk == 0 &&
            gpu == 0
        ) revert ResourceNotEnough();
        workerStatus[id] = 1;
    }

    function workerStop(uint64 id) external returns (uint64) {
        K8sCluster storage w = workers[id];
        if (w.owner == address(0)) revert WorkerNotExist();
        if (w.owner != msg.sender) revert WorkerNotOwnedByCaller();
        if (workerStatus[id] != 1) revert WorkerStatusNotReady();
        workerStatus[id] = 2;
        w.stopBlock = block.number;
        return id;
    }

    // ========== Boot Nodes ==========

    function setBootNodes(uint64[] calldata nodes) external onlyGov {
        uint32 len = 0;
        for (uint32 i = 0; i < nodes.length; i++) {
            bool dup = false;
            for (uint32 j = 0; j < len; j++) {
                if (bootNodes[j] == nodes[i]) {
                    dup = true;
                    break;
                }
            }
            if (!dup) {
                bootNodes[len] = nodes[i];
                len++;
            }
        }
        bootNodesLen = len;
    }

    function getBootNodes() external view returns (SecretNode[] memory) {
        SecretNode[] memory out = new SecretNode[](bootNodesLen);
        for (uint32 i = 0; i < bootNodesLen; i++) {
            out[i] = secrets[bootNodes[i]];
        }
        return out;
    }

    // ========== Secrets ==========

    function getSecrets(
        uint256 start,
        uint256 size
    ) external view returns (uint64[] memory ids, SecretNode[] memory nodes) {
        uint64 total = nextSecretId;
        if (total == 0 || size == 0) {
            return (new uint64[](0), new SecretNode[](0));
        }
        uint64 cur = start >= total ? total - 1 : uint64(start);
        uint64 count = 0;
        uint64 temp = cur;
        while (true) {
            if (secrets[temp].owner != address(0)) count++;
            if (temp == 0 || count >= uint64(size)) break;
            temp--;
        }
        ids = new uint64[](count);
        nodes = new SecretNode[](count);
        uint256 idx = 0;
        while (true) {
            if (secrets[cur].owner != address(0)) {
                ids[idx] = cur;
                nodes[idx] = secrets[cur];
                idx++;
            }
            if (cur == 0 || idx >= count) break;
            cur--;
        }
        return (ids, nodes);
    }

    function getPendingSecrets()
        external
        view
        returns (uint64[] memory, uint32[] memory)
    {
        uint256 len = pendingValidatorIds.length;
        uint64[] memory ids = new uint64[](len);
        uint32[] memory powers = new uint32[](len);
        for (uint256 i = 0; i < len; i++) {
            ids[i] = pendingValidatorIds[i];
            powers[i] = pendingValidators[pendingValidatorIds[i]];
        }
        return (ids, powers);
    }

    function secretRegister(
        bytes calldata name,
        bytes32 validatorId,
        bytes32 p2pId,
        Ip calldata ip,
        uint32 port,
        bytes calldata bls
    ) external returns (uint64) {
        if (nextSecretId == type(uint64).max) revert NodeNotExist();
        if (bls.length == 0) revert InvalidBlsKey();
        uint64 id = nextSecretId;
        nextSecretId++;
        secrets[id] = SecretNode({
            id: id,
            name: name,
            owner: msg.sender,
            level: 0,
            regionId: 0,
            stopBlock: 0,
            validatorId: validatorId,
            p2pId: p2pId,
            startBlock: block.number,
            terminalBlock: 0,
            ip: ip,
            port: port,
            status: 0,
            bls: bls
        });
        secretOfUser[msg.sender] = id;
        if (id == 0) {
            runningValidators[0] = 1;
            runningValidatorIds.push(0);
        }
        return id;
    }

    function secretUpdate(
        uint64 id,
        bytes calldata name,
        Ip calldata ip,
        uint32 port
    ) external {
        SecretNode storage node = secrets[id];
        if (node.owner == address(0)) revert NodeNotExist();
        if (node.owner != msg.sender) revert WorkerNotOwnedByCaller();
        node.name = name;
        node.ip = ip;
        node.port = port;
    }

    function secretDeposit(uint64 id, uint256 deposit) external payable {
        SecretNode storage node = secrets[id];
        if (node.owner == address(0)) revert NodeNotExist();
        if (node.owner != msg.sender) revert WorkerNotOwnedByCaller();
        if (msg.value < deposit) revert DepositNotEnough();
        secretMortgages[id] += msg.value;
    }

    function secretDelete(uint64 id) external {
        SecretNode storage node = secrets[id];
        if (node.owner == address(0)) revert NodeNotExist();
        if (node.owner != msg.sender) revert WorkerNotOwnedByCaller();
        for (uint256 i = 0; i < runningValidatorIds.length; i++) {
            if (runningValidatorIds[i] == id) revert NodeIsRunning();
        }
        for (uint256 i = 0; i < pendingValidatorIds.length; i++) {
            if (pendingValidatorIds[i] == id) revert NodeIsRunning();
        }
        if (secretMortgages[id] != 0) revert NodeIsRunning();
        node.terminalBlock = block.number;
    }

    // ========== Validators ==========

    function getValidators()
        external
        view
        returns (uint64[] memory, SecretNode[] memory, uint32[] memory)
    {
        uint256 count = 0;
        for (uint256 i = 0; i < runningValidatorIds.length; i++) {
            if (runningValidators[runningValidatorIds[i]] > 0) count++;
        }
        uint64[] memory ids = new uint64[](count);
        SecretNode[] memory nodes = new SecretNode[](count);
        uint32[] memory powers = new uint32[](count);
        uint256 idx = 0;
        for (uint256 i = 0; i < runningValidatorIds.length; i++) {
            uint64 vid = runningValidatorIds[i];
            uint32 pow = runningValidators[vid];
            if (pow > 0) {
                ids[idx] = vid;
                nodes[idx] = secrets[vid];
                powers[idx] = pow;
                idx++;
            }
        }
        return (ids, nodes, powers);
    }

    function validatorJoin(uint64 id) external onlyGov {
        if (secrets[id].owner == address(0)) revert NodeNotExist();
        bool found = false;
        for (uint256 i = 0; i < pendingValidatorIds.length; i++) {
            if (pendingValidatorIds[i] == id) {
                found = true;
                break;
            }
        }
        if (!found) {
            pendingValidatorIds.push(id);
        }
        pendingValidators[id] = 1;
    }

    function validatorDelete(uint64 id) external onlyGov {
        bool found = false;
        for (uint256 i = 0; i < pendingValidatorIds.length; i++) {
            if (pendingValidatorIds[i] == id) {
                found = true;
                break;
            }
        }
        if (!found) {
            pendingValidatorIds.push(id);
        }
        pendingValidators[id] = 0;
    }

    function setNextEpoch(uint64) external {
        if (sideChainMultiKey == address(0)) {
            sideChainMultiKey = msg.sender;
        } else {
            if (msg.sender != sideChainMultiKey)
                revert InvalidSideChainCaller();
        }
        uint256 slot = epochSlot;
        if (block.number - lastEpochBlock < slot) revert EpochNotExpired();
        epoch++;
        lastEpochBlock = block.number;
        _calcNewValidators();
    }

    function nextEpochValidators()
        external
        view
        returns (uint64[] memory, SecretNode[] memory, uint32[] memory)
    {
        if (block.number - lastEpochBlock < epochSlot - 5)
            revert EpochNotExpired();
        return _calcValidators();
    }

    function _calcNewValidators() internal {
        (uint64[] memory ids, , uint32[] memory powers) = _calcValidators();
        for (uint256 i = 0; i < runningValidatorIds.length; i++) {
            runningValidators[runningValidatorIds[i]] = 0;
        }
        delete runningValidatorIds;
        for (uint256 i = 0; i < ids.length; i++) {
            if (powers[i] > 0) {
                runningValidators[ids[i]] = powers[i];
                runningValidatorIds.push(ids[i]);
            }
        }
        delete pendingValidatorIds;
    }

    function _calcValidators()
        internal
        view
        returns (uint64[] memory, SecretNode[] memory, uint32[] memory)
    {
        uint256 runningLen = runningValidatorIds.length;
        uint256 pendingLen = pendingValidatorIds.length;
        uint64[] memory tempIds = new uint64[](runningLen + pendingLen);
        uint32[] memory tempPowers = new uint32[](runningLen + pendingLen);
        uint256 count = 0;
        for (uint256 i = 0; i < runningLen; i++) {
            uint64 id = runningValidatorIds[i];
            tempIds[count] = id;
            tempPowers[count] = runningValidators[id];
            count++;
        }
        for (uint256 i = 0; i < pendingLen; i++) {
            uint64 id = pendingValidatorIds[i];
            bool found = false;
            for (uint256 j = 0; j < count; j++) {
                if (tempIds[j] == id) {
                    tempPowers[j] = pendingValidators[id];
                    found = true;
                    break;
                }
            }
            if (!found) {
                tempIds[count] = id;
                tempPowers[count] = pendingValidators[id];
                count++;
            }
        }
        uint256 finalCount = 0;
        for (uint256 i = 0; i < count; i++) {
            if (tempPowers[i] > 0) finalCount++;
        }
        uint64[] memory ids = new uint64[](finalCount);
        SecretNode[] memory nodes = new SecretNode[](finalCount);
        uint32[] memory powers = new uint32[](finalCount);
        uint256 idx = 0;
        for (uint256 i = 0; i < count; i++) {
            if (tempPowers[i] > 0) {
                ids[idx] = tempIds[i];
                nodes[idx] = secrets[tempIds[i]];
                powers[idx] = tempPowers[i];
                idx++;
            }
        }
        return (ids, nodes, powers);
    }

    // ========== Token 积分提现 ==========

    function setTokenContract(address _token) external onlyGov {
        tokenContract = _token;
    }

    function setBlsContract(address _bls) external onlyGov {
        blsContract = _bls;
    }

    function getBlsContract() external view returns (address) {
        return blsContract;
    }

    function setBlsAggPubkey(bytes calldata _aggPubkey) external onlyGov {
        if (blsContract == address(0)) revert BlsContractNotSet();
        (bool success, ) = blsContract.call(abi.encodeWithSignature("setAggPubkey(bytes)", _aggPubkey));
        if (!success) revert CallFailed();
    }

    function withdrawToken(address user, uint256 ethAmount) external onlyGov {
        if (tokenContract == address(0)) revert TokenContractNotSet();

        (bool ok, ) = tokenContract.call(
            abi.encodeWithSignature(
                "withdraw(address,uint256)",
                user,
                ethAmount
            )
        );
        if (!ok) revert TransferFailed();
    }

    receive() external payable {}
}

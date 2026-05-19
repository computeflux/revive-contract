//! Subnet 合约 — PolkaVM/wrevive 迁移版
//! 从 inks/Subnet 迁移，使用 wrevive-api + wrevive-macro。

#![cfg_attr(not(test), no_std)]
#![cfg_attr(not(test), no_main)]

extern crate alloc;

#[cfg(all(not(test), not(feature = "api")))]
#[global_allocator]
static ALLOC: pvm_bump_allocator::BumpAllocator<65536> = pvm_bump_allocator::BumpAllocator::new();

mod datas;
mod errors;

use wrevive_api::*;
use wrevive_macro::{list_2d, mapping, revive_contract, storage};

pub use datas::{
    AssetDeposit, AssetInfo, EpochInfo, Ip, K8sCluster, LevelRequirement, NodeID, RunPrice,
    SecretNode,
};
pub use errors::Error;
pub use primitives::{ensure, ok_or_err};

#[revive_contract]
pub mod subnet {
    use super::*;
    use crate::datas::NodeID;
    use crate::{Error, ensure};

    const GOV_CONTRACT: Storage<Address> = storage!(b"gov_contract");
    const EPOCH_SLOT: Storage<u32> = storage!(b"epoch_slot");
    const EPOCH: Storage<u32> = storage!(b"epoch");
    const LAST_EPOCH_BLOCK: Storage<BlockNumber> = storage!(b"last_epoch_block");
    const TEE_CHAIN_MULTI_KEY: Storage<Address> = storage!(b"tee_chain_multi_key");
    const NEXT_REGION_ID: Storage<u32> = storage!(b"next_region_id");
    const NEXT_WORKER_ID: Storage<u64> = storage!(b"next_worker_id");
    const NEXT_SECRET_ID: Storage<u64> = storage!(b"next_secret_id");
    const NEXT_ASSET_ID: Storage<u32> = storage!(b"next_asset_id");

    const REGIONS: Mapping<u32, Bytes> = mapping!(b"regions");
    const WORKER_STATUS: Mapping<u64, u8> = mapping!(b"worker_status");
    const OWNER_OF_WORKER: Mapping<Address, u64> = mapping!(b"owner_of_worker");
    const MINT_OF_WORKER: Mapping<AccountId, u64> = mapping!(b"mint_of_worker");
    const SECRET_MORTGAGES: Mapping<u64, U256> = mapping!(b"secret_mortgages");
    const LEVEL_PRICES: Mapping<u8, RunPrice> = mapping!(b"level_prices");
    const ASSET_INFOS: Mapping<u32, AssetInfo> = mapping!(b"asset_infos");
    const ASSET_PRICES: Mapping<u32, U256> = mapping!(b"asset_prices");
    const MIN_MORTGAGE_AMOUNT: Storage<U256> = storage!(b"min_mortgage_amount");
    const LEVEL_MIN_MORTGAGES: Mapping<u8, U256> = mapping!(b"level_min_mortgages");

    const WORKERS: Mapping<u64, K8sCluster> = mapping!(b"workers");
    const SECRETS: Mapping<u64, SecretNode> = mapping!(b"secrets");

    const REGION_WORKERS: List2D<u32, u32, u64> = list_2d!(b"region_workers");
    const WORKER_MORTGAGES: List2D<u64, u32, AssetDeposit> = list_2d!(b"worker_mortgages");

    const BOOT_NODES: Mapping<u32, u64> = mapping!(b"boot_nodes");
    const BOOT_NODES_LEN: Storage<u32> = storage!(b"boot_nodes_len");

    const RUNNING_VALIDATORS: Mapping<u64, u32> = mapping!(b"running_validators");
    const RUNNING_VALIDATOR_IDS: Storage<Vec<u64>> = storage!(b"running_validator_ids");
    const PENDING_VALIDATORS: Mapping<u64, u32> = mapping!(b"pending_validators");
    const PENDING_VALIDATOR_IDS: Storage<Vec<u64>> = storage!(b"pending_validator_ids");

    /// Subnet 合约构造函数。
    ///
    /// 该函数在合约部署时自动调用，创建一个新的 Subnet 合约实例。
    /// 当前实现为空操作，合约的初始化逻辑由 `init` 函数完成。
    ///
    /// # 调用权限
    /// 任何人（由部署器在合约部署时自动调用）。
    ///
    /// # 返回值
    /// - `Ok(())`：构造成功。
    #[revive(constructor)]
    pub fn new() -> Result<(), Error> {
        Ok(())
    }

    /// 初始化 Subnet 合约。
    ///
    /// 设置治理合约地址（调用者自身）、epoch 相关参数、各类 ID 计数器的初始值等。
    /// 该函数只能执行一次，若已初始化则直接返回成功而不做任何修改。
    ///
    /// # 调用权限
    /// 任何人（仅在首次调用时有效，调用者将被记录为治理合约地址）。
    ///
    /// # 返回值
    /// - `Ok(())`：初始化成功或已经初始化过。
    #[revive(message, write)]
    pub fn init() -> Result<(), Error> {
        if let Some(_) = GOV_CONTRACT.get() {
            return Ok(());
        }
        let caller = env().caller();
        GOV_CONTRACT.set(&caller);
        EPOCH_SLOT.set(&72000u32);
        EPOCH.set(&0u32);
        LAST_EPOCH_BLOCK.set(&0u32);
        TEE_CHAIN_MULTI_KEY.set(&Address::zero());
        NEXT_REGION_ID.set(&0u32);
        NEXT_WORKER_ID.set(&0u64);
        NEXT_SECRET_ID.set(&0u64);
        NEXT_ASSET_ID.set(&0u32);
        MIN_MORTGAGE_AMOUNT.set(&U256::ZERO);
        Ok(())
    }

    /// 查询当前的 epoch 信息。
    ///
    /// 返回当前 epoch 编号、epoch 间隔（区块数）、上一个 epoch 的区块高度、
    /// 当前区块时间戳以及侧链多签地址等关键信息。
    ///
    /// # 调用权限
    /// 任何人（只读查询）。
    ///
    /// # 返回值
    /// - `EpochInfo`：包含 epoch 编号、epoch_slot、last_epoch_block、now、tee_chain_pub 的结构体。
    #[revive(message)]
    pub fn epoch_info() -> EpochInfo {
        let now = env().now();
        EpochInfo {
            epoch: EPOCH.get().unwrap_or(0),
            epoch_slot: EPOCH_SLOT.get().unwrap_or(72000),
            last_epoch_block: LAST_EPOCH_BLOCK.get().unwrap_or(0),
            now,
            tee_chain_pub: TEE_CHAIN_MULTI_KEY.get().unwrap_or(Address::zero()),
        }
    }

    /// 设置 epoch 的区块间隔（slot）。
    ///
    /// 修改两次 epoch 切换之间所需经过的区块数量。
    ///
    /// # 调用权限
    /// 仅治理合约（gov）可调用。
    ///
    /// # 参数
    /// - `epoch_slot`：新的 epoch 间隔区块数。
    ///
    /// # 返回值
    /// - `Ok(())`：设置成功。
    /// - `Err(Error::MustCallByMainContract)`：调用者不是治理合约。
    #[revive(message, write)]
    pub fn set_epoch_slot(epoch_slot: u32) -> Result<(), Error> {
        ensure_from_gov()?;
        EPOCH_SLOT.set(&epoch_slot);
        Ok(())
    }

    /// 查询侧链多签地址。
    ///
    /// 返回当前设置的侧链多签公钥地址，用于验证侧链发起的调用。
    ///
    /// # 调用权限
    /// 任何人（只读查询）。
    ///
    /// # 返回值
    /// - `Address`：侧链多签地址，若未设置则返回零地址。
    #[revive(message)]
    pub fn tee_chain_key() -> Address {
        TEE_CHAIN_MULTI_KEY.get().unwrap_or(Address::zero())
    }

    /// 注册一个新的区域（Region）。
    ///
    /// 为 Subnet 网络新增一个地理或逻辑区域，区域 ID 自增分配，
    /// 可用于后续 Worker 注册时选择所属区域。
    ///
    /// # 调用权限
    /// 仅治理合约（gov）可调用。
    ///
    /// # 参数
    /// - `name`：区域名称（字节数组）。
    ///
    /// # 返回值
    /// - `Ok(())`：区域设置成功。
    /// - `Err(Error::MustCallByMainContract)`：调用者不是治理合约。
    #[revive(message, write)]
    pub fn set_region(name: Bytes) -> Result<(), Error> {
        ensure_from_gov()?;
        let id = NEXT_REGION_ID.get().unwrap_or(0);
        NEXT_REGION_ID.set(&(id + 1));
        REGIONS.set(&id, &name);
        Ok(())
    }

    /// 根据区域 ID 查询区域名称。
    ///
    /// # 调用权限
    /// 任何人（只读查询）。
    ///
    /// # 参数
    /// - `id`：区域 ID。
    ///
    /// # 返回值
    /// - `Some(Bytes)`：该区域对应的名称。
    /// - `None`：区域不存在。
    #[revive(message)]
    pub fn region(id: u32) -> Option<Bytes> {
        REGIONS.get(&id)
    }

    /// 列出所有已注册的区域。
    ///
    /// 返回所有区域的 ID 和名称列表，按 ID 降序排列，最多返回 1000 条记录。
    ///
    /// # 调用权限
    /// 任何人（只读查询）。
    ///
    /// # 返回值
    /// - `Vec<(u32, Bytes)>`：区域 ID 与名称的列表，降序排列，最多 1000 条。
    #[revive(message)]
    pub fn regions() -> Vec<(u32, Bytes)> {
        let next_id = NEXT_REGION_ID.get().unwrap_or(0);
        let mut out = Vec::new();
        for id in 0..next_id {
            if let Some(name) = REGIONS.get(&id) {
                out.push((id, name));
            }
        }
        out.reverse();
        if out.len() > 1000 {
            out.truncate(1000);
        }
        out
    }

    /// 设置指定等级的运行价格。
    ///
    /// 为 Worker 的不同等级配置对应的运行定价信息。
    ///
    /// # 调用权限
    /// 仅治理合约（gov）可调用。
    ///
    /// # 参数
    /// - `level`：Worker 等级（如 0, 1, 2 ...）。
    /// - `price`：该等级对应的运行价格结构体。
    ///
    /// # 返回值
    /// - `Ok(())`：设置成功。
    /// - `Err(Error::MustCallByMainContract)`：调用者不是治理合约。
    #[revive(message, write)]
    pub fn set_level_price(level: u8, price: RunPrice) -> Result<(), Error> {
        ensure_from_gov()?;
        LEVEL_PRICES.set(&level, &price);
        Ok(())
    }

    /// 查询指定等级的运行价格。
    ///
    /// # 调用权限
    /// 任何人（只读查询）。
    ///
    /// # 参数
    /// - `level`：Worker 等级。
    ///
    /// # 返回值
    /// - `Some(RunPrice)`：该等级对应的价格信息。
    /// - `None`：该等级价格未设置。
    #[revive(message)]
    pub fn level_price(level: u8) -> Option<RunPrice> {
        LEVEL_PRICES.get(&level)
    }

    /// 注册一种新的资产。
    ///
    /// 新增一种可用于 Worker 抵押的资产类型，并为其设置单价。
    /// 资产 ID 自增分配。
    ///
    /// # 调用权限
    /// 仅治理合约（gov）可调用。
    ///
    /// # 参数
    /// - `info`：资产信息结构体（如资产名称、精度等）。
    /// - `price`：资产单价（U256）。
    ///
    /// # 返回值
    /// - `Ok(())`：资产注册成功。
    /// - `Err(Error::MustCallByMainContract)`：调用者不是治理合约。
    #[revive(message, write)]
    pub fn set_asset(info: AssetInfo, price: U256) -> Result<(), Error> {
        ensure_from_gov()?;
        let id = NEXT_ASSET_ID.get().unwrap_or(0);
        NEXT_ASSET_ID.set(&(id + 1));
        ASSET_INFOS.set(&id, &info);
        ASSET_PRICES.set(&id, &price);
        Ok(())
    }

    /// 根据资产 ID 查询资产信息及价格。
    ///
    /// # 调用权限
    /// 任何人（只读查询）。
    ///
    /// # 参数
    /// - `id`：资产 ID。
    ///
    /// # 返回值
    /// - `Some((AssetInfo, U256))`：资产信息和单价。
    /// - `None`：资产不存在。
    #[revive(message)]
    pub fn asset(id: u32) -> Option<(AssetInfo, U256)> {
        let info = ASSET_INFOS.get(&id)?;
        let price = ASSET_PRICES.get(&id)?;
        Some((info, price))
    }

    /// 设置全局最小抵押金额。
    ///
    /// Worker 启动时必须满足总抵押金额不低于该值。
    ///
    /// # 调用权限
    /// 仅治理合约（gov）可调用。
    ///
    /// # 参数
    /// - `amount`：最小抵押金额（U256）。
    ///
    /// # 返回值
    /// - `Ok(())`：设置成功。
    /// - `Err(Error::MustCallByMainContract)`：调用者不是治理合约。
    #[revive(message, write)]
    pub fn set_min_mortgage(amount: U256) -> Result<(), Error> {
        ensure_from_gov()?;
        MIN_MORTGAGE_AMOUNT.set(&amount);
        Ok(())
    }

    /// 查询全局最小抵押金额。
    ///
    /// # 调用权限
    /// 任何人（只读查询）。
    ///
    /// # 返回值
    /// - `U256`：当前全局最小抵押金额，若未设置则返回 0。
    #[revive(message)]
    pub fn min_mortgage() -> U256 {
        MIN_MORTGAGE_AMOUNT.get().unwrap_or(U256::ZERO)
    }

    /// 设置指定等级的最小抵押金额。
    ///
    /// 可为不同等级的 Worker 设置不同的最低抵押要求，
    /// 若某等级未单独设置，则回退到全局最小抵押金额。
    ///
    /// # 调用权限
    /// 仅治理合约（gov）可调用。
    ///
    /// # 参数
    /// - `level`：Worker 等级。
    /// - `amount`：该等级对应的最小抵押金额。
    ///
    /// # 返回值
    /// - `Ok(())`：设置成功。
    /// - `Err(Error::MustCallByMainContract)`：调用者不是治理合约。
    #[revive(message, write)]
    pub fn set_level_min_mortgage(level: u8, amount: U256) -> Result<(), Error> {
        ensure_from_gov()?;
        LEVEL_MIN_MORTGAGES.set(&level, &amount);
        Ok(())
    }

    /// 查询指定等级的最小抵押金额。
    ///
    /// 优先返回该等级单独设置的值；若未设置，则返回全局最小抵押金额；
    /// 若全局也未设置，则返回 0。
    ///
    /// # 调用权限
    /// 任何人（只读查询）。
    ///
    /// # 参数
    /// - `level`：Worker 等级。
    ///
    /// # 返回值
    /// - `U256`：该等级的最小抵押金额。
    #[revive(message)]
    pub fn level_min_mortgage(level: u8) -> U256 {
        LEVEL_MIN_MORTGAGES
            .get(&level)
            .or_else(|| MIN_MORTGAGE_AMOUNT.get())
            .unwrap_or(U256::ZERO)
    }

    /// 计算指定 Worker 已抵押的总资源量。
    ///
    /// 遍历该 Worker 的所有有效抵押记录，累加 CPU、内存、CVM CPU、CVM 内存、磁盘和 GPU 资源。
    /// 已标记为删除（deleted）的抵押记录不参与计算。
    ///
    /// # 调用权限
    /// 任何人（只读查询）。
    ///
    /// # 参数
    /// - `worker_id`：Worker 的唯一 ID。
    ///
    /// # 返回值
    /// - `(cpu, mem, cvm_cpu, cvm_mem, disk, gpu)`：六种资源的总量元组。
    #[revive(message)]
    pub fn worker_total_resources(worker_id: u64) -> (u32, u32, u32, u32, u32, u32) {
        let list = WORKER_MORTGAGES.list_all(&worker_id);
        let mut total_cpu = 0u32;
        let mut total_mem = 0u32;
        let mut total_cvm_cpu = 0u32;
        let mut total_cvm_mem = 0u32;
        let mut total_disk = 0u32;
        let mut total_gpu = 0u32;
        for (_, dep) in list {
            if dep.deleted.is_none() {
                total_cpu = total_cpu.saturating_add(dep.cpu);
                total_mem = total_mem.saturating_add(dep.mem);
                total_cvm_cpu = total_cvm_cpu.saturating_add(dep.cvm_cpu);
                total_cvm_mem = total_cvm_mem.saturating_add(dep.cvm_mem);
                total_disk = total_disk.saturating_add(dep.disk);
                total_gpu = total_gpu.saturating_add(dep.gpu);
            }
        }
        (
            total_cpu,
            total_mem,
            total_cvm_cpu,
            total_cvm_mem,
            total_disk,
            total_gpu,
        )
    }

    /// 计算指定 Worker 已抵押的总资产金额。
    ///
    /// 遍历该 Worker 的所有有效抵押记录，累加抵押金额。
    /// 已标记为删除（deleted）的抵押记录不参与计算。
    ///
    /// # 调用权限
    /// 任何人（只读查询）。
    ///
    /// # 参数
    /// - `worker_id`：Worker 的唯一 ID。
    ///
    /// # 返回值
    /// - `U256`：该 Worker 当前有效的总抵押金额。
    /// 合并计算 Worker 的抵押总额和资源总量，一次遍历完成。
    /// 供 worker_start 使用，避免分别调用两个遍历函数。
    fn worker_total_mortgage_and_resources(worker_id: u64) -> (U256, u32, u32, u32, u32, u32, u32) {
        let list = WORKER_MORTGAGES.list_all(&worker_id);
        let mut total = U256::ZERO;
        let mut total_cpu = 0u32;
        let mut total_mem = 0u32;
        let mut total_cvm_cpu = 0u32;
        let mut total_cvm_mem = 0u32;
        let mut total_disk = 0u32;
        let mut total_gpu = 0u32;
        for (_, dep) in list {
            if dep.deleted.is_none() {
                total = total + dep.amount;
                total_cpu = total_cpu.saturating_add(dep.cpu);
                total_mem = total_mem.saturating_add(dep.mem);
                total_cvm_cpu = total_cvm_cpu.saturating_add(dep.cvm_cpu);
                total_cvm_mem = total_cvm_mem.saturating_add(dep.cvm_mem);
                total_disk = total_disk.saturating_add(dep.disk);
                total_gpu = total_gpu.saturating_add(dep.gpu);
            }
        }
        (
            total,
            total_cpu,
            total_mem,
            total_cvm_cpu,
            total_cvm_mem,
            total_disk,
            total_gpu,
        )
    }

    #[revive(message)]
    pub fn worker_total_mortgage(worker_id: u64) -> U256 {
        let list = WORKER_MORTGAGES.list_all(&worker_id);
        let mut total = U256::ZERO;
        for (_, dep) in list {
            if dep.deleted.is_none() {
                total = total + dep.amount;
            }
        }
        total
    }

    /// 罚没（slash）指定 Worker 的抵押资产。
    ///
    /// 从该 Worker 的抵押记录中按顺序扣除指定金额，将被扣除的代币转账到指定地址。
    /// 若某条抵押记录金额不足，则继续扣除下一条，直到满足罚没金额为止。
    /// 被完全扣除的记录会被标记为删除。
    ///
    /// # 调用权限
    /// 仅 Cloud 合约可调用。
    ///
    /// # 参数
    /// - `worker_id`：目标 Worker 的 ID。
    /// - `amount`：需要罚没的总金额。
    /// - `to`：罚没资金接收地址。
    ///
    /// # 返回值
    /// - `Ok(())`：罚没成功。
    /// - `Err(Error::MustCallByMainContract)`：调用者不是 Cloud 合约。
    /// - `Err(Error::SlashAmountTooLarge)`：罚没金额超过实际抵押总额。
    /// - `Err(Error::WorkerMortgageNotExist)`：抵押记录不存在。
    /// - `Err(Error::TransferFailed)`：转账失败。
    #[revive(message, write)]
    pub fn slash_worker_mortgage(
        worker_id: NodeID,
        amount: U256,
        to: Address,
    ) -> Result<(), Error> {
        ensure_from_gov()?;

        let mut remaining = amount;
        let list = WORKER_MORTGAGES.list_all(&worker_id);

        // 逐条遍历 Worker 的抵押记录，按顺序扣除罚没金额，直至满足目标金额
        // Iterate Worker's mortgage records sequentially, deduct slash amount until target is met
        for (mid, mut dep) in list {
            if dep.deleted.is_some() || remaining == U256::ZERO {
                continue;
            }
            if dep.amount <= remaining {
                // 当前记录金额不足，全额扣除并标记删除（该笔质押已被罚没完毕）
                // Current record insufficient: fully deduct and mark as deleted (this mortgage is fully slashed)
                remaining = remaining - dep.amount;
                dep.amount = U256::ZERO;
                dep.deleted = Some(env().block_number());
                WORKER_MORTGAGES
                    .update(&worker_id, mid, &dep)
                    .ok_or(Error::WorkerMortgageNotExist)?;
            } else {
                // 当前记录金额充足，仅扣除部分金额，保留剩余质押
                // Current record sufficient: partially deduct, retain remaining mortgage
                dep.amount = dep.amount - remaining;
                remaining = U256::ZERO;
                WORKER_MORTGAGES
                    .update(&worker_id, mid, &dep)
                    .ok_or(Error::WorkerMortgageNotExist)?;
            }
        }

        // 遍历结束后剩余金额必须为零，否则说明抵押总额不足以覆盖罚没金额
        // After iteration remaining must be zero; otherwise total mortgage is insufficient to cover slash
        ensure!(remaining == U256::ZERO, Error::SlashAmountTooLarge);
        transfer_native(&to, amount)?;
        Ok(())
    }

    /// 根据 Worker ID 查询 Worker 详情。
    ///
    /// 返回的 Worker 信息中包含从 `WORKER_STATUS` 映射中实时读取的状态字段。
    ///
    /// # 调用权限
    /// 任何人（只读查询）。
    ///
    /// # 参数
    /// - `id`：Worker 的唯一 ID。
    ///
    /// # 返回值
    /// - `Some(K8sCluster)`：Worker 详细信息（含实时状态）。
    /// - `None`：Worker 不存在。
    #[revive(message)]
    pub fn worker(id: NodeID) -> Option<K8sCluster> {
        let mut worker = WORKERS.get(&id)?;
        worker.status = WORKER_STATUS.get(&id).unwrap_or(0);
        Some(worker)
    }

    /// 分页列出 Worker 列表。
    ///
    /// 按 Worker ID 降序排列返回 Worker 信息，支持分页查询。
    /// `start=None` 表示从最新的 Worker 开始查询。
    ///
    /// # 调用权限
    /// 任何人（只读查询）。
    ///
    /// # 参数
    /// - `start`：起始 Worker ID（可选），`None` 表示从最新开始。
    /// - `size`：本次查询返回的最大数量。
    ///
    /// # 返回值
    /// - `Vec<(u64, K8sCluster)>`：Worker ID 与详细信息的列表。
    #[revive(message)]
    pub fn workers(start: Option<u64>, size: u64) -> Vec<(u64, K8sCluster)> {
        let total = NEXT_WORKER_ID.get().unwrap_or(0);
        if total == 0 || size == 0 {
            return Vec::new();
        }
        let mut cur = start.unwrap_or(total - 1);
        if cur >= total {
            cur = total - 1;
        }
        let mut out = Vec::new();
        for _ in 0..size {
            if let Some(mut w) = WORKERS.get(&cur) {
                w.status = WORKER_STATUS.get(&cur).unwrap_or(0);
                out.push((cur, w));
            }
            if cur == 0 {
                break;
            }
            cur -= 1;
        }
        out
    }

    /// 根据用户地址查询其拥有的 Worker。
    ///
    /// 通过 `OWNER_OF_WORKER` 映射查找用户对应的 Worker ID，再返回 Worker 详情。
    ///
    /// # 调用权限
    /// 任何人（只读查询）。
    ///
    /// # 参数
    /// - `user`：用户地址。
    ///
    /// # 返回值
    /// - `Some((u64, K8sCluster))`：用户拥有的 Worker ID 及详细信息。
    /// - `None`：该用户未拥有 Worker。
    #[revive(message)]
    pub fn user_worker(user: Address) -> Option<(u64, K8sCluster)> {
        let id = OWNER_OF_WORKER.get(&user)?;
        let mut worker = WORKERS.get(&id)?;
        worker.status = WORKER_STATUS.get(&id).unwrap_or(0);
        Some((id, worker))
    }

    /// 根据 P2P/Mint ID 查询对应的 Worker。
    ///
    /// 通过 `MINT_OF_WORKER` 映射查找 P2P ID 对应的 Worker ID，再返回 Worker 详情。
    ///
    /// # 调用权限
    /// 任何人（只读查询）。
    ///
    /// # 参数
    /// - `id`：Worker 的 P2P ID（AccountId 类型）。
    ///
    /// # 返回值
    /// - `Some((u64, K8sCluster))`：Worker ID 及详细信息。
    /// - `None`：不存在对应 Worker。
    #[revive(message)]
    pub fn mint_worker(id: AccountId) -> Option<(u64, K8sCluster)> {
        let worker_id = MINT_OF_WORKER.get(&id)?;
        let mut worker = WORKERS.get(&worker_id)?;
        worker.status = WORKER_STATUS.get(&worker_id).unwrap_or(0);
        Some((worker_id, worker))
    }

    /// 注册一个新的 Worker 节点。
    ///
    /// 调用者成为该 Worker 的拥有者，P2P ID 与 Worker ID 绑定。
    /// 每个地址只能注册一个 Worker。Worker 初始状态为 0（未启动）。
    ///
    /// # 调用权限
    /// 任何人（每个地址限注册一个 Worker）。
    ///
    /// # 参数
    /// - `name`：Worker 名称。
    /// - `p2p_id`：Worker 的 P2P 身份 ID。
    /// - `ip`：Worker 的网络 IP 地址。
    /// - `port`：Worker 的服务端口。
    /// - `level`：Worker 等级。
    /// - `region_id`：所属区域 ID（必须已存在）。
    ///
    /// # 返回值
    /// - `Ok(NodeID)`：新注册的 Worker ID。
    /// - `Err(Error::RegionNotExist)`：指定的区域不存在。
    #[revive(message, write)]
    pub fn worker_register(
        name: Bytes,
        p2p_id: AccountId,
        ip: Ip,
        port: u32,
        level: u8,
        region_id: u32,
    ) -> Result<NodeID, Error> {
        REGIONS.get(&region_id).ok_or(Error::RegionNotExist)?;
        let caller = env().caller();
        // 每个地址仅可注册一个 Worker，已注册则返回 WorkerNotOwnedByCaller 作为「已存在」语义
        // Each address can only register one Worker; return WorkerNotOwnedByCaller as "already exists"
        ensure!(
            OWNER_OF_WORKER.get(&caller).is_none(),
            Error::WorkerNotOwnedByCaller
        );
        let worker_id = NEXT_WORKER_ID.get().unwrap_or(0);
        // Worker ID 溢出时返回 WorkerNotExist（触发概率极低，但需防止 panic）
        let next = worker_id.checked_add(1).ok_or(Error::WorkerNotExist)?;
        NEXT_WORKER_ID.set(&next);
        let now = env().block_number();
        let worker = K8sCluster {
            name,
            owner: caller,
            level,
            region_id,
            start_block: now,
            stop_block: None,
            terminal_block: None,
            p2p_id,
            ip,
            port,
            status: 0,
        };
        WORKERS.set(&worker_id, &worker);
        OWNER_OF_WORKER.set(&caller, &worker_id);
        MINT_OF_WORKER.set(&p2p_id, &worker_id);
        REGION_WORKERS.insert(&region_id, &worker_id);
        Ok(worker_id)
    }

    /// 更新 Worker 的基本信息。
    ///
    /// 允许 Worker 拥有者修改 Worker 的名称、IP 地址和端口。
    ///
    /// # 调用权限
    /// 仅 Worker 拥有者可调用。
    ///
    /// # 参数
    /// - `id`：目标 Worker 的 ID。
    /// - `name`：新的 Worker 名称。
    /// - `ip`：新的 IP 地址。
    /// - `port`：新的服务端口。
    ///
    /// # 返回值
    /// - `Ok(())`：更新成功。
    /// - `Err(Error::WorkerNotExist)`：Worker 不存在。
    /// - `Err(Error::WorkerNotOwnedByCaller)`：调用者不是该 Worker 的拥有者。
    #[revive(message, write)]
    pub fn worker_update(id: NodeID, name: Bytes, ip: Ip, port: u32) -> Result<(), Error> {
        let caller = env().caller();
        let mut worker = WORKERS.get(&id).ok_or(Error::WorkerNotExist)?;
        ensure!(worker.owner == caller, Error::WorkerNotOwnedByCaller);
        // 运行中的 Worker 不允许修改 IP，防止劫持活跃 Pod 的网络流量
        // Running workers cannot change IP to prevent hijacking active Pod traffic
        let status = WORKER_STATUS.get(&id).unwrap_or(0);
        if status == 1 {
            ensure!(worker.ip == ip, Error::WorkerStatusNotReady);
        }
        worker.name = name;
        worker.ip = ip;
        worker.port = port;
        WORKERS.set(&id, &worker);
        Ok(())
    }

    /// 为 Worker 添加资源抵押。
    ///
    /// Worker 拥有者通过向合约转入代币来抵押资源。抵押时需声明 CPU、内存、CVM CPU、CVM 内存、
    /// 磁盘和 GPU 数量。实际转账金额必须大于等于声明的 `deposit` 金额。
    /// Worker 必须处于未启动状态（status == 0）才能抵押。
    ///
    /// # 调用权限
    /// 仅 Worker 拥有者可调用。
    ///
    /// # 参数
    /// - `id`：Worker ID。
    /// - `cpu`：抵押的 CPU 资源量。
    /// - `mem`：抵押的内存资源量。
    /// - `cvm_cpu`：抵押的 CVM CPU 资源量。
    /// - `cvm_mem`：抵押的 CVM 内存资源量。
    /// - `disk`：抵押的磁盘资源量。
    /// - `gpu`：抵押的 GPU 资源量。
    /// - `deposit`：声明的抵押金额（实际转账金额必须 ≥ 此值）。
    ///
    /// # 返回值
    /// - `Ok(u32)`：新创建的抵押记录 ID。
    /// - `Err(Error::WorkerNotExist)`：Worker 不存在。
    /// - `Err(Error::WorkerNotOwnedByCaller)`：调用者不是拥有者。
    /// - `Err(Error::WorkerStatusNotReady)`：Worker 已启动，不可抵押。
    /// - `Err(Error::DepositNotEnough)`：实际转账金额不足。
    #[revive(message, write)]
    pub fn worker_mortgage(
        id: NodeID,
        cpu: u32,
        mem: u32,
        cvm_cpu: u32,
        cvm_mem: u32,
        disk: u32,
        gpu: u32,
        deposit: U256,
    ) -> Result<u32, Error> {
        let caller = env().caller();
        let worker = WORKERS.get(&id).ok_or(Error::WorkerNotExist)?;
        ensure!(worker.owner == caller, Error::WorkerNotOwnedByCaller);
        // 允许在未启动（0）或已停止（2）状态下抵押，运行中（1）不可抵押
        // Allow mortgage when not started (0) or stopped (2); not allowed while running (1)
        let status = WORKER_STATUS.get(&id).unwrap_or(0);
        ensure!(status == 0 || status == 2, Error::WorkerStatusNotReady);

        // 校验实际随交易转入的金额 >= 声明的质押金额，确保保证金真实到账
        // Verify actual transferred amount >= declared deposit, ensuring real funds are received
        let transferred = env().value_transferred();
        ensure!(transferred >= deposit, Error::DepositNotEnough);

        // 记录实际转账金额为质押金额（而非声明金额），防止用户虚报质押数额
        // Record actual transferred amount as mortgage (not declared amount), preventing false deposit claims
        let dep = AssetDeposit {
            amount: transferred,
            cpu,
            cvm_cpu,
            mem,
            cvm_mem,
            disk,
            gpu,
            deleted: None,
        };

        let mid = WORKER_MORTGAGES
            .insert(&id, &dep)
            .ok_or(Error::WorkerMortgageNotExist)?;
        Ok(mid)
    }

    /// 解除 Worker 的指定抵押记录并退还资金。
    ///
    /// 将指定的抵押记录标记为删除，并将该记录对应的抵押金额全额退还给 Worker 拥有者。
    /// Worker 必须处于未启动状态（status == 0）才能解抵押。
    ///
    /// # 调用权限
    /// 仅 Worker 拥有者可调用。
    ///
    /// # 参数
    /// - `worker_id`：Worker ID。
    /// - `mortgage_id`：要解除的抵押记录 ID。
    ///
    /// # 返回值
    /// - `Ok(u32)`：已解除的抵押记录 ID。
    /// - `Err(Error::WorkerNotExist)`：Worker 不存在。
    /// - `Err(Error::WorkerNotOwnedByCaller)`：调用者不是拥有者。
    /// - `Err(Error::WorkerStatusNotReady)`：Worker 已启动，不可解抵押。
    /// - `Err(Error::WorkerMortgageNotExist)`：抵押记录不存在。
    /// - `Err(Error::TransferFailed)`：退款转账失败。
    #[revive(message, write)]
    pub fn worker_unmortgage(worker_id: NodeID, mortgage_id: u32) -> Result<u32, Error> {
        let caller = env().caller();
        let worker = WORKERS.get(&worker_id).ok_or(Error::WorkerNotExist)?;
        ensure!(worker.owner == caller, Error::WorkerNotOwnedByCaller);
        // 允许在未启动（0）或已停止（2）状态下解除抵押，运行中（1）不可解抵押
        // Allow unmortgage when not started (0) or stopped (2); not allowed while running (1)
        let status = WORKER_STATUS.get(&worker_id).unwrap_or(0);
        ensure!(status == 0 || status == 2, Error::WorkerStatusNotReady);

        let mut mortgage = WORKER_MORTGAGES
            .get(&worker_id, mortgage_id)
            .ok_or(Error::WorkerMortgageNotExist)?;
        let now = env().block_number();
        mortgage.deleted = Some(now);
        WORKER_MORTGAGES
            .update(&worker_id, mortgage_id, &mortgage)
            .ok_or(Error::WorkerMortgageNotExist)?;
        transfer_native(&worker.owner, mortgage.amount)?;
        Ok(mortgage_id)
    }

    /// 启动 Worker。
    ///
    /// 侧链调用此函数将 Worker 状态置为 1（运行中）。
    /// 启动前会校验 Worker 的总抵押金额是否达到对应等级的最低要求，
    /// 并校验 Worker 是否至少抵押了一种资源（CPU、内存、磁盘、GPU 等）。
    ///
    /// # 调用权限
    /// 仅侧链多签地址（tee_chain）可调用。
    ///
    /// # 参数
    /// - `id`：要启动的 Worker ID。
    ///
    /// # 返回值
    /// - `Ok(())`：启动成功。
    /// - `Err(Error::WorkerNotExist)`：Worker 不存在。
    /// - `Err(Error::InvalidTeeChainCaller)`：调用者不是侧链多签地址。
    /// - `Err(Error::MortgageNotEnough)`：抵押金额不足。
    /// - `Err(Error::ResourceNotEnough)`：未抵押任何资源。
    #[revive(message, write)]
    pub fn worker_start(id: NodeID) -> Result<(), Error> {
        ensure_from_tee_chain()?;
        let worker = WORKERS.get(&id).ok_or(Error::WorkerNotExist)?;

        // 合并一次遍历计算抵押总额和资源总量，避免重复迭代
        // Compute total mortgage and resources in single pass, avoiding double iteration
        let (total, cpu, mem, cvm_cpu, cvm_mem, disk, gpu) =
            worker_total_mortgage_and_resources(id);
        let min = LEVEL_MIN_MORTGAGES
            .get(&worker.level)
            .or_else(|| MIN_MORTGAGE_AMOUNT.get())
            .unwrap_or(U256::ZERO);
        ensure!(total >= min, Error::MortgageNotEnough);
        ensure!(
            cpu > 0 || mem > 0 || cvm_cpu > 0 || cvm_mem > 0 || disk > 0 || gpu > 0,
            Error::ResourceNotEnough
        );

        WORKER_STATUS.set(&id, &1u8);
        Ok(())
    }

    /// 申请停止 Worker。
    ///
    /// Worker 拥有者可以停止运行中的 Worker（status == 1），将状态置为已停止（status == 2）。
    /// 停止后抵押金可通过 `worker_unmortgage` 逐笔解除并退还。
    /// 注意：停止并不自动退还抵押金，需调用方后续手动解除；
    /// Slash 机制在 Worker 停止后仍可对剩余抵押金执行罚没。
    ///
    /// # 调用权限
    /// 仅 Worker 拥有者可调用。
    ///
    /// # 参数
    /// - `id`：要停止的 Worker ID。
    ///
    /// # 返回值
    /// - `Ok(NodeID)`：停止成功，返回 Worker ID。
    /// - `Err(Error::WorkerNotExist)`：Worker 不存在。
    /// - `Err(Error::WorkerNotOwnedByCaller)`：调用者不是拥有者。
    /// - `Err(Error::WorkerStatusNotReady)`：Worker 未处于运行状态（status != 1），无法停止。
    #[revive(message, write)]
    pub fn worker_stop(id: NodeID) -> Result<NodeID, Error> {
        let caller = env().caller();
        let worker = WORKERS.get(&id).ok_or(Error::WorkerNotExist)?;
        ensure!(worker.owner == caller, Error::WorkerNotOwnedByCaller);
        // 仅允许运行中（status == 1）的 Worker 停止，防止重复停止已停止节点
        // Only allow stopping a running Worker (status == 1), preventing double-stop
        ensure!(
            WORKER_STATUS.get(&id).unwrap_or(0) == 1,
            Error::WorkerStatusNotReady
        );
        // 将 Worker 状态置为已停止（2）
        // Set Worker status to stopped (2)
        WORKER_STATUS.set(&id, &2u8);
        let mut worker = WORKERS.get(&id).ok_or(Error::WorkerNotExist)?;
        worker.stop_block = Some(env().block_number());
        WORKERS.set(&id, &worker);
        Ok(id)
    }

    /// 设置引导节点（Boot Nodes）列表。
    ///
    /// 引导节点用于网络发现和初始化连接。传入的节点 ID 列表会自动去重并排序。
    ///
    /// # 调用权限
    /// 仅治理合约（gov）可调用。
    ///
    /// # 参数
    /// - `nodes`：引导节点 ID 列表（Secret Node ID 数组）。
    ///
    /// # 返回值
    /// - `Ok(())`：设置成功。
    /// - `Err(Error::MustCallByMainContract)`：调用者不是治理合约。
    #[revive(message, write)]
    pub fn set_boot_nodes(nodes: Vec<u64>) -> Result<(), Error> {
        ensure_from_gov()?;
        let mut lnodes = nodes;
        lnodes.sort();
        lnodes.dedup();
        let len = lnodes.len() as u32;
        BOOT_NODES_LEN.set(&len);
        for (i, &node_id) in lnodes.iter().enumerate() {
            let k = i as u32;
            BOOT_NODES.set(&k, &node_id);
        }
        Ok(())
    }

    /// 查询当前的引导节点列表。
    ///
    /// 根据 `BOOT_NODES` 中存储的 ID 列表，从 `SECRETS` 映射中查询对应的 SecretNode 详情并返回。
    ///
    /// # 调用权限
    /// 任何人（只读查询）。
    ///
    /// # 返回值
    /// - `Ok(Vec<SecretNode>)`：引导节点的详细信息列表。
    #[revive(message)]
    pub fn boot_nodes() -> Result<Vec<SecretNode>, Error> {
        let len = BOOT_NODES_LEN.get().unwrap_or(0);
        let mut out = Vec::new();
        for i in 0..len {
            if let Some(id) = BOOT_NODES.get(&i) {
                if let Some(node) = SECRETS.get(&id) {
                    out.push(node);
                }
            }
        }
        Ok(out)
    }

    /// 获取当前待处理的 Secret 节点变更列表。
    ///
    /// 返回 pending_secrets 中所有记录，包含 Secret Node ID 和对应的权重（power）。
    /// 这些记录将在下一个 epoch 被合并到运行中的验证者集合。
    ///
    /// # 调用权限
    /// 任何人（只读查询）。
    ///
    /// # 返回值
    /// - `Vec<(u64, u32)>`：待处理 Secret 节点 ID 与权重的列表。
    #[revive(message)]
    pub fn get_pending_secrets() -> Vec<(u64, u32)> {
        let ids = PENDING_VALIDATOR_IDS.get().unwrap_or_default();
        ids.into_iter()
            .map(|id| {
                let power = PENDING_VALIDATORS.get(&id).unwrap_or(0);
                (id, power)
            })
            .collect()
    }

    /// 列出所有已注册的 Secret 节点。
    ///
    /// 返回所有 Secret Node 的 ID 和详细信息，按 ID 降序排列，最多返回 10000 条记录。
    ///
    /// # 调用权限
    /// 任何人（只读查询）。
    ///
    /// # 返回值
    /// - `Vec<(u64, SecretNode)>`：Secret Node ID 与详细信息的列表。
    #[revive(message)]
    pub fn secrets() -> Vec<(u64, SecretNode)> {
        let next_id = NEXT_SECRET_ID.get().unwrap_or(0);
        let mut out = Vec::new();
        for id in 0..next_id {
            if let Some(node) = SECRETS.get(&id) {
                out.push((id, node));
            }
        }
        out.reverse();
        if out.len() > 10000 {
            out.truncate(10000);
        }
        out
    }

    /// 注册一个新的 Secret 节点。
    ///
    /// 调用者成为该 Secret 节点的拥有者。节点初始状态为 0。
    /// 第一个注册的 Secret 节点（id == 0）会自动被加入运行中的验证者集合。
    ///
    /// # 调用权限
    /// 任何人（每个地址限注册一个 Secret 节点）。
    ///
    /// # 参数
    /// - `name`：Secret 节点名称。
    /// - `validator_id`：验证者身份 ID。
    /// - `p2p_id`：P2P 网络身份 ID。
    /// - `ip`：节点 IP 地址。
    /// - `port`：节点服务端口号。
    ///
    /// # 返回值
    /// - `Ok(NodeID)`：新注册的 Secret 节点 ID。
    #[revive(message, write)]
    pub fn secret_register(
        name: Bytes,
        validator_id: AccountId,
        p2p_id: AccountId,
        ip: Ip,
        port: u32,
        bls: Bytes,
    ) -> Result<NodeID, Error> {
        let caller = env().caller();
        ensure!(!bls.is_empty(), Error::InvalidBlsKey);
        let now = env().block_number();
        let node = SecretNode {
            name,
            owner: caller,
            validator_id,
            p2p_id,
            start_block: now,
            terminal_block: None,
            ip,
            port,
            status: 0,
            bls,
        };
        let id = NEXT_SECRET_ID.get().unwrap_or(0);
        let next = id.checked_add(1).ok_or(Error::NodeNotExist)?;
        NEXT_SECRET_ID.set(&next);
        SECRETS.set(&id, &node);
        if id == 0 {
            RUNNING_VALIDATORS.set(&0u64, &1u32);
            let mut ids = Vec::new();
            ids.push(0u64);
            RUNNING_VALIDATOR_IDS.set(&ids);
        }
        Ok(id)
    }

    /// 更新 Secret 节点的基本信息。
    ///
    /// 允许 Secret 节点拥有者修改节点名称、IP 地址和端口。
    ///
    /// # 调用权限
    /// 仅 Secret 节点拥有者可调用。
    ///
    /// # 参数
    /// - `id`：Secret 节点 ID。
    /// - `name`：新的节点名称。
    /// - `ip`：新的 IP 地址。
    /// - `port`：新的服务端口号。
    ///
    /// # 返回值
    /// - `Ok(())`：更新成功。
    /// - `Err(Error::NodeNotExist)`：节点不存在。
    /// - `Err(Error::WorkerNotOwnedByCaller)`：调用者不是节点拥有者。
    #[revive(message, write)]
    pub fn secret_update(id: NodeID, name: Bytes, ip: Ip, port: u32) -> Result<(), Error> {
        let caller = env().caller();
        let mut node = SECRETS.get(&id).ok_or(Error::NodeNotExist)?;
        ensure!(node.owner == caller, Error::WorkerNotOwnedByCaller);
        // 活跃验证者不允许修改 IP，防止劫持共识网络流量
        // Active validators cannot change IP to prevent hijacking consensus traffic
        if RUNNING_VALIDATORS.get(&id).unwrap_or(0) > 0 {
            ensure!(node.ip == ip, Error::WorkerStatusNotReady);
        }
        node.name = name;
        node.ip = ip;
        node.port = port;
        SECRETS.set(&id, &node);
        Ok(())
    }

    /// 向 Secret 节点追加抵押（存款）。
    ///
    /// Secret 节点拥有者可以增加该节点的抵押金额。注意：此函数目前仅从调用参数中读取金额，
    /// 并未检查实际转账金额，具体存款逻辑可能需要配合前端或其他合约调用。
    ///
    /// # 调用权限
    /// 仅 Secret 节点拥有者可调用。
    ///
    /// # 参数
    /// - `id`：Secret 节点 ID。
    /// - `deposit`：要追加的抵押金额。
    ///
    /// # 返回值
    /// - `Ok(())`：存款成功。
    /// - `Err(Error::NodeNotExist)`：节点不存在。
    /// - `Err(Error::WorkerNotOwnedByCaller)`：调用者不是节点拥有者。
    #[revive(message, write)]
    pub fn secret_deposit(id: NodeID, deposit: U256) -> Result<(), Error> {
        let caller = env().caller();
        let node = SECRETS.get(&id).ok_or(Error::NodeNotExist)?;
        ensure!(node.owner == caller, Error::WorkerNotOwnedByCaller);
        let transferred = env().value_transferred();
        ensure!(transferred >= deposit, Error::DepositNotEnough);
        let mut amount = SECRET_MORTGAGES.get(&id).unwrap_or(U256::ZERO);
        // 防止 U256 充务溢出回绕：wrapping_add 后若结果小于加数，说明已进位溢出，饱和到 MAX
        // Prevent U256 deposit overflow wrap-around: if wrapping_add result < operand, overflow occurred; saturate at MAX
        let new_amount = amount.wrapping_add(transferred);
        amount = if new_amount < amount {
            U256::MAX
        } else {
            new_amount
        };
        SECRET_MORTGAGES.set(&id, &amount);
        Ok(())
    }

    /// 删除（注销）Secret 节点。
    ///
    /// 将 Secret 节点标记为终止状态（设置 terminal_block）。
    /// 删除前会检查该节点是否处于运行中或待处理列表，且抵押金额必须为 0。
    ///
    /// # 调用权限
    /// 仅 Secret 节点拥有者可调用。
    ///
    /// # 参数
    /// - `id`：要删除的 Secret 节点 ID。
    ///
    /// # 返回值
    /// - `Ok(())`：删除成功。
    /// - `Err(Error::NodeNotExist)`：节点不存在。
    /// - `Err(Error::WorkerNotOwnedByCaller)`：调用者不是节点拥有者。
    /// - `Err(Error::NodeIsRunning)`：节点仍在运行中或有抵押未清。
    #[revive(message, write)]
    pub fn secret_delete(id: NodeID) -> Result<(), Error> {
        let caller = env().caller();
        let mut node = SECRETS.get(&id).ok_or(Error::NodeNotExist)?;
        ensure!(node.owner == caller, Error::WorkerNotOwnedByCaller);
        let running_ids = RUNNING_VALIDATOR_IDS.get().unwrap_or_default();
        for nid in running_ids.iter() {
            if *nid == id {
                return Err(Error::NodeIsRunning);
            }
        }
        let pending_ids = PENDING_VALIDATOR_IDS.get().unwrap_or_default();
        for nid in pending_ids.iter() {
            if *nid == id {
                return Err(Error::NodeIsRunning);
            }
        }
        if SECRET_MORTGAGES.get(&id).unwrap_or(U256::ZERO) != U256::ZERO {
            return Err(Error::NodeIsRunning);
        }
        node.terminal_block = Some(env().block_number());
        SECRETS.set(&id, &node);
        Ok(())
    }

    /// 查询当前运行中的验证者列表。
    ///
    /// 返回当前 epoch 中正在运行的所有 Secret 验证者节点，包含节点 ID、节点详情和权重（power）。
    ///
    /// # 调用权限
    /// 任何人（只读查询）。
    ///
    /// # 返回值
    /// - `Vec<(u64, SecretNode, u32)>`：验证者 ID、节点详情和权重的列表。
    #[revive(message)]
    pub fn validators() -> Vec<(u64, SecretNode, u32)> {
        let ids = RUNNING_VALIDATOR_IDS.get().unwrap_or_default();
        let mut out = Vec::new();
        for id in ids.into_iter() {
            if let Some(power) = RUNNING_VALIDATORS.get(&id) {
                if power == 0 {
                    continue;
                }
                if let Some(node) = SECRETS.get(&id) {
                    out.push((id, node, power));
                }
            }
        }
        out
    }

    /// 将指定 Secret 节点加入验证者集合（待处理）。
    ///
    /// 将目标节点以权重 1 加入 `PENDING_VALIDATORS`。若该节点已在待处理列表中，
    /// 则将其权重设为 1。该变更将在下一个 epoch 生效。
    ///
    /// # 调用权限
    /// 仅治理合约（gov）可调用。
    ///
    /// # 参数
    /// - `id`：要加入验证者集合的 Secret 节点 ID。
    ///
    /// # 返回值
    /// - `Ok(())`：操作成功。
    /// - `Err(Error::MustCallByMainContract)`：调用者不是治理合约。
    /// - `Err(Error::NodeNotExist)`：节点不存在。
    #[revive(message, write)]
    pub fn validator_join(id: NodeID) -> Result<(), Error> {
        ensure_from_gov()?;
        SECRETS.get(&id).ok_or(Error::NodeNotExist)?;
        let mut ids = PENDING_VALIDATOR_IDS.get().unwrap_or_default();
        let mut found = false;
        for existing_id in ids.iter() {
            if *existing_id == id {
                found = true;
                break;
            }
        }
        if !found {
            ids.push(id);
            PENDING_VALIDATOR_IDS.set(&ids);
        }
        // 将待处理节点同步到 PENDING_VALIDATORS，权重默认设为 1（加入）
        // Sync pending node to PENDING_VALIDATORS with default weight 1 (join)
        PENDING_VALIDATORS.set(&id, &1u32);
        Ok(())
    }

    /// 将指定 Secret 节点从验证者集合中移除（待处理）。
    ///
    /// 将目标节点以权重 0 加入 `PENDING_VALIDATORS`。若该节点已在待处理列表中，
    /// 则将其权重设为 0（表示移除）。该变更将在下一个 epoch 生效。
    ///
    /// # 调用权限
    /// 仅治理合约（gov）可调用。
    ///
    /// # 参数
    /// - `id`：要从验证者集合中移除的 Secret 节点 ID。
    ///
    /// # 返回值
    /// - `Ok(())`：操作成功。
    /// - `Err(Error::MustCallByMainContract)`：调用者不是治理合约。
    #[revive(message, write)]
    pub fn validator_delete(id: NodeID) -> Result<(), Error> {
        ensure_from_gov()?;
        // 校验节点存在，与 validator_join 保持一致
        // Verify node exists, consistent with validator_join
        SECRETS.get(&id).ok_or(Error::NodeNotExist)?;
        let mut ids = PENDING_VALIDATOR_IDS.get().unwrap_or_default();
        let mut found = false;
        for existing_id in ids.iter() {
            if *existing_id == id {
                found = true;
                break;
            }
        }
        if !found {
            ids.push(id);
            PENDING_VALIDATOR_IDS.set(&ids);
        }
        PENDING_VALIDATORS.set(&id, &0u32);
        Ok(())
    }

    /// 推进到下一个 epoch 并更新验证者集合。
    ///
    /// 侧链调用此函数来触发 epoch 切换。首次调用时会将调用者地址记录为侧链多签地址；
    /// 后续调用必须由该多签地址发起。调用时需满足距离上一个 epoch 已超过 `epoch_slot` 区块数。
    /// epoch 切换后会调用 `calc_new_validators` 合并 pending 列表，更新运行中的验证者集合。
    ///
    /// # 调用权限
    /// 首次调用可为任何人（将调用者设为侧链多签地址）；后续仅侧链多签地址可调用。
    ///
    /// # 参数
    /// - `_node_id`：节点 ID（当前未使用，保留用于未来扩展）。
    ///
    /// # 返回值
    /// - `Ok(())`：epoch 切换成功。
    /// - `Err(Error::InvalidTeeChainCaller)`：调用者不是已记录的侧链多签地址。
    /// - `Err(Error::EpochNotExpired)`：当前 epoch 尚未到期。
    #[revive(message, write)]
    pub fn set_next_epoch(_node_id: u64) -> Result<(), Error> {
        let caller = env().caller();
        let now = env().block_number();
        let last_epoch = LAST_EPOCH_BLOCK.get().unwrap_or(0);

        // 首次调用将调用者设为侧链多签地址，后续仅该地址可调用
        // First call sets caller as tee-chain multi-sig address; subsequent calls require this address
        let key = TEE_CHAIN_MULTI_KEY.get().unwrap_or(Address::zero());
        if key == Address::zero() {
            TEE_CHAIN_MULTI_KEY.set(&caller);
        } else {
            ensure!(caller == key, Error::InvalidTeeChainCaller);
        }

        // 校验当前 epoch 是否已到期（距离上次切换达到 epoch_slot 个区块）
        // Verify current epoch has expired (blocks since last switch reached epoch_slot)
        let epoch_slot = EPOCH_SLOT.get().unwrap_or(72000) as u64;
        ensure!(
            (now as u64).saturating_sub(last_epoch as u64) >= epoch_slot,
            Error::EpochNotExpired
        );

        let epoch = EPOCH.get().unwrap_or(0);
        EPOCH.set(&(epoch + 1));
        LAST_EPOCH_BLOCK.set(&now);

        // 推进 epoch 后重新计算验证者集合，合并 pending 和 running 列表
        // Advance epoch and recalculate validator set, merging pending and running lists
        calc_new_validators();
        Ok(())
    }

    /// 查询下一个 epoch 的预期验证者集合。
    ///
    /// 模拟 epoch 切换后的验证者集合：将运行中的验证者与待处理列表合并，
    /// 过滤掉权重为 0 的节点，返回最终的验证者列表。
    /// 此函数仅做模拟计算，不会修改任何状态。
    ///
    /// # 调用权限
    /// 任何人（只读查询）。
    ///
    /// # 返回值
    /// - `Ok(Vec<(u64, SecretNode, u32)>)`：下一个 epoch 的验证者 ID、详情和权重列表。
    /// - `Err(Error::EpochNotExpired)`：当前 epoch 尚未接近到期（距离上次 epoch 不足 `epoch_slot - 5` 个区块）。
    #[revive(message)]
    pub fn next_epoch_validators() -> Result<Vec<(u64, SecretNode, u32)>, Error> {
        let now = env().block_number();
        let last_epoch = LAST_EPOCH_BLOCK.get().unwrap_or(0);
        let epoch_slot = EPOCH_SLOT.get().unwrap_or(72000);
        ensure!(
            (now as u64).saturating_sub(last_epoch as u64) >= (epoch_slot.saturating_sub(5)) as u64,
            Error::EpochNotExpired
        );
        let running_ids = RUNNING_VALIDATOR_IDS.get().unwrap_or_default();
        let pending_ids = PENDING_VALIDATOR_IDS.get().unwrap_or_default();
        let mut runnings: Vec<(u64, u32)> = running_ids
            .into_iter()
            .filter_map(|id| RUNNING_VALIDATORS.get(&id).map(|pow| (id, pow)))
            .collect();
        for pid in pending_ids {
            let ppow = PENDING_VALIDATORS.get(&pid).unwrap_or(0);
            if let Some(r) = runnings.iter_mut().find(|x| x.0 == pid) {
                r.1 = ppow;
            } else {
                runnings.push((pid, ppow));
            }
        }
        runnings.retain(|x| x.1 != 0);
        let out: Vec<(u64, SecretNode, u32)> = runnings
            .into_iter()
            .filter_map(|(id, power)| SECRETS.get(&id).map(|node| (id, node, power)))
            .collect();
        Ok(out)
    }

    fn calc_new_validators() {
        // Epoch 切换时重新计算验证者集合：合并 running + pending，剔除权重为 0 的节点
        // Recalculate validator set during epoch transition: merge running + pending, remove nodes with zero weight
        let running_ids = RUNNING_VALIDATOR_IDS.get().unwrap_or_default();
        let pending_ids = PENDING_VALIDATOR_IDS.get().unwrap_or_default();
        let mut runnings: Vec<(u64, u32)> = running_ids
            .into_iter()
            .filter_map(|id| RUNNING_VALIDATORS.get(&id).map(|pow| (id, pow)))
            .collect();
        // 将 pending 中的变更合并到 running：已存在则更新权重，不存在则新增
        // Merge pending changes into running: update weight if exists, add new if not
        for pid in pending_ids.iter() {
            let ppow = PENDING_VALIDATORS.get(&pid).unwrap_or(0);
            if let Some(r) = runnings.iter_mut().find(|x| x.0 == *pid) {
                r.1 = ppow;
            } else {
                runnings.push((*pid, ppow));
            }
        }
        // 过滤掉权重为 0 的节点（被移除的验证者）
        // Filter out nodes with weight 0 (removed validators)
        runnings.retain(|x| x.1 != 0);
        let new_ids: Vec<u64> = runnings.iter().map(|(id, _)| *id).collect();
        for (nid, pow) in runnings.into_iter() {
            RUNNING_VALIDATORS.set(&nid, &pow);
        }
        RUNNING_VALIDATOR_IDS.set(&new_ids);
        // 清除已处理的 pending 记录，防止存储膨胀
        // Clear processed pending records to prevent storage bloat
        for pid in pending_ids.iter() {
            PENDING_VALIDATORS.clear(&pid);
        }
        PENDING_VALIDATOR_IDS.set(&Vec::new());
    }

    fn ensure_from_gov() -> Result<(), Error> {
        let caller = env().caller();
        let gov = GOV_CONTRACT.get().unwrap_or(Address::zero());
        ensure!(caller == gov, Error::MustCallByMainContract);
        Ok(())
    }

    fn ensure_from_tee_chain() -> Result<(), Error> {
        let caller = env().caller();
        let key = TEE_CHAIN_MULTI_KEY.get().unwrap_or(Address::zero());
        ensure!(caller == key, Error::InvalidTeeChainCaller);
        Ok(())
    }

    fn transfer_native(to: &Address, amount: U256) -> Result<(), Error> {
        env()
            .transfer(to, &amount)
            .map_err(|_| Error::TransferFailed)
    }
}

#[cfg(test)]
mod tests;

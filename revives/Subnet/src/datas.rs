//! 子网合约数据类型（SCALE 编码），共享类型来自 primitives。

use parity_scale_codec::{Decode, Encode};
use wrevive_api::{AccountId, Address, BlockNumber, Bytes, U256};

pub use primitives::{AssetInfo, Ip, K8sCluster, NodeID, RunPrice};

#[derive(Clone, PartialEq, Eq, Debug, Encode, Decode)]
pub struct SecretNode {
    pub name: Bytes,
    pub owner: Address,
    pub validator_id: AccountId,
    pub p2p_id: AccountId,
    pub start_block: BlockNumber,
    pub terminal_block: Option<BlockNumber>,
    pub ip: Ip,
    pub port: u32,
    pub status: u8,
}

#[derive(Clone, PartialEq, Eq, Debug, Encode, Decode)]
pub struct AssetDeposit {
    pub amount: U256,
    pub cpu: u32,
    pub cvm_cpu: u32,
    pub mem: u32,
    pub cvm_mem: u32,
    pub disk: u32,
    pub gpu: u32,
    pub deleted: Option<BlockNumber>,
}

#[derive(Clone, PartialEq, Eq, Debug, Encode, Decode)]
pub struct EpochInfo {
    pub epoch: u32,
    pub epoch_slot: u32,
    pub last_epoch_block: BlockNumber,
    pub now: BlockNumber,
    pub side_chain_pub: Address,
}

/// 硬件等级最低资源规格要求
#[derive(Clone, PartialEq, Eq, Debug, Encode, Decode, Default)]
pub struct LevelRequirement {
    pub min_cpu: u32,
    pub min_mem: u32,
    pub min_cvm_cpu: u32,
    pub min_cvm_mem: u32,
    pub min_disk: u32,
    pub min_gpu: u32,
}

//! 跨合约共享数据类型（SCALE 编码），供 Cloud / Subnet / Pod 等合约统一使用。

use parity_scale_codec::{Decode, Encode};
use wrevive_api::{AccountId, Address, BlockNumber, Bytes, H256};

/// Subnet worker 节点 ID
pub type NodeID = u64;

/// IP 表示（IPv4/IPv6/域名）
#[derive(Clone, PartialEq, Eq, Debug, Encode, Decode, Default)]
pub struct Ip {
    pub ipv4: Option<u32>,
    pub ipv6: Option<u128>,
    pub domain: Option<Bytes>,
}

/// K8s worker 节点信息（Subnet::worker 返回值等）
#[derive(Clone, PartialEq, Eq, Debug, Encode, Decode, Default)]
pub struct K8sCluster {
    pub name: Bytes,
    pub owner: Address,
    pub level: u8,
    pub region_id: u32,
    pub start_block: BlockNumber,
    pub stop_block: Option<BlockNumber>,
    pub terminal_block: Option<BlockNumber>,
    pub p2p_id: AccountId,
    pub ip: Ip,
    pub port: u32,
    pub status: u8,
}

/// 运行价格（用于 mint_pod 等计费）
#[derive(Clone, PartialEq, Eq, Debug, Encode, Decode)]
pub struct RunPrice {
    pub cpu_per: u64,
    pub cvm_cpu_per: u64,
    pub memory_per: u64,
    pub cvm_memory_per: u64,
    pub disk_per: u64,
    pub gpu_per: u64,
}

/// 资产类型（原生 / ERC20，与 Pod/Subnet 一致）
#[derive(Clone, PartialEq, Eq, Debug, Encode, Decode)]
pub enum AssetInfo {
    Native(Bytes),
    ERC20(Bytes, H256),
}

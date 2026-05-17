//! 子网合约错误类型（与 inks Subnet 对齐）

use parity_scale_codec::{Decode, Encode};

#[derive(Clone, Copy, Debug, PartialEq, Eq, Encode, Decode)]
pub enum Error {
    NotEnoughBalance,
    MustCallByMainContract,
    WorkerNotExist,
    WorkerNotOwnedByCaller,
    WorkerStatusNotReady,
    WorkerMortgageNotExist,
    TransferFailed,
    WorkerIsUseByUser,
    NodeNotExist,
    SecretNodeAlreadyExists,
    SetCodeFailed,
    EpochNotExpired,
    InvalidSideChainSignature,
    NodeIsRunning,
    InvalidSideChainCaller,
    RegionNotExist,
    AssetNotExist,
    DepositNotEnough,
    MortgageNotEnough,
    SlashAmountTooLarge,
    CloudContractNotSet,
    ResourceNotEnough,
}

//! Token 合约错误类型

use parity_scale_codec::{Decode, Encode};

#[derive(Clone, Copy, Debug, PartialEq, Eq, Encode, Decode)]
pub enum Error {
    /// 非 Subnet 合约调用提现
    OnlySubnet,
    /// 非 Owner 调用管理接口
    OnlyOwner,
    /// 金额必须大于 0
    AmountMustBeGreaterThanZero,
    /// 余额不足
    InsufficientBalance,
    /// 转账失败
    TransferFailed,
    /// 零地址
    ZeroAddress,
}

//! 代理合约错误类型

use parity_scale_codec::{Decode, Encode};

#[derive(Clone, Copy, Debug, PartialEq, Eq, Encode, Decode)]
pub enum Error {
    /// 非管理员调用升级或转移管理员
    Unauthorized,
    AddressNotFound,
}

#![no_std]

//! 供 PolkaVM/wrevive 合约使用的轻量 primitives：ensure! / ok_or_err! 宏 + 跨合约共享类型。

pub mod types;
pub use types::*;

#[macro_export]
macro_rules! ensure {
    ($condition:expr, $error:expr) => {
        if !$condition {
            return Err($error);
        }
    };
}

#[macro_export]
macro_rules! ok_or_err {
    ($result:expr, $error:expr) => {
        match $result {
            Ok(val) => val,
            Err(_) => return Err($error),
        }
    };
}

//! Token 合约 — PolkaVM/wrevive 版本
//! 积分充值/提现合约，从 Solidity Token.sol 迁移。
//!
//! Subnet 合约作为管理员。用户充值直接调用，提现必须由 Subnet 合约调用。
//! 配合 Proxy 合约实现 UUPS 可升级模式。

#![cfg_attr(not(test), no_std)]
#![cfg_attr(not(test), no_main)]

extern crate alloc;

#[cfg(all(not(test), not(feature = "api")))]
#[global_allocator]
static ALLOC: pvm_bump_allocator::BumpAllocator<65536> = pvm_bump_allocator::BumpAllocator::new();

mod errors;
#[cfg(test)]
mod tests;

use wrevive_api::*;
use wrevive_macro::{mapping, revive_contract, storage};

pub use errors::Error;
pub use primitives::ensure;

#[revive_contract]
pub mod token {
    use super::*;
    use crate::{ensure, Error};

    /// 合约 owner 地址
    const OWNER: Storage<Address> = storage!(b"owner");
    /// Subnet 合约地址（仅该地址可调用 withdraw）
    const SUBNET: Storage<Address> = storage!(b"subnet");
    /// 兑换率：1 ETH = rate 积分
    const RATE: Storage<U256> = storage!(b"rate");
    /// 用户余额：eth 充值量
    const BALANCES: Mapping<Address, U256> = mapping!(b"balances");

    // ========== 构造函数 ==========

    /// Token 合约构造函数。
    ///
    /// 仅分配存储，不做初始化。初始化逻辑由 `init` 完成（代理模式）。
    #[revive(constructor)]
    pub fn new() -> Result<(), Error> {
        Ok(())
    }

    // ========== 初始化 ==========

    /// 代理初始化（替代 constructor）。
    ///
    /// - `owner`：合约管理员地址。传 `None` 则使用 caller。
    ///
    /// # 调用权限
    /// 任何人（仅在首次调用时有效，重复调用无操作）。
    #[revive(message, write)]
    pub fn init(owner: Option<Address>) -> Result<(), Error> {
        // 已初始化则跳过
        if OWNER.get().is_some() {
            return Ok(());
        }
        let api = env();
        let owner_addr = owner.unwrap_or(api.caller());
        OWNER.set(&owner_addr);
        RATE.set(&U256::from(1000u64));
        Ok(())
    }

    // ========== 配置 ==========

    /// 设置 Subnet 合约地址（仅 owner）
    #[revive(message, write)]
    pub fn set_subnet(subnet_addr: Address) -> Result<(), Error> {
        ensure_owner()?;
        ensure!(subnet_addr != Address::zero(), Error::ZeroAddress);
        SUBNET.set(&subnet_addr);
        Ok(())
    }

    /// 设置兑换率（仅 owner）
    #[revive(message, write)]
    pub fn set_rate(new_rate: U256) -> Result<(), Error> {
        ensure_owner()?;
        ensure!(
            new_rate > U256::from(0u64),
            Error::AmountMustBeGreaterThanZero
        );
        RATE.set(&new_rate);
        Ok(())
    }

    // ========== 充值/提现 ==========

    /// 用户充值 ETH 换取积分。
    ///
    /// 调用时需附带 ETH（通过 `transferred_value`）。返回获得的积分数量。
    ///
    /// # 返回值
    /// - `Ok(points_amount)`：充值成功，返回积分数量。
    #[revive(message, write, payable)]
    pub fn recharge() -> Result<U256, Error> {
        let api = env();
        let caller = api.caller();
        let value = api.transferred_value();
        ensure!(value > U256::from(0u64), Error::AmountMustBeGreaterThanZero);

        let rate = RATE.get().unwrap_or(U256::from(1000u64));
        let points_amount = value * rate;

        // 更新余额
        let current = BALANCES.get(&caller).unwrap_or(U256::from(0u64));
        BALANCES.set(&caller, &(current + value));

        Ok(points_amount)
    }

    /// 提现（仅 Subnet 合约可调用）。
    ///
    /// - `user`：提现目标用户地址。
    /// - `eth_amount`：提现 ETH 数量。
    #[revive(message, write)]
    pub fn withdraw(user: Address, eth_amount: U256) -> Result<(), Error> {
        ensure_subnet()?;
        ensure!(
            eth_amount > U256::from(0u64),
            Error::AmountMustBeGreaterThanZero
        );

        let balance = BALANCES.get(&user).unwrap_or(U256::from(0u64));
        ensure!(balance >= eth_amount, Error::InsufficientBalance);

        // 扣减余额
        BALANCES.set(&user, &(balance - eth_amount));

        // 转账 ETH 给用户
        let api = env();
        api.transfer(&user, &eth_amount)
            .map_err(|_| Error::TransferFailed)?;

        Ok(())
    }

    // ========== 管理 ==========

    /// 紧急提现：将合约内所有 ETH 转给 owner。
    #[revive(message, write)]
    pub fn emergency_withdraw() -> Result<(), Error> {
        ensure_owner()?;
        let api = env();
        let contract_balance = api.balance();
        if contract_balance > U256::from(0u64) {
            let owner_addr = OWNER.get().unwrap_or(Address::zero());
            api.transfer(&owner_addr, &contract_balance)
                .map_err(|_| Error::TransferFailed)?;
        }
        Ok(())
    }

    // ========== 查询 ==========

    /// 查询用户余额（ETH 充值量）。
    #[revive(message)]
    pub fn get_balance(user: Address) -> U256 {
        BALANCES.get(&user).unwrap_or(U256::from(0u64))
    }

    /// 将 ETH 数量换算为积分数量。
    #[revive(message)]
    pub fn to_points(eth_amount: U256) -> U256 {
        let rate = RATE.get().unwrap_or(U256::from(1000u64));
        eth_amount * rate
    }

    /// 查询当前兑换率。
    #[revive(message)]
    pub fn get_rate() -> U256 {
        RATE.get().unwrap_or(U256::from(1000u64))
    }

    /// 查询 Subnet 合约地址。
    #[revive(message)]
    pub fn get_subnet() -> Address {
        SUBNET.get().unwrap_or(Address::zero())
    }

    /// 查询合约 owner 地址。
    #[revive(message)]
    pub fn owner() -> Address {
        OWNER.get().unwrap_or(Address::zero())
    }

    // ========== 默认充值（fallback） ==========

    /// receive() 等价：直接向合约转账 ETH 即视为充值。
    #[revive(fallback, payable)]
    pub fn fallback() {
        let api = env();
        let caller = api.caller();
        let value = api.transferred_value();

        if value > U256::from(0u64) {
            let current = BALANCES.get(&caller).unwrap_or(U256::from(0u64));
            BALANCES.set(&caller, &(current + value));
        }
    }

    // ========== 内部辅助 ==========

    /// 确保调用者为 Subnet 合约。
    fn ensure_subnet() -> Result<(), Error> {
        let caller = env().caller();
        let subnet = SUBNET.get().unwrap_or(Address::zero());
        ensure!(caller == subnet, Error::OnlySubnet);
        Ok(())
    }

    /// 确保调用者为合约 owner。
    fn ensure_owner() -> Result<(), Error> {
        let caller = env().caller();
        let owner = OWNER.get().unwrap_or(Address::zero());
        ensure!(caller == owner, Error::OnlyOwner);
        Ok(())
    }
}

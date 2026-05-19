//! Token 合约 — PolkaVM/wrevive 版本
//! 积分充值/提现合约，从 Solidity Token.sol 迁移。
//!
//! 积分规则 | Point rules:
//!   points = value * RATE / TOKEN_UNIT  (Planck → 积分)
//!   eth    = points * TOKEN_UNIT / RATE  (积分 → Planck)
//!   例: DOT=$4, RATE=4000, TOKEN_UNIT=10^15 → 1 DOT = 4000 积分, 1000 积分 ≈ $1
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

use parity_scale_codec::Encode;
use wrevive_api::*;
use wrevive_macro::{mapping, revive_contract, storage};

pub use errors::Error;
pub use primitives::ensure;

/// Relay 事件记录 | Relay event record
#[derive(Clone, Encode, Decode, Debug)]
pub struct EventRecord {
    pub target_contract: Vec<u8>,
    pub event_type: Vec<u8>,
    pub event_data: Vec<u8>,
}

#[revive_contract]
pub mod token {
    use super::*;
    use crate::{Error, EventRecord, ensure};

    const OWNER: Storage<Address> = storage!(b"owner");
    const SUBNET: Storage<Address> = storage!(b"subnet");
    /// 兑换率: 1 DOT = RATE 积分 | RATE points per 1 DOT
    const RATE: Storage<U256> = storage!(b"rate");
    /// 主链币最小单位 (DOT=10^15 Planck, ETH=10^18 Wei)
    const TOKEN_UNIT: Storage<U256> = storage!(b"token_unit");
    /// 用户积分余额 (1 积分 ≈ 1/1000 USD)
    const BALANCES: Mapping<Address, U256> = mapping!(b"balances");
    /// 全局事件 nonce
    const EVENT_NONCE: Storage<u64> = storage!(b"event_nonce");
    /// 事件存储: nonce → EventRecord
    const EVENTS: Mapping<u64, EventRecord> = mapping!(b"events");

    // ========== 构造函数 ==========
    #[revive(constructor)]
    pub fn new() -> Result<(), Error> {
        Ok(())
    }

    // ========== 初始化 ==========
    #[revive(message, write)]
    pub fn init(owner: Option<Address>) -> Result<(), Error> {
        if OWNER.get().is_some() {
            return Ok(());
        }
        let api = env();
        let owner_addr = owner.unwrap_or(api.caller());
        OWNER.set(&owner_addr);
        if RATE.get().is_none() {
            RATE.set(&U256::from(1u64));
        } // $4/DOT → $1=1000pt
        if TOKEN_UNIT.get().is_none() {
            TOKEN_UNIT.set(&U256::from(10_000_000_000u64));
        } // DOT: 10^15 Planck
        if EVENT_NONCE.get().is_none() {
            EVENT_NONCE.set(&0u64);
        }
        Ok(())
    }

    // ========== 配置 ==========
    #[revive(message, write)]
    pub fn set_subnet(subnet_addr: Address) -> Result<(), Error> {
        ensure_owner()?;
        ensure!(subnet_addr != Address::zero(), Error::ZeroAddress);
        SUBNET.set(&subnet_addr);
        emit_event(b"gateway", b"SetSubnet", Encode::encode(&subnet_addr));
        Ok(())
    }

    #[revive(message, write)]
    pub fn set_rate(new_rate: U256) -> Result<(), Error> {
        ensure_owner()?;
        ensure!(
            new_rate > U256::from(0u64),
            Error::AmountMustBeGreaterThanZero
        );
        RATE.set(&new_rate);
        emit_event(b"gateway", b"SetRate", Encode::encode(&new_rate));
        Ok(())
    }

    #[revive(message, write)]
    pub fn set_token_unit(unit: U256) -> Result<(), Error> {
        ensure_owner()?;
        ensure!(unit > U256::from(0u64), Error::AmountMustBeGreaterThanZero);
        TOKEN_UNIT.set(&unit);
        emit_event(b"gateway", b"SetTokenUnit", Encode::encode(&unit));
        Ok(())
    }

    // ========== 充值/提现 (SCALE) ==========
    #[revive(message, write, payable)]
    pub fn recharge() -> Result<U256, Error> {
        recharge_impl()
    }

    #[revive(message, write)]
    pub fn withdraw(user: Address, points: U256) -> Result<(), Error> {
        withdraw_impl(user, points)
    }

    // ========== 充值/提现 (Sol ABI) ==========
    #[revive(message, write, payable, sol)]
    pub fn recharge_sol() -> Result<U256, Error> {
        recharge_impl()
    }

    #[revive(message, write, sol)]
    pub fn withdraw_sol(user: Address, points: U256) -> Result<(), Error> {
        withdraw_impl(user, points)
    }

    // ========== Relay 查询 ==========
    #[revive(message)]
    pub fn get_latest_nonce() -> u64 {
        EVENT_NONCE.get().unwrap_or(0u64)
    }

    #[revive(message)]
    pub fn get_events(from: u64, to: u64) -> Vec<EventRecord> {
        let end = if to == 0 {
            EVENT_NONCE.get().unwrap_or(0u64)
        } else {
            to
        };
        let mut events = Vec::new();
        for n in from..=end {
            if let Some(ev) = EVENTS.get(&n) {
                events.push(ev);
            }
        }
        events
    }

    #[revive(message)]
    pub fn get_event(nonce: u64) -> Option<EventRecord> {
        EVENTS.get(&nonce)
    }

    // ========== 查询 (SCALE) ==========
    #[revive(message)]
    pub fn get_balance(user: Address) -> U256 {
        BALANCES.get(&user).unwrap_or(U256::from(0u64))
    }

    /// DOT → 积分换算 (只读) | DOT → points conversion (read-only)
    #[revive(message)]
    pub fn to_points(dot_amount: U256) -> U256 {
        let rate = RATE.get().unwrap_or(U256::from(1u64));
        let token_unit = TOKEN_UNIT.get().unwrap_or(U256::from(10_000_000_000u64));
        dot_amount * rate / token_unit
    }

    #[revive(message)]
    pub fn get_rate() -> U256 {
        RATE.get().unwrap_or(U256::from(1u64))
    }

    #[revive(message)]
    pub fn get_token_unit() -> U256 {
        TOKEN_UNIT.get().unwrap_or(U256::from(10_000_000_000u64))
    }

    #[revive(message)]
    pub fn get_subnet() -> Address {
        SUBNET.get().unwrap_or(Address::zero())
    }

    #[revive(message)]
    pub fn owner() -> Address {
        OWNER.get().unwrap_or(Address::zero())
    }

    // ========== 查询 (Sol ABI) ==========
    #[revive(message, sol)]
    pub fn get_balance_sol(user: Address) -> U256 {
        BALANCES.get(&user).unwrap_or(U256::from(0u64))
    }

    #[revive(message, sol)]
    pub fn to_points_sol(dot_amount: U256) -> U256 {
        let rate = RATE.get().unwrap_or(U256::from(1u64));
        dot_amount * rate
    }

    #[revive(message, sol)]
    pub fn get_rate_sol() -> U256 {
        RATE.get().unwrap_or(U256::from(1u64))
    }

    #[revive(message, sol)]
    pub fn owner_sol() -> Address {
        OWNER.get().unwrap_or(Address::zero())
    }

    // ========== 管理 ==========
    #[revive(message, write)]
    pub fn emergency_withdraw() -> Result<(), Error> {
        ensure_owner()?;
        let api = env();
        let contract_balance = api.balance();
        if contract_balance > U256::from(0u64) {
            let owner_addr = OWNER.get().unwrap_or(Address::zero());
            api.transfer(&owner_addr, &contract_balance)
                .map_err(|_| Error::TransferFailed)?;
            emit_event(
                b"gateway",
                b"EmergencyWithdraw",
                Encode::encode(&contract_balance),
            );
        }
        Ok(())
    }

    // ========== 默认充值（fallback） ==========
    #[revive(fallback, payable)]
    pub fn fallback() {
        let api = env();
        let value = api.value_transferred();
        if value > U256::from(0u64) {
            let points = value_to_points(value);
            let caller = api.caller();
            let current = BALANCES.get(&caller).unwrap_or(U256::from(0u64));
            BALANCES.set(&caller, &(current + points));
            emit_event(b"gateway", b"Recharge", Encode::encode(&(caller, points)));
        }
    }

    // ========== 内部实现 ==========
    fn recharge_impl() -> Result<U256, Error> {
        let api = env();
        let value = api.value_transferred();
        ensure!(value > U256::from(0u64), Error::AmountMustBeGreaterThanZero);
        let points = value_to_points(value);
        ensure!(
            points > U256::from(0u64),
            Error::AmountMustBeGreaterThanZero
        );
        let caller = api.caller();
        let current = BALANCES.get(&caller).unwrap_or(U256::from(0u64));
        BALANCES.set(&caller, &(current + points));
        emit_event(b"gateway", b"Recharge", Encode::encode(&(caller, points)));
        Ok(points)
    }

    fn withdraw_impl(user: Address, points: U256) -> Result<(), Error> {
        ensure_subnet()?;
        ensure!(
            points > U256::from(0u64),
            Error::AmountMustBeGreaterThanZero
        );
        let balance = BALANCES.get(&user).unwrap_or(U256::from(0u64));
        ensure!(balance >= points, Error::InsufficientBalance);
        let eth_amount = points_to_value(points);
        ensure!(
            eth_amount > U256::from(0u64),
            Error::AmountMustBeGreaterThanZero
        );
        BALANCES.set(&user, &(balance - points));
        env()
            .transfer(&user, &eth_amount)
            .map_err(|_| Error::TransferFailed)?;
        emit_event(b"gateway", b"Withdraw", Encode::encode(&(user, points)));
        Ok(())
    }

    // ========== 积分换算 ==========
    /// Planck → 积分: value * RATE / TOKEN_UNIT
    fn value_to_points(value: U256) -> U256 {
        let rate = RATE.get().unwrap_or(U256::from(1u64));
        let unit = TOKEN_UNIT.get().unwrap_or(U256::from(10_000_000_000u64));
        value * rate / unit
    }

    /// 积分 → Planck: points * TOKEN_UNIT / RATE
    fn points_to_value(points: U256) -> U256 {
        let rate = RATE.get().unwrap_or(U256::from(1u64));
        let unit = TOKEN_UNIT.get().unwrap_or(U256::from(10_000_000_000u64));
        points * unit / rate
    }

    // ========== Relay 内部 ==========
    fn emit_event(target_contract: &[u8], event_type: &[u8], event_data: Vec<u8>) {
        let nonce = EVENT_NONCE.get().unwrap_or(0u64) + 1;
        EVENT_NONCE.set(&nonce);
        EVENTS.set(
            &nonce,
            &EventRecord {
                target_contract: target_contract.to_vec(),
                event_type: event_type.to_vec(),
                event_data,
            },
        );
    }

    // ========== 内部辅助 ==========
    fn ensure_subnet() -> Result<(), Error> {
        let caller = env().caller();
        let subnet = SUBNET.get().unwrap_or(Address::zero());
        ensure!(caller == subnet, Error::OnlySubnet);
        Ok(())
    }

    fn ensure_owner() -> Result<(), Error> {
        let caller = env().caller();
        let owner = OWNER.get().unwrap_or(Address::zero());
        ensure!(caller == owner, Error::OnlyOwner);
        Ok(())
    }
}

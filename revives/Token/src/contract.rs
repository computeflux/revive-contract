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

use pallet_revive_uapi::CallFlags;
use parity_scale_codec::Encode;
use wrevive_api::*;
use wrevive_macro::{mapping, revive_contract, storage};

pub use errors::Error;
pub use primitives::{UniAddr, ensure};

/// Relay 事件记录 | Relay event record
#[derive(Clone, Encode, Decode, Debug)]
pub struct EventRecord {
    pub target_contract: Vec<u8>,
    pub event_type: Vec<u8>,
    pub event_data: Vec<Vec<u8>>,
}

/// ERC20 代币配置 | ERC20 token config
#[derive(Clone, Encode, Decode, Debug, PartialEq, Eq)]
pub struct ERC20TokenConfig {
    pub active: bool,
    pub rate: U256,
    pub unit: U256,
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
    /// ERC20 代币注册表: token_address → config
    const ERC20_TOKENS: Mapping<Address, ERC20TokenConfig> = mapping!(b"erc20_tokens");
    /// ERC20 代币地址列表（用于遍历查询）
    const ERC20_TOKEN_LIST: Storage<Vec<Address>> = storage!(b"erc20_token_list");

    /// Native token 充值开关 (默认开启)
    const NATIVE_ACTIVE: Storage<bool> = storage!(b"native_active");

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
            RATE.set(&U256::from(4000u64));
        } // $4/DOT → $1=1000pt
        if TOKEN_UNIT.get().is_none() {
            TOKEN_UNIT.set(&U256::from(1_000_000_000_000_000u64));
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
        Ok(())
    }

    #[revive(message, write)]
    pub fn set_token_unit(unit: U256) -> Result<(), Error> {
        ensure_owner()?;
        ensure!(unit > U256::from(0u64), Error::AmountMustBeGreaterThanZero);
        TOKEN_UNIT.set(&unit);
        Ok(())
    }

    /// 启用/禁用 Native token 充值
    #[revive(message, write)]
    pub fn set_native_active(active: bool) -> Result<(), Error> {
        ensure_owner()?;
        NATIVE_ACTIVE.set(&active);
        Ok(())
    }

    // ========== ERC20 管理 ==========

    /// 注册/更新 ERC20 代币 | Register / update ERC20 token
    #[revive(message, write)]
    pub fn set_erc20_token(
        token: Address,
        active: bool,
        rate: U256,
        unit: U256,
    ) -> Result<(), Error> {
        ensure_owner()?;
        ensure!(token != Address::zero(), Error::ZeroAddress);
        ensure!(unit > U256::from(0u64), Error::AmountMustBeGreaterThanZero);

        // 新代币加入列表
        if ERC20_TOKENS.get(&token).is_none() {
            let mut list = ERC20_TOKEN_LIST.get().unwrap_or_default();
            list.push(token);
            ERC20_TOKEN_LIST.set(&list);
        }

        ERC20_TOKENS.set(&token, &ERC20TokenConfig { active, rate, unit });
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

    #[revive(message, write)]
    pub fn withdraw_erc20(token: Address, user: Address, points: U256) -> Result<(), Error> {
        withdraw_erc20_impl(token, user, points)
    }

    // ========== 充值/提现 (Sol ABI) ==========
    #[revive(message, write, payable, sol)]
    pub fn recharge_sol() -> Result<U256, Error> {
        recharge_impl()
    }

    #[revive(message, write, sol)]
    pub fn recharge_erc20(token: Address, amount: U256) -> Result<U256, Error> {
        recharge_erc20_impl(token, amount)
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

    /// DOT → 积分换算 (只读) | DOT → points conversion (read-only)
    #[revive(message)]
    pub fn to_points(dot_amount: U256) -> U256 {
        let rate = RATE.get().unwrap_or(U256::from(4000u64));
        let token_unit = TOKEN_UNIT
            .get()
            .unwrap_or(U256::from(1_000_000_000_000_000u64));
        dot_amount * rate / token_unit
    }

    #[revive(message)]
    pub fn get_rate() -> U256 {
        RATE.get().unwrap_or(U256::from(4000u64))
    }

    #[revive(message)]
    pub fn get_token_unit() -> U256 {
        TOKEN_UNIT
            .get()
            .unwrap_or(U256::from(1_000_000_000_000_000u64))
    }

    /// Native token 充值是否启用
    #[revive(message)]
    pub fn get_native_active() -> bool {
        NATIVE_ACTIVE.get().unwrap_or(true) // 默认开启
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
    pub fn to_points_sol(dot_amount: U256) -> U256 {
        let rate = RATE.get().unwrap_or(U256::from(4000u64));
        let token_unit = TOKEN_UNIT
            .get()
            .unwrap_or(U256::from(1_000_000_000_000_000u64));
        dot_amount * rate / token_unit
    }

    #[revive(message, sol)]
    pub fn get_rate_sol() -> U256 {
        RATE.get().unwrap_or(U256::from(4000u64))
    }

    #[revive(message, sol)]
    pub fn owner_sol() -> Address {
        OWNER.get().unwrap_or(Address::zero())
    }

    /// Native token 充值是否启用 (Sol ABI)
    #[revive(message, sol)]
    pub fn get_native_active_sol() -> bool {
        NATIVE_ACTIVE.get().unwrap_or(true) // 默认开启
    }

    // ========== ERC20 查询 ==========
    #[revive(message)]
    pub fn get_erc20_config(token: Address) -> (bool, U256, U256) {
        match ERC20_TOKENS.get(&token) {
            Some(c) => (c.active, c.rate, c.unit),
            None => (false, U256::from(0u64), U256::from(0u64)),
        }
    }

    #[revive(message)]
    pub fn get_erc20_count() -> u64 {
        let list = ERC20_TOKEN_LIST.get().unwrap_or_default();
        list.len() as u64
    }

    /// 返回 ERC20 代币列表 | Return ERC20 token list
    #[revive(message)]
    pub fn get_erc20_list() -> Vec<(Address, bool, U256, U256)> {
        let list = ERC20_TOKEN_LIST.get().unwrap_or_default();
        let mut result = Vec::with_capacity(list.len());
        for addr in &list {
            if let Some(c) = ERC20_TOKENS.get(addr) {
                result.push((*addr, c.active, c.rate, c.unit));
            }
        }
        result
    }

    // ========== ERC20 查询 sol ==========
    #[revive(message, sol)]
    pub fn get_erc20_config_sol(token: Address) -> (bool, U256, U256) {
        match ERC20_TOKENS.get(&token) {
            Some(c) => (c.active, c.rate, c.unit),
            None => (false, U256::from(0u64), U256::from(0u64)),
        }
    }

    /// 查询合约持有的 ERC20 代币余额 | Query ERC20 token balance held by this contract
    #[revive(message)]
    pub fn get_erc20_balance(token: Address) -> U256 {
        let contract_addr = env().address();
        let calldata = build_balance_of_calldata(&contract_addr);
        call_erc20_view(&token, &calldata).unwrap_or(U256::from(0u64))
    }

    /// 已注册 ERC20 代币数量 (Sol ABI)
    #[revive(message, sol)]
    pub fn get_erc20_count_sol() -> U256 {
        let list = ERC20_TOKEN_LIST.get().unwrap_or_default();
        U256::from(list.len() as u64)
    }

    /// 返回 ERC20 代币列表 | Return ERC20 token list
    #[revive(message, sol)]
    pub fn get_erc20_list_sol() -> Vec<(Address, bool, U256, U256)> {
        let list = ERC20_TOKEN_LIST.get().unwrap_or_default();
        let mut result = Vec::with_capacity(list.len());
        for addr in &list {
            if let Some(c) = ERC20_TOKENS.get(addr) {
                result.push((*addr, c.active, c.rate, c.unit));
            }
        }
        result
    }

    // ========== 默认充值（fallback） ==========
    #[revive(fallback, payable)]
    pub fn fallback() {
        let api = env();
        let value = api.value_transferred();
        if value > U256::from(0u64) {
            // 检查 Native token 充值是否启用
            if !NATIVE_ACTIVE.get().unwrap_or(true) {
                api.return_value(ReturnFlags::REVERT, &[]);
            }
            let points = value_to_points(value);
            let caller = api.caller();

            // 积分由 TEE 子链管理，合约只发事件
            let mut args = Vec::new();
            args.push(Encode::encode(&UniAddr {
                t: 2,
                v: caller.as_ref().to_vec(),
            }));
            args.push(Encode::encode(&points));
            emit_event(b"gateway", b"Recharge", args);
        }
    }

    // ========== 内部实现 ==========
    fn recharge_impl() -> Result<U256, Error> {
        ensure_native_active()?;
        let api = env();
        let value = api.value_transferred();
        ensure!(value > U256::from(0u64), Error::AmountMustBeGreaterThanZero);

        let points = value_to_points(value);
        ensure!(
            points > U256::from(0u64),
            Error::AmountMustBeGreaterThanZero
        );

        let caller = api.caller();
        // 积分由 TEE 子链管理，合约只发事件
        let mut args = Vec::new();
        args.push(Encode::encode(&UniAddr {
            t: 2,
            v: caller.as_ref().to_vec(),
        }));
        args.push(Encode::encode(&points));
        emit_event(b"gateway", b"Recharge", args);
        Ok(points)
    }

    fn withdraw_impl(user: Address, points: U256) -> Result<(), Error> {
        ensure_subnet()?;
        ensure!(
            points > U256::from(0u64),
            Error::AmountMustBeGreaterThanZero
        );

        // 积分余额由 TEE 子链校验，合约只负责转账
        let eth_amount = points_to_value(points);
        ensure!(
            eth_amount > U256::from(0u64),
            Error::AmountMustBeGreaterThanZero
        );

        env()
            .transfer(&user, &eth_amount)
            .map_err(|_| Error::TransferFailed)?;

        let mut args = Vec::new();
        args.push(Encode::encode(&UniAddr {
            t: 2,
            v: user.as_ref().to_vec(),
        }));
        args.push(Encode::encode(&points));
        emit_event(b"gateway", b"Withdraw", args);
        Ok(())
    }

    // ========== ERC20 内部实现 ==========
    fn recharge_erc20_impl(token: Address, amount: U256) -> Result<U256, Error> {
        let config = ensure_erc20_active(&token)?;
        ensure!(
            amount > U256::from(0u64),
            Error::AmountMustBeGreaterThanZero
        );

        let caller = env().caller();
        let contract_addr = env().address();

        // 调用 ERC20.transferFrom(caller, self, amount)
        let calldata = build_transfer_from_calldata(&caller, &contract_addr, amount);
        let _ = call_erc20(&token, &calldata).map_err(|_| Error::ERC20TransferFailed)?;

        // 换算积分: points = amount * rate / unit
        let points = amount * config.rate / config.unit;
        ensure!(
            points > U256::from(0u64),
            Error::AmountMustBeGreaterThanZero
        );

        // 积分由 TEE 子链管理，合约只发事件
        let mut args = Vec::new();
        args.push(Encode::encode(&UniAddr {
            t: 2,
            v: caller.as_ref().to_vec(),
        }));
        args.push(Encode::encode(&token));
        args.push(Encode::encode(&amount));
        args.push(Encode::encode(&points));
        emit_event(b"gateway", b"RechargeERC20", args);
        Ok(points)
    }

    fn withdraw_erc20_impl(token: Address, user: Address, points: U256) -> Result<(), Error> {
        ensure_subnet()?;
        let config = ensure_erc20_active(&token)?;
        ensure!(
            points > U256::from(0u64),
            Error::AmountMustBeGreaterThanZero
        );

        // 换算 ERC20 数量: amount = points * unit / rate
        let amount = points * config.unit / config.rate;
        ensure!(
            amount > U256::from(0u64),
            Error::AmountMustBeGreaterThanZero
        );

        // 调用 ERC20.transfer(user, amount)
        let calldata = build_transfer_calldata(&user, amount);
        let _ = call_erc20(&token, &calldata).map_err(|_| Error::ERC20TransferFailed)?;

        let mut args = Vec::new();
        args.push(Encode::encode(&UniAddr {
            t: 2,
            v: user.as_ref().to_vec(),
        }));
        args.push(Encode::encode(&token));
        args.push(Encode::encode(&amount));
        args.push(Encode::encode(&points));
        emit_event(b"gateway", b"WithdrawERC20", args);
        Ok(())
    }

    // ========== 积分换算 ==========
    /// Planck → 积分: value * RATE / TOKEN_UNIT
    fn value_to_points(value: U256) -> U256 {
        let rate = RATE.get().unwrap_or(U256::from(4000u64));
        let unit = TOKEN_UNIT
            .get()
            .unwrap_or(U256::from(1_000_000_000_000_000u64));
        value * rate / unit
    }

    /// 积分 → Planck: points * TOKEN_UNIT / RATE
    fn points_to_value(points: U256) -> U256 {
        let rate = RATE.get().unwrap_or(U256::from(4000u64));
        let unit = TOKEN_UNIT
            .get()
            .unwrap_or(U256::from(1_000_000_000_000_000u64));
        points * unit / rate
    }

    // ========== Relay 内部 ==========
    fn emit_event(target_contract: &[u8], event_type: &[u8], event_data: Vec<Vec<u8>>) {
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

    fn ensure_native_active() -> Result<(), Error> {
        let active = NATIVE_ACTIVE.get().unwrap_or(true);
        ensure!(active, Error::NativeDisabled);
        Ok(())
    }

    /// 校验 ERC20 已注册且激活 | Ensure ERC20 token registered and active
    fn ensure_erc20_active(token: &Address) -> Result<ERC20TokenConfig, Error> {
        let config = ERC20_TOKENS.get(token).ok_or(Error::ERC20NotSupported)?;
        if !config.active {
            return Err(Error::ERC20Inactive);
        }
        Ok(config)
    }

    /// 调用 ERC20 合约 | Call ERC20 contract
    fn call_erc20(contract: &Address, calldata: &[u8]) -> Result<(), Error> {
        let api = env();
        // 用 call 而非 call_evm（与 Proxy 合约保持一致，兼容性更好）
        api.call(
            CallFlags::empty(),
            contract,
            u64::MAX,          // ref_time_limit
            u64::MAX,          // proof_size_limit
            &U256::from(0u64), // deposit
            &U256::from(0u64), // value
            calldata,
            None,
        )
        .map_err(|_| Error::ERC20TransferFailed)?;

        // 解析 bool 返回值: ERC20 transfer 返回 uint256(bool)，大端，最后一字节为 0/1
        let len = api.return_data_size() as usize;
        if len < 32 {
            return Err(Error::ERC20TransferFailed);
        }
        let mut buf = [0u8; 32];
        let mut out = &mut buf[..];
        api.return_data_copy(&mut out, 0);
        if buf[31] != 0 {
            Ok(())
        } else {
            Err(Error::ERC20TransferFailed)
        }
    }

    /// 构造 transfer(address,uint256) calldata
    fn build_transfer_calldata(to: &Address, amount: U256) -> Vec<u8> {
        let mut data = Vec::with_capacity(4 + 32 + 32);
        data.extend_from_slice(&[0xa9, 0x05, 0x9c, 0xbb]); // selector
        data.extend_from_slice(&[0u8; 12]); // left-pad address
        data.extend_from_slice(to.as_ref());
        data.extend_from_slice(&amount.to_be_bytes()); // 大端
        data
    }

    /// 构造 transferFrom(address,address,uint256) calldata
    fn build_transfer_from_calldata(from: &Address, to: &Address, amount: U256) -> Vec<u8> {
        let mut data = Vec::with_capacity(4 + 32 * 3);
        data.extend_from_slice(&[0x23, 0xb8, 0x72, 0xdd]); // selector
        data.extend_from_slice(&[0u8; 12]); // left-pad from
        data.extend_from_slice(from.as_ref());
        data.extend_from_slice(&[0u8; 12]); // left-pad to
        data.extend_from_slice(to.as_ref());
        data.extend_from_slice(&amount.to_be_bytes()); // 大端
        data
    }

    /// 构造 balanceOf(address) calldata
    fn build_balance_of_calldata(account: &Address) -> Vec<u8> {
        let mut data = Vec::with_capacity(4 + 32);
        data.extend_from_slice(&[0x70, 0xa0, 0x82, 0x31]); // selector: balanceOf(address)
        data.extend_from_slice(&[0u8; 12]); // left-pad address
        data.extend_from_slice(account.as_ref());
        data
    }

    /// 调用 ERC20 view 函数，返回 U256 结果
    fn call_erc20_view(contract: &Address, calldata: &[u8]) -> Result<U256, Error> {
        let api = env();
        api.call(
            CallFlags::empty(),
            contract,
            u64::MAX,
            u64::MAX,
            &U256::from(0u64),
            &U256::from(0u64),
            calldata,
            None,
        )
        .map_err(|_| Error::ERC20TransferFailed)?;
        let len = api.return_data_size() as usize;
        if len < 32 {
            return Err(Error::ERC20TransferFailed);
        }
        let mut buf = [0u8; 32];
        let mut out = &mut buf[..];
        api.return_data_copy(&mut out, 0);
        Ok(U256::from_be_bytes(buf))
    }
}

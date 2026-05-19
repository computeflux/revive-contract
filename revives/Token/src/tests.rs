//! Token 合约单元测试。使用 off_chain Engine (wrevive_api::with_engine)。

use super::*;
use wrevive_api::{Address, U256, with_engine};

fn alice() -> Address {
    Address::from([10u8; 20])
}

fn bob() -> Address {
    Address::from([20u8; 20])
}

fn subnet_addr() -> Address {
    Address::from([30u8; 20])
}

/// 部署 + 初始化 Token 合约
fn setup_deployed_and_inited() {
    with_engine(|e| {
        e.reset();
        e.set_caller(alice().0);
        e.set_call_data(&[]);
    });
    let _ = token::new();
    token::init(None).unwrap();
}

/// 部署 + 初始化，设小 TOKEN_UNIT 方便测试
fn setup_with_unit(unit: U256, rate: U256) {
    setup_deployed_and_inited();
    let _ = token::set_token_unit(unit);
    let _ = token::set_rate(rate);
}

// ========== TokenUnit ==========

#[test]
fn token_unit_default() {
    setup_deployed_and_inited();
    assert_eq!(
        token::get_token_unit(),
        U256::from(1_000_000_000_000_000u64)
    );
}

#[test]
fn set_token_unit_works() {
    setup_deployed_and_inited();
    let new_unit = U256::from(1_000_000u64);
    let _ = token::set_token_unit(new_unit);
    assert_eq!(token::get_token_unit(), new_unit);
}

#[test]
fn set_token_unit_by_non_owner_fails() {
    setup_deployed_and_inited();
    with_engine(|e| e.set_caller(bob().0));
    let res = token::set_token_unit(U256::from(1u64));
    assert_eq!(res, Err(Error::OnlyOwner));
}

#[test]
fn set_token_unit_zero_fails() {
    setup_deployed_and_inited();
    let res = token::set_token_unit(U256::from(0u64));
    assert_eq!(res, Err(Error::AmountMustBeGreaterThanZero));
}

// ========== 积分换算 (新公式) ==========

#[test]
fn to_points_with_unit() {
    // unit=1000, rate=2 → 1000 Planck = 2 积分
    setup_with_unit(U256::from(1000u64), U256::from(2u64));
    assert_eq!(token::to_points(U256::from(1000u64)), U256::from(2u64));
    assert_eq!(token::to_points(U256::from(3000u64)), U256::from(6u64));
}

#[test]
fn recharge_converts_planck_to_points() {
    // unit=1000, rate=2 → 5000 Planck = 10 积分
    setup_with_unit(U256::from(1000u64), U256::from(2u64));
    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(5000u64);
    });
    let points = token::recharge().unwrap();
    assert_eq!(points, U256::from(10u64));
    assert_eq!(token::get_balance(alice()), U256::from(10u64));
}

#[test]
fn recharge_accumulates_points() {
    setup_with_unit(U256::from(1000u64), U256::from(2u64));
    // 第一次: 5000 Planck → 10 积分
    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(5000u64);
    });
    let _ = token::recharge().unwrap();
    // 第二次: 2000 Planck → 4 积分
    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(2000u64);
    });
    let _ = token::recharge().unwrap();
    assert_eq!(token::get_balance(alice()), U256::from(14u64));
}

#[test]
fn recharge_small_value_gives_zero_points() {
    // unit=1000, rate=2 → 400 Planck = 0 积分 (截断)
    setup_with_unit(U256::from(1000u64), U256::from(2u64));
    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(400u64);
    });
    let res = token::recharge();
    assert_eq!(res, Err(Error::AmountMustBeGreaterThanZero));
}

#[test]
fn withdraw_converts_points_to_planck() {
    setup_with_unit(U256::from(1000u64), U256::from(2u64));
    let _ = token::set_subnet(subnet_addr());

    // Alice 充 5000 Planck → 10 积分
    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(5000u64);
    });
    let _ = token::recharge().unwrap();

    // Subnet 提 6 积分 → 3000 Planck 转给 Alice
    with_engine(|e| e.set_caller(subnet_addr().0));
    let _ = token::withdraw(alice(), U256::from(6u64));
    assert_eq!(token::get_balance(alice()), U256::from(4u64));
}

#[test]
fn fallback_uses_conversion() {
    setup_with_unit(U256::from(1000u64), U256::from(2u64));
    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(3000u64);
    });
    token::fallback();
    assert_eq!(token::get_balance(alice()), U256::from(6u64));
}

// ========== 部署与初始化 ==========

#[test]
fn deploy_and_default_values() {
    setup_deployed_and_inited();
    assert_eq!(token::owner(), alice());
    assert_eq!(token::get_rate(), U256::from(4000u64));
    assert_eq!(
        token::get_token_unit(),
        U256::from(1_000_000_000_000_000u64)
    );
    assert_eq!(token::get_subnet(), Address::zero());
    assert_eq!(token::get_balance(alice()), U256::from(0u64));
}

#[test]
fn init_is_idempotent() {
    setup_deployed_and_inited();
    token::init(None).unwrap();
    assert_eq!(token::get_rate(), U256::from(4000u64));
}

// ========== 配置 ==========

#[test]
fn set_subnet_works() {
    setup_deployed_and_inited();
    let _ = token::set_subnet(subnet_addr());
    assert_eq!(token::get_subnet(), subnet_addr());
}

#[test]
fn set_subnet_by_non_owner_fails() {
    setup_deployed_and_inited();
    with_engine(|e| e.set_caller(bob().0));
    let res = token::set_subnet(subnet_addr());
    assert_eq!(res, Err(Error::OnlyOwner));
}

#[test]
fn set_subnet_zero_address_fails() {
    setup_deployed_and_inited();
    let res = token::set_subnet(Address::zero());
    assert_eq!(res, Err(Error::ZeroAddress));
}

#[test]
fn set_rate_works() {
    setup_deployed_and_inited();
    let new_rate = U256::from(2000u64);
    let _ = token::set_rate(new_rate);
    assert_eq!(token::get_rate(), new_rate);
}

#[test]
fn set_rate_by_non_owner_fails() {
    setup_deployed_and_inited();
    with_engine(|e| e.set_caller(bob().0));
    let res = token::set_rate(U256::from(2000u64));
    assert_eq!(res, Err(Error::OnlyOwner));
}

#[test]
fn set_rate_zero_fails() {
    setup_deployed_and_inited();
    let res = token::set_rate(U256::from(0u64));
    assert_eq!(res, Err(Error::AmountMustBeGreaterThanZero));
}

// ========== 充值 ==========

#[test]
fn recharge_works() {
    setup_deployed_and_inited();
    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(5u64);
    });
    let points = token::recharge().unwrap();
    assert_eq!(points, U256::from(5000u64)); // 5 * 1000
    assert_eq!(token::get_balance(alice()), U256::from(5u64));
}

#[test]
fn recharge_zero_value_fails() {
    setup_deployed_and_inited();
    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(0u64);
    });
    let res = token::recharge();
    assert_eq!(res, Err(Error::AmountMustBeGreaterThanZero));
}

#[test]
fn recharge_accumulates_balance() {
    setup_deployed_and_inited();
    // 第一次充值
    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(3u64);
    });
    let _ = token::recharge().unwrap();
    // 第二次充值
    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(2u64);
    });
    let _ = token::recharge().unwrap();
    assert_eq!(token::get_balance(alice()), U256::from(5u64));
}

#[test]
fn recharge_different_users() {
    setup_deployed_and_inited();
    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(1u64);
    });
    let _ = token::recharge().unwrap();
    with_engine(|e| {
        e.set_caller(bob().0);
        e.value_transferred = U256::from(2u64);
    });
    let _ = token::recharge().unwrap();
    assert_eq!(token::get_balance(alice()), U256::from(1u64));
    assert_eq!(token::get_balance(bob()), U256::from(2u64));
}

// ========== 提现 ==========

#[test]
fn withdraw_by_subnet_works() {
    setup_deployed_and_inited();
    let _ = token::set_subnet(subnet_addr());

    // Alice 充值
    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(10u64);
    });
    let _ = token::recharge().unwrap();

    // Subnet 提现
    with_engine(|e| e.set_caller(subnet_addr().0));
    let _ = token::withdraw(alice(), U256::from(3u64));
    assert_eq!(token::get_balance(alice()), U256::from(7u64));
}

#[test]
fn withdraw_by_non_subnet_fails() {
    setup_deployed_and_inited();
    let _ = token::set_subnet(subnet_addr());

    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(10u64);
    });
    let _ = token::recharge().unwrap();

    // 非 Subnet 调用提现
    with_engine(|e| e.set_caller(alice().0));
    let res = token::withdraw(alice(), U256::from(1u64));
    assert_eq!(res, Err(Error::OnlySubnet));
}

#[test]
fn withdraw_insufficient_balance_fails() {
    setup_deployed_and_inited();
    let _ = token::set_subnet(subnet_addr());

    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(1u64);
    });
    let _ = token::recharge().unwrap();

    with_engine(|e| e.set_caller(subnet_addr().0));
    let res = token::withdraw(alice(), U256::from(5u64));
    assert_eq!(res, Err(Error::InsufficientBalance));
}

#[test]
fn withdraw_zero_amount_fails() {
    setup_deployed_and_inited();
    let _ = token::set_subnet(subnet_addr());

    with_engine(|e| e.set_caller(subnet_addr().0));
    let res = token::withdraw(alice(), U256::from(0u64));
    assert_eq!(res, Err(Error::AmountMustBeGreaterThanZero));
}

// ========== Fallback 充值 ==========

#[test]
fn fallback_recharge_works() {
    setup_deployed_and_inited();
    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(7u64);
    });
    // 调用 fallback（直接转账）
    token::fallback();
    assert_eq!(token::get_balance(alice()), U256::from(7u64));
}

#[test]
fn fallback_zero_value_does_nothing() {
    setup_deployed_and_inited();
    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(0u64);
    });
    token::fallback();
    assert_eq!(token::get_balance(alice()), U256::from(0u64));
}

// ========== 查询 ==========

#[test]
fn to_points_converts_correctly() {
    setup_deployed_and_inited();
    assert_eq!(
        token::to_points(U256::from(30_000_000_000u64)),
        U256::from(3u64)
    );
    let _ = token::set_rate(U256::from(500u64));
    assert_eq!(
        token::to_points(U256::from(30_000_000_000u64)),
        U256::from(1500u64)
    );
}

#[test]
fn get_balance_returns_zero_for_unknown_user() {
    setup_deployed_and_inited();
    assert_eq!(token::get_balance(bob()), U256::from(0u64));
}

// ========== 紧急提现 ==========

#[test]
fn emergency_withdraw_works() {
    setup_deployed_and_inited();
    // Alice 充值
    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(10u64);
    });
    let _ = token::recharge().unwrap();

    // Owner 紧急提现（caller = alice 即 owner）
    with_engine(|e| e.set_caller(alice().0));
    let _ = token::emergency_withdraw();
}

#[test]
fn emergency_withdraw_by_non_owner_fails() {
    setup_deployed_and_inited();
    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(10u64);
    });
    let _ = token::recharge().unwrap();

    // Bob 调用 emergency_withdraw 应失败
    with_engine(|e| e.set_caller(bob().0));
    let res = token::emergency_withdraw();
    assert_eq!(res, Err(Error::OnlyOwner));
}

// ========== Sol ABI 编码测试 ==========

#[test]
fn sol_recharge_works() {
    setup_deployed_and_inited();
    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(5u64);
    });
    let points = token::recharge_sol().unwrap();
    // 5 ETH * 1000 rate = 5000 points
    assert_eq!(points, U256::from(5000u64));
    assert_eq!(token::get_balance_sol(alice()), U256::from(5u64));
}

#[test]
fn sol_recharge_zero_value_fails() {
    setup_deployed_and_inited();
    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(0u64);
    });
    let res = token::recharge_sol();
    assert_eq!(res, Err(Error::AmountMustBeGreaterThanZero));
}

#[test]
fn sol_withdraw_by_subnet_works() {
    setup_deployed_and_inited();
    let _ = token::set_subnet(subnet_addr());

    // Alice 充值
    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(10u64);
    });
    let _ = token::recharge_sol().unwrap();

    // Subnet 提现
    with_engine(|e| e.set_caller(subnet_addr().0));
    let _ = token::withdraw_sol(alice(), U256::from(3u64));
    assert_eq!(token::get_balance_sol(alice()), U256::from(7u64));
}

#[test]
fn sol_withdraw_without_subnet_fails() {
    setup_deployed_and_inited();
    let _ = token::set_subnet(subnet_addr());

    with_engine(|e| {
        e.set_caller(alice().0);
        e.value_transferred = U256::from(10u64);
    });
    let _ = token::recharge_sol().unwrap();

    // 非 Subnet 调用提现
    with_engine(|e| e.set_caller(bob().0));
    let res = token::withdraw_sol(alice(), U256::from(1u64));
    assert_eq!(res, Err(Error::OnlySubnet));
}

#[test]
fn sol_get_rate_works() {
    setup_deployed_and_inited();
    assert_eq!(token::get_rate_sol(), U256::from(4000u64));
    let _ = token::set_rate(U256::from(500u64));
    assert_eq!(token::get_rate_sol(), U256::from(500u64));
}

#[test]
fn sol_owner_matches_openzeppelin() {
    setup_deployed_and_inited();
    // owner_sol 的选择器 0x8da5cb5b 与 OpenZeppelin Ownable.owner() 一致
    assert_eq!(token::owner_sol(), alice());
}

#[test]
fn sol_to_points_converts() {
    setup_deployed_and_inited();
    assert_eq!(token::to_points_sol(U256::from(2u64)), U256::from(2000u64));
}

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
        e.set_caller(alice().into());
        e.set_call_data(&[]);
    });
    let _ = token::new();
    token::init(None).unwrap();
}

/// 部署 + 初始化，设小 TOKEN_UNIT 方便测试
/// unit=1, rate=1000 → value * 1000 / 1 = value * 1000 (方便验算)
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
    with_engine(|e| e.set_caller(bob().into()));
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
fn recharge_returns_points() {
    // unit=1000, rate=2 → 5000 Planck = 10 积分
    setup_with_unit(U256::from(1000u64), U256::from(2u64));
    with_engine(|e| {
        e.set_caller(alice().into());
        e.value_transferred = U256::from(5000u64);
    });
    let points = token::recharge().unwrap();
    assert_eq!(points, U256::from(10u64));
    // 积分由 TEE 管理，合约不存储余额
}

#[test]
fn recharge_accumulates_points() {
    setup_with_unit(U256::from(1000u64), U256::from(2u64));
    // 第一次: 5000 Planck → 10 积分
    with_engine(|e| {
        e.set_caller(alice().into());
        e.value_transferred = U256::from(5000u64);
    });
    let p1 = token::recharge().unwrap();
    assert_eq!(p1, U256::from(10u64));
    // 第二次: 2000 Planck → 4 积分
    with_engine(|e| {
        e.set_caller(alice().into());
        e.value_transferred = U256::from(2000u64);
    });
    let p2 = token::recharge().unwrap();
    assert_eq!(p2, U256::from(4u64));
}

#[test]
fn recharge_small_value_gives_zero_points() {
    // unit=1000, rate=2 → 400 Planck = 0 积分 (截断)
    setup_with_unit(U256::from(1000u64), U256::from(2u64));
    with_engine(|e| {
        e.set_caller(alice().into());
        e.value_transferred = U256::from(400u64);
    });
    let res = token::recharge();
    assert_eq!(res, Err(Error::AmountMustBeGreaterThanZero));
}

#[test]
fn withdraw_returns_native_coin() {
    setup_with_unit(U256::from(1000u64), U256::from(2u64));
    let _ = token::set_subnet(subnet_addr());

    // Alice 充 5000 Planck
    with_engine(|e| {
        e.set_caller(alice().into());
        e.value_transferred = U256::from(5000u64);
    });
    let _ = token::recharge().unwrap();

    // Subnet 提 6 积分 → 3000 Planck 转给 Alice（积分余额由 TEE 校验）
    with_engine(|e| e.set_caller(subnet_addr().into()));
    let res = token::withdraw(alice(), U256::from(6u64), 1u64);
    assert_eq!(res, Ok(()));
}

#[test]
fn fallback_uses_conversion() {
    setup_with_unit(U256::from(1000u64), U256::from(2u64));
    with_engine(|e| {
        e.set_caller(alice().into());
        e.value_transferred = U256::from(3000u64);
    });
    // fallback 不 panic 即成功（积分由 TEE 管理）
    token::fallback();
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
    with_engine(|e| e.set_caller(bob().into()));
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
    with_engine(|e| e.set_caller(bob().into()));
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
    setup_with_unit(U256::from(1u64), U256::from(1000u64));
    with_engine(|e| {
        e.set_caller(alice().into());
        e.value_transferred = U256::from(5u64);
    });
    let points = token::recharge().unwrap();
    assert_eq!(points, U256::from(5000u64)); // 5 * 1000
}

#[test]
fn recharge_zero_value_fails() {
    setup_deployed_and_inited();
    with_engine(|e| {
        e.set_caller(alice().into());
        e.value_transferred = U256::from(0u64);
    });
    let res = token::recharge();
    assert_eq!(res, Err(Error::AmountMustBeGreaterThanZero));
}

#[test]
fn recharge_multiple_times_succeeds() {
    setup_with_unit(U256::from(1u64), U256::from(1000u64));
    with_engine(|e| {
        e.set_caller(alice().into());
        e.value_transferred = U256::from(3u64);
    });
    let _ = token::recharge().unwrap();
    with_engine(|e| {
        e.set_caller(alice().into());
        e.value_transferred = U256::from(2u64);
    });
    let _ = token::recharge().unwrap();
    // 积分由 TEE 管理，不检查合约内余额
}

#[test]
fn recharge_different_users() {
    setup_with_unit(U256::from(1u64), U256::from(1000u64));
    with_engine(|e| {
        e.set_caller(alice().into());
        e.value_transferred = U256::from(1u64);
    });
    let p1 = token::recharge().unwrap();
    assert_eq!(p1, U256::from(1000u64));
    with_engine(|e| {
        e.set_caller(bob().into());
        e.value_transferred = U256::from(2u64);
    });
    let p2 = token::recharge().unwrap();
    assert_eq!(p2, U256::from(2000u64));
}

// ========== 提现 ==========

#[test]
fn withdraw_by_subnet_works() {
    setup_with_unit(U256::from(1u64), U256::from(1000u64));
    let _ = token::set_subnet(subnet_addr());

    // Alice 充值
    with_engine(|e| {
        e.set_caller(alice().into());
        e.value_transferred = U256::from(10u64);
    });
    let _ = token::recharge().unwrap();

    // Subnet 提现（积分余额由 TEE 校验）
    with_engine(|e| e.set_caller(subnet_addr().into()));
    let res = token::withdraw(alice(), U256::from(3000u64), 2u64);
    assert_eq!(res, Ok(()));
}

#[test]
fn withdraw_by_non_subnet_fails() {
    setup_with_unit(U256::from(1u64), U256::from(1000u64));
    let _ = token::set_subnet(subnet_addr());

    with_engine(|e| {
        e.set_caller(alice().into());
        e.value_transferred = U256::from(10u64);
    });
    let _ = token::recharge().unwrap();

    // 非 Subnet 调用提现
    with_engine(|e| e.set_caller(alice().into()));
    let res = token::withdraw(alice(), U256::from(1000u64), 3u64);
    assert_eq!(res, Err(Error::OnlySubnet));
}

#[test]
fn withdraw_zero_amount_fails() {
    setup_deployed_and_inited();
    let _ = token::set_subnet(subnet_addr());

    with_engine(|e| e.set_caller(subnet_addr().into()));
    let res = token::withdraw(alice(), U256::from(0u64), 4u64);
    assert_eq!(res, Err(Error::AmountMustBeGreaterThanZero));
}

// ========== Fallback 充值 ==========

#[test]
fn fallback_recharge_works() {
    setup_with_unit(U256::from(1u64), U256::from(1000u64));
    with_engine(|e| {
        e.set_caller(alice().into());
        e.value_transferred = U256::from(7u64);
    });
    token::fallback();
    // 不 panic 即成功
}

#[test]
fn fallback_zero_value_does_nothing() {
    setup_deployed_and_inited();
    with_engine(|e| {
        e.set_caller(alice().into());
        e.value_transferred = U256::from(0u64);
    });
    token::fallback();
    // 不 panic 即成功
}

// ========== 查询 ==========

#[test]
fn to_points_converts_correctly() {
    setup_deployed_and_inited();
    assert_eq!(
        token::to_points(U256::from(1_000_000_000_000_000u64)),
        U256::from(4000u64)
    );
    let _ = token::set_rate(U256::from(500u64));
    assert_eq!(
        token::to_points(U256::from(1_000_000_000_000_000u64)),
        U256::from(500u64)
    );
}

// ========== Sol ABI 编码测试 ==========

#[test]
fn sol_recharge_works() {
    setup_with_unit(U256::from(1u64), U256::from(1000u64));
    with_engine(|e| {
        e.set_caller(alice().into());
        e.value_transferred = U256::from(5u64);
    });
    let points = token::recharge_sol().unwrap();
    assert_eq!(points, U256::from(5000u64));
}

#[test]
fn sol_recharge_zero_value_fails() {
    setup_deployed_and_inited();
    with_engine(|e| {
        e.set_caller(alice().into());
        e.value_transferred = U256::from(0u64);
    });
    let res = token::recharge_sol();
    assert_eq!(res, Err(Error::AmountMustBeGreaterThanZero));
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
    assert_eq!(token::owner_sol(), alice());
}

#[test]
fn sol_to_points_converts() {
    setup_deployed_and_inited();
    assert_eq!(
        token::to_points_sol(U256::from(1_000_000_000_000_000u64)),
        U256::from(4000u64)
    );
}

// ========== ERC20 测试 ==========

fn erc20_addr() -> Address {
    Address::from([40u8; 20])
}

fn setup_erc20(rate: U256, unit: U256) {
    setup_deployed_and_inited();
    let _ = token::set_erc20_token(erc20_addr(), true, rate, unit);
}

#[test]
fn set_erc20_token_works() {
    setup_deployed_and_inited();
    let res = token::set_erc20_token(
        erc20_addr(),
        true,
        U256::from(1000u64),
        U256::from(1_000_000u64),
    );
    assert_eq!(res, Ok(()));
    let (active, rate, unit) = token::get_erc20_config(erc20_addr());
    assert_eq!(active, true);
    assert_eq!(rate, U256::from(1000u64));
    assert_eq!(unit, U256::from(1_000_000u64));
}

#[test]
fn set_erc20_token_by_non_owner_fails() {
    setup_deployed_and_inited();
    with_engine(|e| e.set_caller(bob().into()));
    let res = token::set_erc20_token(
        erc20_addr(),
        true,
        U256::from(1000u64),
        U256::from(1_000_000u64),
    );
    assert_eq!(res, Err(Error::OnlyOwner));
}

#[test]
fn set_erc20_token_zero_address_fails() {
    setup_deployed_and_inited();
    let res = token::set_erc20_token(
        Address::zero(),
        true,
        U256::from(1000u64),
        U256::from(1_000_000u64),
    );
    assert_eq!(res, Err(Error::ZeroAddress));
}

#[test]
fn set_erc20_token_zero_unit_fails() {
    setup_deployed_and_inited();
    let res = token::set_erc20_token(erc20_addr(), true, U256::from(1000u64), U256::from(0u64));
    assert_eq!(res, Err(Error::AmountMustBeGreaterThanZero));
}

#[test]
fn deactivate_erc20_token_works() {
    setup_erc20(U256::from(1000u64), U256::from(1_000_000u64));
    let _ = token::set_erc20_token(
        erc20_addr(),
        false,
        U256::from(1000u64),
        U256::from(1_000_000u64),
    );
    let (active, _, _) = token::get_erc20_config(erc20_addr());
    assert_eq!(active, false);
}

#[test]
fn get_erc20_config_unknown_returns_default() {
    setup_deployed_and_inited();
    let (active, rate, dec) = token::get_erc20_config(erc20_addr());
    assert_eq!(active, false);
    assert_eq!(rate, U256::from(0u64));
    assert_eq!(dec, U256::from(0u64));
}

#[test]
fn get_erc20_config_works() {
    setup_erc20(U256::from(500u64), U256::from(1_000_000_000_000_000_000u64));
    let (active, rate, unit) = token::get_erc20_config(erc20_addr());
    assert_eq!(active, true);
    assert_eq!(rate, U256::from(500u64));
    assert_eq!(unit, U256::from(1_000_000_000_000_000_000u64));
}

#[test]
fn get_erc20_list_returns_registered_tokens() {
    setup_deployed_and_inited();
    assert_eq!(token::get_erc20_count(), 0u64);
    // 注册两个代币
    let _ = token::set_erc20_token(
        erc20_addr(),
        true,
        U256::from(1000u64),
        U256::from(1_000_000u64),
    );
    let addr2 = Address::from([50u8; 20]);
    let _ = token::set_erc20_token(addr2, true, U256::from(500u64), U256::from(10_000u64));
    assert_eq!(token::get_erc20_count(), 2u64);
    // get_erc20_list 返回 Vec
    let list = token::get_erc20_list();
    assert_eq!(list.len(), 2);
    assert_eq!(list[0].0, erc20_addr());
    assert!(list[0].1);
    assert_eq!(list[1].0, addr2);
    assert_eq!(list[1].1, true);
    assert_eq!(list[1].2, U256::from(500u64));
    assert_eq!(list[1].3, U256::from(10_000u64));
    // 更新已有代币不增加数量
    let _ = token::set_erc20_token(
        erc20_addr(),
        false,
        U256::from(2000u64),
        U256::from(2_000_000u64),
    );
    assert_eq!(token::get_erc20_count(), 2u64);
}

// ========== ERC20 充值校验 ==========

#[test]
fn recharge_erc20_unregistered_fails() {
    setup_deployed_and_inited();
    with_engine(|e| e.set_caller(alice().into()));
    let res = token::recharge_erc20(erc20_addr(), U256::from(100u64));
    assert_eq!(res, Err(Error::ERC20NotSupported));
}

#[test]
fn recharge_erc20_inactive_fails() {
    setup_erc20(U256::from(1000u64), U256::from(1_000_000u64));
    let _ = token::set_erc20_token(
        erc20_addr(),
        false,
        U256::from(1000u64),
        U256::from(1_000_000u64),
    );
    with_engine(|e| e.set_caller(alice().into()));
    let res = token::recharge_erc20(erc20_addr(), U256::from(100u64));
    assert_eq!(res, Err(Error::ERC20Inactive));
}

#[test]
fn recharge_erc20_zero_amount_fails() {
    setup_erc20(U256::from(1000u64), U256::from(1_000_000u64));
    with_engine(|e| e.set_caller(alice().into()));
    let res = token::recharge_erc20(erc20_addr(), U256::from(0u64));
    assert_eq!(res, Err(Error::AmountMustBeGreaterThanZero));
}

// ========== ERC20 提现校验 ==========

#[test]
fn withdraw_erc20_by_non_subnet_fails() {
    setup_erc20(U256::from(1000u64), U256::from(1_000_000u64));
    let _ = token::set_subnet(subnet_addr());
    with_engine(|e| e.set_caller(alice().into()));
    let res = token::withdraw_erc20(erc20_addr(), alice(), U256::from(1000u64), 5u64);
    assert_eq!(res, Err(Error::OnlySubnet));
}

#[test]
fn withdraw_erc20_unregistered_fails() {
    setup_deployed_and_inited();
    let _ = token::set_subnet(subnet_addr());
    with_engine(|e| e.set_caller(subnet_addr().into()));
    let res = token::withdraw_erc20(erc20_addr(), alice(), U256::from(1000u64), 6u64);
    assert_eq!(res, Err(Error::ERC20NotSupported));
}

#[test]
fn withdraw_erc20_inactive_fails() {
    setup_erc20(U256::from(1000u64), U256::from(1_000_000u64));
    let _ = token::set_subnet(subnet_addr());
    let _ = token::set_erc20_token(
        erc20_addr(),
        false,
        U256::from(1000u64),
        U256::from(1_000_000u64),
    );
    with_engine(|e| e.set_caller(subnet_addr().into()));
    let res = token::withdraw_erc20(erc20_addr(), alice(), U256::from(1000u64), 7u64);
    assert_eq!(res, Err(Error::ERC20Inactive));
}

#[test]
fn withdraw_erc20_zero_points_fails() {
    setup_erc20(U256::from(1000u64), U256::from(1_000_000u64));
    let _ = token::set_subnet(subnet_addr());
    with_engine(|e| e.set_caller(subnet_addr().into()));
    let res = token::withdraw_erc20(erc20_addr(), alice(), U256::from(0u64), 8u64);
    assert_eq!(res, Err(Error::AmountMustBeGreaterThanZero));
}

#[test]
fn withdraw_duplicate_nonce_returns_ok() {
    setup_with_unit(U256::from(1u64), U256::from(1000u64));
    let _ = token::set_subnet(subnet_addr());

    with_engine(|e| e.set_caller(subnet_addr().into()));
    let res = token::withdraw(alice(), U256::from(1000u64), 9u64);
    assert_eq!(res, Ok(()));

    // 重复 nonce 直接返回 Ok，不报错
    with_engine(|e| e.set_caller(subnet_addr().into()));
    let res = token::withdraw(alice(), U256::from(1000u64), 9u64);
    assert_eq!(res, Ok(()));
}

#[test]
fn withdraw_erc20_duplicate_nonce_returns_ok() {
    setup_erc20(U256::from(1000u64), U256::from(1_000_000u64));
    let _ = token::set_subnet(subnet_addr());

    with_engine(|e| e.set_caller(subnet_addr().into()));
    let res = token::withdraw_erc20(erc20_addr(), alice(), U256::from(1000u64), 10u64);
    assert_eq!(res, Ok(()));

    // 重复 nonce 直接返回 Ok，不报错
    with_engine(|e| e.set_caller(subnet_addr().into()));
    let res = token::withdraw_erc20(erc20_addr(), alice(), U256::from(1000u64), 10u64);
    assert_eq!(res, Ok(()));
}

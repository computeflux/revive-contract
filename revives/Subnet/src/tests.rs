//! Unit tests for Subnet contract. Uses off_chain Engine (wrevive_api::with_engine).

use super::*;
use wrevive_api::{AccountId, Address, U256, with_engine};

fn gov_caller() -> [u8; 20] {
    [1u8; 20]
}

/// Deploy (new) + init with gov as caller so that subsequent gov-only calls succeed.
fn setup_deployed_and_inited() {
    with_engine(|e| {
        e.reset();
        e.set_caller(gov_caller());
        e.set_call_data(&[]);
    });
    let _ = subnet::new();
    subnet::init().unwrap();
}

#[test]
fn deploy_and_epoch_info() {
    with_engine(|e| {
        e.reset();
        e.set_caller(gov_caller());
        e.set_call_data(&[]);
    });
    let _ = subnet::new();
    let info = subnet::epoch_info();
    assert_eq!(info.epoch, 0);
    assert_eq!(info.epoch_slot, 72000);
    assert_eq!(subnet::side_chain_key(), Address::zero());
}

#[test]
fn init_sets_gov_and_is_idempotent() {
    with_engine(|e| {
        e.reset();
        e.set_caller(gov_caller());
        e.set_call_data(&[]);
    });
    let _ = subnet::new();
    subnet::init().unwrap();
    assert_eq!(subnet::side_chain_key(), Address::zero());
    // second init is no-op
    subnet::init().unwrap();
}

#[test]
fn set_epoch_slot_only_by_gov() {
    setup_deployed_and_inited();
    let _ = subnet::set_epoch_slot(36000);
    let info = subnet::epoch_info();
    assert_eq!(info.epoch_slot, 36000);

    with_engine(|e| e.set_caller([99u8; 20]));
    let res = subnet::set_epoch_slot(1);
    assert_eq!(res, Err(Error::MustCallByMainContract));
}

#[test]
fn set_region_and_region() {
    setup_deployed_and_inited();
    let name = b"eu-west".to_vec();
    let _ = subnet::set_region(name.clone());
    assert_eq!(subnet::region(0), Some(name));
    assert_eq!(subnet::region(1), None);
}

#[test]
fn regions_list_after_set_region() {
    setup_deployed_and_inited();
    let a = b"region-a".to_vec();
    let b = b"region-b".to_vec();
    let _ = subnet::set_region(a.clone());
    let _ = subnet::set_region(b.clone());
    let list = subnet::regions();
    assert_eq!(list.len(), 2);
    assert_eq!(list[0].0, 1);
    assert_eq!(list[0].1, b);
    assert_eq!(list[1].0, 0);
    assert_eq!(list[1].1, a);
}

#[test]
fn set_level_price_and_level_price() {
    setup_deployed_and_inited();
    let price = RunPrice {
        cpu_per: 1,
        cvm_cpu_per: 2,
        memory_per: 3,
        cvm_memory_per: 4,
        disk_per: 5,
        gpu_per: 6,
    };
    let _ = subnet::set_level_price(0, price.clone());
    assert_eq!(subnet::level_price(0), Some(price));
}

#[test]
fn set_asset_and_asset() {
    setup_deployed_and_inited();
    let info = AssetInfo::Native(alloc::vec::Vec::new());
    let price = U256::from(1000u64);
    let _ = subnet::set_asset(info.clone(), price);
    let got = subnet::asset(0).expect("asset");
    assert_eq!(got.0, info);
    assert_eq!(got.1, price);
}

#[test]
fn set_region_non_gov_reverts() {
    setup_deployed_and_inited();
    with_engine(|e| e.set_caller([99u8; 20]));
    let res = subnet::set_region(b"x".to_vec());
    assert_eq!(res, Err(Error::MustCallByMainContract));
}

#[test]
fn get_pending_secrets_empty() {
    setup_deployed_and_inited();
    let list = subnet::get_pending_secrets();
    assert!(list.is_empty());
}

// ---------- worker ----------

fn default_ip() -> Ip {
    Ip {
        ipv4: Some(3232263885),
        ipv6: None,
        domain: None,
    }
}

fn account_id_from_u8(v: u8) -> AccountId {
    AccountId::from([v; 32])
}

#[test]
fn worker_register_and_worker_and_workers() {
    setup_deployed_and_inited();
    let _ = subnet::set_region(b"eu".to_vec());
    with_engine(|e| e.set_caller([10u8; 20]));
    let worker_id = subnet::worker_register(
        b"worker-0".to_vec(),
        account_id_from_u8(1),
        default_ip(),
        30333,
        1,
        0,
    )
    .unwrap();
    assert_eq!(worker_id, 0);

    let w = subnet::worker(worker_id).expect("worker");
    assert_eq!(w.name, b"worker-0".to_vec());
    assert_eq!(w.owner, Address::from([10u8; 20]));
    assert_eq!(w.level, 1);
    assert_eq!(w.region_id, 0);
    assert_eq!(w.port, 30333);
    assert_eq!(w.status, 0);

    let list = subnet::workers(None, 10);
    assert_eq!(list.len(), 1);
    assert_eq!(list[0].0, 0);

    let user_w = subnet::user_worker(Address::from([10u8; 20])).expect("user_worker");
    assert_eq!(user_w.0, 0);
}

#[test]
fn worker_register_region_not_exist_fails() {
    setup_deployed_and_inited();
    with_engine(|e| e.set_caller([10u8; 20]));
    let res = subnet::worker_register(
        b"w".to_vec(),
        account_id_from_u8(1),
        default_ip(),
        30333,
        1,
        99, // region 99 not set
    );
    assert_eq!(res, Err(Error::RegionNotExist));
}

#[test]
fn worker_update_by_owner_ok() {
    setup_deployed_and_inited();
    let _ = subnet::set_region(b"eu".to_vec());
    with_engine(|e| e.set_caller([10u8; 20]));
    let wid = subnet::worker_register(
        b"old".to_vec(),
        account_id_from_u8(1),
        default_ip(),
        30333,
        1,
        0,
    )
    .unwrap();
    let _ = subnet::worker_update(wid, b"new".to_vec(), default_ip(), 30334);
    let w = subnet::worker(wid).unwrap();
    assert_eq!(w.name, b"new".to_vec());
    assert_eq!(w.port, 30334);
}

#[test]
fn worker_update_by_non_owner_fails() {
    setup_deployed_and_inited();
    let _ = subnet::set_region(b"eu".to_vec());
    with_engine(|e| e.set_caller([10u8; 20]));
    let wid = subnet::worker_register(
        b"w".to_vec(),
        account_id_from_u8(1),
        default_ip(),
        30333,
        1,
        0,
    )
    .unwrap();
    with_engine(|e| e.set_caller([99u8; 20]));
    let res = subnet::worker_update(wid, b"x".to_vec(), default_ip(), 1);
    assert_eq!(res, Err(Error::WorkerNotOwnedByCaller));
}

#[test]
fn worker_mortgage_by_owner() {
    setup_deployed_and_inited();
    let _ = subnet::set_region(b"eu".to_vec());
    with_engine(|e| {
        e.set_caller([10u8; 20]);
        e.value_transferred = U256::from(1000u64);
    });
    let wid = subnet::worker_register(
        b"w".to_vec(),
        account_id_from_u8(1),
        default_ip(),
        30333,
        1,
        0,
    )
    .unwrap();
    with_engine(|e| e.value_transferred = U256::from(1000u64));
    let mid = subnet::worker_mortgage(wid, 2, 4, 0, 0, 10, 0, U256::from(1000u64)).unwrap();
    assert_eq!(mid, 0);
}

#[test]
fn worker_stop_by_owner_when_running() {
    setup_deployed_and_inited();
    let _ = subnet::set_region(b"eu".to_vec());

    // 注册 Worker
    with_engine(|e| e.set_caller([10u8; 20]));
    let wid = subnet::worker_register(
        b"w".to_vec(),
        account_id_from_u8(1),
        default_ip(),
        30333,
        1,
        0,
    )
    .unwrap();

    // 抵押资源（status 必须为 0 才能抵押）
    with_engine(|e| {
        e.set_caller([10u8; 20]);
        e.value_transferred = U256::from(1000u64);
    });
    let _ = subnet::worker_mortgage(wid, 2, 4, 0, 0, 10, 0, U256::from(1000u64)).unwrap();

    // 以侧链地址（Address::zero() = [0u8;20]）启动 Worker
    with_engine(|e| e.set_caller([0u8; 20]));
    subnet::worker_start(wid).expect("worker_start should succeed");
    assert_eq!(subnet::worker(wid).unwrap().status, 1);

    // Owner 停止 Worker
    with_engine(|e| e.set_caller([10u8; 20]));
    let got = subnet::worker_stop(wid).unwrap();
    assert_eq!(got, wid);
    assert_eq!(subnet::worker(wid).unwrap().status, 2);
}

#[test]
fn worker_stop_by_non_owner_fails() {
    setup_deployed_and_inited();
    let _ = subnet::set_region(b"eu".to_vec());
    with_engine(|e| e.set_caller([10u8; 20]));
    let wid = subnet::worker_register(
        b"w".to_vec(),
        account_id_from_u8(1),
        default_ip(),
        30333,
        1,
        0,
    )
    .unwrap();
    with_engine(|e| e.set_caller([99u8; 20]));
    let res = subnet::worker_stop(wid);
    assert_eq!(res, Err(Error::WorkerNotOwnedByCaller));
}

// ---------- boot_nodes ----------

#[test]
fn set_boot_nodes_and_boot_nodes() {
    setup_deployed_and_inited();
    let _ = subnet::set_boot_nodes(alloc::vec![2, 0, 1]);
    let nodes = subnet::boot_nodes().unwrap();
    assert!(nodes.is_empty()); // no SECRETS registered yet, so boot_nodes returns empty list
}

// ---------- secret ----------

#[test]
fn secret_register_and_secrets() {
    setup_deployed_and_inited();
    with_engine(|e| e.set_caller([20u8; 20]));
    let id = subnet::secret_register(
        b"node0".to_vec(),
        account_id_from_u8(10),
        account_id_from_u8(11),
        default_ip(),
        30110,
    )
    .unwrap();
    assert_eq!(id, 0);

    let list = subnet::secrets();
    assert_eq!(list.len(), 1);
    assert_eq!(list[0].0, 0);
    assert_eq!(list[0].1.name, b"node0".to_vec());
}

#[test]
fn secret_register_two_then_secrets() {
    setup_deployed_and_inited();
    with_engine(|e| e.set_caller([20u8; 20]));
    let _ = subnet::secret_register(
        b"n0".to_vec(),
        account_id_from_u8(10),
        account_id_from_u8(11),
        default_ip(),
        30110,
    )
    .unwrap();
    with_engine(|e| e.set_caller([21u8; 20]));
    let _ = subnet::secret_register(
        b"n1".to_vec(),
        account_id_from_u8(12),
        account_id_from_u8(13),
        default_ip(),
        30120,
    )
    .unwrap();
    let list = subnet::secrets();
    assert_eq!(list.len(), 2);
}

#[test]
fn secret_update_by_owner() {
    setup_deployed_and_inited();
    with_engine(|e| e.set_caller([20u8; 20]));
    let id = subnet::secret_register(
        b"old".to_vec(),
        account_id_from_u8(10),
        account_id_from_u8(11),
        default_ip(),
        30110,
    )
    .unwrap();
    let _ = subnet::secret_update(id, b"new".to_vec(), default_ip(), 30111);
    let list = subnet::secrets();
    assert_eq!(list[0].1.name, b"new".to_vec());
    assert_eq!(list[0].1.port, 30111);
}

#[test]
fn secret_deposit_by_owner() {
    setup_deployed_and_inited();
    with_engine(|e| e.set_caller([20u8; 20]));
    let id = subnet::secret_register(
        b"n".to_vec(),
        account_id_from_u8(10),
        account_id_from_u8(11),
        default_ip(),
        30110,
    )
    .unwrap();
    let _ = subnet::secret_deposit(id, U256::from(500u64));
    // no direct reader for SECRET_MORTGAGES; secret_delete would fail if deposit != 0
}

// ---------- validator_join / validator_delete ----------

#[test]
fn validator_join_only_by_gov() {
    setup_deployed_and_inited();
    with_engine(|e| e.set_caller([20u8; 20]));
    let id = subnet::secret_register(
        b"n".to_vec(),
        account_id_from_u8(10),
        account_id_from_u8(11),
        default_ip(),
        30110,
    )
    .unwrap();
    with_engine(|e| e.set_caller(gov_caller()));
    let _ = subnet::validator_join(id);
    let pending = subnet::get_pending_secrets();
    assert_eq!(pending.len(), 1);
    assert_eq!(pending[0].0, id);
    assert_eq!(pending[0].1, 1);
}

#[test]
fn validator_join_non_gov_fails() {
    setup_deployed_and_inited();
    with_engine(|e| e.set_caller([20u8; 20]));
    let id = subnet::secret_register(
        b"n".to_vec(),
        account_id_from_u8(10),
        account_id_from_u8(11),
        default_ip(),
        30110,
    )
    .unwrap();
    with_engine(|e| e.set_caller([99u8; 20]));
    let res = subnet::validator_join(id);
    assert_eq!(res, Err(Error::MustCallByMainContract));
}

#[test]
fn validator_join_nonexistent_node_fails() {
    setup_deployed_and_inited();
    with_engine(|e| e.set_caller(gov_caller()));
    let res = subnet::validator_join(999);
    assert_eq!(res, Err(Error::NodeNotExist));
}

#[test]
fn validator_delete_only_by_gov() {
    setup_deployed_and_inited();
    with_engine(|e| e.set_caller([20u8; 20]));
    let id = subnet::secret_register(
        b"n".to_vec(),
        account_id_from_u8(10),
        account_id_from_u8(11),
        default_ip(),
        30110,
    )
    .unwrap();
    with_engine(|e| e.set_caller(gov_caller()));
    let _ = subnet::validator_join(id);
    let _ = subnet::validator_delete(id);
    let pending = subnet::get_pending_secrets();
    assert_eq!(pending.len(), 1);
    assert_eq!(pending[0].1, 0);
}

#[test]
fn validators_after_first_secret_register() {
    setup_deployed_and_inited();
    with_engine(|e| e.set_caller([20u8; 20]));
    let _ = subnet::secret_register(
        b"n0".to_vec(),
        account_id_from_u8(10),
        account_id_from_u8(11),
        default_ip(),
        30110,
    )
    .unwrap();
    let v = subnet::validators();
    assert_eq!(v.len(), 1);
    assert_eq!(v[0].0, 0);
    assert_eq!(v[0].2, 1);
}

#[test]
fn worker_start_only_by_side_chain_fails() {
    setup_deployed_and_inited();
    let _ = subnet::set_region(b"eu".to_vec());
    with_engine(|e| e.set_caller([10u8; 20]));
    let wid = subnet::worker_register(
        b"w".to_vec(),
        account_id_from_u8(1),
        default_ip(),
        30333,
        1,
        0,
    )
    .unwrap();
    with_engine(|e| e.set_caller([99u8; 20]));
    let res = subnet::worker_start(wid);
    assert_eq!(res, Err(Error::InvalidSideChainCaller));
}

#[test]
fn set_next_epoch_before_epoch_slot_returns_epoch_not_expired() {
    setup_deployed_and_inited();
    with_engine(|e| e.set_caller(gov_caller()));
    // off_chain block_number() returns 0, last_epoch=0, epoch_slot=72000 => (0-0) < 72000
    let res = subnet::set_next_epoch(0);
    assert_eq!(res, Err(Error::EpochNotExpired));
}

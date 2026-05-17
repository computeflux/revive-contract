#![cfg_attr(not(test), no_std)]
#![cfg_attr(not(test), no_main)]

extern crate alloc;

#[cfg(all(not(test), not(feature = "api")))]
#[global_allocator]
static ALLOC: pvm_bump_allocator::BumpAllocator<65536> = pvm_bump_allocator::BumpAllocator::new();

mod errors;

use pallet_revive_uapi::CallFlags;
use wrevive_api::{Address, Encode, Env, ReturnFlags, Storage, U256, env};
use wrevive_macro::{revive_contract, storage};

pub use errors::Error;
pub use primitives::ensure;

#[revive_contract]
pub mod proxy {
    use super::*;

    const PROXY_IMPLEMENTATION: Storage<Address> = storage!(b"_proxy_implementation");
    const PROXY_ADMIN: Storage<Address> = storage!(b"_proxy_admin");

    /// 部署代理：设置实现合约地址与管理员。若不传 admin，则使用 caller 为管理员。
    #[revive(constructor)]
    pub fn new(implementation: Address, admin: Option<Address>) -> Result<(), Error> {
        let caller = env().caller();
        let admin_addr = admin.unwrap_or(caller);
        PROXY_IMPLEMENTATION.set(&implementation);
        PROXY_ADMIN.set(&admin_addr);
        Ok(())
    }

    /// 当前实现合约地址
    #[revive(message)]
    pub fn get_implementation() -> Address {
        PROXY_IMPLEMENTATION.get().unwrap_or(Address::zero())
    }

    /// 管理员地址（有权调用 upgrade）
    #[revive(message)]
    pub fn get_admin() -> Address {
        PROXY_ADMIN.get().unwrap_or(Address::zero())
    }

    /// 升级实现合约（仅管理员可调）
    #[revive(message, write)]
    pub fn upgrade(implementation: Address) -> Result<(), Error> {
        ensure_admin()?;
        PROXY_IMPLEMENTATION.set(&implementation);
        Ok(())
    }

    /// 将管理员转移给新地址（仅当前管理员可调）
    #[revive(message, write)]
    pub fn transfer_admin(new_admin: Address) -> Result<(), Error> {
        ensure_admin()?;
        PROXY_ADMIN.set(&new_admin);
        Ok(())
    }

    /// 未匹配到本合约 message 时，将调用数据 delegate_call 到实现合约，并通过 delegate_call 的返回数据原样返回。
    #[revive(fallback)]
    pub fn fallback() {
        let api = env();
        let callee = PROXY_IMPLEMENTATION.get().unwrap_or(Address::zero());
        if callee == Address::zero() {
            let error = Error::AddressNotFound;
            api.return_value(ReturnFlags::REVERT, &Encode::encode(&error));
        }
        let call_data_len = api.call_data_size() as usize;
        let call_data = api.call_data_copy(0, call_data_len);

        let result = api.delegate_call(
            CallFlags::empty(),
            &callee,
            u64::MAX,
            u64::MAX,
            &U256::MAX,
            &call_data,
            None,
        );

        let len = api.return_data_size() as usize;
        let mut full = alloc::vec![0u8; len];
        let mut slice = full.as_mut_slice();
        api.return_data_copy(&mut slice, 0);

        let flags = match result {
            Ok(()) => ReturnFlags::empty(),
            Err(_) => ReturnFlags::REVERT,
        };

        api.return_value(flags, &full);
    }

    fn ensure_admin() -> Result<(), Error> {
        let caller = env().caller();
        let admin = PROXY_ADMIN.get().unwrap_or(Address::zero());
        ensure!(caller == admin, Error::Unauthorized);
        Ok(())
    }
}

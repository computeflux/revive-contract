# Dtoken Contracts (Hardhat)

<p align="right">
  <strong>English</strong> | <a href="./README.zh-CN.md">简体中文</a>
</p>

Smart contracts for Dtoken marketplace, built with Hardhat. This folder includes compile, test, deploy and (optional) verify scripts for Polkadot Hub testnet and EVM networks.

## Prerequisites

- Node.js + npm
- A private key for deployment (never commit it)

## Setup

```bash
npm install
cp .env.example .env
```

## Common commands

```bash
npm run compile
npm run test
```

### Deploy

```bash
npm run deploy:polkadot
```

### Verify (Polkadot)

```bash
npm run verify:polkadot
```

## Generate Go bindings (for `token` backend)

This repository provides a helper script:

```bash
./generate-go-binding.sh
```

It requires `abigen` and writes the binding to `../service/ModelMarketplaceV3.go` (in the `token` workspace).

## Related projects

- Token service (DTOKEN): `../token`
- DApp: `../dapp`
- Admin dashboard: `../dashboard`
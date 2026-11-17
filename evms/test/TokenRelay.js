// Relay 查询接口测试 | Relay query interface tests
//
// 验证 Token.sol（EVM 版）与 revives/Token（PolkaVM 版）的 relay 事件语义一致：
//   - getLatestNonce() / getEvent(nonce) / getEvents(from, to)
//   - EventRecord{targetContract="gateway", eventType="Recharge"|"WithdrawalDone"|"WithdrawalCancelled"}
//   - eventData 为 SCALE 编码，字节与 PolkaVM 版 emit_event 一致
//   - 写入点与链上日志一一对应：pending 提现不产生 relay 事件

const { expect } = require("chai");
const { ethers } = require("hardhat");

const RATE = 4000n;
const UNIT = 10n ** 18n;

/// SCALE u256：32 字节小端
function scaleU256(v) {
  const hex = BigInt(v).toString(16).padStart(64, "0");
  const bytes = [];
  for (let i = 0; i < 32; i++) bytes.push(hex.slice(i * 2, i * 2 + 2));
  return "0x" + bytes.reverse().join("");
}

/// SCALE u64：8 字节小端
function scaleU64(v) {
  const hex = BigInt(v).toString(16).padStart(16, "0");
  const bytes = [];
  for (let i = 0; i < 8; i++) bytes.push(hex.slice(i * 2, i * 2 + 2));
  return "0x" + bytes.reverse().join("");
}

/// SCALE UniAddr{t=2, v=20 字节地址}：u32 小端 + compact(20)=0x50 + 地址
function scaleUniAddr(addr) {
  return "0x02000000" + "50" + addr.slice(2).toLowerCase();
}

/// 字符串 → bytes（用于比较 bytes 字段）
function toBytes(s) {
  return ethers.hexlify(ethers.toUtf8Bytes(s));
}

/// getEvent 与 ethers BaseContract.getEvent 同名，需显式取函数
async function getEvent(token, nonce) {
  return token.getFunction("getEvent")(nonce);
}

async function deployToken(owner) {
  const Token = await ethers.getContractFactory("Token");
  const impl = await Token.deploy();
  await impl.waitForDeployment();

  const initData = Token.interface.encodeFunctionData("initialize", [owner.address]);
  // ERC1967Proxy 未被 contracts 引用，直接使用 OZ 自带编译产物
  const ozProxy = require("@openzeppelin/contracts/build/contracts/ERC1967Proxy.json");
  const Proxy = new ethers.ContractFactory(ozProxy.abi, ozProxy.bytecode, owner);
  const proxy = await Proxy.deploy(await impl.getAddress(), initData);
  await proxy.waitForDeployment();

  return Token.attach(await proxy.getAddress());
}

describe("Token relay queries", function () {
  let owner, subnet, user, other;
  let token;

  beforeEach(async function () {
    [owner, subnet, user, other] = await ethers.getSigners();
    token = await deployToken(owner);
    await (await token.connect(owner).setSubnet(subnet.address)).wait();

    // 代理存储不会继承实现合约构造函数中的状态变量初始值，按文档默认值显式设置
    await (await token.connect(owner).setNativeActive(true)).wait();
    await (await token.connect(subnet).setRechargeFeeRate(500n)).wait();
    await (await token.connect(subnet).setWithdrawFeeRate(500n)).wait();
  });

  it("starts empty", async function () {
    expect(await token.getLatestNonce()).to.equal(0n);
    expect((await token.getEvents(0, 0)).length).to.equal(0);

    const rec = await getEvent(token, 1);
    expect(rec.targetContract).to.equal("0x");
  });

  it("records Recharge with SCALE-encoded UniAddr and points", async function () {
    const value = ethers.parseEther("1");
    const fee = (value * 500n) / 10000n;
    const points = ((value - fee) * RATE) / UNIT;

    await (await token.connect(user).recharge({ value })).wait();

    expect(await token.getLatestNonce()).to.equal(1n);

    const rec = await getEvent(token, 1);
    expect(rec.targetContract).to.equal(toBytes("gateway"));
    expect(rec.eventType).to.equal(toBytes("Recharge"));
    expect(rec.eventData.length).to.equal(2);
    expect(rec.eventData[0]).to.equal(scaleUniAddr(user.address));
    expect(rec.eventData[0].length).to.equal(2 + 25 * 2); // 25 字节
    expect(rec.eventData[1]).to.equal(scaleU256(points));

    // 非 2 的幂幂等点：0xED8 = 3800，验证小端
    expect(rec.eventData[1]).to.equal(
      "0x" + "d80e" + "00".repeat(30)
    );
  });

  it("records Recharge on plain value transfer (receive)", async function () {
    const value = ethers.parseEther("2");
    await (await user.sendTransaction({ to: await token.getAddress(), value })).wait();

    expect(await token.getLatestNonce()).to.equal(1n);
    const rec = await getEvent(token, 1);
    expect(rec.eventType).to.equal(toBytes("Recharge"));
    expect(rec.eventData[0]).to.equal(scaleUniAddr(user.address));
  });

  it("records WithdrawalDone for immediate withdrawal, nonce is idempotent", async function () {
    await (await token.connect(user).recharge({ value: ethers.parseEther("1") })).wait();

    const wdNonce = 77n;
    await (await token.connect(subnet).withdraw(user.address, 2000n, wdNonce)).wait();

    expect(await token.getLatestNonce()).to.equal(2n);
    const rec = await getEvent(token, 2);
    expect(rec.eventType).to.equal(toBytes("WithdrawalDone"));
    expect(rec.eventData.length).to.equal(1);
    expect(rec.eventData[0]).to.equal(scaleU64(wdNonce));
    expect(rec.eventData[0]).to.equal("0x4d00000000000000");

    // 重复提交同一 nonce：幂等，不产生新事件
    await (await token.connect(subnet).withdraw(user.address, 2000n, wdNonce)).wait();
    expect(await token.getLatestNonce()).to.equal(2n);
  });

  it("does not record an event for pending withdrawal; claim records WithdrawalDone", async function () {
    await (await token.connect(user).recharge({ value: ethers.parseEther("1") })).wait();

    const wdNonce = 78n;
    const afterRecharge = await token.getLatestNonce();
    expect(afterRecharge).to.equal(1n);

    // 8000 积分 = 2 ETH > 合约余额 → PENDING，不产生 relay 事件
    await (await token.connect(subnet).withdraw(user.address, 8000n, wdNonce)).wait();
    expect(await token.getLatestNonce()).to.equal(afterRecharge);
    expect((await token.getPendingWithdrawal(wdNonce)).status).to.equal(0n); // PENDING

    // 注入流动性（普通转账会走 receive → 记录一条 Recharge）
    await (await owner.sendTransaction({ to: await token.getAddress(), value: ethers.parseEther("2") })).wait();
    const beforeClaim = await token.getLatestNonce();

    await (await token.connect(user).claimWithdrawal(wdNonce)).wait();

    expect(await token.getLatestNonce()).to.equal(beforeClaim + 1n);
    const rec = await getEvent(token, beforeClaim + 1n);
    expect(rec.eventType).to.equal(toBytes("WithdrawalDone"));
    expect(rec.eventData[0]).to.equal(scaleU64(wdNonce));

    // 重复 claim 必须失败（状态已变更）
    await expect(token.connect(user).claimWithdrawal(wdNonce)).to.be.reverted;
  });

  it("records WithdrawalCancelled", async function () {
    await (await token.connect(user).recharge({ value: ethers.parseEther("1") })).wait();

    const wdNonce = 79n;
    await (await token.connect(subnet).withdraw(user.address, 100000n, wdNonce)).wait();
    expect(await token.getLatestNonce()).to.equal(1n);

    await (await token.connect(user).cancelWithdrawal(wdNonce)).wait();

    expect(await token.getLatestNonce()).to.equal(2n);
    const rec = await getEvent(token, 2);
    expect(rec.eventType).to.equal(toBytes("WithdrawalCancelled"));
    expect(rec.eventData[0]).to.equal(scaleU64(wdNonce));
  });

  it("getEvents ranges match get_latest_nonce / get_event semantics", async function () {
    await (await token.connect(user).recharge({ value: ethers.parseEther("1") })).wait();
    await (await token.connect(subnet).withdraw(user.address, 1000n, 1n)).wait();
    await (await token.connect(subnet).withdraw(user.address, 1000n, 2n)).wait();

    expect(await token.getLatestNonce()).to.equal(3n);

    // to == 0 → 查询到最新
    expect((await token.getEvents(0, 0)).length).to.equal(3);
    // 单点
    const one = await token.getEvents(2, 2);
    expect(one.length).to.equal(1);
    expect(one[0].eventType).to.equal(toBytes("WithdrawalDone"));
    // 越界 from > latest → 空
    expect((await token.getEvents(4, 0)).length).to.equal(0);
    // to 超出最新 → 截断到最新
    expect((await token.getEvents(1, 100)).length).to.equal(3);
  });

  it("only subnet can create withdrawal events", async function () {
    await (await token.connect(user).recharge({ value: ethers.parseEther("1") })).wait();
    await expect(token.connect(other).withdraw(user.address, 1000n, 9n)).to.be.revertedWith(
      "only subnet contract"
    );
    expect(await token.getLatestNonce()).to.equal(1n);
  });
});

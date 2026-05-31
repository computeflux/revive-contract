// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.20;

/**
 * @title List2D
 * @notice 模拟 revives 中 List2D 行为的通用存储库。
 *         每个列表有独立的自增 k2，删除不影响其他条目的 k2，分页基于 k2 降序。
 */
library List2D {
    struct List {
        uint64 nextK2;
        uint64 count;
        mapping(uint64 => bytes) values;
        mapping(uint64 => bool) exists;
    }

    function insert(List storage self, bytes memory value) internal returns (uint64) {
        uint64 k2 = self.nextK2;
        self.nextK2++;
        self.values[k2] = value;
        self.exists[k2] = true;
        self.count++;
        return k2;
    }

    function get(List storage self, uint64 k2) internal view returns (bytes memory) {
        require(self.exists[k2], "List2D: not found");
        return self.values[k2];
    }

    function update(List storage self, uint64 k2, bytes memory value) internal {
        require(self.exists[k2], "List2D: not found");
        self.values[k2] = value;
    }

    function clear(List storage self, uint64 k2) internal {
        require(self.exists[k2], "List2D: not found");
        delete self.values[k2];
        self.exists[k2] = false;
        self.count--;
    }

    function len(List storage self) internal view returns (uint64) {
        return self.count;
    }

    /**
     * @notice 按 k2 降序分页查询。
     * @param start 起始 k2（不包含，从此值向下遍历）。若 >= nextK2 则从 nextK2-1 开始。
     * @param size  最大返回数量。
     * @return k2s    返回条目的 k2 列表。
     * @return values 返回条目的 bytes 值列表。
     */
    function descList(List storage self, uint64 start, uint64 size) internal view returns (uint64[] memory k2s, bytes[] memory values) {
        if (self.count == 0 || size == 0) {
            return (new uint64[](0), new bytes[](0));
        }
        uint64 cur = start;
        if (cur >= self.nextK2) {
            if (self.nextK2 == 0) {
                return (new uint64[](0), new bytes[](0));
            }
            cur = self.nextK2 - 1;
        }
        uint64 allocSize = size > self.count ? self.count : size;
        uint64[] memory tempK2s = new uint64[](allocSize);
        bytes[] memory tempValues = new bytes[](allocSize);
        uint64 idx = 0;
        while (true) {
            if (self.exists[cur]) {
                tempK2s[idx] = cur;
                tempValues[idx] = self.values[cur];
                idx++;
            }
            if (cur == 0 || idx >= size) break;
            cur--;
        }
        if (idx < size) {
            uint64[] memory outK2s = new uint64[](idx);
            bytes[] memory outValues = new bytes[](idx);
            for (uint64 i = 0; i < idx; i++) {
                outK2s[i] = tempK2s[i];
                outValues[i] = tempValues[i];
            }
            return (outK2s, outValues);
        }
        return (tempK2s, tempValues);
    }

    /**
     * @notice 按 k2 升序分页查询（用于 pod 等需要正向遍历的场景）。
     */
    function ascList(List storage self, uint64 start, uint64 size) internal view returns (uint64[] memory k2s, bytes[] memory values) {
        if (self.count == 0 || size == 0) {
            return (new uint64[](0), new bytes[](0));
        }
        uint64 cur = start;
        if (cur >= self.nextK2) {
            return (new uint64[](0), new bytes[](0));
        }
        uint64 allocSize = size > self.count ? self.count : size;
        uint64[] memory tempK2s = new uint64[](allocSize);
        bytes[] memory tempValues = new bytes[](allocSize);
        uint64 idx = 0;
        while (true) {
            if (self.exists[cur]) {
                tempK2s[idx] = cur;
                tempValues[idx] = self.values[cur];
                idx++;
            }
            cur++;
            if (cur >= self.nextK2 || idx >= size) break;
        }
        if (idx < size) {
            uint64[] memory outK2s = new uint64[](idx);
            bytes[] memory outValues = new bytes[](idx);
            for (uint64 i = 0; i < idx; i++) {
                outK2s[i] = tempK2s[i];
                outValues[i] = tempValues[i];
            }
            return (outK2s, outValues);
        }
        return (tempK2s, tempValues);
    }

    /**
     * @notice 列出所有存在的 k2（降序，无数量限制，仅用于小列表）。
     */
    function listAllK2sDesc(List storage self) internal view returns (uint64[] memory) {
        if (self.count == 0) return new uint64[](0);
        uint64[] memory temp = new uint64[](self.count);
        uint64 idx = 0;
        for (uint64 k = self.nextK2; k > 0; ) {
            k--;
            if (self.exists[k]) {
                temp[idx] = k;
                idx++;
            }
        }
        return temp;
    }
}

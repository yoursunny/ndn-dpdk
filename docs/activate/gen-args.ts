import type { ActivateGenArgs, EalConfig } from "@usnistgov/ndn-dpdk";
import stdout from "stdout-stream";

const eal: EalConfig = {
  pciDevices: [
    "0000:04:00.0",
  ],
};

const args: ActivateGenArgs = {
  eal,
  mempool: {
    DIRECT: { capacity: 1048575, dataroom: 9128 },
    INDIRECT: { capacity: 1048575 },
  },
};

stdout.write(JSON.stringify(args));

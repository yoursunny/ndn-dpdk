#include "flowdef.h"
#include "../core/base16.h"
#include "../core/logger.h"
#include "hdr-impl.h"

N_LOG_INIT(EthFlowDef);

#define MASK(field) memset(&(field), 0xFF, sizeof(field))

__attribute__((nonnull)) static inline void
AppendItem(EthFlowDef* flow, size_t* i, enum rte_flow_item_type typ, const void* spec,
           const void* mask, __rte_unused size_t size) {
  flow->pattern[*i].type = typ;
  flow->pattern[*i].spec = spec;
  flow->pattern[*i].mask = mask;
  ++(*i);
  NDNDPDK_ASSERT(*i < RTE_DIM(flow->pattern));
}

__attribute__((nonnull)) static inline void
PrepareVxlan(const EthLocator* loc, struct rte_vxlan_hdr* vxlanSpec,
             struct rte_vxlan_hdr* vxlanMask, struct rte_ether_hdr* innerEthSpec,
             struct rte_ether_hdr* innerEthMask) {
  MASK(vxlanMask->vni);
  PutVxlanHdr((uint8_t*)vxlanSpec, loc->vxlan);

  MASK(innerEthMask->dst_addr);
  MASK(innerEthMask->src_addr);
  MASK(innerEthMask->ether_type);
  PutEtherHdr((uint8_t*)innerEthSpec, loc->innerRemote, loc->innerLocal, 0, EtherTypeNDN);
}

__attribute__((nonnull)) static inline void
GeneratePattern(EthFlowDef* flow, size_t specLen[], const EthLocator* loc, EthLocatorClass c,
                EthFlowFlags flowFlags) {
  size_t i = 0;
#define APPEND(typ, field)                                                                         \
  AppendItem(flow, &i, RTE_FLOW_ITEM_TYPE_##typ, &flow->field##Spec, &flow->field##Mask,           \
             (specLen[i] = sizeof(flow->field##Spec)))

  if (c.passthru) {
    if (flowFlags & EthFlowFlagsPassthruArp) {
      MASK(flow->ethMask.hdr.ether_type);
      flow->ethSpec.hdr.ether_type = rte_cpu_to_be_16(RTE_ETHER_TYPE_ARP);
      APPEND(ETH, eth);
    } else {
      flow->attr.priority = 1;
    }
    return;
  }

  MASK(flow->ethMask.hdr.dst_addr);
  PutEtherHdr((uint8_t*)(&flow->ethSpec.hdr), loc->remote, loc->local, loc->vlan, c.etherType);
  if (c.multicast) {
    flow->ethSpec.hdr.dst_addr = loc->remote;
  } else {
    MASK(flow->ethMask.hdr.src_addr);
  }
  APPEND(ETH, eth);

  if (loc->vlan != 0) {
    flow->vlanMask.hdr.vlan_tci = rte_cpu_to_be_16(0x0FFF); // don't mask PCP & DEI bits
    PutVlanHdr((uint8_t*)(&flow->vlanSpec.hdr), loc->vlan, c.etherType);
    APPEND(VLAN, vlan);
  }

  if (!c.udp) {
    // don't mask EtherType for IPv4/IPv6 - rejected by i40e driver
    MASK(flow->ethMask.hdr.ether_type);
    MASK(flow->vlanMask.hdr.eth_proto);
    return;
  }
  // several drivers do not support ETH+IP combination, so clear ETH spec
  flow->pattern[0].spec = NULL;

  if (c.v4) {
    MASK(flow->ip4Mask.hdr.src_addr);
    MASK(flow->ip4Mask.hdr.dst_addr);
    PutIpv4Hdr((uint8_t*)(&flow->ip4Spec.hdr), loc->remoteIP, loc->localIP);
    APPEND(IPV4, ip4);
  } else {
    MASK(flow->ip6Mask.hdr.src_addr);
    MASK(flow->ip6Mask.hdr.dst_addr);
    PutIpv6Hdr((uint8_t*)(&flow->ip6Spec.hdr), loc->remoteIP, loc->localIP);
    APPEND(IPV6, ip6);
  }

  if (c.tunnel != 'V') { // VXLAN packet can have any UDP source port
    MASK(flow->udpMask.hdr.src_port);
  }
  MASK(flow->udpMask.hdr.dst_port);
  PutUdpHdr((uint8_t*)(&flow->udpSpec.hdr), loc->remoteUDP, loc->localUDP);
  APPEND(UDP, udp);

  switch (c.tunnel) {
    case 'V': {
      if (flowFlags & EthFlowFlagsVxRaw) {
        struct {
          struct rte_vxlan_hdr vxlan;
          struct rte_ether_hdr eth;
        } __rte_aligned(2) spec = {0}, mask = {0};
        PrepareVxlan(loc, &spec.vxlan, &mask.vxlan, &spec.eth, &mask.eth);
        static_assert(sizeof(spec) == 4 + 16 + 2, "");
        rte_mov16(flow->rawSpecBuf, RTE_PTR_ADD(&spec, 4));
        rte_mov16(flow->rawMaskBuf, RTE_PTR_ADD(&mask, 4));

        flow->rawSpec.relative = 1;
        flow->rawSpec.offset = 4;
        flow->rawSpec.length = 16;
        flow->rawSpec.pattern = flow->rawSpecBuf;
        flow->rawMask = rte_flow_item_raw_mask;
        flow->rawMask.pattern = flow->rawMaskBuf;
        APPEND(RAW, raw);
      } else {
        PrepareVxlan(loc, &flow->vxlanSpec.hdr, &flow->vxlanMask.hdr, &flow->innerEthSpec.hdr,
                     &flow->innerEthMask.hdr);
        APPEND(VXLAN, vxlan);
        APPEND(ETH, innerEth);
      }
      break;
    }
    case 'G': {
      EthGtpHdr spec = {0};
      PutGtpHdr((uint8_t*)&spec, true, loc->ulTEID, loc->ulQFI);
      rte_memcpy(&flow->gtpSpec.hdr, &spec.hdr, sizeof(flow->gtpSpec.hdr));
      MASK(flow->gtpMask.hdr.teid);

      if (flowFlags & EthFlowFlagsGtp) {
        APPEND(GTP, gtp);
      } else {
        APPEND(GTPU, gtp);
      }
      break;
    }
  }

#undef APPEND
}

__attribute__((nonnull)) static inline void
CleanPattern(EthFlowDef* flow, size_t specLen[]) {
  for (int i = 0;; ++i) {
    size_t itemLen = specLen[i];
    struct rte_flow_item* item = &flow->pattern[i];
    switch (item->type) {
      case RTE_FLOW_ITEM_TYPE_END:
        return;
      case RTE_FLOW_ITEM_TYPE_RAW:
        itemLen = offsetof(struct rte_flow_item_raw, pattern);
        break;
      default:
        break;
    }

    if (item->spec == NULL) {
      item->mask = NULL;
      continue;
    }

    uint8_t* spec = (uint8_t*)item->spec;
    const uint8_t* mask = (const uint8_t*)item->mask;
    for (size_t j = 0; j < itemLen; ++j) {
      spec[j] &= mask[j];
    }
  }
}

__attribute__((nonnull)) static inline void
AppendAction(EthFlowDef* flow, size_t* i, enum rte_flow_action_type typ, const void* conf) {
  flow->actions[*i].type = typ;
  flow->actions[*i].conf = conf;
  ++(*i);
  NDNDPDK_ASSERT(*i < RTE_DIM(flow->pattern));
}

__attribute__((nonnull)) static inline EthFlowFlags
GenerateActions(EthFlowDef* flow, EthLocatorClass c, EthFlowFlags flowFlags, uint32_t mark,
                const uint16_t queues[], int nQueues) {
  EthFlowFlags addFlowFlags = 0;

  size_t i = 0;
#define APPEND(typ, field) AppendAction(flow, &i, RTE_FLOW_ACTION_TYPE_##typ, &flow->field##Act)

  NDNDPDK_ASSERT(nQueues >= 1);
  if (nQueues == 1) {
    flow->queueAct.index = queues[0];
    APPEND(QUEUE, queue);
  } else {
    flow->rssAct.level = 1;
    flow->rssAct.types = c.v4 ? RTE_ETH_RSS_NONFRAG_IPV4_UDP : RTE_ETH_RSS_NONFRAG_IPV6_UDP,
    flow->rssAct.queue_num = RTE_MIN((uint32_t)nQueues, RTE_DIM(flow->rssQueues));
    rte_memcpy(flow->rssQueues, queues, sizeof(queues[0]) * flow->rssAct.queue_num);
    flow->rssAct.queue = flow->rssQueues;
    APPEND(RSS, rss);
  }

  if (!(((flowFlags & EthFlowFlagsRssUnmarked) && nQueues > 1) ||
        ((flowFlags & EthFlowFlagsEtherUnmarked) && !c.udp))) {
    flow->markAct.id = mark;
    APPEND(MARK, mark);
    addFlowFlags |= EthFlowFlagsMarked;
  }

#undef APPEND
  return addFlowFlags;
}

__attribute__((nonnull)) static inline void
PrintDef(const EthFlowDef* flow, size_t specLen[]) {
  for (int i = 0;; ++i) {
    const struct rte_flow_item* item = &flow->pattern[i];
    char b16Spec[Base16_BufferSize(64)] = {'-', 0};
    char b16Mask[Base16_BufferSize(64)] = {'-', 0};
    if (item->spec != NULL && item->mask != NULL) {
      NDNDPDK_ASSERT(specLen[i] <= 64);
      Base16_Encode(b16Spec, sizeof(b16Spec), item->spec, specLen[i]);
      Base16_Encode(b16Mask, sizeof(b16Mask), item->mask, specLen[i]);
    }
    const char* typeName = NULL;
    if (rte_flow_conv(RTE_FLOW_CONV_OP_ITEM_NAME_PTR, &typeName, sizeof(&typeName),
                      (const void*)(uintptr_t)item->type, NULL) <= 0) {
      typeName = "-";
    }
    N_LOGD("^ pattern index=%d type=%d type-name=%s spec=%s mask=%s", i, (int)item->type, typeName,
           b16Spec, b16Mask);
    if (item->type == RTE_FLOW_ITEM_TYPE_END) {
      break;
    }
  }

  for (int i = 0;; ++i) {
    const struct rte_flow_action* action = &flow->actions[i];
    const char* typeName = NULL;
    if (rte_flow_conv(RTE_FLOW_CONV_OP_ACTION_NAME_PTR, &typeName, sizeof(&typeName),
                      (const void*)(uintptr_t)action->type, NULL) <= 0) {
      typeName = "-";
    }
    N_LOGD("^ action index=%d type=%d type-name=%s", i, (int)action->type, typeName);
    if (action->type == RTE_FLOW_ACTION_TYPE_END) {
      break;
    }
  }
}

void
EthFlowDef_Prepare(EthFlowDef* flow, const EthLocator* loc, EthFlowFlags* flowFlags, uint32_t mark,
                   const uint16_t queues[], int nQueues) {
  EthLocatorClass c = EthLocator_Classify(loc);
  *flow = (const EthFlowDef){0};
  flow->attr.ingress = 1;

  size_t specLen[RTE_DIM(flow->pattern)];
  GeneratePattern(flow, specLen, loc, c, *flowFlags);
  CleanPattern(flow, specLen);
  *flowFlags |= GenerateActions(flow, c, *flowFlags, mark, queues, nQueues);

  if (N_LOG_ENABLED(DEBUG)) {
    N_LOGD("Prepare loc=%p flow-flags=%08" PRIx32, loc, *flowFlags);
    N_LOGD("^ attr group=%" PRIu32 " priority=%" PRIu32, flow->attr.group, flow->attr.priority);
    PrintDef(flow, specLen);
  }
}

void
EthFlowDef_UpdateError(const EthFlowDef* flow, struct rte_flow_error* error) {
  ptrdiff_t offset = RTE_PTR_DIFF(error->cause, flow);
  if (offset >= 0 && (size_t)offset < sizeof(*flow)) {
    error->cause = (const void*)offset;
  }
}
